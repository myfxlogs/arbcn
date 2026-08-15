// Package collect：采集框架（docs/design/02-monitor-architecture.md §4/§10）。
// 每个数据源 = 一个 collector，插件化注册；每源独立 goroutine、独立间隔/重试，
// 单源故障不拖垮全局。采集事实经 Sink 落库，规则引擎只认 Fact 不认来源。
package collect

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"arbcn/internal/fact"
)

// Collector 是单一数据源的最小抽象（§4）：轮询只读公开 API，产出归一 Fact。
// 铁律：无密钥、无写动作（§1）；实现必须尊重 ctx 取消（优雅退出依赖它）。
type Collector interface {
	Kind() string                                  // 如 fact.KindFunding；一个 collector 只产一种 Kind
	Poll(ctx context.Context) ([]fact.Fact, error) // 只读公开 API
}

// Named 把注册名、默认轮询间隔与 collector 绑在一起（Scheduler 的输入单元）。
type Named struct {
	Name      string
	Interval  time.Duration
	Collector Collector
}

// Registry 名称 → collector 注册表；重名注册拒绝（配置错误 fail fast）。
type Registry struct {
	mu sync.RWMutex
	m  map[string]Collector
}

// NewRegistry 返回空注册表。
func NewRegistry() *Registry {
	return &Registry{m: make(map[string]Collector)}
}

// Register 注册 collector；空名 / nil / 重名返回错误。
func (r *Registry) Register(name string, c Collector) error {
	if name == "" || c == nil {
		return fmt.Errorf("collect: register %q: name and collector required", name)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, dup := r.m[name]; dup {
		return fmt.Errorf("collect: register %q: duplicate", name)
	}
	r.m[name] = c
	return nil
}

// Get 按名取 collector；未注册返回 false。
func (r *Registry) Get(name string) (Collector, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.m[name]
	return c, ok
}

// Names 返回全部注册名（升序，输出稳定）。
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.m))
	for n := range r.m {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}
