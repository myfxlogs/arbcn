package collect

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"arbcn/internal/fact"
)

// testFact 快速构造测试事实（默认心跳时序无关值）。
func testFact(kind, venue, symbol string, value float64, ts time.Time) fact.Fact {
	return fact.Fact{Kind: kind, Venue: venue, Symbol: symbol, Value: value, Ts: ts}
}

// TestDedupSameValueTsSkipped：喂相同 (value, ts) 两批 → 只落一批（不调 Sink）。
// 删掉 dedup.go 的跳过分支（continue）本测试必红。
func TestDedupSameValueTsSkipped(t *testing.T) {
	ts := time.Unix(1000, 0)
	f := testFact(fact.KindFunding, "binance", "BTC", 10.5, ts)
	sink := &recSink{}
	d := newDedupSet()

	if err := d.emit(context.Background(), []fact.Fact{f}, sink.emit); err != nil {
		t.Fatalf("emit(1): %v", err)
	}
	if err := d.emit(context.Background(), []fact.Fact{f}, sink.emit); err != nil {
		t.Fatalf("emit(2): %v", err)
	}
	if sink.count() != 1 {
		t.Fatalf("sink 收到 %d 批, want 1（相同 value+ts 去重）", sink.count())
	}
}

// TestDedupValueChangeLands：value 变 → 照常落库。
func TestDedupValueChangeLands(t *testing.T) {
	ts := time.Unix(1000, 0)
	sink := &recSink{}
	d := newDedupSet()
	fs := []fact.Fact{
		testFact(fact.KindFunding, "binance", "BTC", 10.5, ts),
		testFact(fact.KindFunding, "binance", "BTC", 10.6, ts), // value 变
	}
	if err := d.emit(context.Background(), fs, sink.emit); err != nil {
		t.Fatalf("emit: %v", err)
	}
	if sink.count() != 2 {
		t.Fatalf("sink 收到 %d 条, want 2（value 变化不落库 = 失真）", sink.count())
	}
}

// TestDedupTsChangeLands：ts 变（value 相同）→ 照常落库。
func TestDedupTsChangeLands(t *testing.T) {
	sink := &recSink{}
	d := newDedupSet()
	fs := []fact.Fact{
		testFact(fact.KindFunding, "binance", "BTC", 10.5, time.Unix(1000, 0)),
		testFact(fact.KindFunding, "binance", "BTC", 10.5, time.Unix(1001, 0)), // ts 变
	}
	if err := d.emit(context.Background(), fs, sink.emit); err != nil {
		t.Fatalf("emit: %v", err)
	}
	if sink.count() != 2 {
		t.Fatalf("sink 收到 %d 条, want 2（ts 变化不落库 = 失真）", sink.count())
	}
}

// TestDedupKeyIsolation：venue 参与 key——不同源同名 symbol 不冲突（M2-a §3.1 边界）。
func TestDedupKeyIsolation(t *testing.T) {
	ts := time.Unix(1000, 0)
	sink := &recSink{}
	d := newDedupSet()
	fs := []fact.Fact{
		testFact(fact.KindFunding, "binance", "BTC", 10.5, ts),
		testFact(fact.KindFunding, "okx", "BTC", 10.5, ts),     // 同 symbol 异 venue
		testFact(fact.KindTicker, "okx", "BTC", 10.5, ts),      // 同 venue/symbol 异 kind
		testFact(fact.KindFunding, "okx", "ETH", 10.5, ts),     // 同 kind/venue 异 symbol
		testFact(fact.KindFunding, "binance", "BTC", 10.5, ts), // 重复 → 应被去重
	}
	if err := d.emit(context.Background(), fs, sink.emit); err != nil {
		t.Fatalf("emit: %v", err)
	}
	if sink.count() != 4 {
		t.Fatalf("sink 收到 %d 条, want 4（仅第 5 条与第 1 条同 key 被去重）", sink.count())
	}
}

// TestDedupSinkFailureDoesNotConsume：[对抗测试锚点] Sink 失败 → 去重状态不推进，
// 重试可重新投递。原实现先 d.last[k]=f 再 next()，失败后相同 (value,ts) 被当重复
// 跳过 → 数据永久丢失（R1#1）。删掉 dedup.go 的 `if err := next(...); err != nil { return err }`
// 后返回前的状态推进守卫 → 本测试必红。
func TestDedupSinkFailureDoesNotConsume(t *testing.T) {
	ts := time.Unix(1000, 0)
	f := testFact(fact.KindFunding, "binance", "BTC", 10.5, ts)
	d := newDedupSet()
	attempts, got := 0, 0
	sink := func(_ context.Context, fs []fact.Fact) error {
		attempts++
		if attempts == 1 {
			return errors.New("sink: db down")
		}
		got += len(fs)
		return nil
	}

	// 第一次：Sink 失败 → 返回错误（调用方退避重试）。
	if err := d.emit(context.Background(), []fact.Fact{f}, sink); err == nil {
		t.Fatal("emit(1) = nil, want error（Sink 失败须返回）")
	}
	// 重试：相同 (value,ts) 必须重新投递（不被去重吞掉）。
	if err := d.emit(context.Background(), []fact.Fact{f}, sink); err != nil {
		t.Fatalf("emit(2) = %v, want nil（重试投递）", err)
	}
	if got != 1 {
		t.Fatalf("sink 收到 %d 条, want 1（失败轮不得吞数据）", got)
	}
	// 第三次：已成功投递 → 去重生效。
	if err := d.emit(context.Background(), []fact.Fact{f}, sink); err != nil {
		t.Fatalf("emit(3) = %v, want nil", err)
	}
	if got != 1 {
		t.Fatalf("sink 收到 %d 条, want 1（成功后去重）", got)
	}
}

// TestDedupHeartbeatUnaffected：heartbeat 的 value 每周期变化 → 照常落库。
// 生产上心跳不流经 Scheduler Sink（alert.Heartbeat 直连 st.InsertFacts），
// 即便流入本包装也不受影响（value 持续变化）。
func TestDedupHeartbeatUnaffected(t *testing.T) {
	ts := time.Unix(1000, 0)
	sink := &recSink{}
	d := newDedupSet()
	fs := []fact.Fact{
		testFact(fact.KindHeartbeat, "collector", "binance_funding", 0.1, ts),
		testFact(fact.KindHeartbeat, "collector", "binance_funding", 0.2, ts), // value 增长
		testFact(fact.KindHeartbeat, "collector", "binance_funding", 0.3, ts),
	}
	if err := d.emit(context.Background(), fs, sink.emit); err != nil {
		t.Fatalf("emit: %v", err)
	}
	if sink.count() != 3 {
		t.Fatalf("sink 收到 %d 条, want 3（心跳 value 变化全部落库）", sink.count())
	}
}

// TestSchedulerDedupSuppressesRepeats：Scheduler 级——Collector 无感返回相同
// (value, ts)，Dedup 开启时只落第一批；多源并发访问共享 map（-race 覆盖互斥）。
func TestSchedulerDedupSuppressesRepeats(t *testing.T) {
	fc := &fakeCollector{
		kind: fact.KindFunding,
		out:  []fact.Fact{testFact(fact.KindFunding, "binance", "BTC", 10.95, time.Unix(0, 0))},
	}
	sink := &recSink{}
	s := &Scheduler{
		Sources: []Named{
			{Name: "binance_funding", Interval: 2 * time.Millisecond, Collector: fc},
			{Name: "okx_funding", Interval: 2 * time.Millisecond,
				Collector: &fakeCollector{kind: fact.KindFunding,
					out: []fact.Fact{testFact(fact.KindFunding, "okx", "BTC", 10.95, time.Unix(0, 0))}}},
		},
		Sink:   sink.emit,
		Jitter: func() float64 { return 0.5 },
		Dedup:  true,
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { _ = s.Run(ctx); close(done) }()

	// 多轮轮询后：两源各只落 1 条（同 key 相同 value+ts 全被去重）。
	waitFor(t, 200*time.Millisecond, func() bool { return fc.pollCount() >= 4 })
	cancel()
	<-done

	if sink.count() != 2 {
		t.Fatalf("sink 收到 %d 条, want 2（每源只落第一批，后续重复被去重）", sink.count())
	}
}

// mutCollector：每轮 value 递增的假源（验证值变化照常落库）。
type mutCollector struct {
	mu  sync.Mutex
	val float64
}

func (c *mutCollector) Kind() string { return fact.KindFunding }

func (c *mutCollector) Poll(ctx context.Context) ([]fact.Fact, error) {
	c.mu.Lock()
	c.val++
	v := c.val
	c.mu.Unlock()
	return []fact.Fact{testFact(fact.KindFunding, "binance", "BTC", v, time.Unix(0, 0))}, nil
}

// TestSchedulerDedupAllowsChange：值变化 → 每批都落库（去重不吞真实变化）。
func TestSchedulerDedupAllowsChange(t *testing.T) {
	fc := &mutCollector{}
	sink := &recSink{}
	s := &Scheduler{
		Sources: []Named{{Name: "binance_funding", Interval: time.Millisecond, Collector: fc}},
		Sink:    sink.emit,
		Jitter:  func() float64 { return 0.5 },
		Dedup:   true,
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { _ = s.Run(ctx); close(done) }()

	waitFor(t, 200*time.Millisecond, func() bool { return sink.count() >= 3 })
	cancel()
	<-done

	if sink.count() < 3 {
		t.Fatalf("sink 收到 %d 条, want ≥3（value 每轮变化应全部落库）", sink.count())
	}
}
