package collect

import (
	"context"
	"testing"
	"time"

	"arbcn/internal/fact"
)

// testSched：固定抖动（factor 1.0 → 精确间隔）的确定性调度器。
func testSched(srcs []Named, sink Sink) (*Scheduler, context.Context, context.CancelFunc) {
	s := &Scheduler{
		Sources: srcs,
		Sink:    sink,
		Jitter:  func() float64 { return 0.5 },
	}
	ctx, cancel := context.WithCancel(context.Background())
	return s, ctx, cancel
}

// TestSchedulerPollsAndSinks：轮询发生、事实经 Sink 到达、取消后 Run 及时返回。
func TestSchedulerPollsAndSinks(t *testing.T) {
	fc := &fakeCollector{
		kind: fact.KindFunding,
		out:  []fact.Fact{{Kind: fact.KindFunding, Venue: "binance", Symbol: "BTC", Value: 10.95}},
	}
	sink := &recSink{}
	s, ctx, cancel := testSched([]Named{{Name: "binance_funding", Interval: 5 * time.Millisecond, Collector: fc}}, sink.emit)

	done := make(chan struct{})
	go func() { _ = s.Run(ctx); close(done) }()

	waitFor(t, 200*time.Millisecond, func() bool { return fc.pollCount() >= 4 })
	waitFor(t, 200*time.Millisecond, func() bool { return sink.count() >= 4 })

	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not return after cancel")
	}
}

// TestSchedulerBackoff：前 N 次失败触发退避重试（attempt 递增），成功后事实到达 Sink。
func TestSchedulerBackoff(t *testing.T) {
	fc := &fakeCollector{
		kind:  fact.KindFunding,
		failN: 2,
		out:   []fact.Fact{{Kind: fact.KindFunding, Venue: "binance", Symbol: "BTC"}},
	}
	sink := &recSink{}
	var backoffCalls []int
	s := &Scheduler{
		Sources: []Named{{Name: "binance_funding", Interval: time.Hour, Collector: fc}},
		Sink:    sink.emit,
		Jitter:  func() float64 { return 0.5 },
		Backoff: func(attempt int) time.Duration {
			backoffCalls = append(backoffCalls, attempt)
			return time.Millisecond
		},
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { _ = s.Run(ctx); close(done) }()

	waitFor(t, 200*time.Millisecond, func() bool { return sink.count() >= 1 })
	cancel()
	<-done

	if len(backoffCalls) < 2 {
		t.Fatalf("backoff called %d times, want >= 2", len(backoffCalls))
	}
	if backoffCalls[0] != 1 || backoffCalls[1] != 2 {
		t.Fatalf("backoff attempts = %v, want [1 2 ...]", backoffCalls)
	}
}

// TestSchedulerIsolation：A 持续失败不影响 B 正常轮询与落库。
func TestSchedulerIsolation(t *testing.T) {
	bad := &fakeCollector{kind: fact.KindFunding, failN: 1 << 30}
	good := &fakeCollector{
		kind: fact.KindTicker,
		out:  []fact.Fact{{Kind: fact.KindTicker, Venue: "okx", Symbol: "BTC"}},
	}
	sink := &recSink{}
	s := &Scheduler{
		Sources: []Named{
			{Name: "bad_funding", Interval: 2 * time.Millisecond, Collector: bad},
			{Name: "okx_ticker", Interval: 2 * time.Millisecond, Collector: good},
		},
		Sink:    sink.emit,
		Jitter:  func() float64 { return 0.5 },
		Backoff: func(int) time.Duration { return time.Millisecond },
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { _ = s.Run(ctx); close(done) }()

	waitFor(t, 200*time.Millisecond, func() bool { return good.pollCount() >= 4 })
	if bad.pollCount() < 2 {
		t.Fatalf("bad source polls = %d, want >= 2 (failure retries)", bad.pollCount())
	}
	if sink.count() == 0 {
		t.Fatal("sink got no facts from healthy source while other failed")
	}
	cancel()
	<-done
}

// TestSchedulerGracefulShutdown：在飞 Poll 被 ctx 取消，Run 及时返回（不等退避/间隔）。
func TestSchedulerGracefulShutdown(t *testing.T) {
	blocked := &fakeCollector{kind: fact.KindFunding, delay: 10 * time.Second}
	sink := &recSink{}
	s, ctx, cancel := testSched([]Named{{Name: "blocked", Interval: time.Hour, Collector: blocked}}, sink.emit)

	done := make(chan struct{})
	go func() { _ = s.Run(ctx); close(done) }()

	waitFor(t, time.Second, func() bool { return blocked.pollCount() >= 1 })
	cancel()
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Run did not return promptly after cancel (blocked Poll not interrupted)")
	}
	if sink.count() != 0 {
		t.Fatalf("sink got %d facts, want 0 (poll was canceled)", sink.count())
	}
}

// TestSchedulerInvalidConfig：nil Sink / 空源名 / nil collector 在 Run 时 fail fast。
func TestSchedulerInvalidConfig(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	c := &fakeCollector{kind: fact.KindFunding}
	noop := func(context.Context, []fact.Fact) error { return nil }
	cases := []Scheduler{
		{Sources: []Named{{Name: "x", Collector: c}}},
		{Sources: []Named{{Name: "", Collector: c}}, Sink: noop},
		{Sources: []Named{{Name: "x"}}, Sink: noop},
	}
	for i, s := range cases {
		if err := s.Run(ctx); err == nil {
			t.Errorf("case %d: Run = nil error, want error", i)
		}
	}
}
