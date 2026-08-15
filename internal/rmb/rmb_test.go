package rmb

import (
	"math"
	"testing"
	"time"

	"arbcn/internal/fact"
)

var testNow = time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC)

func fxFact(value float64, ts time.Time) *fact.Fact {
	return &fact.Fact{Kind: fact.KindFX, Venue: "sina", Symbol: "USDCNH", Value: value, Unit: fact.UnitPrice, Ts: ts}
}

// TestConvertCoveredWithFX：覆盖 kind + 汇率可用 → RMBValue = Value − 年化升值，
// FXRate/FXAvailable 回填；原始 Fact 不被改写（不污染，02 §8）。
func TestConvertCoveredWithFX(t *testing.T) {
	facts := []fact.Fact{
		{Kind: fact.KindFunding, Venue: "binance", Symbol: "BTC", Value: 6.0, Unit: fact.UnitPctAnnualized, Ts: testNow},
		{Kind: fact.KindDefiRate, Venue: "aave", Symbol: "USDC", Value: 4.5, Unit: fact.UnitPctAnnualized, Ts: testNow},
		{Kind: fact.KindDepositRate, Venue: "manual", Symbol: "USD_3M", Value: 3.1, Unit: fact.UnitPctAnnualized, Ts: testNow},
	}
	fx := fxFact(7.25, testNow)
	got := Convert(facts, fx, 0.03)

	if len(got) != 3 {
		t.Fatalf("Convert len = %d, want 3", len(got))
	}
	for i, want := range []float64{5.97, 4.47, 3.07} {
		if math.Abs(got[i].RMBValue-want) > 1e-9 {
			t.Errorf("got[%d].RMBValue = %v, want %v", i, got[i].RMBValue, want)
		}
		if !got[i].FXAvailable || got[i].FXRate != 7.25 {
			t.Errorf("got[%d] fx = %v/%v, want available/7.25", i, got[i].FXAvailable, got[i].FXRate)
		}
		// 原始事实不污染（P3/02 §8）：Value 保持 USD 原值。
		if got[i].Fact.Value != facts[i].Value {
			t.Errorf("got[%d].Fact.Value = %v, want 原值 %v（不污染）", i, got[i].Fact.Value, facts[i].Value)
		}
	}
}

// TestConvertFXMissingDegrades：汇率缺失（fx=nil）→ 覆盖 kind 回退 USD 原值 +
// FXAvailable=false（"汇率不可用"标记），不崩溃、不静默用错值（03-m2-spec §4）。
// [对抗测试锚点] 删除 Convert 中 `if fx == nil` 早退分支 → 本测试必红（§11 锚点模式）。
func TestConvertFXMissingDegrades(t *testing.T) {
	facts := []fact.Fact{
		{Kind: fact.KindFunding, Venue: "binance", Symbol: "BTC", Value: 6.0},
		{Kind: fact.KindDefiRate, Venue: "aave", Symbol: "USDC", Value: 4.5},
	}
	got := Convert(facts, nil, 0)
	for i, c := range got {
		if c.RMBValue != facts[i].Value {
			t.Errorf("got[%d].RMBValue = %v, want USD 原值 %v（汇率缺失回退）", i, c.RMBValue, facts[i].Value)
		}
		if c.FXAvailable {
			t.Errorf("got[%d].FXAvailable = true, want false（汇率缺失 → 前端「汇率不可用」）", i)
		}
	}
}

// TestConvertNonCoveredUnchanged：未覆盖 kind（RMB 计价 / N/A）→ RMBValue 原样、
// FXAvailable=false（不折算）。
func TestConvertNonCoveredUnchanged(t *testing.T) {
	facts := []fact.Fact{
		{Kind: fact.KindReverseRepo, Venue: "sse", Symbol: "GC001", Value: 2.0}, // RMB 计价，不折算
		{Kind: fact.KindCalendar, Venue: "rule", Symbol: "quarter_end", Value: 12},
		{Kind: fact.KindHeartbeat, Venue: "collector", Symbol: "fx", Value: 1.0},
	}
	fx := fxFact(7.25, testNow)
	got := Convert(facts, fx, 0.03)
	for i, c := range got {
		if c.RMBValue != facts[i].Value {
			t.Errorf("got[%d].RMBValue = %v, want 原样 %v", i, c.RMBValue, facts[i].Value)
		}
		if c.FXAvailable {
			t.Errorf("got[%d].FXAvailable = true, want false（未覆盖 kind）", i)
		}
	}
}

// TestAnnualizedRMBAppreciation：年化人民币升值率口径——
// 一年跨度 7.25→7.03（USDCNH 贬 3.03%）→ 年化升值 ≈ +3.03%。
func TestAnnualizedRMBAppreciation(t *testing.T) {
	series := []fact.Fact{
		{Kind: fact.KindFX, Value: 7.25, Ts: testNow.Add(-365 * 24 * time.Hour)},
		{Kind: fact.KindFX, Value: 7.03, Ts: testNow},
	}
	got := AnnualizedRMBAppreciation(series)
	if math.Abs(got-0.0303) > 1e-3 {
		t.Errorf("appreciation = %v, want ≈ 0.0303（RMB 年化升值）", got)
	}

	// 反向：USDCNH 走强（USD 升值）→ 负升值（RMB 贬值）。
	series[1].Value = 7.47
	if got := AnnualizedRMBAppreciation(series); got < -0.031 || got > -0.029 {
		t.Errorf("appreciation(USD 走强) = %v, want ≈ −0.0303", got)
	}
}

// TestAnnualizedRMBAppreciationEdge：序列不足 / 无时间跨度 / 首值非法 → 0（趋势不可用）。
func TestAnnualizedRMBAppreciationEdge(t *testing.T) {
	if got := AnnualizedRMBAppreciation(nil); got != 0 {
		t.Errorf("nil = %v, want 0", got)
	}
	if got := AnnualizedRMBAppreciation([]fact.Fact{{Value: 7.25, Ts: testNow}}); got != 0 {
		t.Errorf("单点 = %v, want 0", got)
	}
	// 无时间跨度（同刻两条）。
	if got := AnnualizedRMBAppreciation([]fact.Fact{
		{Value: 7.25, Ts: testNow}, {Value: 7.03, Ts: testNow},
	}); got != 0 {
		t.Errorf("零跨度 = %v, want 0", got)
	}
	// 首值 ≤ 0（非法汇率）。
	if got := AnnualizedRMBAppreciation([]fact.Fact{
		{Value: 0, Ts: testNow.Add(-24 * time.Hour)}, {Value: 7.03, Ts: testNow},
	}); got != 0 {
		t.Errorf("首值非法 = %v, want 0", got)
	}
}
