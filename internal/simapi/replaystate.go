// GetReplayState 回放证伪门禁证据面（D-065 修订）。切出本文件防 service.go 超 450 行。
// 只读编排：每策略 kind 历史 rate 事实 → 按 (venue,symbol) 分组 → 每对回放 → overall 聚合
// → 只读展示。计算全走 internal/sim 纯函数（replay.go，P3 单源，Driver 门禁同款判据），
// 本文件只做数据面编排；零写路径（只展示当前回放判据，不触发订单管线）、零网络零密钥
// （D-010）。P4 可检查性：门禁休眠（no_window）也可见，不靠拒单负样本才暴露。
package simapi

import (
	"context"
	"sort"
	"time"

	"connectrpc.com/connect"

	"arbcn/internal/fact"
	"arbcn/internal/sim"
	simv1 "arbcn/internal/simapi/gen/arbcn/sim/v1"
	"arbcn/internal/store"
)

// GetReplayState 每策略 kind 当前回放判据。数据面 = kind 自己的 rate 事实历史
// （ReplayHistoryDays 窗口），按 (venue,symbol) 分组逐对回放 → OverallReplay 样本加权
// 聚合。查询失败 → 报错（证据面不编造数据）；无事实 → no_window（D-061 ②）。
func (s *Service) GetReplayState(ctx context.Context, _ *connect.Request[simv1.GetReplayStateRequest]) (*connect.Response[simv1.GetReplayStateResponse], error) {
	resp := &simv1.GetReplayStateResponse{HistoryDays: int32(sim.ReplayHistoryDays)}
	from := s.now().Add(-time.Duration(sim.ReplayHistoryDays) * 24 * time.Hour)

	kinds := []string{store.SimKindFundingHedge, store.SimKindCarryAsset, store.SimKindRepo}
	for _, k := range kinds {
		rateKind, tier, friction, ok := sim.ReplayKindConfig(k)
		if !ok {
			continue // 未知 kind 不展示（表 SSOT，不该发生）
		}
		fs, err := s.st.QueryFacts(ctx, store.FactQuery{Kind: rateKind, From: from, Limit: 500_000})
		if err != nil {
			return nil, storeErr(err)
		}
		// 按 (venue,symbol) 分组逐对回放（门禁按对拒单，overall 只作展示聚合）。
		groups := map[string][]fact.Fact{}
		for _, f := range fs {
			key := f.Venue + "|" + f.Symbol
			groups[key] = append(groups[key], f)
		}
		pairs := make([]sim.ReplayPair, 0, len(groups))
		for _, g := range groups {
			pairs = append(pairs, sim.ComputeReplay(g, tier, friction))
		}
		sort.Slice(pairs, func(i, j int) bool { // 确定性展示顺序
			return pairs[i].Venue+pairs[i].Symbol < pairs[j].Venue+pairs[j].Symbol
		})
		o := sim.OverallReplay(pairs, tier)
		best, worst := o.BestNetAnn, o.WorstNetAnn
		if o.WindowCount == 0 { // 无窗口时 ±Inf 无展示意义 → 归零（诚实：无窗口非"极值无穷"）
			best, worst = 0, 0
		}
		resp.Kinds = append(resp.Kinds, &simv1.ReplayStateKind{
			Kind: k, RateKind: rateKind,
			Verdict: o.Verdict, Note: o.Note,
			WindowCount: int32(o.WindowCount), TotalSamples: int32(o.TotalSamples),
			HighSamples: int32(o.HighSamples),
			MeanNetAnn:  o.MeanNetAnn, BestNetAnn: best, WorstNetAnn: worst,
			TierPct: tier, FrictionPct: friction,
		})
	}
	return connect.NewResponse(resp), nil
}
