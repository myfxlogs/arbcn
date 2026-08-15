package sim

import (
	"math"
	"strings"
	"testing"
)

// TestConfirmDriftPasses：无漂移 → 通过（04-m3-spec §10.3 锚点 `(100,5,100,5)` → 过）。
func TestConfirmDriftPasses(t *testing.T) {
	reject, reason := ConfirmDriftCheck(100, 5, 100, 5)
	if reject || reason != "" {
		t.Fatalf("ConfirmDriftCheck(100,5,100,5) = (%v, %q), want (false, \"\")", reject, reason)
	}
	// 边界内：ref 漂移 1.99% / 年化变化 19.8% → 仍过（未过线）。
	reject, _ = ConfirmDriftCheck(100, 5, 101.99, 5)
	if reject {
		t.Fatal("ConfirmDriftCheck(100,5,101.99,5) = reject, want pass（1.99% < 2%）")
	}
	reject, _ = ConfirmDriftCheck(100, 5, 100, 5.99)
	if reject {
		t.Fatal("ConfirmDriftCheck(100,5,100,5.99) = reject, want pass（19.8% < 20%）")
	}
	// 边界恰好在 2.00% / 20.00% → 过（口径 >，不含等于）。
	reject, _ = ConfirmDriftCheck(100, 5, 102.00, 5)
	if reject {
		t.Fatal("ConfirmDriftCheck(100,5,102.00,5) = reject, want pass（恰 2.00% 未过线）")
	}
	reject, _ = ConfirmDriftCheck(100, 5, 100, 6.00)
	if reject {
		t.Fatal("ConfirmDriftCheck(100,5,100,6.00) = reject, want pass（恰 20.00% 未过线）")
	}
}

// TestConfirmDriftRejectsDrift：[对抗测试锚点 §10.3] 删 `> 0.02` 漂移比较 → 本测试必红。
// ref 漂移 2.01% → 拒（独立触发）；年化变化 20.2% → 拒（独立触发）。
func TestConfirmDriftRejectsDrift(t *testing.T) {
	reject, reason := ConfirmDriftCheck(100, 5, 102.01, 5)
	if !reject {
		t.Fatal("ConfirmDriftCheck(100,5,102.01,5) = pass, want reject（ref 漂移 2.01%）")
	}
	if !strings.Contains(reason, "ref_price 漂移") {
		t.Fatalf("reason = %q, want 含 ref_price 漂移", reason)
	}

	reject, reason = ConfirmDriftCheck(100, 5, 100, 6.01)
	if !reject {
		t.Fatal("ConfirmDriftCheck(100,5,100,6.01) = pass, want reject（年化变化 20.2%）")
	}
	if !strings.Contains(reason, "预期年化变化") {
		t.Fatalf("reason = %q, want 含 预期年化变化", reason)
	}
}

// TestConfirmDriftFailClosed：[对抗测试锚点 §10.3] 删 fail-closed 守卫 → 本测试必红。
// 零 ref（生成时无参考价）/ NaN / ±Inf 输入 → 拒（practices #7：NaN < x 恒 false，
// 不得静默放行）。
func TestConfirmDriftFailClosed(t *testing.T) {
	cases := []struct {
		name           string
		genRef, genSpr, curRef, curSpr float64
	}{
		{"零 genRef", 0, 5, 100, 5},
		{"零 genSpread", 100, 0, 100, 5},
		{"NaN curRef", 100, 5, math.NaN(), 5},
		{"NaN genSpread", 100, math.NaN(), 100, 5},
		{"+Inf curSpread", 100, 5, 100, math.Inf(1)},
		{"-Inf curRef", 100, 5, math.Inf(-1), 5},
		{"NaN 双输入", math.NaN(), 5, math.NaN(), 5},
	}
	for _, c := range cases {
		reject, reason := ConfirmDriftCheck(c.genRef, c.genSpr, c.curRef, c.curSpr)
		if !reject {
			t.Errorf("%s = pass, want reject（fail-closed）", c.name)
		}
		if !strings.Contains(reason, "SPREAD_DRIFT") {
			t.Errorf("%s reason = %q, want 含 SPREAD_DRIFT", c.name, reason)
		}
	}
}
