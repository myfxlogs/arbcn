package collect

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"arbcn/internal/fact"
)

// Sink 接收轮询产出（把 Fact 写入存储层）；返回错误触发该源重试退避。
type Sink func(ctx context.Context, fs []fact.Fact) error

// Scheduler 每源一个 goroutine（§10）：独立间隔 + 抖动（防限流）+ 独立重试退避；
// 单源故障不影响其他源。Run 阻塞至 ctx 取消且全部 goroutine 退出（优雅退出）。
type Scheduler struct {
	Sources     []Named
	Sink        Sink                            // 必填；事实落库的唯一下游
	PollTimeout time.Duration                   // 单次 Poll 超时；0 = 10s
	Backoff     func(attempt int) time.Duration // 失败退避；0 = 1s,2s,…封顶 32s
	Jitter      func() float64                  // [0,1) 间隔抖动因子；0 = math/rand
	Log         *slog.Logger                    // 0 = slog.Default()
	// OnSuccess 在每次轮询成功（Poll+Sink 均无错）后回调（源名, 成功时刻）。
	// 心跳元监控（internal/alert.Heartbeat.Record）挂接此处，只读不干预调度。
	OnSuccess func(name string, at time.Time)
	// Dedup 为 true 时按 (kind, venue, symbol) 去重：与上条 (value, ts) 相同 → 跳过
	// 落库（M2-a §3.1）。共享 map + 互斥，对全部源统一生效；Collector 无感。
	Dedup bool

	dedup *dedupSet // 惰性装配（Run 前置，避免 goroutine 竞态）
}

// Run 启动全部源并阻塞直至 ctx 取消。装配错误（nil Sink / 空源）直接返回；
// 运行期错误走重试退避 + 日志，不中断进程。
func (s *Scheduler) Run(ctx context.Context) error {
	if s.Sink == nil {
		return errors.New("collect: scheduler: nil Sink")
	}
	for _, src := range s.Sources {
		if src.Name == "" || src.Collector == nil {
			return fmt.Errorf("collect: scheduler: invalid source %q", src.Name)
		}
	}
	if s.Dedup {
		s.dedup = newDedupSet()
	}
	var wg sync.WaitGroup
	for _, src := range s.Sources {
		wg.Add(1)
		go func(src Named) {
			defer wg.Done()
			s.runSource(ctx, src)
		}(src)
	}
	<-ctx.Done()
	wg.Wait()
	return nil
}

// runSource 单源主循环：先轮询（boot 即取数），成功后等待抖动间隔；
// 失败独立重试退避；ctx 取消立即退出（不等待退避/间隔）。
func (s *Scheduler) runSource(ctx context.Context, src Named) {
	log := s.log().With("source", src.Name, "kind", src.Collector.Kind())
	attempt := 0
	for {
		if err := s.pollOnce(ctx, src); err != nil {
			if ctx.Err() != nil {
				return // 优雅退出
			}
			attempt++
			wait := s.backoff(attempt)
			log.Warn("poll failed, backing off", "attempt", attempt, "retry_in", wait.String(), "err", err)
			if !sleepCtx(ctx, wait) {
				return
			}
			continue
		}
		attempt = 0
		if s.OnSuccess != nil {
			s.OnSuccess(src.Name, time.Now())
		}
		wait := s.nextWait(src.Interval)
		log.Debug("poll ok", "next_in", wait.String())
		if !sleepCtx(ctx, wait) {
			return
		}
	}
}

// pollOnce 单次轮询 + 落库：Poll 带超时；Sink 失败视同本轮失败（退避重试）。
func (s *Scheduler) pollOnce(ctx context.Context, src Named) error {
	pctx, cancel := context.WithTimeout(ctx, s.pollTimeout())
	defer cancel()
	fs, err := src.Collector.Poll(pctx)
	if err != nil {
		return fmt.Errorf("poll: %w", err)
	}
	if err := s.sinkFacts(ctx, fs); err != nil {
		return fmt.Errorf("sink: %w", err)
	}
	return nil
}

// sinkFacts 落库入口：Dedup 开启时经去重包装（M2-a §3.1），否则直通 Sink。
func (s *Scheduler) sinkFacts(ctx context.Context, fs []fact.Fact) error {
	if !s.Dedup {
		return s.Sink(ctx, fs)
	}
	return s.dedup.emit(ctx, fs, s.Sink)
}

func (s *Scheduler) pollTimeout() time.Duration {
	if s.PollTimeout > 0 {
		return s.PollTimeout
	}
	return 10 * time.Second
}

func (s *Scheduler) backoff(attempt int) time.Duration {
	if s.Backoff != nil {
		return s.Backoff(attempt)
	}
	return time.Duration(1<<min(attempt, 5)) * time.Second // 1s..32s 封顶
}

// nextWait 在 interval 基础上加 ±10% 抖动，错开多源同刻请求（防限流）。
func (s *Scheduler) nextWait(interval time.Duration) time.Duration {
	j := rand.Float64()
	if s.Jitter != nil {
		j = s.Jitter()
	}
	return time.Duration(float64(interval) * (0.9 + 0.2*j))
}

func (s *Scheduler) log() *slog.Logger {
	if s.Log != nil {
		return s.Log
	}
	return slog.Default()
}

// sleepCtx 可中断等待；ctx 取消返回 false（调用方退出循环）。
func sleepCtx(ctx context.Context, d time.Duration) bool {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-t.C:
		return true
	}
}
