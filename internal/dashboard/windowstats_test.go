package dashboard

import (
	"testing"

	"arbcn/internal/fact"
)

// ffact 构造一条 funding fact（单位 pct_annualized 百分点点数）。
func ffact(v float64) fact.Fact {
	return fact.Fact{Kind: fact.KindFunding, Venue: "okx", Symbol: "BTC", Value: v, Unit: fact.UnitPctAnnualized}
}

// mk 从值列表构造 facts。
func mk(vals ...float64) []fact.Fact {
	fs := make([]fact.Fact, 0, len(vals))
	for _, v := range vals {
		fs = append(fs, ffact(v))
	}
	return fs
}

func TestComputeFundingWindowStats_Basic(t *testing.T) {
	// 10 份读数：6 正 + 4 负，已知 min/max/mean/正费率占比。
	fs := mk(4.0, 5.0, 6.0, 7.0, 8.0, 9.0, -1.0, -2.0, -3.0, -4.0)
	s := ComputeFundingWindowStats(fs)
	if s.Count != 10 {
		t.Fatalf("Count = %d, want 10", s.Count)
	}
	if s.Min != -4.0 {
		t.Errorf("Min = %v, want -4", s.Min)
	}
	if s.Max != 9.0 {
		t.Errorf("Max = %v, want 9", s.Max)
	}
	if got := s.Mean; got < 2.88 || got > 2.92 {
		t.Errorf("Mean = %v, want 2.9", got)
	}
	if got := s.PositiveShare; got < 0.59 || got > 0.61 {
		t.Errorf("PositiveShare = %v, want 0.6", got)
	}
	if s.Class != WindowWatch {
		t.Errorf("Class = %q, want watch（正费率占比 60%% 在 50–90%% 区间）", s.Class)
	}
	if s.Note == "" {
		t.Error("Note 应为空/含判据说明，不应为空串")
	}
}

func TestComputeFundingWindowStats_Empty(t *testing.T) {
	// 空数据 → 不 panic（占比不除零）+ 不编造（practices #7）。
	s := ComputeFundingWindowStats(nil)
	if s.Count != 0 {
		t.Errorf("Count = %d, want 0", s.Count)
	}
	if s.Class != WindowNot {
		t.Errorf("Class = %q, want not", s.Class)
	}
	if s.Note == "" || s.Note == "无" {
		t.Errorf("Note 应明示无数据，got %q", s.Note)
	}
}

func TestComputeFundingWindowStats_SparseSamples(t *testing.T) {
	// 样本 < 3 → 附加「样本过少仅供参考」（诚实标注不虚称窗口可信）。
	s := ComputeFundingWindowStats(mk(4.0, 5.0))
	if s.Count != 2 {
		t.Fatalf("Count = %d, want 2", s.Count)
	}
	if s.Note == "" || !contains(s.Note, "样本过少") {
		t.Errorf("Note 应含「样本过少」, got %q", s.Note)
	}
	// 3 份 = 到样本下限，不附加。
	s3 := ComputeFundingWindowStats(mk(4.0, 5.0, 6.0))
	if contains(s3.Note, "样本过少") {
		t.Errorf("3 份不应标样本过少, got %q", s3.Note)
	}
}

func TestClassifyFundingWindow_Matrix(t *testing.T) {
	cases := []struct {
		name  string
		s     FundingWindowStats
		class string
	}{
		// high：均值 ≥ 15%（D-016 高费率窗口档），即使占比不足 90% 也优先（档位最直接）。
		{"high mean 20", FundingWindowStats{Count: 10, Mean: 20, PositiveShare: 0.8}, WindowHigh},
		// tradable：正费率占比 ≥ 90% 且均值 ≥ 0。
		{"tradable 100% pos", FundingWindowStats{Count: 10, Mean: 5, PositiveShare: 1.0}, WindowTradable},
		{"tradable 90% pos", FundingWindowStats{Count: 10, Mean: 2, PositiveShare: 0.9}, WindowTradable},
		// watch：50%–90% 且均值 ≥ 0。
		{"watch 50% pos", FundingWindowStats{Count: 10, Mean: 1, PositiveShare: 0.5}, WindowWatch},
		{"watch 80% pos", FundingWindowStats{Count: 10, Mean: 3, PositiveShare: 0.8}, WindowWatch},
		// not：占比 < 50%。
		{"not 40% pos", FundingWindowStats{Count: 10, Mean: 1, PositiveShare: 0.4}, WindowNot},
		// not：均值为负（即使占比高——负均值说明负读数幅度大）。
		{"not negative mean", FundingWindowStats{Count: 10, Mean: -0.5, PositiveShare: 0.95}, WindowNot},
		// not：无数据。
		{"not empty", FundingWindowStats{Count: 0}, WindowNot},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			class, note := ClassifyFundingWindow(c.s)
			if class != c.class {
				t.Errorf("class = %q, want %q (note=%q)", class, c.class, note)
			}
			if c.s.Count > 0 && note == "" {
				t.Error("有数据时 note 不应为空（判据自明性）")
			}
		})
	}
}

// 对抗锚点：删「high 档优先」分支 → 本测试必红（mean 20 + 80% 会误判 tradable）。
func TestClassifyFundingWindow_HighTierAnchor(t *testing.T) {
	class, _ := ClassifyFundingWindow(FundingWindowStats{Count: 10, Mean: 20, PositiveShare: 0.8})
	if class != WindowHigh {
		t.Fatalf("高费率窗口档判据被破坏：mean 20 → %q, want high", class)
	}
}

// 对抗锚点：删「tradable ≥90% 占比」判据 → 本测试必红（40% 占比会被误判 tradable）。
func TestClassifyFundingWindow_TradableAnchor(t *testing.T) {
	class, _ := ClassifyFundingWindow(FundingWindowStats{Count: 10, Mean: 5, PositiveShare: 0.4})
	if class != WindowNot {
		t.Fatalf("正费率占比判据被破坏：40%% → %q, want not", class)
	}
}

// 对抗锚点：删「空数据守卫」→ 本测试必红（空窗口不得给可交易假象，D-019）。
func TestClassifyFundingWindow_EmptyAnchor(t *testing.T) {
	class, note := ClassifyFundingWindow(FundingWindowStats{Count: 0})
	if class != WindowNot {
		t.Fatalf("空窗口守卫被破坏：→ %q, want not", class)
	}
	if note == "" {
		t.Fatal("空窗口 note 应明示无数据")
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
