package sim

import (
	"context"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"arbcn/internal/fact"
)

// fundingFacts 生成 n 条 8h 间隔、年化相同的 funding 事实（从 ts 起）。
func fundingFacts(ts time.Time, n int, annualized float64) []fact.Fact {
	out := make([]fact.Fact, n)
	for i := 0; i < n; i++ {
		out[i] = fact.Fact{
			Kind: fact.KindFunding, Venue: "binance", Symbol: "BTC",
			Value: annualized, Ts: ts.Add(time.Duration(i) * 8 * time.Hour), Src: "test",
		}
	}
	return out
}

// TestReportCumulative：[对抗测试锚点 §9.5 S4] 实际累计 = Σ 每期分数费率 × 名义。
// 删 report.go ComputeSeries 中 sumRate 的 Σ（如改成只取首条）→ 本测试必红。
// 数值独立推导：10.95%/1095 = 0.0001/期 × 10000 = 1.0/期 × 3 期 = 3.0。
func TestReportCumulative(t *testing.T) {
	fs := fundingFacts(t0, 3, 10.95)
	r := ComputeSeries("binance", "BTC", fs, 10_000, defaultFrictionRate)

	if math.Abs(r.ActualCumulative-3.0) > 1e-9 {
		t.Fatalf("ActualCumulative = %v, want 3.0（3 期 × 0.0001 × 10000）", r.ActualCumulative)
	}
	// 理论累计 = 窗口均值年化分数 × 名义 × 天数/365：0.1095×10000×0.6667/365 = 2.0。
	if math.Abs(r.TheoreticalCumulative-2.0) > 1e-6 {
		t.Fatalf("TheoreticalCumulative = %v, want 2.0", r.TheoreticalCumulative)
	}
	if math.Abs(r.Residual-1.0) > 1e-6 {
		t.Fatalf("Residual = %v, want 1.0（实际−理论）", r.Residual)
	}
}

// TestReportResidualDistribution：残差序列（实际累计 − 理论直线）均值/σ/半衰期。
// 恒定年化 → 逐点残差恒为每期常数，均值 = 常数，σ = 0；|残差| 不减半 → +Inf。
func TestReportResidualDistribution(t *testing.T) {
	fs := fundingFacts(t0, 3, 10.95)
	r := ComputeSeries("binance", "BTC", fs, 10_000, defaultFrictionRate)
	if math.Abs(r.ResidualMean-1.0) > 1e-9 {
		t.Fatalf("ResidualMean = %v, want 1.0", r.ResidualMean)
	}
	if math.Abs(r.ResidualStd-0) > 1e-9 {
		t.Fatalf("ResidualStd = %v, want 0（恒定年化无离散）", r.ResidualStd)
	}
	if !math.IsInf(r.HalfLifeDays, 1) {
		t.Fatalf("HalfLifeDays = %v, want +Inf（残差恒定不减半）", r.HalfLifeDays)
	}
}

// TestReportNetAnnualized：摩擦后年化净收益 = (实际 − 名义×摩擦)/名义 × 365/天数 × 100。
// 名义 10000、摩擦 0.2% = 20 > 实际 3 → 净负，≥5% 不成立。
func TestReportNetAnnualized(t *testing.T) {
	fs := fundingFacts(t0, 3, 10.95)
	r := ComputeSeries("binance", "BTC", fs, 10_000, defaultFrictionRate)
	if r.Pass5Pct {
		t.Fatalf("Pass5Pct = true, want false（摩擦 20 > 收益 3）")
	}
	if r.NetAnnualized >= 0 {
		t.Fatalf("NetAnnualized = %v, want 负（摩擦覆盖）", r.NetAnnualized)
	}
}

// TestReportEmptySeries：无事实 → 全零/零值报告（不 panic，HalfLife +Inf）。
func TestReportEmptySeries(t *testing.T) {
	r := ComputeSeries("binance", "BTC", nil, 10_000, defaultFrictionRate)
	if r.ActualCumulative != 0 || r.TheoreticalCumulative != 0 || r.Pass5Pct {
		t.Fatalf("empty series = %+v, want 零值", r)
	}
	if !math.IsInf(r.HalfLifeDays, 1) {
		t.Fatalf("HalfLifeDays = %v, want +Inf", r.HalfLifeDays)
	}
}

// TestRenderMarkdown：周频报告 markdown 独占段（begin/end 标记 + 表头 + 每行一条）。
func TestRenderMarkdown(t *testing.T) {
	fs := fundingFacts(t0, 3, 10.95)
	r := ComputeSeries("binance", "BTC", fs, 10_000, defaultFrictionRate)
	md := RenderMarkdown([]SeriesReport{r})
	if !strings.Contains(md, reportBeginMarker) || !strings.Contains(md, reportEndMarker) {
		t.Fatalf("markdown 缺 begin/end 标记:\n%s", md)
	}
	if !strings.Contains(md, "| venue | symbol |") || !strings.Contains(md, "| binance | BTC |") {
		t.Fatalf("markdown 缺表头/数据行:\n%s", md)
	}
	// 独占段：begin 在 end 之前，且除标记外同文件无重复 begin（P3 单事实源）。
	i, j := strings.Index(md, reportBeginMarker), strings.Index(md, reportEndMarker)
	if i < 0 || j < 0 || i >= j {
		t.Fatalf("begin/end 顺序错: begin=%d end=%d", i, j)
	}
	if strings.Count(md, reportBeginMarker) != 1 {
		t.Fatalf("begin 标记重复出现")
	}
}

// TestRenderReportQuery：Driver.RenderReport 按 HistoryDays 窗口查询 funding →
// 按 (venue,symbol) 分组出报告。HistoryDays=0 → 禁用返回空。
func TestRenderReportQuery(t *testing.T) {
	d, st := newDriver(t, DefaultConfig())
	st.facts = fundingFacts(t0, 3, 10.95)
	reports, err := d.RenderReport(context.Background())
	if err != nil {
		t.Fatalf("RenderReport: %v", err)
	}
	if len(reports) != 1 || reports[0].Venue != "binance" || reports[0].Symbol != "BTC" {
		t.Fatalf("reports = %+v, want 1 条 binance/BTC", reports)
	}

	cfg := DefaultConfig()
	cfg.HistoryDays = 0 // 禁用
	d0, _ := newDriver(t, cfg)
	r0, err := d0.RenderReport(context.Background())
	if err != nil || len(r0) != 0 {
		t.Fatalf("HistoryDays=0: reports=%v err=%v, want 空/无错", r0, err)
	}
}

// TestAtomicWriteReport：原子写盘成功 + 内容落盘（temp+rename 机制冒烟）。
func TestAtomicWriteReport(t *testing.T) {
	path := t.TempDir() + "/sub/sim_report.md"
	if err := atomicWriteReport(path, []byte(reportBeginMarker+"\ncontent\n"+reportEndMarker+"\n")); err != nil {
		t.Fatalf("atomicWriteReport: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(got), "content") {
		t.Fatalf("file = %q, want content", got)
	}
}
