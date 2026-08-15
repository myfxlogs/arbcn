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
	ID          int64
	Name        string // 唯一键
	Kind        string // 命中哪个 Fact 流（fact.Kind*）
	Cond        string // 条件表达式，由表达式求值器解析
	Level       string // LevelInfo / LevelWarn / LevelCritical
	Enabled     bool
	Venue       string // 实体 scope：逗号分隔 IN 列表；空 = 不限
	Symbol      string // 实体 scope：同上
	IntervalSec int    // 评估间隔（秒）；≤0 = 默认 300
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

// Alert 是 alerts 表的持久化记录（状态机状态转变时由规则引擎写入，§4）。
// ID/RuleName/Delivered/Acked 是读取路径字段：写入时忽略。
type Alert struct {
	ID        int64 // 主键（读取时回填）
	RuleID    int64
	RuleName  string    // JOIN rules.name（邮件标题用；规则存在性由外键保证）
	Ts        time.Time // 零值 = 写入时刻
	Level     string    // LevelInfo / LevelWarn / LevelCritical
	Message   string
	Delivered bool // M1-f 投递状态（迁移 0003）
	Acked     bool // M1-g 仪表盘确认状态（0001 已建列；独立于 Delivered）
}

// RuleState 是触发器视图的联表投影（rules LEFT JOIN trigger_states）。
// 无状态行 = 尚未评估，语义等同 armed；Since 零值 = 尚未转变。
type RuleState struct {
	RuleName  string
	State     string // StateArmed / StateActive / StateResolved
	Since     time.Time
	LastValue *float64 // NULL = 从未评估
}

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
	// LatestFacts 每 (kind, venue, symbol) 返回一条最新事实（按 ts 取最大）；
	// 空参数 = 不过滤（M1-g 仪表盘机会面板快照）。
	LatestFacts(ctx context.Context, kind, venue, symbol string) ([]fact.Fact, error)
	// UpsertRule 按 Name 幂等写入，返回 id。
	UpsertRule(ctx context.Context, r Rule) (int64, error)
	ListRules(ctx context.Context) ([]Rule, error)
	// GetTriggerState 无记录时返回 ErrNotFound。
	GetTriggerState(ctx context.Context, ruleID int64) (TriggerState, error)
	// PutTriggerState upsert；Since 零值 = now。
	PutTriggerState(ctx context.Context, s TriggerState) error
	// InsertAlert 追加告警行（acked=false；Ts 零值 = now）。
	InsertAlert(ctx context.Context, a Alert) error
	// PendingAlerts 返回 delivered=false 的告警（ts 升序，最多 limit 条；
	// limit ≤ 0 = 默认 100）。Alerter 消费未投递行（M1-f）。
	PendingAlerts(ctx context.Context, limit int) ([]Alert, error)
	// MarkAlertDelivered 标记单条告警已投递（M1-f）。
	MarkAlertDelivered(ctx context.Context, id int64) error
	// ListAlerts 时间降序分页返回告警流（含 acked；ts DESC, id DESC；
	// limit ≤ 0 = 默认 100，offset < 0 = 0）（M1-g 仪表盘）。
	ListAlerts(ctx context.Context, limit, offset int) ([]Alert, error)
	// AckAlert 单条确认（幂等；未知 id 无操作）（M1-g 仪表盘）。
	AckAlert(ctx context.Context, id int64) error
	// ListTriggerStates 返回全部规则的触发器视图（未评估规则 = armed）
	// （M1-g 仪表盘）。
	ListTriggerStates(ctx context.Context) ([]RuleState, error)
}
