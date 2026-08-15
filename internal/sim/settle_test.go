package sim

import (
	"math"
	"testing"

	"arbcn/internal/fact"
)

// TestSettleFundingPnl：单期结算 pnl = funding_rate × notional（04-m3-spec §3.2）。
// [对抗测试锚点] 删除 SettleFundingPnl 中的 `rate * notional` 乘法 → 本测试必红。
func TestSettleFundingPnl(t *testing.T) {
	if got := SettleFundingPnl(0.01, 10_000); got != 100 {
		t.Fatalf("SettleFundingPnl(0.01, 10000) = %v, want 100（rate×notional）", got)
	}
	if got := SettleFundingPnl(0, 10_000); got != 0 {
		t.Fatalf("SettleFundingPnl(0, 10000) = %v, want 0（零费率）", got)
	}
	if got := SettleFundingPnl(-0.005, 10_000); got != -50 {
		t.Fatalf("SettleFundingPnl(-0.005, 10000) = %v, want -50（负费率 = 支付）", got)
	}
}

// TestPer8hRate：年化费率（百分点点数）→ 单期（8h）分数费率。
// 追溯复审 H1：先 ÷100 转分数（点数≠分数），再 ÷（3×365=1095）。原实现缺 ÷100 →
// 每 8h 费率虚高 100 倍；[对抗测试锚点] 删 ÷100 → 本测试必红。
func TestPer8hRate(t *testing.T) {
	if got := Per8hRate(10.95); math.Abs(got-0.0001) > 1e-15 {
		t.Fatalf("Per8hRate(10.95) = %v, want 0.0001（10.95%%/100/1095）", got)
	}
}

// TestRMBDayEnd：日终 RMB 折算复用 internal/rmb（USD 年化费率 − 年化人民币升值），
// RMB PnL = rmbRate × notional × 持有天数/365。
// rmb 口径：RMBValue = Value − appreciation（同数值尺度——要扣 3 个百分点传 3.0）。
// 追溯复审 H1：rmbRate 是百分点点数，乘名义前 ÷100 转分数费率。
// 例：年化 6%、升值 3 个百分点 → rmbRate=3 点 → 单日 10000 名义 PnL = 0.03×10000/365。
func TestRMBDayEnd(t *testing.T) {
	fx := &fact.Fact{Kind: fact.KindFX, Venue: "sina", Symbol: "USDCNH", Value: 7.25, Ts: t0}
	got := RMBDayEnd(6, 10_000, 1, fx, 3.0)
	want := 3.0 / 100 * 10_000 / DaysPerYear
	if math.Abs(got-want) > 1e-6 {
		t.Fatalf("RMBDayEnd = %v, want %v（RMB 净费率 3 点 ÷100 × 名义 × 1 天/365）", got, want)
	}

	// 持有 30 天 → ×30。
	got30 := RMBDayEnd(6, 10_000, 30, fx, 3.0)
	if math.Abs(got30-want*30) > 1e-6 {
		t.Fatalf("RMBDayEnd(30d) = %v, want %v", got30, want*30)
	}
}

// TestRMBDayEndFXMissing：汇率缺失（fx=nil）→ 回退 USD 原值口径（与 rmb 同语义），
// RMBValue = 年化 6%（不扣升值）。
func TestRMBDayEndFXMissing(t *testing.T) {
	got := RMBDayEnd(6, 10_000, 1, nil, 0)
	want := 6.0 / 100 * 10_000 / DaysPerYear // 6 点 ÷100 = 0.06 分数
	if math.Abs(got-want) > 1e-6 {
		t.Fatalf("RMBDayEnd(nil fx) = %v, want %v（USD 原值口径）", got, want)
	}
}

// TestRMBDayEndAppreciationAbsorbs：汇率升值完全吞噬费率 → RMB 净 PnL ≤ 0。
func TestRMBDayEndAppreciationAbsorbs(t *testing.T) {
	fx := &fact.Fact{Kind: fact.KindFX, Venue: "sina", Symbol: "USDCNH", Value: 7.25, Ts: t0}
	got := RMBDayEnd(3, 10_000, 1, fx, 3.0) // 3 − 3 = 0
	if got != 0 {
		t.Fatalf("RMBDayEnd(3, 3 点升值) = %v, want 0（完全吞噬）", got)
	}
}
