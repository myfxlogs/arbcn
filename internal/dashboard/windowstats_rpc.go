// ListFundingWindowStats 7d 费率窗口统计 RPC（D-064）。只读编排：QueryFacts
// （Kind=funding, From=now−7d）→ 聚合 + 逐 venue|symbol 窗口统计 + 判据。零写
// 路径（只读证据面，不碰任何执行门禁）；数据面故障 → storeErr（不编造）。
package dashboard

import (
	"context"
	"sort"
	"time"

	"connectrpc.com/connect"

	dashboardv1 "arbcn/internal/dashboard/gen/arbcn/dashboard/v1"
	"arbcn/internal/fact"
	"arbcn/internal/store"
)

// maxWindowPairs per_pair 返回行数上限（均值降序前 N，防大面刷爆前端）。
const maxWindowPairs = 50

// ListFundingWindowStats 返回最近 7 天 funding 窗口统计：overall = 全部监控面
// funding 读数聚合（「当前是否处于可交易窗口」判据主答案），per_pair = 逐
// venue|symbol 明细（均值降序前 50）。只覆盖监控面内 venue/symbol（诚实标注，
// 与 GetPerformanceReport 环境统计同口径，D-062）。窗口内无数据 → overall
// class=not + note「无数据」（不编造，practices #7）。
func (s *Service) ListFundingWindowStats(ctx context.Context, _ *connect.Request[dashboardv1.ListFundingWindowStatsRequest]) (*connect.Response[dashboardv1.ListFundingWindowStatsResponse], error) {
	now := time.Now()
	if s.Now != nil {
		now = s.Now()
	}
	since := now.Add(-time.Duration(FundingWindowDays) * 24 * time.Hour)

	fs, err := s.st.QueryFacts(ctx, store.FactQuery{Kind: fact.KindFunding, From: since, Limit: 100000})
	if err != nil {
		return nil, storeErr(err)
	}

	overall := ComputeFundingWindowStats(fs)

	// per_pair 分组：key = venue|symbol。
	byKey := map[string][]fact.Fact{}
	for _, f := range fs {
		key := f.Venue + "|" + f.Symbol
		byKey[key] = append(byKey[key], f)
	}
	pairs := make([]*dashboardv1.FundingWindowPair, 0, len(byKey))
	for _, pfs := range byKey {
		st := ComputeFundingWindowStats(pfs)
		pairs = append(pairs, &dashboardv1.FundingWindowPair{
			Venue:  pfs[0].Venue,
			Symbol: pfs[0].Symbol,
			Stats:  toWindowStatsProto(st),
		})
	}
	// 均值降序，前 maxWindowPairs。
	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].Stats.Mean > pairs[j].Stats.Mean
	})
	if len(pairs) > maxWindowPairs {
		pairs = pairs[:maxWindowPairs]
	}

	resp := &dashboardv1.ListFundingWindowStatsResponse{
		WindowDays: FundingWindowDays,
		Overall:    toWindowStatsProto(overall),
		PerPair:    pairs,
	}
	return connect.NewResponse(resp), nil
}

// toWindowStatsProto FundingWindowStats → proto（无逻辑，纯映射）。
func toWindowStatsProto(s FundingWindowStats) *dashboardv1.FundingWindowStats {
	return &dashboardv1.FundingWindowStats{
		Count:         int32(s.Count),
		Min:           s.Min,
		Max:           s.Max,
		Mean:          s.Mean,
		PositiveShare: s.PositiveShare,
		Class:         s.Class,
		Note:          s.Note,
	}
}
