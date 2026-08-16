// 进化建议引擎 L0（D-044）：把模拟盘负样本 / 事实库截面异常 / 无单水位 / 数据源停更
// 加工成「待核实证据候选」供决策层参考。只读证据表面——永不自动改规则/执行
// （决策在环，D-019/D-043 同源）。按需计算（on-demand pull）：四信号都便宜，
// 不加表、不加后台循环；前端沿用 useSnapshot 的 60s poll。
package dashboard

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	dashboardv1 "arbcn/internal/dashboard/gen/arbcn/dashboard/v1"
	"arbcn/internal/fact"
	"arbcn/internal/store"
)

// —— 信号常量（集中一处；注释锚定与规则引擎的同步点，改动须两处核对）——
const (
	// anomalyFactor 截面异常判定倍率：value > anomalyFactor×median 即标尖峰。
	// 中位数法对利率整体上行（regime shift）稳健——全员上涨时中位数同步上移不误报。
	anomalyFactor = 2.0
	// minAnomalySamples 截面异常最少样本数（<3 中位数无统计意义，不判定）。
	minAnomalySamples = 3
	// noOrderWindowDays 无单提示窗口：近 N 天无 active 建议单才提示。
	noOrderWindowDays = 7
	// fundingWindowEntryPct funding 窗口档入场门槛 %（与 internal/rule/defaults.go
	// funding 窗口档 avg_30d 阈值同步；改动须两处核对）。
	fundingWindowEntryPct = 15.0
	// insightOrderLimit 模拟单拉取上限（个人模拟盘单日个位数，1000 远覆盖）。
	insightOrderLimit = 1000
	// defiLookback 截面数据回看窗口（取每池最新值即可，30d 覆盖采集间隔）。
	defiLookback = 30 * 24 * time.Hour
	// fundingLookback 无单提示的 funding 最新值回看窗口（24h 内必有最新 funding）。
	fundingLookback = 24 * time.Hour
)

// insightItem 包内信号结构（映射到 proto Insight；id 为稳定 key）。
type insightItem struct {
	id       string
	category string
	severity string
	title    string
	detail   string
	actions  []string
}

// ListInsights 进化建议 L0：四类信号汇总，severity 排序（critical>warn>info）。
// 只读证据表面——任何信号都只是「待核实候选」，动作一律指向 D# 人工决策。
func (s *Service) ListInsights(ctx context.Context, _ *connect.Request[dashboardv1.ListInsightsRequest]) (*connect.Response[dashboardv1.ListInsightsResponse], error) {
	now := s.now()
	items := []insightItem{}

	// 1. 拒单分布 + 近 7 天 active 单计数（同一份订单列表）。
	orders, err := s.st.ListSimOrders(ctx, insightOrderLimit, 0)
	if err != nil {
		return nil, storeErr(err)
	}
	items = append(items, rejectDistribution(orders, 3)...)
	recent := 0
	windowFrom := now.AddDate(0, 0, -noOrderWindowDays)
	for _, o := range orders {
		switch o.Status {
		case store.SimStatusSuggested, store.SimStatusConfirmed, store.SimStatusFilled:
			if !o.Ts.Before(windowFrom) {
				recent++
			}
		}
	}

	// 2. DeFi 截面异常（取每池 30d 内最新值）。
	defi, err := s.st.QueryFacts(ctx, store.FactQuery{Kind: fact.KindDefiRate, From: now.Add(-defiLookback)})
	if err != nil {
		return nil, storeErr(err)
	}
	items = append(items, defiAnomalies(defi, anomalyFactor)...)

	// 3. 连续无单提示（当前 funding 距窗口门槛）。
	funding, err := s.st.QueryFacts(ctx, store.FactQuery{Kind: fact.KindFunding, From: now.Add(-fundingLookback)})
	if err != nil {
		return nil, storeErr(err)
	}
	if h, ok := noOrderHint(funding, recent); ok {
		items = append(items, h)
	}

	// 4. 数据源停更（复用已测 sourceHealth；单源失败不阻断其他信号）。
	for _, src := range s.sources {
		status, _, _, err := s.sourceHealth(ctx, src, now)
		if err != nil {
			continue
		}
		switch status {
		case StatusDown:
			items = append(items, insightItem{
				id:       "source_down:" + src.Name,
				category: "data",
				severity: "critical",
				title:    "数据源停更",
				detail:   fmt.Sprintf("数据源 %s 超过 2×轮询间隔无新数据（down）", src.Name),
				actions:  []string{"排查 collector（网络/限流/端点），恢复后事实库自动续更"},
			})
		case StatusStale:
			items = append(items, insightItem{
				id:       "source_stale:" + src.Name,
				category: "data",
				severity: "warn",
				title:    "数据源延迟",
				detail:   fmt.Sprintf("数据源 %s 最近一次事实超过轮询间隔（stale）", src.Name),
				actions:  []string{"观察下一轮是否恢复；持续则排查 collector"},
			})
		}
	}

	// 5. 经验库匹配（D-046）：确定性签名命中已吸收条目 → 只读呈现（category=knowledge）。
	// 复用 signal 2 的 defi 截面数据面；funding 30d 单独查（尖峰/跨所分歧数据面）。
	km, err := s.knowledgeMatches(ctx, now, defi)
	if err != nil {
		return nil, storeErr(err)
	}
	items = append(items, km...)

	// severity 排序，同类内保序（SliceStable）。
	sort.SliceStable(items, func(i, j int) bool { return severityRank(items[i].severity) < severityRank(items[j].severity) })

	out := make([]*dashboardv1.Insight, 0, len(items))
	for _, it := range items {
		out = append(out, toInsightProto(it, now))
	}
	return connect.NewResponse(&dashboardv1.ListInsightsResponse{Insights: out}), nil
}

// rejectDistribution 拒单原因分布：过滤 status=rejected，risk_flags 逐条展开计数
// （单订单可含多 flag），按计数降序取前 maxN。空 flags / 无拒单 → nil。
func rejectDistribution(orders []store.SimOrder, maxN int) []insightItem {
	counts := map[string]int{}
	total := 0
	for _, o := range orders {
		if o.Status != store.SimStatusRejected {
			continue
		}
		total++
		for _, flag := range o.RiskFlags {
			counts[flag]++
		}
	}
	if len(counts) == 0 {
		return nil
	}
	type fc struct {
		flag string
		n    int
	}
	all := make([]fc, 0, len(counts))
	for flag, n := range counts {
		all = append(all, fc{flag, n})
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].n != all[j].n {
			return all[i].n > all[j].n
		}
		return all[i].flag < all[j].flag
	})
	if len(all) > maxN {
		all = all[:maxN]
	}
	var b strings.Builder
	fmt.Fprintf(&b, "近 %d 笔拒单主因：", total)
	for i, x := range all {
		if i > 0 {
			b.WriteString("、")
		}
		fmt.Fprintf(&b, "%s ×%d", x.flag, x.n)
	}
	return []insightItem{{
		id:       "reject_dist",
		category: "risk",
		severity: "info",
		title:    "模拟盘拒单原因分布",
		detail:   b.String(),
		actions:  []string{"对照 main 拒因核对门禁阈值是否合理，需调整走 D#"},
	}}
}

// defiPoint 截面最新 defi_rate 值（按 (venue,symbol) 聚合）。
type defiPoint struct {
	value  float64
	venue  string
	symbol string
	ts     time.Time
}

// defiAnomalies 截面异常：取每 (venue,symbol) 最新 defi_rate，任一值 >
// factor×median 即标尖峰（样本 < minAnomalySamples 不判定；NaN/Inf 跳过，practices #7）。
func defiAnomalies(facts []fact.Fact, factor float64) []insightItem {
	latest := map[string]defiPoint{}
	for _, f := range facts {
		if math.IsNaN(f.Value) || math.IsInf(f.Value, 0) {
			continue
		}
		key := f.Venue + "\x00" + f.Symbol
		if cur, ok := latest[key]; !ok || f.Ts.After(cur.ts) {
			latest[key] = defiPoint{f.Value, f.Venue, f.Symbol, f.Ts}
		}
	}
	if len(latest) < minAnomalySamples {
		return nil
	}
	vals := make([]float64, 0, len(latest))
	for _, p := range latest {
		vals = append(vals, p.value)
	}
	med := median(vals)
	// 防御：median NaN/≤0 无判定意义（事实入库已挡 NaN，双保险）。
	if math.IsNaN(med) || med <= 0 {
		return nil
	}
	var out []insightItem
	for _, p := range latest {
		if p.value > factor*med {
			out = append(out, insightItem{
				id:       "defi_anomaly:" + p.venue + ":" + p.symbol,
				category: "anomaly",
				severity: "warn",
				title:    "DeFi 利率异常尖峰",
				detail:   fmt.Sprintf("%s（%s）利率 %.2f%% 远超同类中位 %.2f%%（~%.1f×）", p.symbol, p.venue, p.value, med, p.value/med),
				actions:  []string{"实盘决策前按 D-028 核实均值与资金利用率，勿按单点尖峰操作"},
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].id < out[j].id })
	return out
}

// noOrderHint 连续无单提示：近 noOrderWindowDays 无 active 建议单，且当前 funding
// 最高值 < fundingWindowEntryPct → 提示（无单 + 距门槛远 = 应评估门槛或接受常态）。
func noOrderHint(latestFunding []fact.Fact, recentOrderCount int) (insightItem, bool) {
	if recentOrderCount > 0 {
		return insightItem{}, false
	}
	// QueryFacts 按 ts ASC，逐个覆盖即得每实体最新 funding 值。
	latest := map[string]float64{}
	for _, f := range latestFunding {
		if math.IsNaN(f.Value) || math.IsInf(f.Value, 0) {
			continue
		}
		latest[f.Venue+"\x00"+f.Symbol] = f.Value
	}
	top := math.NaN()
	for _, v := range latest {
		if math.IsNaN(top) || v > top {
			top = v
		}
	}
	if math.IsNaN(top) || top >= fundingWindowEntryPct {
		return insightItem{}, false
	}
	return insightItem{
		id:       "no_order",
		category: "opportunity",
		severity: "info",
		title:    "近 7 天无模拟建议单",
		detail:   fmt.Sprintf("当前 funding 最高年化 %.2f%%，低于 funding 窗口档门槛 %.0f%%（差 %.2f%%）", top, fundingWindowEntryPct, fundingWindowEntryPct-top),
		actions:  []string{"评估 funding 窗口档门槛是否过高（走 D#），或接受当前无机会为常态（宁缺毋滥）"},
	}, true
}

// median 中位数（未排序输入亦可；空 → NaN）。
func median(xs []float64) float64 {
	if len(xs) == 0 {
		return math.NaN()
	}
	sorted := append([]float64(nil), xs...)
	sort.Float64s(sorted)
	n := len(sorted)
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2
}

// severityRank severity → 排序权重（critical=0 最先）。
func severityRank(s string) int {
	switch s {
	case "critical":
		return 0
	case "warn":
		return 1
	default:
		return 2 // info
	}
}

// toInsightProto 映射包内信号 → proto（at = 生成时刻，注入时钟）。
func toInsightProto(it insightItem, now time.Time) *dashboardv1.Insight {
	return &dashboardv1.Insight{
		Id:       it.id,
		Category: it.category,
		Severity: it.severity,
		Title:    it.title,
		Detail:   it.detail,
		At:       timestamppb.New(now),
		Actions:  it.actions,
	}
}
