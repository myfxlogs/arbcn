package main

import (
	"context"
	"errors"
	"testing"
	"time"
)

// fakeSeeder 记录失败序列：前 failN 次返回错误，之后成功。
type fakeSeeder struct {
	failN   int
	calls   int
	capital float64
}

func (f *fakeSeeder) InitSimAccount(_ context.Context, capital float64) error {
	f.calls++
	f.capital = capital
	if f.calls <= f.failN {
		return errors.New("connection refused")
	}
	return nil
}

// sleepRec 记录 sleep 调用并立即放行（不真等）。
type sleepRec struct {
	calls int
	dur   []time.Duration
}

func (s *sleepRec) sleep(d time.Duration) bool {
	s.calls++
	s.dur = append(s.dur, d)
	return true
}

// TestSeedSimAccountRetrySurvivesBootRace（对抗锚点）：PG 开机慢 2 个重试周期内就绪
// -> seed 最终成功。删重试循环（首次失败即返回）-> 本测试必红。
func TestSeedSimAccountRetrySurvivesBootRace(t *testing.T) {
	f := &fakeSeeder{failN: 2}
	s := &sleepRec{}
	err := seedSimAccountRetry(context.Background(), f, 100_000, 9, 10*time.Second, s.sleep)
	if err != nil {
		t.Fatalf("want nil err after transient failures, got %v", err)
	}
	if f.calls != 3 {
		t.Fatalf("want 3 attempts (2 fail + 1 ok), got %d", f.calls)
	}
	if f.capital != 100_000 {
		t.Fatalf("capital not passed through: %v", f.capital)
	}
	if s.calls != 2 {
		t.Fatalf("want 2 sleeps, got %d", s.calls)
	}
	for _, d := range s.dur {
		if d != 10*time.Second {
			t.Fatalf("sleep duration %v != configured wait", d)
		}
	}
}

// TestSeedSimAccountRetryBounded（对抗锚点）：持续失败 -> 恰好 attempts 次后返回错误，
// 不无限循环。删有界（attempts 不封顶）-> calls 断言必红。
func TestSeedSimAccountRetryBounded(t *testing.T) {
	f := &fakeSeeder{failN: 1 << 30} // 永远失败
	s := &sleepRec{}
	err := seedSimAccountRetry(context.Background(), f, 1, 9, 10*time.Second, s.sleep)
	if err == nil {
		t.Fatal("want error after exhausted attempts, got nil")
	}
	if f.calls != 9 {
		t.Fatalf("want exactly 9 attempts, got %d", f.calls)
	}
	if s.calls != 8 {
		t.Fatalf("want 8 sleeps (last failure no sleep), got %d", s.calls)
	}
}

// TestSeedSimAccountRetryFirstTryOK：PG 已就绪（正常重启路径）-> 零重试零等待。
func TestSeedSimAccountRetryFirstTryOK(t *testing.T) {
	f := &fakeSeeder{}
	s := &sleepRec{}
	if err := seedSimAccountRetry(context.Background(), f, 1, 9, 10*time.Second, s.sleep); err != nil {
		t.Fatalf("want nil, got %v", err)
	}
	if f.calls != 1 || s.calls != 0 {
		t.Fatalf("want 1 call 0 sleeps, got %d calls %d sleeps", f.calls, s.calls)
	}
}

// TestSeedSimAccountRetryCtxCancel：shutdown 信号到达（sleep 中 ctx 取消）-> 立即退出
// 不拖到耗尽（优雅停机不被 60s 重试窗口卡住）。
func TestSeedSimAccountRetryCtxCancel(t *testing.T) {
	f := &fakeSeeder{failN: 1 << 30}
	ctx, cancel := context.WithCancel(context.Background())
	s := func(time.Duration) bool {
		cancel()
		return false
	}
	err := seedSimAccountRetry(ctx, f, 1, 9, 10*time.Second, s)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("want context.Canceled, got %v", err)
	}
	if f.calls != 1 {
		t.Fatalf("want 1 attempt then ctx exit, got %d", f.calls)
	}
}
