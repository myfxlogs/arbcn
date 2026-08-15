// 元监控心跳发射方（M1-e 契约定稿，dialogue #26）：
// kind=heartbeat、symbol=源名、venue=collector、value=距该源上次成功轮询秒数
// ÷ 该源轮询间隔（错过的窗口数）；规则 collector_heartbeat 阈值 last_24h > 2
// 已就位（internal/rule/defaults.go）。
//
// 发射方 = 独立定时器：每源一个 goroutine，与采集循环解耦——源停摆后仍持续
// 发射且 value 随停摆增长；若心跳随源停摆而停，停摆源不再"自报"，检测即失效。
//
// 挂接方式（只读）：M1-h 接线时对每个启用源调 Track(name, interval)，
// Scheduler.OnSuccess = h.Record 登记成功时刻；Run 启动发射循环。
package alert

import (
	"context"
	"errors"
	"log/slog"
	"sort"
	"sync"
	"time"

	"arbcn/internal/fact"
	"arbcn/internal/store"
)

// Heartbeat 是心跳发射方实例；Now 为测试注入时钟，0 = time.Now。
type Heartbeat struct {
	St   store.Store      // 必填；心跳 fact 落库（InsertFacts）
	Now  func() time.Time // 0 = time.Now
	Emit time.Duration    // 每源发射间隔；0 = 30s
	Log  *slog.Logger     // 0 = slog.Default()

	mu      sync.Mutex
	tracked map[string]time.Duration // 源名 → 轮询间隔（Track 声明）
	lastOK  map[string]time.Time     // 源名 → 上次成功轮询（Record 写，发射读）
}

// Track 声明被监控的源（Run 前调用）。无效参数拒绝（配置错误不静默）。
func (h *Heartbeat) Track(name string, interval time.Duration) {
	if name == "" || interval <= 0 {
		h.log().Warn("heartbeat: Track rejected", "name", name, "interval", interval)
		return
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.tracked == nil {
		h.tracked = map[string]time.Duration{}
		h.lastOK = map[string]time.Time{}
	}
	h.tracked[name] = interval
	if _, ok := h.lastOK[name]; !ok {
		h.lastOK[name] = h.now() // 起点 = 声明时刻；首轮成功会重置
	}
}

// Record 是 Scheduler.OnSuccess 的回调实现：登记源最近一次成功轮询时刻。
// 未 Track 的源名忽略（不会为其启动发射 goroutine）。
func (h *Heartbeat) Record(name string, at time.Time) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.tracked[name]; !ok {
		return
	}
	h.lastOK[name] = at
}

// Run 每源一个发射 goroutine；阻塞至 ctx 取消且全部退出（优雅退出）。
// 发射先于等待（boot 即发一条，value≈0）。
func (h *Heartbeat) Run(ctx context.Context) error {
	if h.St == nil {
		return errors.New("alert: heartbeat: nil store")
	}
	var wg sync.WaitGroup
	for _, name := range h.trackedNames() {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			h.runSource(ctx, name)
		}(name)
	}
	<-ctx.Done()
	wg.Wait()
	return nil
}

// runSource 单源发射循环：落库失败只记日志（下一周期自然重试），不崩不阻塞他源。
func (h *Heartbeat) runSource(ctx context.Context, name string) {
	for {
		if err := h.St.InsertFacts(ctx, []fact.Fact{h.next(name)}); err != nil && ctx.Err() == nil {
			h.log().Warn("heartbeat insert failed", "source", name, "err", err)
		}
		if !sleepCtx(ctx, h.emit()) {
			return
		}
	}
}

// next 计算本周期心跳 fact；value = 距上次成功轮询秒数 ÷ 轮询间隔。
func (h *Heartbeat) next(name string) fact.Fact {
	now := h.now()
	h.mu.Lock()
	iv := h.tracked[name]
	last := h.lastOK[name]
	h.mu.Unlock()
	return fact.Fact{
		Kind:   fact.KindHeartbeat,
		Venue:  "collector",
		Symbol: name,
		Value:  now.Sub(last).Seconds() / iv.Seconds(),
		Unit:   fact.UnitRatio,
		Ts:     now,
		Src:    "heartbeat",
	}
}

func (h *Heartbeat) trackedNames() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	names := make([]string, 0, len(h.tracked))
	for n := range h.tracked {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func (h *Heartbeat) emit() time.Duration {
	if h.Emit > 0 {
		return h.Emit
	}
	return 30 * time.Second
}

func (h *Heartbeat) now() time.Time {
	if h.Now != nil {
		return h.Now()
	}
	return time.Now()
}

func (h *Heartbeat) log() *slog.Logger {
	if h.Log != nil {
		return h.Log
	}
	return slog.Default()
}
