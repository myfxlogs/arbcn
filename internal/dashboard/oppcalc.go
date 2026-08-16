// 机会实算卡（D-046）：把业主不在场时的「算账」写成确定性公式——瞬时费率 / 30 日均值 /
// 保本持续天数 / 扣摩擦净年化 / 三档判定 / 中文模板叙述（D-043 模板化叙述，非 LLM 生成）。
//
// 纯函数无 I/O（对抗测试可直接喂数值）；只读证据表面（practices #20）：实算卡只说「这笔账
// 划不划算」，执行门禁仍由现有规则引擎 + 门禁把关——D-016 15%/20% 激活线、SignalToOrder
// 的 MinSpread/CarryMinSpread、carry 白名单一律不动。
//
// 摩擦口径（业主核实）：两交易所均为普通主户费率（非 VIP、无抵扣）——现货 taker 0.1%×2 +
// 永续 taker 0.05%×2 = 0.3%（双开双平）。与 sim/report.go 的 defaultFrictionRate=0.2% 口径
// 独立：那是周报的通用摩擦模型，本卡是 funding_hedge 双 taker 专项。后续启用 BNB 抵扣 /
// 升 VIP 档 → 改 env（ARBCN_OPP_FRICTION_FUNDING），不改代码。
package dashboard

import (
	"fmt"
	"math"
	"time"

	"arbcn/internal/fact"
	"arbcn/internal/store"
)

// 实算卡常量（单点定义；与 internal/rule/defaults.go funding 窗口档、D-021 稳定币基档
// 同步点，改动须两处核对）。
const (
	// defaultOppFrictionPct funding_hedge 双开双平 taker 摩擦 %（现货 0.1%×2 + 永续
	// 0.05%×2 = 0.3%）。普通主户费率已由业主核实；配置优先（Service.OppFrictionFunding）。
	defaultOppFrictionPct = 0.3
	// stableBasePct 判定基准：稳定币基档机会成本 %（D-021）。净年化 > 此值才「可抓」。
	stableBasePct = 4.5
	// oppSpikeFactor 尖峰判定倍率：瞬时 > 2×30 日均值 = 尖峰陷阱（与 insights.go
	// defiAnomalies 中位数×2 因子同口径）。
	oppSpikeFactor = 2.0
	// oppAvgDays 净年化持有口径：30 日（与 avg_30d 窗口一致）。一次性摩擦的年化摊薄
	// = friction × 365/oppAvgDays。
	oppAvgDays = 30.0
	// oppAvgWindow 30 日均值回看窗口。
	oppAvgWindow = 30 * 24 * time.Hour
)

// 判定值域（Rating* 常量；前端映射中文徽标，见 web/src 实算卡区块）。
const (
	RatingGrab      = "grab"
	RatingBreakeven = "breakeven"
	RatingTrap      = "trap"
)

// OppCard 机会实算卡：确定性算账结果。Narrative 为中文模板叙述（D-043）。
// Avg30 样本不足 = NaN（前端显示「—」，判定退化为观望，宁缺毋滥）。
type OppCard struct {
	Kind          string // store.SimKindFundingHedge / SimKindCarryAsset / SimKindRepo
	Venue         string
	Symbol        string
	Inst          float64 // 瞬时年化 %（最新事实值）
	Avg30         float64 // 30 日均值 %（NaN = 样本不足）
	BreakEvenDays float64 // 保本持续天数（friction>0 且 Inst>0；否则 0 = 不适用）
	NetAnnualized float64 // 扣摩擦净年化 %（30 日持有口径；NaN = 均值样本不足）
	Friction      float64 // 摩擦假设 %
	Rating        string  // RatingGrab / RatingBreakeven / RatingTrap
	Narrative     string  // 中文模板叙述
}

// ratingFor 三档判定：坑（当前费率 ≤0 / 净年化 ≤0 / 瞬时 >2×均值尖峰）→ 打平 →
// 可抓（净年化 > 稳定币基档机会成本）。avg30 = NaN 时跳过尖峰判定（样本不足不判尖峰）。
//
// [对抗测试锚点] 删任一分支（全标/全不标）→ oppcalc_test.go 必红。结论对合理摩擦区间
// （0.1–0.5%）稳健：尖峰/净年化符号不受摩擦精细值影响。
func ratingFor(inst, avg30, net float64) string {
	if inst <= 0 || net <= 0 {
		return RatingTrap
	}
	if avg30 > 0 && inst > oppSpikeFactor*avg30 {
		return RatingTrap
	}
	if net > stableBasePct {
		return RatingGrab
	}
	return RatingBreakeven
}

// fundingCard 构建 funding_hedge 实算卡。friction 为配置摩擦 %。
//
// [对抗测试锚点] 删公式（BreakEvenDays=f×365/Inst、NetAnnualized=Avg30−f×365/30）→
// oppcalc_test.go 必红。
func fundingCard(venue, symbol string, inst, avg30, friction float64) OppCard {
	c := OppCard{
		Kind: store.SimKindFundingHedge, Venue: venue, Symbol: symbol,
		Inst: inst, Avg30: avg30, Friction: friction,
	}
	if math.IsNaN(avg30) {
		// 均值样本不足 → 净年化显式 NaN（勿留 0 值：ratingFor 会把 0 误读为「净 ≤0 坑」）。
		c.NetAnnualized = math.NaN()
	} else {
		// 30 日持有口径：一次性摩擦按持有期年化摊薄（f×365/30）。
		c.NetAnnualized = avg30 - friction*365/oppAvgDays
	}
	// 保本天数：瞬时费率持续多少天可覆盖摩擦（现货+永续双 taker）。
	if friction > 0 && inst > 0 {
		c.BreakEvenDays = friction * 365 / inst
	}
	c.Rating = ratingFor(inst, avg30, c.NetAnnualized)
	c.Narrative = fundingNarrative(c)
	return c
}

// carryCard 构建 carry_asset 实算卡（稳定币生息，持有生息无开平仓摩擦 → friction=0）。
func carryCard(venue, symbol string, inst, avg30 float64) OppCard {
	c := OppCard{
		Kind: store.SimKindCarryAsset, Venue: venue, Symbol: symbol,
		Inst: inst, Avg30: avg30, Friction: 0,
	}
	if math.IsNaN(avg30) {
		c.NetAnnualized = math.NaN() // 均值不足 → 显式 NaN（同上，防 0 值误判坑）
	} else {
		c.NetAnnualized = avg30 // 无方向摩擦，净年化 ≈ 均值
	}
	c.Rating = ratingFor(inst, avg30, c.NetAnnualized)
	c.Narrative = carryNarrative(c)
	return c
}

// repoCard 构建 repo 实算卡（逆回购，当日时点利率即机会本身；无方向摩擦）。
// avg30 仅供对照展示，不参与尖峰判定（季末时点冲高就是机会，不是陷阱）。
func repoCard(venue, symbol string, inst, avg30 float64) OppCard {
	c := OppCard{
		Kind: store.SimKindRepo, Venue: venue, Symbol: symbol,
		Inst: inst, Avg30: avg30, Friction: 0,
	}
	c.NetAnnualized = inst // 时点利率口径
	c.Rating = ratingFor(inst, math.NaN(), c.NetAnnualized)
	c.Narrative = repoNarrative(c)
	return c
}

// fundingNarrative funding_hedge 中文模板叙述（D-043：模板化，非 LLM 生成）。
func fundingNarrative(c OppCard) string {
	switch {
	case c.Inst <= 0:
		return fmt.Sprintf("当前资金费率 %.2f%% ≤ 0，现货+永续对冲开仓即反向付费——判定：不抓。", c.Inst)
	case math.IsNaN(c.Avg30):
		return fmt.Sprintf("瞬时费率 %.2f%%，按 %.2f%% 摩擦（现货+永续双 taker）需持续约 %.0f 天保本；30 日历史样本不足——判定：观望，待数据累积。",
			c.Inst, c.Friction, c.BreakEvenDays)
	}
	f := c.Friction
	switch c.Rating {
	case RatingTrap:
		if c.Inst > oppSpikeFactor*c.Avg30 {
			return fmt.Sprintf("瞬时费率 %.2f%% 为 30 日均值 %.2f%% 的 %.1f 倍尖峰，按 %.2f%% 摩擦（现货+永续双 taker）需持续约 %.0f 天才保本；30 日持有扣摩擦净年化 %.2f%%（机会成本基准 %.1f%%）——判定：尖峰陷阱，不抓。",
				c.Inst, c.Avg30, c.Inst/c.Avg30, f, c.BreakEvenDays, c.NetAnnualized, stableBasePct)
		}
		return fmt.Sprintf("30 日均值 %.2f%% 扣 %.2f%% 摩擦（现货+永续双 taker）后，30 日持有净年化 %.2f%% ≤ 0，费率不足以覆盖交易成本——判定：不抓。",
			c.Avg30, f, c.NetAnnualized)
	case RatingGrab:
		return fmt.Sprintf("30 日均值 %.2f%% 扣 %.2f%% 摩擦后净年化 %.2f%% > 稳定币基档 %.1f%%（保本约 %.0f 天）——判定：可抓，执行前过 D-016 门禁。",
			c.Avg30, f, c.NetAnnualized, stableBasePct, c.BreakEvenDays)
	default:
		return fmt.Sprintf("30 日均值 %.2f%% 扣 %.2f%% 摩擦后净年化 %.2f%%，介于 0 与稳定币基档 %.1f%% 之间——判定：打平/观望，机会成本不占优。",
			c.Avg30, f, c.NetAnnualized, stableBasePct)
	}
}

// carryNarrative carry_asset 中文模板叙述。
func carryNarrative(c OppCard) string {
	if math.IsNaN(c.Avg30) {
		return fmt.Sprintf("瞬时生息年化 %.2f%%，30 日样本不足——判定：观望，待数据累积。", c.Inst)
	}
	switch c.Rating {
	case RatingTrap:
		if c.Inst > oppSpikeFactor*c.Avg30 {
			return fmt.Sprintf("瞬时利率 %.2f%% 为 30 日均值 %.2f%% 的 %.1f 倍尖峰，均值口径净年化 %.2f%%（机会成本基准 %.1f%%）——判定：尖峰陷阱，等回落再评估。",
				c.Inst, c.Avg30, c.Inst/c.Avg30, c.NetAnnualized, stableBasePct)
		}
		return fmt.Sprintf("30 日均值 %.2f%% 净年化 ≤ 0——判定：不抓。", c.Avg30)
	case RatingGrab:
		return fmt.Sprintf("30 日均值 %.2f%% 净年化 > 稳定币基档 %.1f%%——判定：可抓（注意赎回/脱锚 caveat，先核实资产结构）。",
			c.Avg30, stableBasePct)
	default:
		return fmt.Sprintf("30 日均值 %.2f%% 净年化介于 0 与稳定币基档 %.1f%% 之间——判定：打平/观望。",
			c.Avg30, stableBasePct)
	}
}

// repoNarrative repo 中文模板叙述（标注时点属性）。
func repoNarrative(c OppCard) string {
	switch c.Rating {
	case RatingTrap:
		return fmt.Sprintf("当日逆回购年化 %.2f%%（时点利率）≤ 0——判定：无意义，不抓。", c.Inst)
	case RatingGrab:
		return fmt.Sprintf("当日逆回购年化 %.2f%%（时点利率）> 稳定币基档 %.1f%%——判定：时点可抓（现金管理档时点操作）。",
			c.Inst, stableBasePct)
	default:
		return fmt.Sprintf("当日逆回购年化 %.2f%%（时点利率），低于稳定币基档 %.1f%%——判定：打平/观望，现金留基档更优。",
			c.Inst, stableBasePct)
	}
}

// meanFacts 窗口事实均值（跳过 NaN/Inf 样本，practices #7）；空序列 → NaN。
// 值按事实口径原样（funding/defi_rate 均为 pct_annualized 百分点点数）。
func meanFacts(facts []fact.Fact) float64 {
	sum, n := 0.0, 0
	for _, f := range facts {
		if math.IsNaN(f.Value) || math.IsInf(f.Value, 0) {
			continue
		}
		sum += f.Value
		n++
	}
	if n == 0 {
		return math.NaN()
	}
	return sum / float64(n)
}

// groupByVenueSymbol 按 venue\x00symbol 分组事实（QueryFacts 已按 ts 升序，保留组内序）。
func groupByVenueSymbol(facts []fact.Fact) map[string][]fact.Fact {
	out := map[string][]fact.Fact{}
	for _, f := range facts {
		key := f.Venue + "\x00" + f.Symbol
		out[key] = append(out[key], f)
	}
	return out
}
