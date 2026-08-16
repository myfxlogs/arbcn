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

// LedgerEntry 是 ledger 表的持久化记录（M2-b §6 台账起步）。
// 出入金流水：date 出入金日期 / channel 通道 / currency 币种 / amount 金额（正入负出）/
// fee_rate 费率 % / tier 档位（entry 自带，不推断）/ note 备注。
type LedgerEntry struct {
	ID       int64     // 主键（读取时回填）
	Date     time.Time // 出入金日期
	Channel  string    // 通道（binance / okx / 民营定期 / 逆回购 ...）
	Currency string    // 币种（RMB / USD / USDT / USDC / BTC ...）
	Amount   float64   // 金额；正 = 入金，负 = 出金
	FeeRate  float64   // 费率 %（0 = 无）
	Tier     string    // 档位（Tier* 常量；entry 自带，不推断）
	Note     string    // 备注
}

// Tier 值域（D-026 三档 + 持有层单列可选；entry 自带，前端映射中文标签）。
const (
	TierProtectedConvexity = "protected_convexity" // 保本凸性（10 万 A 半：民营定期+现金管理+期权凸性）
	TierStableBase         = "stable_base"         // 稳定币基档（10 万 B 半：CEX 定期 + 自托管 DeFi）
	TierCashManagement     = "cash_management"     // 现金管理（国内底仓：货基/逆回购时点）
	TierHolding            = "holding"             // 持有层（业主自选方向敞口，单列；TRX 等）
)

// TierSummary 按 tier 归因汇总（M2-b §6：GROUP BY tier 简单分组）。
type TierSummary struct {
	Tier       string  // 档位（Tier* 常量）
	Inflow     float64 // 累计入金（amount > 0 和）
	Outflow    float64 // 累计出金（amount < 0 的绝对值）
	Net        float64 // 净额（Inflow − Outflow）
	EntryCount int     // 笔数
}

// SimOrder 是 sim_orders 表的持久化记录（04-m3-spec §1.1 建议订单）。
// 生成时风险门禁已执行（risk_flags 落库）；拒单保留为负样本（status=rejected + note）。
type SimOrder struct {
	ID             int64     // 主键（读取时回填）
	Ts             time.Time // 生成时刻
	SrcRule        string    // 触发规则名（funding_warn / defi_* 等）
	Kind           string    // 套利类型（SimKind*）
	Venue          string    // sim_local / binance_testnet / okx_demo
	Symbol         string    // 标的（如 BTCUSDT）
	Side           string    // SimSide*（long / short / hedge）
	Qty            float64   // 名义数量（quote 币种，模拟 USD）
	RefPrice       float64   // 生成时参考价
	ExpectedSpread float64   // 预期年化价差 %
	RiskFlags      []string  // 门禁未过标记（Risk*；空 = 全过）
	Status         string    // SimStatus*
	Note           string    // 拒单原因 / 结算备注
}

// EntityHit 是规则命中实体的描述（M3-b §9.2：S1 驱动组装 Signal 的输入）。
// 由规则引擎在 armed→active 转变时从命中匹配映射而来（含全局模式：venue/symbol 为空）。
type EntityHit struct {
	Venue  string
	Symbol string
	Value  float64
}

// SimPosition 是 sim_positions 表的持久化记录（04-m3-spec §1.2 模拟成交腿）。
// hedge = 两行（现货 long + 永续 short），carry/repo = 一行。pnl 按 funding 周期累计。
type SimPosition struct {
	ID        int64     // 主键（读取时回填）
	OrderID   int64     // 来源订单（sim_orders.id）
	Ts        time.Time // 建仓时刻
	Kind      string    // SimKind*
	Venue     string
	Symbol    string
	Side      string // 腿方向（long / short）
	Qty       float64
	RefPrice  float64
	Funding   bool      // 资金费率结算腿（funding_hedge 永续腿 / carry 生息腿）
	PnL       float64   // 累计已结算 PnL（模拟 USD）
	Status    string    // SimPosStatus*
	UpdatedAt time.Time // 最近结算时刻（读取时回填）
}

// TestnetAccount 是 sim_testnet_accounts 表的一行（D-040 测试网账户区数据面）。
// Source 是主键（sim_testnet_binance / sim_testnet_okx）；UpdatedAt 读取时回填。
// EquityUSD 口径因 Source 而异（诚实标注，前端明示）：binance = 稳定币合计近似
// （无行情折算非稳定币），okx = totalEq（交易所精确）。
type TestnetAccount struct {
	Source       string                 // 探针源（simtestnet.SourceBinanceTestnet / SourceOKXDemo）
	AccountAlias string                 // binance accountAlias / okx 无
	EquityUSD    float64                // 折合 USD（口径见上）
	Details      []TestnetAccountDetail // 每资产余额明细（JSONB）
	UpdatedAt    time.Time              // 最近一次余额查询成功时刻（读取时回填）
}

// TestnetAccountDetail 单资产余额。Asset/Balance 保留 API 原字符串（避免浮点精度）；
// EquityUSD 有 USD 折算则填（okx eqUsd / binance 稳定币 = balance），未知 = 0（前端标 —）。
type TestnetAccountDetail struct {
	Asset     string
	Balance   string
	EquityUSD float64
}

// KnowledgeEntry 是 knowledge_entries 表的一行（D-046 市场结构经验库）。
// 吸收 = 人工 + D#（internal/knowledge.Defaults 落盘，git 跟踪）；匹配 = 确定性签名纯函数；
// 呈现 = 只读——系统永不自动吸收/自动改 verdict（practices #20）。ID/Ts 读取时回填。
type KnowledgeEntry struct {
	ID             int64      // 主键（读取时回填）
	Ts             time.Time  // 吸收时刻（读取时回填）
	Signature      string     // 受控签名键（knowledge.Signature*）
	Venue          string     // seed 实例 venue（溯源用）
	Symbol         string     // seed 实例 symbol
	Verdict        string     // 人工判定（D# 落）
	Rationale      string     // 判定依据（中文）
	Source         string     // 出处（对话 #N / D#）
	Status         string     // active / superseded / retracted（D# 演进）
	ValidatedAt    *time.Time // 复核时刻；nil = 待复核
	ValidationNote string     // 复核结论
}

// SimOrder 值域（与 DB CHECK / spec 一致）。
const (
	SimStatusSuggested = "suggested" // 生成时默认（门禁全过）
	SimStatusConfirmed = "confirmed" // 人工确认（M3-c UI 流）
	SimStatusFilled    = "filled"    // 本地模拟成交
	SimStatusRejected  = "rejected"  // 门禁未过（负样本）
	SimStatusExpired   = "expired"

	SimKindFundingHedge = "funding_hedge" // 现货+永续对冲（cash-and-carry）
	SimKindCarryAsset   = "carry_asset"   // 白名单生息资产（sUSDe/USDe 等）
	SimKindRepo         = "repo"          // 逆回购/现金等价（天然无方向敞口）

	SimSideLong  = "long"
	SimSideShort = "short"
	SimSideHedge = "hedge" // 对冲结构（订单级；sim_positions 腿级用 long/short）

	SimPosStatusOpen    = "open"
	SimPosStatusSettled = "settled"
)

// Store 是监控管线的持久化抽象（facts/rules/trigger_states）。
type Store interface {
	// InsertFacts 批量写入；空切片无操作；含非法 Fact 整批拒绝。
	InsertFacts(ctx context.Context, facts []fact.Fact) error
	// QueryFacts 按 FactQuery 过滤（索引 (kind, symbol, ts) 友好）。
	QueryFacts(ctx context.Context, q FactQuery) ([]fact.Fact, error)
	// LatestFacts 每 (kind, venue, symbol) 返回一条最新事实（按 ts 取最大）；
	// 空参数 = 不过滤（M1-g 仪表盘机会面板快照）。
	LatestFacts(ctx context.Context, kind, venue, symbol string) ([]fact.Fact, error)
	// UpsertRule 按 Name 确保规则存在并返回 id；已存在不覆盖（保留 DB 人工编辑，02 §4）。
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
	// ListUnacked 返回全部未读告警（acked=false；ts DESC, id DESC）。
	// 未读数小，一次拉全（M2-a §1.1 铃铛）。
	ListUnacked(ctx context.Context) ([]Alert, error)
	// AckAll 全部已读（单事务 UPDATE alerts SET acked=true WHERE acked=false），
	// 返回本次确认的告警数（M2-a §1.2）。
	AckAll(ctx context.Context) (int64, error)
	// InsertLedgerEntry 追加台账行（M2-b §6）；返回新行 id。
	InsertLedgerEntry(ctx context.Context, e LedgerEntry) (int64, error)
	// ListLedgerEntries 按 date DESC, id DESC 分页返回台账流水。
	// limit ≤ 0 = 默认 100，offset < 0 = 0。
	ListLedgerEntries(ctx context.Context, limit, offset int) ([]LedgerEntry, error)
	// LedgerSummary 按 tier 分组归因汇总（GROUP BY tier；空 tier 归入 ""）。
	LedgerSummary(ctx context.Context) ([]TierSummary, error)

	// InsertSimOrder 追加建议订单（sim_orders）；返回新行 id。ts 零值 = now。
	// kind/symbol/side 必填；qty ≤ 0 拒绝（M3-a §3）。
	InsertSimOrder(ctx context.Context, o SimOrder) (int64, error)
	// ListSimOrders 按 ts DESC, id DESC 分页返回建议订单（稳定排序）。
	// limit ≤ 0 = 默认 100，offset < 0 = 0。
	ListSimOrders(ctx context.Context, limit, offset int) ([]SimOrder, error)
	// GetSimOrder 按 id 取单条订单；未知 id 返回 ErrNotFound。
	GetSimOrder(ctx context.Context, id int64) (SimOrder, error)
	// UpdateSimOrderStatus 更新订单状态（suggested→confirmed→filled/rejected/expired）；
	// note 非空时覆盖备注。未知 id 无操作。
	UpdateSimOrderStatus(ctx context.Context, id int64, status, note string) error
	// FillSimOrder 原子成交（M3-a 复审 M1）：单事务内「订单 confirmed→filled + 建全部
	// sim_positions 腿」，任一失败整体回滚——不留"filled 但缺腿"的半对冲状态（D-019）。
	// 订单不存在或非 confirmed → 拒绝（防状态漂移/并发双插）。legs 的 OrderID 缺省回填 id。
	FillSimOrder(ctx context.Context, id int64, note string, legs []SimPosition) error
	// AcceptSimOrder 人工确认原子成交（M3-c C3，practices #8）：单事务内
	// suggested→confirmed→filled + 建全部 sim_positions 腿；任一守卫 RowsAffected 为 0
	// 整体回滚。事务内 confirmed 是中间态，外部只见 suggested/filled（无"已确认未成交"
	// 悬挂）；并发双确认 → 守卫 1（status='suggested'）拦第二次（无重复建腿）。
	// 语义与 FillSimOrder（confirmed→filled）互补：AcceptSimOrder 是 M3-c 人工流从
	// suggested 一次性确认成交。legs 的 OrderID 缺省回填 id。
	AcceptSimOrder(ctx context.Context, id int64, note string, legs []SimPosition) error
	// RejectSimOrder 确认时拒单（M3-c C3）：原子置 rejected + note 覆盖 + risk_flags
	// 追加 flags（去重）。仅 status='suggested' 时生效（RowsAffected 守卫），未知 id /
	// 非 suggested → 报错（并发/状态漂移最后防线）。拒单 = 负样本保留（04-m3-spec §4）。
	RejectSimOrder(ctx context.Context, id int64, reason string, flags ...string) error
	// TodaySimNotional 当日（[当日 00:00, now) 本地日）活跃订单
	// （suggested/confirmed/filled）名义和——DAILY_OVER 门禁数据面（04-m3-spec §4）。
	TodaySimNotional(ctx context.Context, now time.Time) (float64, error)
	// UpsertTestnetAccount 幂等 upsert 测试网账户快照（source 主键；updated_at = DB now()）。
	// Source 必填；Details 保留原序（D-040 探针成功路径）。
	UpsertTestnetAccount(ctx context.Context, a TestnetAccount) error
	// ListTestnetAccounts 返回全部账户快照（source ASC）；无数据 = 空切片。
	ListTestnetAccounts(ctx context.Context) ([]TestnetAccount, error)
	// InsertSimPosition 追加模拟成交腿；返回新行 id。order_id 必填，qty ≤ 0 拒绝。
	InsertSimPosition(ctx context.Context, p SimPosition) (int64, error)
	// ListSimPositions 按 ts DESC, id DESC 分页返回持仓腿（稳定排序）。
	ListSimPositions(ctx context.Context, limit, offset int) ([]SimPosition, error)
	// ListOpenSimPositions 返回 open 持仓腿（symbol 空 = 不限；venue 空 = 不限；
	// ts ASC 建仓序）。M3-b §9.3：按 (symbol,venue) 分组结算，避免跨 venue 污染。
	ListOpenSimPositions(ctx context.Context, symbol, venue string) ([]SimPosition, error)
	// SettleSimPosition 结算更新持仓腿：pnl += addPnl，status 覆盖（settled 关闭），
	// updated_at = now。未知 id 无操作。
	SettleSimPosition(ctx context.Context, id int64, addPnl float64, status string) error

	// ListKnowledgeEntries 返回经验库全部条目（signature ASC 稳定排序；D-046 浏览）。
	ListKnowledgeEntries(ctx context.Context) ([]KnowledgeEntry, error)
	// UpsertKnowledgeEntry 按 signature 确保条目存在并返回 id（镜像 UpsertRule：已存在
	// **不覆盖**，保留 DB 后续人工修订）。seed 落盘幂等。
	UpsertKnowledgeEntry(ctx context.Context, e KnowledgeEntry) (int64, error)
	// ReviewKnowledgeEntry 人工复核（D-054）：写 validated_at=now + 生命周期 status +
	// 可选 verdict 判定文本（空 = 保留原判定）+ validation_note，只改判定记录不改规则/
	// 门禁；未知 signature 返回 ErrNotFound。
	ReviewKnowledgeEntry(ctx context.Context, signature, status, verdict, note string) error
}
