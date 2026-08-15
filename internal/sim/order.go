package sim

import (
	"fmt"
	"math"
	"slices"
	"strings"
	"time"

	"arbcn/internal/store"
)

// 风险门禁标记（04-m3-spec §1.1 risk_flags 值域；生成时落库）。
const (
	RiskUNHEDGED  = "UNHEDGED"  // 对冲缺腿 / 非白名单方向性敞口（不赌原则 D-019）
	RiskSpreadLow = "SPREAD_LOW" // 预期年化价差 < 门槛（默认 5%，摩擦覆盖）
	RiskSizeOver  = "SIZE_OVER"  // 单笔名义 > 模拟资金 20%
	RiskDailyOver = "DAILY_OVER" // 当日新增名义 > 模拟资金 50%
	RiskWhitelist = "WHITELIST"  // carry_asset 标的未在白名单（sUSDe/USDe 等）
	RiskInvalid   = "INVALID_INPUT" // 非有限值 / 未知 kind / 负日累计：门禁不可静默绕过的兜底
)

// Signal 是订单生成器的信号输入（04-m3-spec §3.1）：规则命中 + 机会面板快照。
// 由调用方（本地模拟盘驱动）从 facts 与 sim 状态组装；SignalToOrder 只读它，无 I/O。
type Signal struct {
	RuleName       string    // 触发规则名（src_rule，如 funding_warn / defi_large_tier_change）
	Kind           string    // 套利类型（store.SimKindFundingHedge / CarryAsset / Repo）
	Symbol         string    // 标的（如 BTC / ETH / USDT）
	Venue          string    // 模拟盘 venue（sim_local / binance_testnet / okx_demo）
	RefPrice       float64   // 生成时参考价（行情）
	ExpectedSpread float64   // 预期年化价差 %；≤0 且 FundingAnn>0 → 由 FundingAnn 回填
	FundingAnn     float64   // 年化资金费率 %（funding_* 命中；预期价差推断源）
	SpotPrice      float64   // 现货价（funding_hedge 现货腿；≤0 = 缺腿）
	PerpPrice      float64   // 永续价（funding_hedge 永续腿；≤0 = 缺腿）
	Notional       float64   // 建议名义（quote 币种，模拟 USD）；≤0 = 按单笔上限默认
	CarryWhite     bool      // carry_asset 白名单命中
	DayNotional    float64   // 当日已累计活跃订单名义（DAILY_OVER 数据面，调用方查询）
	Ts             time.Time // 生成时刻（零值 → 由调用方回填）
}

// SignalToOrder 信号 → 建议订单（04-m3-spec §3.1 核心转换器，纯函数、无 I/O）。
// 生成时执行六道风险门禁（§4），risk_flags 落库；任一未过 → status=rejected + note
// （拒单记录保留为负样本，§4"拒单不是失败"）。门禁数值来自 cfg（定稿默认）。
//
// [对抗测试锚点]（§11 锚点模式 + practices #4/#5）：
//   - 删"无对冲拒单"分支 → TestSignalToOrderRejectsUnhedged 必红
//   - 删"预期价差 < 阈值拒单" → TestSignalToOrderRejectsLowSpread 必红
//   - 删"单笔超限拒单" → TestSignalToOrderRejectsSizeOver 必红
//   - 删"日累计超限拒单" → TestSignalToOrderRejectsDailyOver 必红
//   - 删"carry 白名单拒单" → TestSignalToOrderRejectsCarryWhitelist 必红
func SignalToOrder(sig Signal, cfg Config) store.SimOrder {
	spread := sig.ExpectedSpread
	// L2：仅"未提供"（==0）才由 FundingAnn 回填；显式负值 = 坏信号，不得被覆盖后
	// 放行（原 `<= 0` 会让 -3 的亏损信号被 +8 的 gross funding 掩盖，门禁失效）。
	if spread == 0 && sig.FundingAnn > 0 {
		spread = sig.FundingAnn
	}
	notional := sig.Notional
	if notional <= 0 {
		notional = cfg.Capital * cfg.MaxSizePct
	}

	o := store.SimOrder{
		Ts: sig.Ts, SrcRule: sig.RuleName, Kind: sig.Kind, Venue: sig.Venue,
		Symbol: sig.Symbol, Qty: notional, RefPrice: sig.RefPrice,
		ExpectedSpread: spread, RiskFlags: []string{}, Status: store.SimStatusSuggested,
	}
	switch sig.Kind {
	case store.SimKindFundingHedge:
		o.Side = store.SimSideHedge
	case store.SimKindCarryAsset, store.SimKindRepo:
		o.Side = store.SimSideLong
	}

	var reasons []string
	reject := func(flag, reason string) {
		if !slices.Contains(o.RiskFlags, flag) {
			o.RiskFlags = append(o.RiskFlags, flag) // 去重：同门禁多原因只记一个标记
		}
		reasons = append(reasons, reason)
	}

	// 有限性守卫（M3-a 复审 M3）：Go 中 `NaN < x` / `NaN > x` 恒 false——NaN/±Inf 输入会
	// 静默绕过全部数值门禁，且 qty=NaN 落库后污染日累计 SUM（DAILY_OVER 永久失效）。
	// 任一非有限值 → 拒单，门禁不可被坏输入架空。
	for _, v := range []struct {
		name string
		val  float64
	}{
		{"expected_spread", sig.ExpectedSpread}, {"funding_ann", sig.FundingAnn},
		{"spot_price", sig.SpotPrice}, {"perp_price", sig.PerpPrice},
		{"ref_price", sig.RefPrice}, {"notional", sig.Notional},
		{"day_notional", sig.DayNotional},
	} {
		if math.IsNaN(v.val) || math.IsInf(v.val, 0) {
			reject(RiskInvalid, v.name+" 非有限值")
		}
	}
	// L3：负日累计压低 DAILY_OVER 分母 → 拒单（数据面 SUM 恒 ≥0，负值 = 调用方 bug）。
	if sig.DayNotional < 0 {
		reject(RiskInvalid, "day_notional 为负")
	}
	// L1：未知 kind 不留 side 空/静默单腿——ConfirmAndFill 的 default 分支会把任意未知
	// kind 当成单 funding 腿建仓（M3-a 复审 L1）。
	if o.Side == "" {
		reject(RiskInvalid, "未知套利 kind "+sig.Kind)
	}

	// 对冲腿完整（kind=funding_hedge 必须双腿齐备：现货+永续，方向对冲）。
	// repo = 现金等价（逆回购），天然无方向敞口，视作已对冲。
	hedged := false
	switch sig.Kind {
	case store.SimKindFundingHedge:
		hedged = sig.SpotPrice > 0 && sig.PerpPrice > 0
	case store.SimKindRepo:
		hedged = true
	}
	if sig.Kind == store.SimKindFundingHedge && !hedged {
		reject(RiskUNHEDGED, "funding_hedge 缺腿（现货/永续价格缺失）")
	}

	// 方向性非白名单拒单（不赌原则 D-019：无对冲的方向性建议拒单）。
	if !hedged && !sig.CarryWhite {
		reject(RiskUNHEDGED, "非白名单方向性敞口（不赌原则 D-019）")
	}

	// 预期年化价差 < 门槛（默认 5%，摩擦覆盖）。
	if spread < cfg.MinSpread {
		reject(RiskSpreadLow, fmt.Sprintf("预期年化价差 %.2f%% < 门槛 %.2f%%", spread, cfg.MinSpread))
	}

	// 单笔名义 > 模拟资金 20%。
	if notional > cfg.Capital*cfg.MaxSizePct {
		reject(RiskSizeOver, fmt.Sprintf("单笔名义 %.0f > 模拟资金 %.2f%%", notional, cfg.Capital*cfg.MaxSizePct))
	}

	// 当日累计新增名义 > 模拟资金 50%。
	if sig.DayNotional+notional > cfg.Capital*cfg.MaxDailyPct {
		reject(RiskDailyOver, fmt.Sprintf("日累计名义 %.0f > 模拟资金 %.2f%%", sig.DayNotional+notional, cfg.Capital*cfg.MaxDailyPct))
	}

	// carry_asset 标的须在显式白名单（sUSDe/USDe 等生息资产）。
	if sig.Kind == store.SimKindCarryAsset && !sig.CarryWhite {
		reject(RiskWhitelist, "carry_asset 标的未在白名单")
	}

	if len(o.RiskFlags) > 0 {
		o.Status = store.SimStatusRejected
		o.Note = strings.Join(reasons, "; ")
	}
	return o
}
