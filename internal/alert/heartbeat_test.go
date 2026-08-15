package alert

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"arbcn/internal/fact"
)

// t0 心跳测试基准时钟。
var t0 = time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)

func TestHeartbeatValueGrowsOnStall(t *testing.T) {
	clk := &fakeClock{t: t0}
	h := &Heartbeat{St: newMemStore(), Now: clk.now, Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	h.Track("binance_funding", 5*time.Minute)

	// 停摆 12 分钟（无 Record）：value = 12×60/300 = 2.4 > 2（规则阈值），
	// 且随停摆继续增长（4.0）——契约：独立发射方，值随停摆增长。
	clk.t = t0.Add(12 * time.Minute)
	f1 := h.next("binance_funding")
	if f1.Value != 2.4 {
		t.Fatalf("stalled value = %v, want 2.4 (>2 触发规则)", f1.Value)
	}
	clk.t = t0.Add(20 * time.Minute)
	f2 := h.next("binance_funding")
	if f2.Value <= f1.Value || f2.Value != 4.0 {
		t.Fatalf("value not growing: %v → %v, want 2.4 → 4.0", f1.Value, f2.Value)
	}
	for _, f := range []fact.Fact{f1, f2} {
		if f.Kind != fact.KindHeartbeat || f.Venue != "collector" || f.Symbol != "binance_funding" || f.Unit != fact.UnitRatio {
			t.Errorf("fact shape = %+v, want kind=heartbeat venue=collector symbol=binance_funding unit=ratio", f)
		}
	}
}

// TestHeartbeatRecordResets：成功轮询回调后 value 归零并重新从该时刻起算。
func TestHeartbeatRecordResets(t *testing.T) {
	clk := &fakeClock{t: t0}
	h := &Heartbeat{St: newMemStore(), Now: clk.now, Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	h.Track("okx_funding", 5*time.Minute)

	clk.t = t0.Add(10 * time.Minute)
	if v := h.next("okx_funding").Value; v != 2.0 {
		t.Fatalf("value before success = %v, want 2.0", v)
	}
	h.Record("okx_funding", clk.t) // 调度器成功回调
	clk.t = t0.Add(11 * time.Minute)
	if v := h.next("okx_funding").Value; v != 0.2 {
		t.Fatalf("value after reset = %v, want 0.2", v)
	}
}

// TestHeartbeatEmitsWhileStalled：独立定时器——无任何成功回调仍持续发射，
// 且 value 单调不减（真实时钟，间隔 1s，发射 2ms）。
func TestHeartbeatEmitsWhileStalled(t *testing.T) {
	st := newMemStore()
	h := &Heartbeat{St: st, Emit: 2 * time.Millisecond, Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	h.Track("okx_funding", time.Second)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { _ = h.Run(ctx); close(done) }()

	waitFor(t, 500*time.Millisecond, func() bool { return len(st.factsCopy()) >= 5 })
	fs := st.factsCopy()
	for i, f := range fs {
		if f.Kind != fact.KindHeartbeat || f.Symbol != "okx_funding" || f.Venue != "collector" {
			t.Fatalf("fact %d = %+v, want heartbeat/okx_funding/collector", i, f)
		}
		if i > 0 && f.Value+1e-9 < fs[i-1].Value {
			t.Fatalf("value decreased: %v → %v", fs[i-1].Value, f.Value)
		}
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not return after cancel")
	}
}

// TestHeartbeatTrackGuard：无效 Track 拒绝、未 Track 源的 Record 忽略；
// 无有效源时 Run 只等待 ctx（无 goroutine）。
func TestHeartbeatTrackGuard(t *testing.T) {
	h := &Heartbeat{St: newMemStore(), Log: slog.New(slog.NewTextHandler(io.Discard, nil))}
	h.Track("", 5*time.Minute)
	h.Track("bad_interval", 0)
	h.Record("ghost", time.Now()) // 未 Track：忽略不 panic
	if n := len(h.trackedNames()); n != 0 {
		t.Fatalf("tracked = %v, want empty", h.trackedNames())
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { _ = h.Run(ctx); close(done) }()
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not return after cancel")
	}
}

// TestHeartbeatNilStore：装配错误 fail fast。
func TestHeartbeatNilStore(t *testing.T) {
	h := &Heartbeat{}
	if err := h.Run(context.Background()); err == nil {
		t.Fatal("Run with nil store = nil error, want error")
	}
}
