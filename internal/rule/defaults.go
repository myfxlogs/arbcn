// 首版规则集（docs/design/02-monitor-architecture.md §7 表 10 条全落；
// 阈值锚 docs/handoff/facts.md）。规则 = 配置行，M1-h 接线时经 Seed 落库；
// 改阈值 = 改 DB 一行，不发布版本（§4）。
//
// scope 约定：venue/symbol 逗号分隔 = IN 列表（空 = 不限）；逐实体独立聚合，
// 任一实体命中即告警。chg 语义：环比 = 最新采集值 − 紧邻上一采集值
// （±0.5% 用 "||" 双比较表达）。
//
// 各规则口径说明（实现决策，供复审）：
//   - funding_warn/critical：symbol=BTC,ETH 即 §7 "BTC/ETH" 行；venue 不限
//     （同一币种跨所费率口径一致）。
//   - trx_funding_positive：穿越型。"此前 48h 均值" = avg_48h@24h（[−72h,−24h)）。
//   - ladder_trap：跨实体比较用显式 scope 聚合。数据源约定 venue=binance_ear /
//     symbol=USDT_H（头条档）、USDT_L（大额档）；v1 无自动采集（Binance Earn
//     人工补录见 dialogue #25），规则声明式就位，数据到达即生效。
//   - reverse_repo_timing："前 1 天 + 当日 10:30" 简化为 last_24h <= 1（1h 评估
//     间隔近似 10:30 时间门；"未配置提醒"无 v1 存储面，告警即提醒）。
//   - collector_heartbeat：心跳 fact 契约（M1-f 发射，本规则只实现条件）——
//     kind=heartbeat、symbol=源名、value=距该源上次成功轮询秒数 ÷ 该源轮询间隔
//     （错过的窗口数）。发射方必须是独立定时器（源停摆后心跳仍持续、值随停摆
//     增长），否则停摆源不再"自报"而检测失效。阈值 2 = 错过 ≥2 个窗口。
//     "数据库不可达"由 /healthz degraded 覆盖（main.go 已有），不落规则。
package rule

import (
	"context"
	"fmt"

	"arbcn/internal/fact"
	"arbcn/internal/store"
)

// Defaults 返回首版 10 规则。
func Defaults() []store.Rule {
	return []store.Rule{
		{
			Name: "funding_warn", Kind: fact.KindFunding, Symbol: "BTC,ETH",
			Cond: "avg_30d > 15", Level: store.LevelWarn, Enabled: true, IntervalSec: 300,
		},
		{
			Name: "funding_critical", Kind: fact.KindFunding, Symbol: "BTC,ETH",
			Cond: "avg_30d > 20", Level: store.LevelCritical, Enabled: true, IntervalSec: 300,
		},
		{
			Name: "trx_funding_positive", Kind: fact.KindFunding, Symbol: "TRX",
			Cond: "avg_24h > 0 && avg_48h@24h <= 0", Level: store.LevelWarn, Enabled: true, IntervalSec: 300,
		},
		{
			Name: "defi_large_tier_change", Kind: fact.KindDefiRate,
			Cond: "chg_1h >= 0.5 || chg_1h <= -0.5", Level: store.LevelInfo, Enabled: true, IntervalSec: 1800,
		},
		{
			Name: "ladder_trap", Kind: fact.KindDefiRate,
			Cond:  "avg_7d(binance_ear, USDT_H) > 3 * avg_7d(binance_ear, USDT_L)",
			Level: store.LevelWarn, Enabled: true, IntervalSec: 1800,
		},
		{
			Name: "reverse_repo_timing", Kind: fact.KindCalendar,
			Cond: "last_24h <= 1", Level: store.LevelWarn, Enabled: true, IntervalSec: 3600,
		},
		{
			Name: "usdcnh_buy_line", Kind: fact.KindFX, Symbol: "USDCNH",
			Cond: "last_24h < 6.6", Level: store.LevelWarn, Enabled: true, IntervalSec: 300,
		},
		{
			Name: "iv_opportunity", Kind: fact.KindIV, Symbol: "BTC",
			Cond: "last_24h < p25_30d", Level: store.LevelInfo, Enabled: true, IntervalSec: 1800,
		},
		{
			Name: "nonstable_quote_change", Kind: fact.KindDefiRate, Symbol: "WETH,ETH,WBTC,BTC",
			Cond: "chg_1h >= 0.5 || chg_1h <= -0.5", Level: store.LevelWarn, Enabled: true, IntervalSec: 1800,
		},
		{
			Name: "collector_heartbeat", Kind: fact.KindHeartbeat,
			Cond: "last_24h > 2", Level: store.LevelCritical, Enabled: true, IntervalSec: 60,
		},
	}
}

// Seed 幂等写入 Defaults（UpsertRule），返回写入条数。
func Seed(ctx context.Context, st store.Store) (int, error) {
	n := 0
	for _, r := range Defaults() {
		if _, err := st.UpsertRule(ctx, r); err != nil {
			return n, fmt.Errorf("rule: seed %q: %w", r.Name, err)
		}
		n++
	}
	return n, nil
}
