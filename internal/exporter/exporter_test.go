package exporter

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"arbcn/internal/fact"
	"arbcn/internal/store"
)

// fakeFacts：最小 factsSource 假实现（只读 LatestFacts 面）。
type fakeFacts struct{ facts []fact.Fact }

func (f *fakeFacts) LatestFacts(_ context.Context, kind, venue, symbol string) ([]fact.Fact, error) {
	out := []fact.Fact{}
	for _, x := range f.facts {
		if kind != "" && x.Kind != kind {
			continue
		}
		if venue != "" && x.Venue != venue {
			continue
		}
		if symbol != "" && x.Symbol != symbol {
			continue
		}
		out = append(out, x)
	}
	return out, nil
}

var (
	expNow = time.Date(2026, 8, 15, 18, 14, 0, 0, time.UTC)
	expFacts = []fact.Fact{
		{Kind: fact.KindFunding, Venue: "binance", Symbol: "BTC", Value: 6.84, Unit: fact.UnitPctAnnualized, Ts: expNow, Src: "binance"},
		{Kind: fact.KindFX, Venue: "sina", Symbol: "USDCNH", Value: 7.03, Unit: fact.UnitPrice, Ts: expNow, Src: "sina"},
		{Kind: fact.KindDefiRate, Venue: "aave", Symbol: "USDC", Value: 4.67, Unit: fact.UnitPctAnnualized, Ts: expNow, Src: "defillama"},
		{Kind: fact.KindHeartbeat, Venue: "binance", Symbol: "binance", Value: 0, Ts: expNow, Src: "heartbeat"}, // 内部遥测，不导出
	}
)

func newTestExporter(t *testing.T) (*Exporter, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "facts.md")
	x := New(&fakeFacts{facts: expFacts}, path)
	x.Now = func() time.Time { return expNow }
	return x, path
}

// TestExportCreatesSection：首次导出 → 文件含段标记 + 新快照表格；
// 快照含市场事实、排除 heartbeat。
func TestExportCreatesSection(t *testing.T) {
	x, path := newTestExporter(t)
	if err := x.Export(context.Background()); err != nil {
		t.Fatalf("Export: %v", err)
	}
	doc, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	s := string(doc)
	if !strings.Contains(s, beginMarker) || !strings.Contains(s, endMarker) {
		t.Errorf("段标记缺失：\n%s", s)
	}
	for _, want := range []string{
		"### 快照 2026-08-15 18:14（现行）",
		"| funding BTC@binance | 6.84 | pct_annualized | 2026-08-15 18:14 | binance |",
		"| defi_rate USDC@aave | 4.67 | pct_annualized | 2026-08-15 18:14 | defillama |",
		"| fx USDCNH@sina | 7.03 | price | 2026-08-15 18:14 | sina |",
	} {
		if !strings.Contains(s, want) {
			t.Errorf("快照缺 %q", want)
		}
	}
	if strings.Contains(s, "heartbeat") {
		t.Error("快照不应含 heartbeat 内部遥测")
	}
}

// TestExportMarksOldExpiredNotDeleted：[对抗测试锚点 §11②]——
// 二次导出 → 旧「现行」快照改标「已过期（被新快照取代）」且旧值逐行保留；
// 删除 writeSection 里 strings.Replace("（现行）", expiredMark) 一行 → 本测试必红。
func TestExportMarksOldExpiredNotDeleted(t *testing.T) {
	x, path := newTestExporter(t)
	ctx := context.Background()

	if err := x.Export(ctx); err != nil {
		t.Fatalf("Export(1): %v", err)
	}
	expNow2 := expNow.Add(20 * time.Minute)
	x.Now = func() time.Time { return expNow2 }
	if err := x.Export(ctx); err != nil {
		t.Fatalf("Export(2): %v", err)
	}

	doc, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	s := string(doc)

	if !strings.Contains(s, "### 快照 2026-08-15 18:34（现行）") {
		t.Errorf("新快照缺现行头：\n%s", s)
	}
	if !strings.Contains(s, "### 快照 2026-08-15 18:14（已过期 · 被 2026-08-15 18:34 快照取代）") {
		t.Errorf("旧快照未标已过期：\n%s", s)
	}
	if !strings.Contains(s, "| funding BTC@binance | 6.84 | pct_annualized | 2026-08-15 18:14 | binance |") {
		t.Errorf("旧快照值被删除：\n%s", s)
	}
	if strings.Index(s, "18:34（现行）") > strings.Index(s, "18:14（已过期") {
		t.Error("新快照应在旧快照之前")
	}
}

// TestExportPreservesHandContent：段外手工内容（段前/段后）逐字保留。
func TestExportPreservesHandContent(t *testing.T) {
	x, path := newTestExporter(t)
	before := "# 市场事实库\n\n## 1. 国内利率\n\n| 事实 | 值 |\n|------|-----|\n| 民营定期 | 2.15% |\n"
	after := "\n## 5. 业主渠道现状\n\n| 事实 | 状态 |\n|------|------|\n| Binance 账户 | 已有 ✓ |\n"
	if err := os.WriteFile(path, []byte(before+after), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := x.Export(context.Background()); err != nil {
		t.Fatalf("Export: %v", err)
	}
	doc, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	s := string(doc)
	if !strings.Contains(s, "## 1. 国内利率") || !strings.Contains(s, "| 民营定期 | 2.15% |") {
		t.Errorf("段前手工内容丢失：\n%s", s)
	}
	if !strings.Contains(s, "## 5. 业主渠道现状") || !strings.Contains(s, "| Binance 账户 | 已有 ✓ |") {
		t.Errorf("段后手工内容丢失：\n%s", s)
	}
	if strings.Index(s, beginMarker) > strings.Index(s, endMarker) {
		t.Error("段标记顺序错误")
	}
}

// TestExportRewritesInPlace：二次导出 → 段内更新，段外手工内容保留且不重复。
func TestExportRewritesInPlace(t *testing.T) {
	x, path := newTestExporter(t)
	hand := "# 市场事实库\n\n## 1. 国内利率\n\n| 事实 | 值 |\n|------|-----|\n| 民营定期 | 2.15% |\n"
	if err := os.WriteFile(path, []byte(hand), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := x.Export(ctx); err != nil {
		t.Fatalf("Export(1): %v", err)
	}
	x.Now = func() time.Time { return expNow.Add(time.Hour) }
	if err := x.Export(ctx); err != nil {
		t.Fatalf("Export(2): %v", err)
	}
	doc, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	s := string(doc)
	if n := strings.Count(s, beginMarker); n != 1 {
		t.Errorf("begin 标记出现 %d 次, want 1", n)
	}
	if n := strings.Count(s, "## 1. 国内利率"); n != 1 {
		t.Errorf("手工节出现 %d 次, want 1（重写不重复）", n)
	}
}

// TestOnRuleActiveCoalesces：规则触发信号非阻塞投递，突发合并为一次导出。
func TestOnRuleActiveCoalesces(t *testing.T) {
	x, _ := newTestExporter(t)
	for i := 0; i < 5; i++ {
		x.OnRuleActive(context.Background(), store.Rule{}, nil)
	}
	if len(x.trigger) != 1 {
		t.Errorf("trigger 队列 len = %d, want 1（合并）", len(x.trigger))
	}
}

// TestExportWriteError：目标路径不可写 → Export 返回错误（不吞）。
func TestExportWriteError(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "facts.md")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	x := New(&fakeFacts{facts: expFacts}, dir)
	x.Now = func() time.Time { return expNow }
	if err := x.Export(context.Background()); err == nil {
		t.Error("Export on dir path = nil, want error")
	}
}

// TestRunExportsOnTrigger：Run 循环消费规则触发信号（关键规则激活 → 立即导出）。
// boot 导出后，OnRuleActive 触发 → 新快照（新时刻）出现；ctx 取消 → Run 退出。
func TestRunExportsOnTrigger(t *testing.T) {
	x, path := newTestExporter(t)
	// 并发安全时钟：与 Run goroutine 并发推进，用 atomic.Value 存 time.Time（D-067，同
	// TestRunThrottlesRuleTrigger；裸变量赋值在 -race 下检测为 data race）。
	var clock atomic.Value
	clock.Store(expNow)
	x.Now = func() time.Time { return clock.Load().(time.Time) }
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- x.Run(ctx) }()

	waitFor(t, 2*time.Second, func() bool {
		b, _ := os.ReadFile(path)
		return strings.Contains(string(b), "### 快照 2026-08-15 18:14（现行）")
	})

	// 触发规则事件：新时刻（18:44）快照应写入。
	expNow2 := expNow.Add(30 * time.Minute)
	clock.Store(expNow2)
	x.OnRuleActive(context.Background(), store.Rule{}, nil)
	waitFor(t, 2*time.Second, func() bool {
		b, _ := os.ReadFile(path)
		return strings.Contains(string(b), "### 快照 2026-08-15 18:44（现行）")
	})

	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Run: %v", err)
	}
}

// TestExportCapsSnapshotCount：[对抗测试锚点 D-066]——
// 连续导出 maxSnapshots+1 次 → 段内快照数恒为 maxSnapshots（最旧整体移除）；
// 删除 writeSection 里 truncateSnapshots 调用 → 第 maxSnapshots+1 份残留必红。
func TestExportCapsSnapshotCount(t *testing.T) {
	x, path := newTestExporter(t)
	ctx := context.Background()
	for i := 0; i < maxSnapshots+1; i++ {
		at := expNow.Add(time.Duration(i) * time.Minute)
		x.Now = func() time.Time { return at }
		if err := x.Export(ctx); err != nil {
			t.Fatalf("Export(%d): %v", i, err)
		}
	}
	doc, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	s := string(doc)
	if n := strings.Count(s, "### 快照 "); n != maxSnapshots {
		t.Errorf("快照数 = %d, want %d（封顶）", n, maxSnapshots)
	}
	// 最新份在最前、最旧份（首次导出时刻）已被移除。
	latest := expNow.Add(maxSnapshots * time.Minute).Format(timeLayout)
	if !strings.Contains(s, "### 快照 "+latest+"（现行）") {
		t.Errorf("最新快照 %s 缺失", latest)
	}
	if strings.Contains(s, "### 快照 "+expNow.Format(timeLayout)) {
		t.Error("最旧快照未移除（应被封顶淘汰）")
	}
	// 节头重置到段顶：位于最新快照之前。
	if hi, si := strings.Index(s, "## 监控快照"), strings.Index(s, "### 快照 "); hi < 0 || hi > si {
		t.Error("节头未重置到段顶（应在最新快照之前）")
	}
}

// TestRunThrottlesRuleTrigger：[对抗测试锚点 D-066]——
// 规则触发导出距上次成功导出 < ruleTriggerThrottle 被合并跳过；≥ 阈值才出新快照。
// 删除 Run 里节流判断 → 5min 内二次触发即出第二快照必红。
func TestRunThrottlesRuleTrigger(t *testing.T) {
	x, path := newTestExporter(t)
	// 并发安全时钟：Run goroutine 内 x.now() 与测试推进时钟并发，裸变量赋值有
	// data race（-race 必红）——用 atomic.Value 存 time.Time（写一次闭包，
	// 之后测试只 Store 推进），读走 Load，无同步开销外的竞态。
	var clock atomic.Value
	clock.Store(expNow)
	x.Now = func() time.Time { return clock.Load().(time.Time) }
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- x.Run(ctx) }()

	waitFor(t, 2*time.Second, func() bool {
		b, _ := os.ReadFile(path)
		return strings.Contains(string(b), "### 快照 2026-08-15 18:14（现行）")
	})
	// 距 boot 导出 +5min（< 阈值）：触发被节流，无新快照。
	early := expNow.Add(5 * time.Minute)
	clock.Store(early)
	x.OnRuleActive(context.Background(), store.Rule{}, nil)
	time.Sleep(300 * time.Millisecond)
	b, _ := os.ReadFile(path)
	if strings.Contains(string(b), "### 快照 "+early.Format(timeLayout)+"（现行）") {
		t.Error("节流失效：<10min 的规则触发仍写出新快照")
	}
	// 距 boot 导出 +12min（≥ 阈值）：触发生效，出新快照。
	late := expNow.Add(12 * time.Minute)
	clock.Store(late)
	x.OnRuleActive(context.Background(), store.Rule{}, nil)
	waitFor(t, 2*time.Second, func() bool {
		b, _ := os.ReadFile(path)
		return strings.Contains(string(b), "### 快照 "+late.Format(timeLayout)+"（现行）")
	})

	cancel()
	if err := <-done; err != nil {
		t.Fatalf("Run: %v", err)
	}
}

// waitFor 轮询等待条件成立（collect/rule 包同款）。
func waitFor(t *testing.T, d time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("condition not met in time")
}

// 编译期断言：fakeFacts 满足 factsSource。
var _ factsSource = (*fakeFacts)(nil)
