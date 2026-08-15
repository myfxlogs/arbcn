// Package fact：统一事实模型（docs/design/02-monitor-architecture.md §4）。
// 规则引擎只认 Fact 不认来源；Value 口径由 Kind 约定。
package fact

import (
	"fmt"
	"time"
)

// Kind 常量表（§4 Fact.Kind；按 §5 数据源扩充：ticker / deposit_rate）。
const (
	KindFunding     = "funding"      // 永续资金费率（UnitPctAnnualized）
	KindDefiRate    = "defi_rate"    // 链上/DeFi 收益产品利率（UnitPctAnnualized）
	KindReverseRepo = "reverse_repo" // 交易所逆回购 GC001/R-001（UnitPctAnnualized）
	KindDepositRate = "deposit_rate" // 银行挂牌存款利率（§5 Domestic；UnitPctAnnualized）
	KindFX          = "fx"           // 汇率 USDCNH（UnitPrice）
	KindIV          = "iv"           // 隐含波动率 DVOL（UnitPct）
	KindCalendar    = "calendar"     // 日历事件（季末/月末/国债发行；UnitDays）
	KindTicker      = "ticker"       // 行情价格（§5 Exchange collector；UnitPrice）
	KindHeartbeat   = "heartbeat"    // collector 心跳（UnitRatio；M1-f 发射，M1-e 只消费）
)

// Unit 常量表（§4；口径由 Kind 约定）。
const (
	UnitPctAnnualized = "pct_annualized" // 年化 %
	UnitPrice         = "price"          // 价格（如 USDCNH 7.25）
	UnitPct           = "pct"            // 百分比（如 IV 45%）
	UnitDays          = "days"           // 天数（日历事件倒计时）
	UnitRatio         = "ratio"          // 无量纲比率（心跳 = 距上次成功轮询秒数 ÷ 该源轮询间隔）
)

// Fact 是采集 → 归一 → 规则的唯一数据载体。
type Fact struct {
	Kind   string
	Venue  string
	Symbol string
	Value  float64
	Unit   string
	Ts     time.Time
	Src    string
}

// validKinds 是已知 Kind 的值域（P4：值域约束前置到模型层）。
var validKinds = map[string]bool{
	KindFunding:     true,
	KindDefiRate:    true,
	KindReverseRepo: true,
	KindDepositRate: true,
	KindFX:          true,
	KindIV:          true,
	KindCalendar:    true,
	KindTicker:      true,
	KindHeartbeat:   true,
}

// ValidKind 报告 k 是否为已知 Kind。
func ValidKind(k string) bool { return validKinds[k] }

// Validate 校验 Fact；未知 Kind 拒绝。
func (f Fact) Validate() error {
	if !ValidKind(f.Kind) {
		return fmt.Errorf("fact: unknown kind %q", f.Kind)
	}
	return nil
}
