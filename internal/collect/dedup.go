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

// emit 过滤重复 fact，仅对去重后保留的 batch 调用 next（Sink）。整批全重复 →
// 不调 next 直接返回 nil。互斥覆盖 map 读改写。
//
// 去重状态只在 next 成功后推进（R1#1 裁定）：若 Sink 失败（如 PG 暂不可达），
// 本轮事实不得被标记已见——否则退避重试后相同 (value,ts) 会被当重复跳过，
// 数据永久丢失且心跳误判成功，DB 故障被静默屏蔽。batch 内重复用局部 seen 判定
//（同一批出现重复 key 也去重，见 TestDedupKeyIsolation）；跨批靠 d.last。
// 成功前不写 d.last 的窗口内，多源并发产出同一 key 的概率可忽略（每 key 单一生产者）。
func (d *dedupSet) emit(ctx context.Context, fs []fact.Fact, next Sink) error {
	d.mu.Lock()
	kept := make([]fact.Fact, 0, len(fs))
	keys := make([]dedupKey, 0, len(fs))
	seen := make(map[dedupKey]fact.Fact, len(fs)) // 本轮已保留（batch 内去重）
	for _, f := range fs {
		k := dedupKey{f.Kind, f.Venue, f.Symbol}
		prev, ok := d.last[k]
		if !ok {
			prev, ok = seen[k]
		}
		if ok && prev.Value == f.Value && prev.Ts.Equal(f.Ts) {
			continue
		}
		seen[k] = f
		kept = append(kept, f)
		keys = append(keys, k)
	}
	d.mu.Unlock()

	if len(kept) == 0 {
		return nil
	}
	if err := next(ctx, kept); err != nil {
		return err // Sink 失败：去重状态未推进，重试可重新投递
	}
	d.mu.Lock()
	for i, f := range kept {
		d.last[keys[i]] = f
	}
	d.mu.Unlock()
	return nil
}
