package collect

import (
	"context"
	"sync"

	"arbcn/internal/fact"
)

// dedupKey 是连续重复事实去重的唯一键：按 (kind, venue, symbol) 记忆最后 (value, ts)。
// venue 参与 key（不同源同名 symbol 不冲突，M2-a §3.1 边界）。
type dedupKey struct {
	kind, venue, symbol string
}

// dedupSet 是 Scheduler Sink 的幂等去重包装（M2-a §3.1，P3 单点）。
// 相同 (value, ts) 跳过（不调 Sink）；value 变或 ts 变 → 照常落库。
// 多源 goroutine 共享 → 内部互斥保护。heartbeat fact 不流经本包装（alert.Heartbeat
// 直接 st.InsertFacts），故不受影响；即便流入，其 value 每周期变化也会照常落库。
type dedupSet struct {
	mu   sync.Mutex
	last map[dedupKey]fact.Fact
}

// newDedupSet 返回空去重集。
func newDedupSet() *dedupSet {
	return &dedupSet{last: make(map[dedupKey]fact.Fact)}
}

// emit 过滤与上条重复的 fact，仅对去重后保留的 batch 调用 next（Sink）。
// 整批全重复 → 不调 next 直接返回 nil。互斥覆盖 map 读改写。
func (d *dedupSet) emit(ctx context.Context, fs []fact.Fact, next Sink) error {
	d.mu.Lock()
	kept := make([]fact.Fact, 0, len(fs))
	for _, f := range fs {
		k := dedupKey{f.Kind, f.Venue, f.Symbol}
		if prev, ok := d.last[k]; ok && prev.Value == f.Value && prev.Ts.Equal(f.Ts) {
			continue
		}
		d.last[k] = f
		kept = append(kept, f)
	}
	d.mu.Unlock()

	if len(kept) == 0 {
		return nil
	}
	return next(ctx, kept)
}
