package collect

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"arbcn/internal/fact"
)

// fakeCollector：可控失败/延迟/计数的假源（registry/config/sched 测试共用）。
type fakeCollector struct {
	kind  string
	out   []fact.Fact
	delay time.Duration

	mu    sync.Mutex
	failN int // 前 failN 次 Poll 返回错误
	polls int
}

func (f *fakeCollector) Kind() string { return f.kind }

func (f *fakeCollector) Poll(ctx context.Context) ([]fact.Fact, error) {
	f.mu.Lock()
	f.polls++
	fail := f.failN > 0
	if fail {
		f.failN--
	}
	f.mu.Unlock()
	if f.delay > 0 {
		select {
		case <-time.After(f.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if fail {
		return nil, errors.New("fake poll failure")
	}
	return f.out, nil
}

func (f *fakeCollector) pollCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.polls
}

// recSink：记录收到的事实（并发安全）。
type recSink struct {
	mu sync.Mutex
	fs []fact.Fact
}

func (r *recSink) emit(_ context.Context, fs []fact.Fact) error {
	r.mu.Lock()
	r.fs = append(r.fs, fs...)
	r.mu.Unlock()
	return nil
}

func (r *recSink) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.fs)
}

// waitFor：cond 在 d 内成立，否则失败（异步断言轮询条件）。
func waitFor(t *testing.T, d time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("waitFor: condition not met")
}
