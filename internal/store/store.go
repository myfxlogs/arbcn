// Package store：存储层接口（docs/design/02-monitor-architecture.md §4）。
// 纯接口 + 持久化记录类型，无实现依赖；PG 实现在 pgstore。
package store

import (
	"context"
	"errors"
	"time"

	"arbcn/internal/fact"
)

// ErrNotFound：查询无结果（如触发器状态尚未初始化）。
var ErrNotFound = errors.New("store: not found")

// Rule 是 rules 表的持久化记录（规则 = 配置，非代码，§4）。
type Rule struct {
	ID      int64
	Name    string // 唯一键
	Kind    string // 命中哪个 Fact 流（fact.Kind*）
	Cond    string // 条件表达式，由表达式求值器解析
	Level   string // LevelInfo / LevelWarn / LevelCritical
	Enabled bool
}

// Rule.Level 值域（与 DB CHECK 一致）。
const (
	LevelInfo     = "info"
	LevelWarn     = "warn"
	LevelCritical = "critical"
)

// TriggerState 是 trigger_states 表的持久化记录（告警状态机，§4）。
type TriggerState struct {
	RuleID    int64
	State     string // StateArmed / StateActive / StateResolved
	Since     time.Time
	LastValue float64
}

// TriggerState.State 值域（与 DB CHECK 一致）。
const (
	StateArmed    = "armed"
	StateActive   = "active"
	StateResolved = "resolved"
)

// FactQuery 时间窗查询条件；空字段 = 不筛选。
// 窗口 [From, To)；To 零值 = now；Limit ≤ 0 = 默认 1000。结果按 ts 升序。
type FactQuery struct {
	Kind   string
	Venue  string
	Symbol string
	From   time.Time
	To     time.Time
	Limit  int
}

// Store 是监控管线的持久化抽象（facts/rules/trigger_states）。
type Store interface {
	// InsertFacts 批量写入；空切片无操作；含非法 Fact 整批拒绝。
	InsertFacts(ctx context.Context, facts []fact.Fact) error
	// QueryFacts 按 FactQuery 过滤（索引 (kind, symbol, ts) 友好）。
	QueryFacts(ctx context.Context, q FactQuery) ([]fact.Fact, error)
	// UpsertRule 按 Name 幂等写入，返回 id。
	UpsertRule(ctx context.Context, r Rule) (int64, error)
	ListRules(ctx context.Context) ([]Rule, error)
	// GetTriggerState 无记录时返回 ErrNotFound。
	GetTriggerState(ctx context.Context, ruleID int64) (TriggerState, error)
	// PutTriggerState upsert；Since 零值 = now。
	PutTriggerState(ctx context.Context, s TriggerState) error
}
