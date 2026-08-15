// M3-b §9.5 历史收敛周频统计报告（04-m3-spec §5.3 落地）。
// 纯函数计算：实际累计 funding vs 理论累计、残差分布、收敛半衰期、摩擦后净收益 vs 5% 门槛。
// 收敛统计的唯一证据 = 历史数据回填（§5.3/D-036）；前向模拟只证机制（§5.2）。
// 本文件零网络零密钥（§9.4）：只有纯数学 + markdown 渲染。
package sim

import (
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"arbcn/internal/fact"
	"arbcn/internal/store"
)

// defaultFrictionRate 摩擦模型（04-m3-spec §5.2）：双开双平 taker 手续费 + 滑点估计，
// 占名义比例。0.2% 为定稿默认。
const defaultFrictionRate = 0.002

// minSpreadPct 门槛对照（§9.5：摩擦后净收益 vs 5% 门槛）。
const minSpreadPct = 5.0

// 周频报告段标记（facts.md 同风格独占段，P3 单事实源）。
const (
	reportBeginMarker = "<!-- ARBCN-SIM-REPORT-BEGIN -->"
	reportEndMarker   = "<!-- ARBCN-SIM-REPORT-END -->"
)

// SeriesReport 单 (venue,symbol) 周频统计（§9.5）。
type SeriesReport struct {
	Venue                string
	Symbol               string
	ActualCumulative     float64 // 实际累计 funding 收益（Σ 每期分数费率 × 名义，模拟 USD）
	TheoreticalCumulative float64 // 理论累计（窗口均值年化分数 × 名义 × 天数/365）
	Residual             float64 // 实际 − 理论
	ResidualMean         float64 // 残差分布均值
	ResidualStd          float64 // 残差分布 σ
	HalfLifeDays         float64 // |残差| 减半所需天数（窗口内未减半 = +Inf）
	NetAnnualized        float64 // 摩擦后年化净收益 %
	Pass5Pct             bool    // NetAnnualized ≥ 5%
}

// ComputeSeries 计算单 (venue,symbol) 的收敛统计（纯函数、零 I/O）。
// fs 为 kind=funding 的年化事实（无需有序，函数内排序）；notional 为模拟名义。
//
// [对抗测试锚点] §9.5：删除实际累计的 Σ（Per8hRate×notional）→
// report_test.go TestReportCumulative 累计断言必红。
func ComputeSeries(venue, symbol string, fs []fact.Fact, notional, frictionRate float64) SeriesReport {
	fs = sortedFacts(fs)
	// H1 复审：实时 funding collector 以采集时刻作 Ts（binance premiumIndex.time /
	// okx time.Now()），每 5m 轮询在 facts 表落 ~96 行同值事实。周报若把每行当一期，
	// Σ 累计会虚高近两数量级（残留差信号作废）。按 8h 结算桶折叠为每期一行，Σ 恢复
	// "一期一次"。桶宽与下方 ÷(3×365) 的 8h 除数一致（当前符号集 BTC/ETH/TRX 均 8h
	// 结算；将来接入 4h 合约须同改这两处）。
	fs = dedupSettleBuckets(fs)
	r := SeriesReport{Venue: venue, Symbol: symbol, HalfLifeDays: math.Inf(1)}
	if len(fs) == 0 {
		return r
	}
	// 实际累计 = Σ 每期分数费率 × 名义（H1 刻度：点数 ÷100 ÷1095 转单期分数）。
	sumRate := 0.0
	for _, f := range fs {
		sumRate += f.Value
	}
	actual := sumRate / 100 / (PeriodsPerDay * DaysPerYear) * notional

	// 理论累计 = 窗口均值年化（分数）× 名义 × 天数/365。
	meanAnnualized := sumRate / float64(len(fs))
	days := fs[len(fs)-1].Ts.Sub(fs[0].Ts).Hours() / 24
	theoretical := meanAnnualized / 100 * notional * days / DaysPerYear

	r.ActualCumulative = actual
	r.TheoreticalCumulative = theoretical
	r.Residual = actual - theoretical

	// 残差分布：逐结算点的累计残差序列（实际累计 − 理论直线）。
	var series []float64
	runActual := 0.0
	for i, f := range fs {
		runActual += f.Value / 100 / (PeriodsPerDay * DaysPerYear) * notional
		runDays := fs[i].Ts.Sub(fs[0].Ts).Hours() / 24
		runTheory := meanAnnualized / 100 * notional * runDays / DaysPerYear
		series = append(series, runActual-runTheory)
	}
	r.ResidualMean = mean(series)
	r.ResidualStd = stddev(series)
	r.HalfLifeDays = halfLifeDays(series, fs)

	// 摩擦后年化净收益 %（实际 − 摩擦成本）/ 名义 × 365/天数 × 100。
	net := actual - notional*frictionRate
	spanDays := math.Max(days, 1e-9)
	r.NetAnnualized = net / notional * DaysPerYear / spanDays * 100
	r.Pass5Pct = r.NetAnnualized >= minSpreadPct
	return r
}

// dedupSettleBuckets 把同一 8h 结算桶内的多行折叠为一行（保留桶内最早一行）。
// 输入须已按 Ts 升序。实时 funding collector 每 5m 轮询落一行（采集时刻作 Ts），
// 一个结算期 ~96 行同值——折叠后每期一行，Σ 恢复"一期一次"。
//
// [对抗测试锚点] §9.5 S4（H1 复审）：删除折叠 → report_test.go TestReportDedupsFlood
// （96 行/桶不虚高断言）必红。
func dedupSettleBuckets(fs []fact.Fact) []fact.Fact {
	const bucket = int64(8 * 3600) // 8h 结算桶（秒）；与 Per8hRate 除数一致
	out := make([]fact.Fact, 0, len(fs))
	last := int64(math.MinInt64)
	for _, f := range fs {
		b := f.Ts.Unix() / bucket
		if b == last {
			continue
		}
		last = b
		out = append(out, f)
	}
	return out
}

// sortedFacts 按 Ts 升序返回副本（纯函数内不修改输入）。
func sortedFacts(fs []fact.Fact) []fact.Fact {
	out := append([]fact.Fact(nil), fs...)
	sort.Slice(out, func(i, j int) bool { return out[i].Ts.Before(out[j].Ts) })
	return out
}

func mean(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := 0.0
	for _, x := range xs {
		s += x
	}
	return s / float64(len(xs))
}

func stddev(xs []float64) float64 {
	if len(xs) < 2 {
		return 0
	}
	m := mean(xs)
	s := 0.0
	for _, x := range xs {
		d := x - m
		s += d * d
	}
	return math.Sqrt(s / float64(len(xs)-1))
}

// halfLifeDays 返回 |残差| 相对首结算点减半所需天数；窗口内未减半 → +Inf。
// （L2 复审：实现为"相对首点"，注释原写"滚动窗口"系口径偏差，已对齐实现。）
func halfLifeDays(series []float64, fs []fact.Fact) float64 {
	if len(series) < 2 {
		return math.Inf(1)
	}
	start := math.Abs(series[0])
	if start == 0 {
		return 0
	}
	target := start / 2
	for i := 1; i < len(series); i++ {
		if math.Abs(series[i]) <= target {
			return fs[i].Ts.Sub(fs[0].Ts).Hours() / 24
		}
	}
	return math.Inf(1)
}

// RenderMarkdown 渲染周频报告为 markdown 独占段（纯函数；仿 exporter facts.md 段风格）。
func RenderMarkdown(reports []SeriesReport) string {
	sorted := append([]SeriesReport(nil), reports...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].Venue != sorted[j].Venue {
			return sorted[i].Venue < sorted[j].Venue
		}
		return sorted[i].Symbol < sorted[j].Symbol
	})
	var b strings.Builder
	b.WriteString(reportBeginMarker + "\n")
	b.WriteString("## sim_report（模拟盘周频统计 · M3-b §9.5）\n\n")
	b.WriteString("> 收敛统计唯一证据 = 历史 funding 数据回填（§5.3/D-036）；前向模拟只证机制（§5.2）。\n")
	b.WriteString("> 实际/理论累计 = 模拟 USD（名义）；残差 = 实际 − 理论；半衰期 = |残差| 减半所需天数（+Inf = 窗口内未减半）。\n")
	b.WriteString("> 摩擦后年化 = (实际 − 名义×摩擦率)/名义 × 365/天数 × 100；摩擦率默认 0.2%（双开双平 taker + 滑点）。\n\n")
	b.WriteString("| venue | symbol | 实际累计 | 理论累计 | 残差 | 残差均值 | 残差σ | 半衰期(天) | 摩擦后年化% | ≥5% |\n")
	b.WriteString("|-------|--------|---------|---------|------|---------|-------|-----------|------------|-----|\n")
	for _, r := range sorted {
		b.WriteString(fmt.Sprintf("| %s | %s | %.2f | %.2f | %.4f | %.4f | %.4f | %s | %.2f | %v |\n",
			r.Venue, r.Symbol, r.ActualCumulative, r.TheoreticalCumulative, r.Residual,
			r.ResidualMean, r.ResidualStd, halfLifeStr(r.HalfLifeDays), r.NetAnnualized, r.Pass5Pct))
	}
	b.WriteString(reportEndMarker + "\n")
	return b.String()
}

func halfLifeStr(d float64) string {
	if math.IsInf(d, 1) {
		return "+Inf"
	}
	return fmt.Sprintf("%.1f", d)
}

// RenderReport 查询 HistoryDays 窗口内 funding 历史 → 每 (venue,symbol) ComputeSeries →
// 渲染 markdown。HistoryDays ≤ 0 = 禁用（返回空）。数据面定位：收敛统计只来自历史（§5.3）。
func (d *Driver) RenderReport(ctx context.Context) ([]SeriesReport, error) {
	if d.cfg.HistoryDays <= 0 {
		return nil, nil
	}
	from := d.now().Add(-time.Duration(d.cfg.HistoryDays) * 24 * time.Hour)
	// H1 复审：Limit 须盖住实时洪水总量（365d × 3 符号 × 2 venue × ~288 行/日 ≈ 63 万行）；
	// 200k 会截断到最旧端、丢最近 ~4 个月，窗口整体错位。2M 留足余量，ComputeSeries 内
	// dedupSettleBuckets 再把每 8h 桶折叠为一行（~6.5k 行）。
	fs, err := d.st.QueryFacts(ctx, store.FactQuery{Kind: fact.KindFunding, From: from, Limit: 2_000_000})
	if err != nil {
		return nil, fmt.Errorf("sim report: query facts: %w", err)
	}
	groups := map[[2]string][]fact.Fact{}
	for _, f := range fs {
		key := [2]string{f.Venue, f.Symbol}
		groups[key] = append(groups[key], f)
	}
	keys := make([][2]string, 0, len(groups))
	for k := range groups {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i][0] != keys[j][0] {
			return keys[i][0] < keys[j][0]
		}
		return keys[i][1] < keys[j][1]
	})
	notional := d.cfg.Capital * d.cfg.MaxSizePct
	reports := make([]SeriesReport, 0, len(keys))
	for _, k := range keys {
		reports = append(reports, ComputeSeries(k[0], k[1], groups[k], notional, defaultFrictionRate))
	}
	return reports, nil
}

// renderReport 渲染并原子写入 sim_report 文件（settle loop 每 7 tick 调用）。
func (d *Driver) renderReport(ctx context.Context) error {
	if d.cfg.HistoryDays <= 0 || d.cfg.ReportPath == "" {
		return nil
	}
	reports, err := d.RenderReport(ctx)
	if err != nil {
		return err
	}
	if len(reports) == 0 {
		return nil
	}
	return atomicWriteReport(d.cfg.ReportPath, []byte(RenderMarkdown(reports)))
}

// atomicWriteReport 临时文件 + rename 原子落盘（仿 exporter.atomicWrite）。
func atomicWriteReport(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("sim report: mkdir %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".sim-report-*.tmp")
	if err != nil {
		return fmt.Errorf("sim report: create temp: %w", err)
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("sim report: write temp: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("sim report: close temp: %w", err)
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		return fmt.Errorf("sim report: rename: %w", err)
	}
	return nil
}
