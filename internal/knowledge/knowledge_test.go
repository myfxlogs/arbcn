package knowledge

import (
	"math"
	"testing"
)

// TestFundingSpikeTrap：>factor×avg30 才判尖峰；avg30≤0 / NaN 输入不判（practices #7）。
// [对抗测试锚点] 删 `inst > factor*avg30` 判断 → 必红。
func TestFundingSpikeTrap(t *testing.T) {
	if !FundingSpikeTrap(9.14, 4.16, Factor) {
		t.Error("9.14 vs 4.16 应判尖峰（>2×）")
	}
	if FundingSpikeTrap(5.0, 4.16, Factor) {
		t.Error("5.0 vs 4.16 不应判尖峰（<2×）")
	}
	if FundingSpikeTrap(5.0, 0, Factor) || FundingSpikeTrap(5.0, -1, Factor) {
		t.Error("avg30≤0 不应判尖峰")
	}
	if FundingSpikeTrap(math.NaN(), 4.16, Factor) || FundingSpikeTrap(9.14, math.NaN(), Factor) {
		t.Error("NaN 输入不应判尖峰")
	}
	if FundingSpikeTrap(9.14, 4.16, 4.0) {
		t.Error("factor=4 时 9.14 不应判尖峰（阈值 16.64）")
	}
}

// TestDefiPoolSpikes：中位数×factor 标尖峰；样本 <3 不判；NaN 跳过；返回排序确定性。
// [对抗测试锚点] 删 `value > factor*median` 判断 → 必红。
func TestDefiPoolSpikes(t *testing.T) {
	pool := map[string]float64{
		"aave-v3:USDC": 3.5,
		"morpho:USDC":  3.9,
		"ethena:USDC":  4.0,
		"aave-v3:USDT": 12.57,
	}
	// 中位 3.95，×2 → 7.9：只标 12.57。
	out := DefiPoolSpikes(pool, Factor)
	if len(out) != 1 || out[0] != "aave-v3:USDT" {
		t.Errorf("spikes = %v, want [aave-v3:USDT]", out)
	}
	// 样本 <3 → 空。
	if out := DefiPoolSpikes(map[string]float64{"a": 3.0, "b": 4.0}, Factor); out != nil {
		t.Errorf("2 样本 spikes = %v, want nil", out)
	}
	// NaN 值跳过后有效 <3 → 空（practices #7）。
	if out := DefiPoolSpikes(map[string]float64{"a": 3.0, "b": math.NaN(), "c": 4.0}, Factor); out != nil {
		t.Errorf("含 NaN spikes = %v, want nil", out)
	}
	// 排序确定性：两个尖峰按 key 升序。
	out2 := DefiPoolSpikes(map[string]float64{"z:z": 20.0, "a:a": 20.0, "b:b": 1.0, "c:c": 1.0}, 1.1)
	if len(out2) != 2 || out2[0] != "a:a" || out2[1] != "z:z" {
		t.Errorf("spikes = %v, want [a:a z:z]（排序）", out2)
	}
}

// TestCrossVenueDivergence：一正一负（TRX 案例）与大差距同号都判；单值/小差距/NaN 后不足不判。
// [对抗测试锚点] 删 `max-min >= minSpread` 判断 → 必红。
func TestCrossVenueDivergence(t *testing.T) {
	// TRX 案例：binance +2.3 vs okx −3.5（差 5.8 ≥ 4）→ 分歧。
	if !CrossVenueDivergence([]float64{2.3, -3.5}, MinCrossVenueSpread()) {
		t.Error("一正一负（TRX 案例差 5.8）应判分歧")
	}
	if !CrossVenueDivergence([]float64{8.0, 2.0}, MinCrossVenueSpread()) {
		t.Error("同号大差距（差 6.0）应判分歧")
	}
	if CrossVenueDivergence([]float64{5.0, 3.0}, MinCrossVenueSpread()) {
		t.Error("小差距（差 2.0 < 4）不应判分歧")
	}
	if CrossVenueDivergence([]float64{5.0}, MinCrossVenueSpread()) {
		t.Error("单 venue 不应判分歧")
	}
	if CrossVenueDivergence(nil, MinCrossVenueSpread()) {
		t.Error("空值不应判分歧")
	}
	// NaN 值跳过：2.3 + NaN + −3.5 → 有效 [2.3, −3.5] 差 5.8 → 分歧。
	if !CrossVenueDivergence([]float64{2.3, math.NaN(), -3.5}, MinCrossVenueSpread()) {
		t.Error("NaN 跳过后 TRX 值仍应判分歧")
	}
}

// TestDefaults：3 条 seed 签名稳定 + 字段完整 + status active。
func TestDefaults(t *testing.T) {
	d := Defaults()
	if len(d) != 3 {
		t.Fatalf("Defaults = %d 条, want 3", len(d))
	}
	sigs := map[string]bool{}
	for _, e := range d {
		sigs[e.Signature] = true
		if e.Verdict == "" || e.Rationale == "" || e.Source == "" || e.Status != "active" {
			t.Errorf("seed %s 字段不完整: %+v", e.Signature, e)
		}
	}
	for _, s := range []string{SignatureFundingSpikeTrap, SignatureDefiSinglePoolSpike, SignatureFundingCrossVenueDiverg} {
		if !sigs[s] {
			t.Errorf("缺签名 %s", s)
		}
	}
}
