// Package knowledge：市场结构经验库（D-046）。
//
// 吸收 = 人工 + D#（Defaults 落盘，git 跟踪；boot 幂等 upsert）；匹配 = 确定性签名纯函数
// （本包探测器，可对抗测试）；呈现 = 只读 knowledge_match insight（dashboard）+ 浏览 RPC。
// 系统只匹配与呈现，永不自动吸收、永不自动改 verdict（practices #20：优化方向由数据提示、
// 动作由决策层 D# 拍板）。
package knowledge

import (
	"math"
	"sort"

	"arbcn/internal/store"
)

// Signature 受控签名词（新签名 = 新 D#，git 留痕）。探测器按签名匹配知识条目；
// 未在 Defaults/DB 中的签名 → 无条目 → 不产信号（宁缺毋滥）。
const (
	// SignatureFundingSpikeTrap 资金费率尖峰陷阱：瞬时费率 ≫ 30 日均值，按瞬时值开仓不划算。
	SignatureFundingSpikeTrap = "funding:spike_trap"
	// SignatureDefiSinglePoolSpike 单池利率尖峰：截面中位数×2 判定，多不可持续。
	SignatureDefiSinglePoolSpike = "defi:single_pool_spike"
	// SignatureFundingCrossVenueDiverg 跨所资金费率显著分歧（可能真实市场结构，先核实）。
	SignatureFundingCrossVenueDiverg = "funding:cross_venue_divergence"
)

// Factor 尖峰判定倍率（与 dashboard.defiAnomalies 中位数×2 同口径；regime shift 稳健）。
const Factor = 2.0

// minCrossVenueSpread 跨所分歧最小价差点数（年化百分点）。TRX 案例 binance +2.3 vs
// okx −3.5 → 差 5.8 ≥ 4；低于此的常规噪音不标分歧。
const minCrossVenueSpread = 4.0

// Defaults 返回当前已核实的经验条目（D-046 seed，git 跟踪；boot 幂等 upsert）。
// 每条对应一个已识别的市场结构模式；verdict/rationale 为人工判定（D# 落），
// 后续重匹配 + 人工复核 → 新 D# 更新条目（不自动演进）。
func Defaults() []store.KnowledgeEntry {
	return []store.KnowledgeEntry{
		{
			Signature: SignatureFundingSpikeTrap,
			Verdict:   "坑",
			Rationale: "瞬时费率冲高多为尖峰陷阱：ETH@okx 曾瞬时 9.14% vs 30 日均值 4.16%，按 0.3% 摩擦需持续约 12 天才保本，扣摩擦净年化远低于稳定币基档 4.5%——别按瞬时值开仓，先看 30 日均值与保本天数。",
			Source:    "对话 #64 / D-016 / D-043",
			Status:    "active",
		},
		{
			Signature: SignatureDefiSinglePoolSpike,
			Verdict:   "坑·核实",
			Rationale: "单池利率尖峰多不可持续：Aave USDT 借贷利率曾瞬时 12.57%（截面中位数×2 判定），随后回落——按截面中位数×2 核对单池利率，勿按单点尖峰配置资金。",
			Source:    "对话 #63 / D-043",
			Status:    "active",
		},
		{
			Signature: SignatureFundingCrossVenueDiverg,
			Verdict:   "已核实·真实分歧",
			Rationale: "跨所资金费率分歧可能是真实市场结构而非数据错：TRX funding binance +2.3% vs okx −3.5% 经实测为真实分歧，双边可同时吃两所正负费率——遇跨所分歧先核实再判定，勿当数据异常忽略。",
			Source:    "对话 #50 / D-016",
			Status:    "active",
		},
	}
}

// FundingSpikeTrap 瞬时费率是否为尖峰陷阱：inst > factor×avg30（avg30>0 才有尖峰意义）。
// NaN/Inf/非正值 → false（不判定，宁缺毋滥，practices #7）。
//
// [对抗测试锚点] 删 `inst > factor*avg30` 判断（全判/全不判）→ knowledge_test.go 必红。
func FundingSpikeTrap(inst, avg30, factor float64) bool {
	if math.IsNaN(inst) || math.IsNaN(avg30) || avg30 <= 0 || inst <= 0 {
		return false
	}
	return inst > factor*avg30
}

// DefiPoolSpikes 截面尖峰：返回 value > factor×median 的 key 集合（样本 <3 或中位数
// ≤0 → 空）。key 为调用方实体的稳定标识（如 "venue\x00symbol"）。与 dashboard.
// defiAnomalies 的中位数×2 同口径；NaN/Inf 跳过。返回按 key 排序（确定性）。
//
// [对抗测试锚点] 删 `value > factor*median` 判断 → knowledge_test.go 必红。
func DefiPoolSpikes(poolValues map[string]float64, factor float64) []string {
	clean := map[string]float64{}
	for k, v := range poolValues {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			continue
		}
		clean[k] = v
	}
	if len(clean) < 3 {
		return nil
	}
	vals := make([]float64, 0, len(clean))
	for _, v := range clean {
		vals = append(vals, v)
	}
	med := median(vals)
	if math.IsNaN(med) || med <= 0 {
		return nil
	}
	var out []string
	for k, v := range clean {
		if v > factor*med {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}

// CrossVenueDivergence 跨所资金费率显著分歧：≥2 个 venue 值且 max−min ≥ minSpread
// 百分点（覆盖一正一负——TRX 案例——与大差距同号两种情况）。NaN/Inf 值跳过；有效
// 数 <2 → false。minSpread ≤ 0 → false。
//
// [对抗测试锚点] 删 `max-min >= minSpread` 判断 → knowledge_test.go 必红。
func CrossVenueDivergence(values []float64, minSpread float64) bool {
	if minSpread <= 0 {
		return false
	}
	clean := make([]float64, 0, len(values))
	for _, v := range values {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			continue
		}
		clean = append(clean, v)
	}
	if len(clean) < 2 {
		return false
	}
	min, max := clean[0], clean[0]
	for _, v := range clean[1:] {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	return max-min >= minSpread
}

// MinCrossVenueSpread 导出跨所分歧阈值（供测试/调用方引用常量口径）。
func MinCrossVenueSpread() float64 { return minCrossVenueSpread }

// median 中位数（未排序输入亦可；空 → NaN）。
func median(xs []float64) float64 {
	if len(xs) == 0 {
		return math.NaN()
	}
	sorted := append([]float64(nil), xs...)
	sort.Float64s(sorted)
	n := len(sorted)
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2
}
