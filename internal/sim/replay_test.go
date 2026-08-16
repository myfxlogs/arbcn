package sim

import (
	"strings"
	"testing"
	"time"

	"arbcn/internal/fact"
	"arbcn/internal/store"
)

// ffactAt 构造一条指定 ts 的 funding fact（pct_annualized 百分点点数）。
func ffactAt(ts time.Time, v float64) fact.Fact {
	return fact.Fact{Kind: fact.KindFunding, Venue: "okx", Symbol: "BTC", Value: v, Unit: fact.UnitPctAnnualized, Ts: ts}
}

// run 构造一段连续窗口（ts 每 2h 递增），返回事实切片。
func runAt(base time.Time, vals ...float64) []fact.Fact {
	fs := make([]fact.Fact, 0, len(vals))
	for i, v := range vals {
		fs = append(fs, ffactAt(base.Add(time.Duration(i)*2*time.Hour), v))
	}
	return fs
}

// tier15 测试高费率档（funding_hedge D-016 15% 档）。
const tier15 = 15.0

// friction 测试一次性摩擦 %（D-046 默认 0.3）。
const friction = 0.3

// TestReplayKindConfig：每策略回放档表正确（rateKind/tier/friction），未知 kind → ok=false。
// [对抗测试锚点] D-065：删任一表行 → 本测试必红；repo 摩擦误配 0.3% → TestReplayRepoZeroFriction 必红。
func TestReplayKindConfig(t *testing.T) {
	cases := []struct {
		kind                string
		wantRate            string
		wantTier, wantFric  float64
	}{
		{store.SimKindFundingHedge, fact.KindFunding, 15, 0.3},
		{store.SimKindCarryAsset, fact.KindDefiRate, 8, 0.3},
		{store.SimKindRepo, fact.KindReverseRepo, 5, 0},
	}
	for _, c := range cases {
		rate, tier, fric, ok := ReplayKindConfig(c.kind)
		if !ok || rate != c.wantRate || tier != c.wantTier || fric != c.wantFric {
			t.Errorf("ReplayKindConfig(%s) = (%s,%v,%v,%v), want (%s,%v,%v,true)", c.kind, rate, tier, fric, ok, c.wantRate, c.wantTier, c.wantFric)
		}
	}
	if _, _, _, ok := ReplayKindConfig("nonsense_kind"); ok {
		t.Fatal("未知 kind 必须 ok=false（不判不拒）")
	}
}

// TestReplayRepoZeroFriction：repo 摩擦必须 0——单个 ≥5% 读数（时点逆回购季末）不得被
// 0.3% 年化摊成 falsified（frictionAnn=109.5% 会错杀真实 repo 机会，D-065 修订值锚点）。
// [对抗测试锚点] 把 replayGateCfgs repo 摩擦改成 0.3 → 本测试必红。
func TestReplayRepoZeroFriction(t *testing.T) {
	fs := runAt(t0.Add(-30*24*time.Hour), 5.5) // 单读数 ≥5% 档
	p := ComputeReplay(fs, 5, 0)                // repo 档：tier 5 / 摩擦 0
	if p.Verdict != ReplayPass {
		t.Fatalf("repo 单读数 5.5%% 摩擦 0 → 净 5.5%% > 基档 4.5%%，want pass, got %q (net=%v)", p.Verdict, p.MeanNetAnn)
	}
	if p.Windows[0].FrictionAnn != 0 {
		t.Errorf("FrictionAnn = %v, want 0（repo 无 taker 费）", p.Windows[0].FrictionAnn)
	}
}

func TestComputeReplay_NoWindow(t *testing.T) {
	// 全部 <15% → window_count=0，verdict=no_window，note 明示环境无窗口（D-061 ②）。
	base := t0.Add(-90 * 24 * time.Hour)
	fs := runAt(base, 5, 6, 7, 8, 9, 10, 4, 3)
	p := ComputeReplay(fs, tier15, friction)
	if p.WindowCount != 0 {
		t.Fatalf("WindowCount = %d, want 0", p.WindowCount)
	}
	if p.Verdict != ReplayNoWindow {
		t.Errorf("Verdict = %q, want no_window（全部 <15%%）", p.Verdict)
	}
	if p.Note == "" || !strings.Contains(p.Note, "环境无窗口") {
		t.Errorf("Note 应明示环境无窗口, got %q", p.Note)
	}
	if p.HighSamples != 0 {
		t.Errorf("HighSamples = %d, want 0", p.HighSamples)
	}
}

func TestComputeReplay_Empty(t *testing.T) {
	// 空数据 → 不 panic + 不编造（practices #7）。
	p := ComputeReplay(nil, tier15, friction)
	if p.Verdict != ReplayNoWindow || p.Note == "" {
		t.Fatalf("空数据 verdict=%q note=%q, want no_window + 明示无数据", p.Verdict, p.Note)
	}
	if p.TotalSamples != 0 || p.WindowCount != 0 {
		t.Errorf("空数据 TotalSamples/WindowCount = %d/%d, want 0/0", p.TotalSamples, p.WindowCount)
	}
}

func TestComputeReplay_FalsifiedShortWindow(t *testing.T) {
	// 短窗口摊不动摩擦 → 均值净年化 ≤0 → falsified（结构性证伪）。
	// mean 16%、3 份 span 4h → 按最短 1 天摊 → frictionAnn = 0.3×365/1 = 109.5
	// → net = −93.5（1 天窗口只能抓到 16/365 ≈ 0.04% 实际收益，0.3% 往返成本必亏）。
	base := t0.Add(-30 * 24 * time.Hour)
	fs := runAt(base, 15, 16, 17) // 3 份，span 4h → 按 replayMinWindowDays 摊
	p := ComputeReplay(fs, tier15, friction)
	if p.WindowCount != 1 {
		t.Fatalf("WindowCount = %d, want 1", p.WindowCount)
	}
	if p.Verdict != ReplayFalsified {
		t.Fatalf("Verdict = %q, want falsified（短窗口覆盖不了摩擦）", p.Verdict)
	}
	if p.MeanNetAnn >= 0 {
		t.Errorf("MeanNetAnn = %v, want < 0（frictionAnn=%v ≥ mean 16）", p.MeanNetAnn, p.Windows[0].FrictionAnn)
	}
	if len(p.Windows) != 1 || p.Windows[0].Samples != 3 {
		t.Errorf("Windows = %d 个/样本 %d, want 1 个/3 样本", len(p.Windows), p.Windows[0].Samples)
	}
}

func TestComputeReplay_PassLongWindow(t *testing.T) {
	// 长窗口摊得动摩擦 → 净年化 > 稳定币基档 → pass（证伪未发生）。
	// mean 18%、窗口 90 天 → frictionAnn = 0.3×365/90 = 1.22 → net ≈ 16.78 > 4.5。
	base := t0.Add(-120 * 24 * time.Hour)
	var fs []fact.Fact
	for i := 0; i < 45; i++ { // 45 份 × 2h ≈ 90 天跨度
		fs = append(fs, ffactAt(base.Add(time.Duration(i)*2*24*time.Hour), 18))
	}
	p := ComputeReplay(fs, tier15, friction)
	if p.WindowCount != 1 {
		t.Fatalf("WindowCount = %d, want 1", p.WindowCount)
	}
	if p.Verdict != ReplayPass {
		t.Fatalf("Verdict = %q, want pass（长窗口净正）", p.Verdict)
	}
	if p.MeanNetAnn < 10 || p.MeanNetAnn > 20 {
		t.Errorf("MeanNetAnn = %v, want ≈16.78（18 − 0.3×365/90）", p.MeanNetAnn)
	}
}

func TestComputeReplay_Watch(t *testing.T) {
	// 窗口存在但净 ∈(0, 4.5] → watch（门禁无经济意义）。
	// mean 15.2%、窗口 8 天（5 份 × 2 天跨度）→ frictionAnn = 0.3×365/8 = 13.69
	// → net ≈ 1.51 ∈(0, 4.5]（若跨度只有 6 天 → net −3.05 ≤0 → falsified，本测试必红，
	// 正是「短窗口摊不动摩擦」与「中窗口净不抵基档」的边界区分）。
	base := t0.Add(-40 * 24 * time.Hour)
	var fs []fact.Fact
	for i := 0; i < 5; i++ { // 5 份 × 2 天 = 8 天跨度
		fs = append(fs, ffactAt(base.Add(time.Duration(i)*2*24*time.Hour), 15.2))
	}
	p := ComputeReplay(fs, tier15, friction)
	if p.Verdict != ReplayWatch {
		t.Fatalf("Verdict = %q, want watch（净 ∈(0,4.5]）net=%v", p.Verdict, p.MeanNetAnn)
	}
	if p.MeanNetAnn <= 0 || p.MeanNetAnn > replayStableBasePct {
		t.Errorf("MeanNetAnn = %v, want ∈(0, 4.5]", p.MeanNetAnn)
	}
}

func TestComputeReplay_Segmentation(t *testing.T) {
	// 高-低-高 → 两个窗口（读数落到 <15 断开）。
	base := t0.Add(-60 * 24 * time.Hour)
	fs := append(runAt(base, 16, 17), ffactAt(base.Add(6*time.Hour), 5)) // 高×2 → 低
	fs = append(fs, ffactAt(base.Add(8*time.Hour), 19), ffactAt(base.Add(10*time.Hour), 20))
	p := ComputeReplay(fs, tier15, friction)
	if p.WindowCount != 2 {
		t.Fatalf("WindowCount = %d, want 2（低读数断开）", p.WindowCount)
	}
	if len(p.Windows) != 2 || p.Windows[0].Samples != 2 || p.Windows[1].Samples != 2 {
		t.Errorf("Windows 明细 = %d 个/各样本 %d,%d, want 2 个/2,2", len(p.Windows), p.Windows[0].Samples, p.Windows[1].Samples)
	}
}

func TestComputeReplay_Boundary(t *testing.T) {
	// 边界：v=15.0 计入窗口（≥ tier）；v=14.99 断开。
	base := t0.Add(-20 * 24 * time.Hour)
	fs := runAt(base, 15.0, 14.99, 15.5)
	p := ComputeReplay(fs, tier15, friction)
	if p.WindowCount != 2 {
		t.Fatalf("WindowCount = %d, want 2（15.0 计入、14.99 断开）", p.WindowCount)
	}
	if p.HighSamples != 2 {
		t.Errorf("HighSamples = %d, want 2（15.0 与 15.5）", p.HighSamples)
	}
}

func TestComputeReplay_Unsorted(t *testing.T) {
	// 乱序输入 → 排序后窗口扫描仍正确（不依赖采样顺序）。
	base := t0.Add(-20 * 24 * time.Hour)
	fs := []fact.Fact{
		ffactAt(base.Add(4*time.Hour), 17),
		ffactAt(base.Add(0), 16),
		ffactAt(base.Add(6*time.Hour), 5),
		ffactAt(base.Add(2*time.Hour), 18),
	}
	p := ComputeReplay(fs, tier15, friction)
	if p.WindowCount != 1 || p.Windows[0].Samples != 3 {
		t.Fatalf("乱序 → WindowCount=%d Samples=%d, want 1/3", p.WindowCount, p.Windows[0].Samples)
	}
	if p.HighSamples != 3 {
		t.Errorf("HighSamples = %d, want 3", p.HighSamples)
	}
}

func TestOverallReplay_Aggregates(t *testing.T) {
	// 跨对聚合：okx/BTC no_window + binance/ETH pass → overall 合并窗口、样本加权净年化、
	// verdict 取 overall 自己的判定（复用 classifyReplay）。
	base := t0.Add(-120 * 24 * time.Hour)
	pairNoWin := ComputeReplay(runAt(base.Add(-30*24*time.Hour), 5, 6, 7), tier15, friction) // okx/BTC
	var ethFs []fact.Fact
	for i := 0; i < 45; i++ {
		f := ffactAt(base.Add(time.Duration(i)*2*24*time.Hour), 18)
		f.Venue, f.Symbol = "binance", "ETH"
		ethFs = append(ethFs, f)
	}
	pairPass := ComputeReplay(ethFs, tier15, friction) // binance/ETH 长窗口 pass
	o := OverallReplay([]ReplayPair{pairNoWin, pairPass}, tier15)
	if o.WindowCount != 1 {
		t.Fatalf("overall.WindowCount = %d, want 1（no_window 对不贡献窗口）", o.WindowCount)
	}
	if o.Verdict != ReplayPass {
		t.Errorf("overall.Verdict = %q, want pass（样本加权净年化正）", o.Verdict)
	}
	if o.HighSamples != pairPass.HighSamples {
		t.Errorf("overall.HighSamples = %d, want %d", o.HighSamples, pairPass.HighSamples)
	}
	if o.TotalSamples != pairNoWin.TotalSamples+pairPass.TotalSamples {
		t.Errorf("overall.TotalSamples = %d, want %d", o.TotalSamples, pairNoWin.TotalSamples+pairPass.TotalSamples)
	}
}

func TestOverallReplay_AllNoWindow(t *testing.T) {
	// 全部对无窗口 → overall no_window + 明示环境无窗口（不 panic）。
	base := t0.Add(-30 * 24 * time.Hour)
	p1 := ComputeReplay(runAt(base, 5, 6), tier15, friction)
	p2 := ComputeReplay(runAt(base.Add(-10*24*time.Hour), 3, 4), tier15, friction)
	o := OverallReplay([]ReplayPair{p1, p2}, tier15)
	if o.Verdict != ReplayNoWindow || o.WindowCount != 0 {
		t.Fatalf("overall verdict=%q count=%d, want no_window/0", o.Verdict, o.WindowCount)
	}
}

// 对抗锚点：删「≥tier 计入窗口」判定 → 边界测试必红。
func TestReplayBoundaryAnchor(t *testing.T) {
	base := t0.Add(-20 * 24 * time.Hour)
	fs := runAt(base, 14.99, 15.0)
	p := ComputeReplay(fs, tier15, friction)
	if p.WindowCount != 1 {
		t.Fatalf("15.0 必须计入窗口（≥%v），got WindowCount=%d", tier15, p.WindowCount)
	}
}

// 对抗锚点：删「摩擦年化摊薄」→ 短窗口 falsified 测试必红（net 会恒正）。
func TestReplayFrictionAnchor(t *testing.T) {
	base := t0.Add(-30 * 24 * time.Hour)
	p := ComputeReplay(runAt(base, 15, 16, 17), tier15, friction)
	if p.Windows[0].FrictionAnn == 0 {
		t.Fatal("摩擦年化摊薄必须 > 0，删摊薄 → 本测试必红")
	}
	if p.Verdict != ReplayFalsified {
		t.Fatalf("短窗口必须 falsified（摩擦未被摊薄则恒正），got %q", p.Verdict)
	}
}

// 对抗锚点：删「no_window 环境无窗口守卫」→ 空/全低输入必红（不得误判成 pass/watch）。
func TestReplayNoWindowAnchor(t *testing.T) {
	p := ComputeReplay(nil, tier15, friction)
	if p.Verdict != ReplayNoWindow {
		t.Fatalf("空数据必须 no_window，删守卫 → 本测试必红，got %q", p.Verdict)
	}
}
