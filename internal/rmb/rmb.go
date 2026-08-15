// Package rmb：RMB 折算层（docs/design/02-monitor-architecture.md §8，M2-b §4）。
// 展示/查询层折算：USD 计价事实 × 当日 USDCNH → RMB 净收益视角；原始事实不污染。
// 纯函数 + 无存储依赖（存储查询由调用方 dashboard 服务完成），便于对抗测试。
//
// 口径（D-023 例：稳定币 4.5–6% 折算人民币后，年化升值 3% 情景净 1.5–3%）：
//
//	RMB 净收益 ≈ USD 收益率 − 年化人民币升值率
//
// 刻度约定：收益与升值都按**百分点点数**（6.0 = 6%）计算，避免单元混用
// （追溯复审 R6#1 裁定：AnnualizedRMBAppreciation 返回点数，与 Value 同刻度）。
// 年化人民币升值率由 fx 事实序列（ts 升序）在回看窗口内计算；序列不足 2 点 →
// 趋势不可用（按 0 处理，只回填当日汇率）。汇率缺失（fx 为 nil）→ 回退 USD 原值
// + 汇率不可用标记，不崩溃、不静默用错值（03-m2-spec §4）。
package rmb

import (
	"time"

	"arbcn/internal/fact"
)

// CoveredKinds 需 RMB 折算的 kind（非 RMB 计价者，03-m2-spec §4 覆盖项）。
var CoveredKinds = map[string]bool{
	fact.KindFunding:     true,
	fact.KindDefiRate:    true,
	fact.KindDepositRate: true,
}

// TrendDays 年化汇率趋势回看窗口（天数）。设计决定：03-m2-spec §4 只给"当日
// USDCNH"，"净收益"的年化口径从 D-023 例推得，窗口 30d 与规则引擎 30d 滚动同源。
// 若决策层改口径，只改此常量 + 测试。
const TrendDays = 30

// Converted 折算结果：原事实 + RMB 视角值 + 汇率元数据（原始 Fact 不被改写）。
type Converted struct {
	Fact        fact.Fact
	RMBValue    float64 // RMB 净收益视角；覆盖 kind 且汇率可用 = Value − 年化升值（点数）；否则 = Value
	FXRate      float64 // 当日 USDCNH；0 = 不可用
	FXAvailable bool    // 汇率可用（false → 前端显示"汇率不可用"，不静默用错值）
}

// AnnualizedRMBAppreciation 由 fx 事实序列（ts 升序）计算年化人民币升值率
// （**百分点点数**，与事实 Value 同刻度，R6#1 裁定）：
//
//	raw = (last/first − 1) / 天数 × 365 × 100   （正 = USDCNH 走强 = 美元升值）
//	return = −raw                                （正 = 人民币升值；6% 收益率 − 3 点升值 = 3% 净）
//
// 序列 < 2 点 / 无时间跨度 / 首值 ≤ 0 → 0（趋势不可用）。调用方保证序列 ts 升序。
func AnnualizedRMBAppreciation(series []fact.Fact) float64 {
	if len(series) < 2 {
		return 0
	}
	first, last := series[0], series[len(series)-1]
	days := last.Ts.Sub(first.Ts).Hours() / 24
	if days <= 0 || first.Value <= 0 {
		return 0
	}
	raw := (last.Value/first.Value - 1) / days * 365 * 100
	return -raw
}

// Convert 对 facts 做 RMB 折算：
//   - 覆盖 kind（funding/defi_rate/deposit_rate）且 fx 非 nil →
//     RMBValue = Value − appreciation（USD 收益率 − 年化人民币升值，两者都是点数），
//     FXRate/FXAvailable 回填；
//   - 覆盖 kind 且 fx nil（汇率缺失）→ RMBValue = Value（USD 原值），FXAvailable=false；
//   - 未覆盖 kind → RMBValue = Value（不折算），FXAvailable=false（N/A）。
func Convert(facts []fact.Fact, fx *fact.Fact, appreciation float64) []Converted {
	out := make([]Converted, 0, len(facts))
	for _, f := range facts {
		c := Converted{Fact: f, RMBValue: f.Value}
		if !CoveredKinds[f.Kind] {
			out = append(out, c) // 不折算（RMB 计价或 N/A）
			continue
		}
		if fx == nil {
			out = append(out, c) // 汇率缺失：USD 原值 + FXAvailable=false（前端"汇率不可用"）
			continue
		}
		c.RMBValue = f.Value - appreciation
		c.FXRate = fx.Value
		c.FXAvailable = true
		out = append(out, c)
	}
	return out
}

// SeriesWindow 返回 [now−TrendDays, now) 的查询起始时刻（供调用方 QueryFacts）。
func SeriesWindow(now time.Time) time.Time {
	return now.Add(-time.Duration(TrendDays) * 24 * time.Hour)
}
