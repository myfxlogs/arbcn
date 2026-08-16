// 首版规则集（docs/design/02-monitor-architecture.md §7 表 10 条全落 +
// D-041 演练档 funding_drill；阈值锚 docs/handoff/facts.md）。规则 = 配置行，
// M1-h 接线时经 Seed 落库；改阈值 = 改 DB 一行，不发布版本（§4）。
//
// scope 约定：venue/symbol 逗号分隔 = IN 列表（空 = 不限）；逐实体独立聚合，
// 任一实体命中即告警。chg 语义：环比 = 最新采集值 − 紧邻上一采集值
// （±0.5% 用 "||" 双比较表达）。
//
// 各规则口径说明（实现决策，供复审）：
//   - funding_warn/critical：symbol=BTC,ETH 即 §7 "BTC/ETH" 行；venue 不限
//     （同一币种跨所费率口径一致）。
//   - funding_drill（D-041 演练档）：funding_hedge 演练带 [5%,15%)——高于 5%
//     （跨过 SPREAD_LOW 门禁）且低于 funding_warn 门槛 15%（不重复真实窗口档）；
//     只喂模拟盘演练确认→成交→8h 结算，Info 级不打扰。上限 15 与 funding_warn
//     同源（同库规则表，改一须改二）。
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

// ruleLabels 规则名 → 中文标签（告警消息与前端展示共用语义；规则名本身是
// 稳定标识符不改动，P3）。未知名回退原名（新规则未映射时优雅降级）。
var ruleLabels = map[string]string{
	"funding_warn":            "资金费率预警",
	"funding_critical":        "资金费率激活",
	"funding_drill":           "资金费率演练档",
	"trx_funding_positive":    "TRX 费率转正",
	"defi_large_tier_change":  "金额档利率变动",
	"ladder_trap":             "阶梯陷阱识别",
	"reverse_repo_timing":     "逆回购时点",
	"usdcnh_buy_line":         "汇率加仓线",
	"iv_opportunity":          "IV 机会",
	"nonstable_quote_change":  "计价币种陷阱",
	"collector_heartbeat":     "采集器心跳",
}

// ruleLabel 取规则中文标签；未知回退原名。
func ruleLabel(name string) string {
	if l, ok := ruleLabels[name]; ok {
		return l
	}
	return name
}

// Defaults 返回首版 11 规则（10 + D-041 演练档）。
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
			Name: "funding_drill", Kind: fact.KindFunding, Symbol: "BTC,ETH",
			Cond: "avg_30d > 5 && avg_30d < 15", Level: store.LevelInfo, Enabled: true, IntervalSec: 300,
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

// Seed 确保 10 条默认规则存在（R2#1 裁定：已存在的规则保留 DB 状态，不覆盖人工编辑；
// 代码默认值只引导新装，改代码阈值不影响已部署 DB，须显式迁移/SQL 才更新）。
// 返回处理条数（幂等恒 = len(Defaults)，与 integration_test 锚点一致）。
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
