// 规则引擎（docs/design/02-monitor-architecture.md §4/§7）：
// 从 store 加载声明式规则 → 解析 Cond（坏表达式启动即报错）→ 按每规则
// 独立间隔评估 → 告警状态机（armed→active→resolved，状态转变才写 alerts 行，
// 持续满足只更新 trigger_states.last_value）。
//
// 评估模型：规则 scope（venue/symbol，逗号分隔 = IN 列表）逐实体独立聚合，
// 任一实体命中即告警（如 funding 规则对 BTC/ETH 各算各的 30d 均值）；
// 含显式 scope 聚合的条件（阶梯陷阱）在规则全部事实集上全局评估一次。
package rule

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"

	"arbcn/internal/fact"
	"arbcn/internal/store"
)

// maxFacts 单规则单轮查询上限（30d × 5m 频率 ≈ 8.6k 行，留足余量）。
const maxFacts = 100_000

// Config 是引擎的注入项（与 collect.Scheduler 同构的注入风格）。
type Config struct {
	Now      func() time.Time               // 0 = time.Now（测试注入时钟）
	Interval func(store.Rule) time.Duration // 评估间隔覆盖；0 = 规则 interval_sec
	Log      *slog.Logger                   // 0 = slog.Default()
}

// Engine 是规则引擎实例。
type Engine struct {
	st    store.Store
	rules []evalRule
	now   func() time.Time
	ival  func(store.Rule) time.Duration
	log   *slog.Logger
}

// evalRule 是加载时解析好的单规则评估单元。
type evalRule struct {
	r        store.Rule
	cond     *condExpr
	lookback time.Duration
	venues   map[string]bool
	symbols  map[string]bool
}

// New 加载并解析全部规则；坏表达式 / 空规则集在此报错（配置错误 fail fast）。
func New(ctx context.Context, st store.Store, cfg Config) (*Engine, error) {
	if st == nil {
		return nil, errors.New("rule: engine: nil store")
	}
	rules, err := st.ListRules(ctx)
	if err != nil {
		return nil, fmt.Errorf("rule: engine: load rules: %w", err)
	}
	if len(rules) == 0 {
		return nil, errors.New("rule: engine: no rules in store")
	}
	ev := make([]evalRule, 0, len(rules))
	for _, r := range rules {
		c, err := parseCond(r.Cond)
		if err != nil {
			return nil, fmt.Errorf("rule %q: %w", r.Name, err)
		}
		ev = append(ev, evalRule{
			r:        r,
			cond:     c,
			lookback: c.lookback(),
			venues:   splitSet(r.Venue),
			symbols:  splitSet(r.Symbol),
		})
	}
	e := &Engine{st: st, rules: ev, now: cfg.Now, ival: cfg.Interval, log: cfg.Log}
	if e.now == nil {
		e.now = time.Now
	}
	if e.log == nil {
		e.log = slog.Default()
	}
	return e, nil
}

// EvaluateAll 全部启用规则各评估一轮（单规则失败即中止并返回错误）。
func (e *Engine) EvaluateAll(ctx context.Context) error {
	for _, r := range e.rules {
		if !r.r.Enabled {
			continue
		}
		if _, err := e.evaluateRule(ctx, r); err != nil {
			return err
		}
	}
	return nil
}

// Run 每启用规则一个 goroutine 按各自间隔评估（§7 调度）；阻塞至 ctx 取消且
// 全部退出。单规则失败按退避重试、不影响其他规则（与 collect.Scheduler 同构）。
func (e *Engine) Run(ctx context.Context) error {
	var wg sync.WaitGroup
	for _, r := range e.rules {
		if !r.r.Enabled {
			continue
		}
		wg.Add(1)
		go func(r evalRule) {
			defer wg.Done()
			e.runRule(ctx, r)
		}(r)
	}
	<-ctx.Done()
	wg.Wait()
	return nil
}

// runRule 单规则循环：先评估（boot 即评一次），成功后按间隔等待；
// 失败独立退避（1s..32s 封顶）；ctx 取消立即退出。
func (e *Engine) runRule(ctx context.Context, r evalRule) {
	attempt := 0
	for {
		if _, err := e.evaluateRule(ctx, r); err != nil {
			if ctx.Err() != nil {
				return
			}
			attempt++
			wait := time.Duration(1<<min(attempt, 5)) * time.Second
			e.log.Warn("rule eval failed, backing off", "rule", r.r.Name, "attempt", attempt, "retry_in", wait.String(), "err", err)
			if !sleepCtx(ctx, wait) {
				return
			}
			continue
		}
		attempt = 0
		if !sleepCtx(ctx, e.interval(r.r)) {
			return
		}
	}
}

// interval 规则评估间隔：Config.Interval 覆盖优先，否则 interval_sec（≤0 = 5m）。
func (e *Engine) interval(r store.Rule) time.Duration {
	if e.ival != nil {
		return e.ival(r)
	}
	if r.IntervalSec <= 0 {
		return 5 * time.Minute
	}
	return time.Duration(r.IntervalSec) * time.Second
}

// sleepCtx 可中断等待；ctx 取消返回 false。
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

// evaluateRule 单规则一轮：查询窗口事实 → 判定 → 状态机 + 落库。返回本轮是否命中。
func (e *Engine) evaluateRule(ctx context.Context, r evalRule) (bool, error) {
	fs, err := e.st.QueryFacts(ctx, store.FactQuery{
		Kind:  r.r.Kind,
		From:  e.now().Add(-r.lookback),
		Limit: maxFacts,
	})
	if err != nil {
		return false, fmt.Errorf("rule %q: query facts: %w", r.r.Name, err)
	}
	fs = filterScope(fs, r.venues, r.symbols)
	hit, matches := e.judge(r.cond, fs)
	st, err := e.st.GetTriggerState(ctx, r.r.ID)
	if err != nil && !errors.Is(err, store.ErrNotFound) {
		return false, fmt.Errorf("rule %q: get trigger state: %w", r.r.Name, err)
	}
	if errors.Is(err, store.ErrNotFound) {
		st = store.TriggerState{RuleID: r.r.ID, State: store.StateArmed}
	}
	return e.transition(ctx, r.r, st, hit, matches)
}

// judge 求值：全局模式（含显式 scope 聚合）一次；否则逐实体，任一命中即真。
func (e *Engine) judge(c *condExpr, fs []fact.Fact) (bool, []match) {
	if c.scoped {
		if ok, v := c.eval(fs, e.now()); ok {
			return true, []match{{value: v}}
		}
		return false, nil
	}
	var matches []match
	for key, grp := range groupByEntity(fs) {
		if ok, v := c.eval(grp, e.now()); ok {
			matches = append(matches, match{venue: key[0], symbol: key[1], value: v})
		}
	}
	sort.Slice(matches, func(i, j int) bool { // 消息输出稳定
		if matches[i].venue != matches[j].venue {
			return matches[i].venue < matches[j].venue
		}
		return matches[i].symbol < matches[j].symbol
	})
	return len(matches) > 0, matches
}

// groupByEntity 按 (venue,symbol) 分组（每组 ts 升序，来自 QueryFacts）。
func groupByEntity(fs []fact.Fact) map[[2]string][]fact.Fact {
	groups := make(map[[2]string][]fact.Fact, 8)
	for _, f := range fs {
		key := [2]string{f.Venue, f.Symbol}
		groups[key] = append(groups[key], f)
	}
	return groups
}

// filterScope 按规则 scope 过滤（空集 = 不限）。
func filterScope(fs []fact.Fact, venues, symbols map[string]bool) []fact.Fact {
	if len(venues) == 0 && len(symbols) == 0 {
		return fs
	}
	out := make([]fact.Fact, 0, len(fs))
	for _, f := range fs {
		if len(venues) > 0 && !venues[f.Venue] {
			continue
		}
		if len(symbols) > 0 && !symbols[f.Symbol] {
			continue
		}
		out = append(out, f)
	}
	return out
}

// splitSet 解析逗号分隔的 scope 字段；空 = 空集（不限）。
func splitSet(s string) map[string]bool {
	set := map[string]bool{}
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			set[p] = true
		}
	}
	return set
}
