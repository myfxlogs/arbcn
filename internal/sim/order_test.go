package sim

import (
	"math"
	"slices"
	"testing"
	"time"

	"arbcn/internal/store"
)

// t0 是测试基准时钟。
var t0 = time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)

// validSignal 返回一条全门禁通过的资金费率对冲信号（对照 §4 六道门禁）。
func validSignal() Signal {
	return Signal{
		RuleName: "funding_warn", Kind: store.SimKindFundingHedge, Symbol: "BTC",
		Venue: "sim_local", RefPrice: 60000, ExpectedSpread: 10, FundingAnn: 10,
		SpotPrice: 60000, PerpPrice: 60050, Notional: 10000, CarryWhite: false,
		DayNotional: 0, Ts: t0,
	}
}

func hasFlag(o store.SimOrder, f string) bool { return slices.Contains(o.RiskFlags, f) }

// TestSignalToOrderValidHedge：有效资金费率对冲 → suggested、无 risk_flags、
// side=hedge、expected_spread 取快照值。
func TestSignalToOrderValidHedge(t *testing.T) {
	o := SignalToOrder(validSignal(), DefaultConfig())
	if o.Status != store.SimStatusSuggested {
		t.Fatalf("status = %q, want suggested", o.Status)
	}
	if len(o.RiskFlags) != 0 {
		t.Fatalf("risk_flags = %v, want empty", o.RiskFlags)
	}
	if o.Side != store.SimSideHedge || o.Symbol != "BTC" || o.SrcRule != "funding_warn" {
		t.Fatalf("order = %+v", o)
	}
	if o.ExpectedSpread != 10 || o.Qty != 10000 || o.RefPrice != 60000 {
		t.Fatalf("spread/qty/ref = %v/%v/%v, want 10/10000/60000", o.ExpectedSpread, o.Qty, o.RefPrice)
	}
}

// TestSignalToOrderRejectsUnhedged：funding_hedge 缺腿（现货价格缺失）→ UNHEDGED 拒单。
// CarryWhite=true 隔离方向性门禁，保证只由"缺腿"分支触发（干净锚点）。
// [对抗测试锚点] 删除 SignalToOrder 中 `sig.Kind == ... && !hedged` 拒单分支 → 本测试必红。
func TestSignalToOrderRejectsUnhedged(t *testing.T) {
	sig := validSignal()
	sig.SpotPrice = 0 // 缺现货腿
	sig.CarryWhite = true
	o := SignalToOrder(sig, DefaultConfig())
	if o.Status != store.SimStatusRejected || !hasFlag(o, RiskUNHEDGED) {
		t.Fatalf("status/flags = %q/%v, want rejected/UNHEDGED", o.Status, o.RiskFlags)
	}
	if o.Note == "" {
		t.Fatalf("note = empty, want 拒单原因")
	}
}

// TestSignalToOrderRejectsDirectional：非白名单方向性（carry_asset 未白名单）→
// UNHEDGED（方向性敞口，不赌原则 D-019）拒单。
// [对抗测试锚点] 删除 `!hedged && !sig.CarryWhite` 拒单分支 → 本测试必红。
func TestSignalToOrderRejectsDirectional(t *testing.T) {
	sig := validSignal()
	sig.Kind = store.SimKindCarryAsset
	sig.CarryWhite = false // 未白名单
	o := SignalToOrder(sig, DefaultConfig())
	if o.Status != store.SimStatusRejected || !hasFlag(o, RiskUNHEDGED) {
		t.Fatalf("status/flags = %q/%v, want rejected/UNHEDGED", o.Status, o.RiskFlags)
	}
}

// TestSignalToOrderRejectsLowSpread：预期年化价差 < 5% → SPREAD_LOW 拒单。
// [对抗测试锚点] 删除 `spread < cfg.MinSpread` 拒单分支 → 本测试必红。
func TestSignalToOrderRejectsLowSpread(t *testing.T) {
	sig := validSignal()
	sig.ExpectedSpread = 3
	sig.FundingAnn = 0 // 禁用推断，显式价差 3% < 门槛 5%
	o := SignalToOrder(sig, DefaultConfig())
	if o.Status != store.SimStatusRejected || !hasFlag(o, RiskSpreadLow) {
		t.Fatalf("status/flags = %q/%v, want rejected/SPREAD_LOW", o.Status, o.RiskFlags)
	}
}

// TestSignalToOrderRejectsSizeOver：单笔名义 > 模拟资金 20%（100_000×0.20=20_000）
// → SIZE_OVER 拒单。
// [对抗测试锚点] 删除 `notional > cfg.Capital*cfg.MaxSizePct` 拒单分支 → 本测试必红。
func TestSignalToOrderRejectsSizeOver(t *testing.T) {
	sig := validSignal()
	sig.Notional = 25_000 // > 20_000
	o := SignalToOrder(sig, DefaultConfig())
	if o.Status != store.SimStatusRejected || !hasFlag(o, RiskSizeOver) {
		t.Fatalf("status/flags = %q/%v, want rejected/SIZE_OVER", o.Status, o.RiskFlags)
	}
}

// TestSignalToOrderRejectsDailyOver：当日累计 + 单笔 > 模拟资金 50%（50_000）→ DAILY_OVER。
// [对抗测试锚点] 删除 `sig.DayNotional+notional > cfg.Capital*cfg.MaxDailyPct` 分支 → 必红。
func TestSignalToOrderRejectsDailyOver(t *testing.T) {
	sig := validSignal()
	sig.DayNotional = 45_000 // +10_000 = 55_000 > 50_000
	o := SignalToOrder(sig, DefaultConfig())
	if o.Status != store.SimStatusRejected || !hasFlag(o, RiskDailyOver) {
		t.Fatalf("status/flags = %q/%v, want rejected/DAILY_OVER", o.Status, o.RiskFlags)
	}
}

// TestSignalToOrderRejectsCarryWhitelist：carry_asset 未白名单 → WHITELIST 拒单。
// [对抗测试锚点] 删除 `sig.Kind == ... && !sig.CarryWhite` 拒单分支 → 本测试必红。
func TestSignalToOrderRejectsCarryWhitelist(t *testing.T) {
	sig := validSignal()
	sig.Kind = store.SimKindCarryAsset
	sig.CarryWhite = false
	o := SignalToOrder(sig, DefaultConfig())
	if o.Status != store.SimStatusRejected || !hasFlag(o, RiskWhitelist) {
		t.Fatalf("status/flags = %q/%v, want rejected/WHITELIST", o.Status, o.RiskFlags)
	}
}

// TestSignalToOrderCarryWhitePasses：carry_asset 已白名单 + 价差达标 → suggested
// （白名单豁免"无对冲拒单"，§4 白名单行）。
func TestSignalToOrderCarryWhitePasses(t *testing.T) {
	sig := validSignal()
	sig.Kind = store.SimKindCarryAsset
	sig.CarryWhite = true
	o := SignalToOrder(sig, DefaultConfig())
	if o.Status != store.SimStatusSuggested || len(o.RiskFlags) != 0 {
		t.Fatalf("status/flags = %q/%v, want suggested/empty", o.Status, o.RiskFlags)
	}
	if o.Side != store.SimSideLong {
		t.Fatalf("side = %q, want long（carry 生息腿）", o.Side)
	}
}

// TestSignalToOrderRepoPasses：repo（现金等价）天然无方向敞口 → 不需对冲/白名单 → suggested。
func TestSignalToOrderRepoPasses(t *testing.T) {
	sig := validSignal()
	sig.Kind = store.SimKindRepo
	sig.CarryWhite = false
	o := SignalToOrder(sig, DefaultConfig())
	if o.Status != store.SimStatusSuggested || len(o.RiskFlags) != 0 {
		t.Fatalf("status/flags = %q/%v, want suggested/empty", o.Status, o.RiskFlags)
	}
	if o.Side != store.SimSideLong {
		t.Fatalf("side = %q, want long", o.Side)
	}
}

// TestSignalToOrderSpreadInference：expected_spread ≤ 0 且 funding 可用 → 由 FundingAnn 回填。
func TestSignalToOrderSpreadInference(t *testing.T) {
	sig := validSignal()
	sig.ExpectedSpread = 0
	sig.FundingAnn = 8
	o := SignalToOrder(sig, DefaultConfig())
	if o.ExpectedSpread != 8 {
		t.Fatalf("expected_spread = %v, want 8（funding 回填）", o.ExpectedSpread)
	}
}

// TestSignalToOrderDefaultNotional：Notional ≤ 0 → 按单笔上限默认（capital×20%）。
func TestSignalToOrderDefaultNotional(t *testing.T) {
	sig := validSignal()
	sig.Notional = 0
	o := SignalToOrder(sig, DefaultConfig())
	if o.Qty != 20_000 {
		t.Fatalf("qty = %v, want 20000（默认单笔上限）", o.Qty)
	}
}

// TestSignalToOrderNegativeNotionalFallback：Notional ≤ 0（未显式预算）→ 按单笔上限
// 兜底默认，qty > 0（纯函数不 panic；存储层 qty ≤ 0 拒绝在 pgstore 兜底）。
func TestSignalToOrderNegativeNotionalFallback(t *testing.T) {
	sig := validSignal()
	sig.Notional = -5
	o := SignalToOrder(sig, DefaultConfig())
	if o.Qty <= 0 {
		t.Fatalf("qty = %v, want > 0（负名义兜底）", o.Qty)
	}
}

// TestSignalToOrderRejectsNaN：[对抗测试锚点] Go 中 NaN 使 `<`/`>` 门禁恒 false——
// 若无有限性守卫，NaN 价差/名义会静默绕过六道门禁且 qty=NaN 污染日累计（M3-a 复审 M3）。
// 删 order.go 有限性守卫循环 → 本测试必红。
func TestSignalToOrderRejectsNaN(t *testing.T) {
	for name, mut := range map[string]func(*Signal){
		"expected_spread": func(s *Signal) { s.ExpectedSpread = math.NaN() },
		"notional":        func(s *Signal) { s.Notional = math.NaN() },
		"spot_price":      func(s *Signal) { s.SpotPrice = math.NaN() },
		"day_notional":    func(s *Signal) { s.DayNotional = math.NaN() },
	} {
		sig := validSignal()
		mut(&sig)
		o := SignalToOrder(sig, DefaultConfig())
		if o.Status != store.SimStatusRejected || !hasFlag(o, RiskInvalid) {
			t.Errorf("%s: status/flags = %q/%v, want rejected/INVALID_INPUT", name, o.Status, o.RiskFlags)
		}
	}
}

// TestSignalToOrderRejectsUnknownKind：[对抗测试锚点] 未知 kind 不得静默变单腿建仓
// （ConfirmAndFill default 分支会把任意未知 kind 当 funding 腿）——删 `o.Side == ""`
// 拒单分支 → 本测试必红（M3-a 复审 L1）。
func TestSignalToOrderRejectsUnknownKind(t *testing.T) {
	sig := validSignal()
	sig.Kind = "mystery"
	sig.CarryWhite = true // 隔离方向性门禁，保证只由未知 kind 分支触发
	o := SignalToOrder(sig, DefaultConfig())
	if o.Status != store.SimStatusRejected || !hasFlag(o, RiskInvalid) || o.Side != "" {
		t.Fatalf("status/flags/side = %q/%v/%q, want rejected/INVALID_INPUT/空", o.Status, o.RiskFlags, o.Side)
	}
}

// TestSignalToOrderNegativeSpreadNotMasked：[对抗测试锚点] 显式负价差（亏损信号）不得被
// FundingAnn 回填掩盖——原 `<= 0` 会让 -3 被 +8 覆盖后放行；改 `== 0` 后负价差 → SPREAD_LOW。
func TestSignalToOrderNegativeSpreadNotMasked(t *testing.T) {
	sig := validSignal()
	sig.ExpectedSpread = -3 // 显式亏损信号
	sig.FundingAnn = 8      // gross funding 为正，但显式价差是负的
	o := SignalToOrder(sig, DefaultConfig())
	if o.Status != store.SimStatusRejected || !hasFlag(o, RiskSpreadLow) {
		t.Fatalf("status/flags = %q/%v, want rejected/SPREAD_LOW（负价差不被掩盖）", o.Status, o.RiskFlags)
	}
	if o.ExpectedSpread != -3 {
		t.Fatalf("expected_spread = %v, want -3（保持显式值）", o.ExpectedSpread)
	}
}

// TestSignalToOrderRejectsNegativeDayNotional：[对抗测试锚点] 负日累计压低 DAILY_OVER
// 分母可绕过单日 50% 门禁——负值视为调用方 bug 拒单（M3-a 复审 L3）。
func TestSignalToOrderRejectsNegativeDayNotional(t *testing.T) {
	sig := validSignal()
	sig.DayNotional = -10_000
	o := SignalToOrder(sig, DefaultConfig())
	if o.Status != store.SimStatusRejected || !hasFlag(o, RiskInvalid) {
		t.Fatalf("status/flags = %q/%v, want rejected/INVALID_INPUT", o.Status, o.RiskFlags)
	}
}
