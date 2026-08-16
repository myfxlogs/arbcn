// 机会实算卡 RPC（D-046）：编排存储层 → 纯函数算账 → proto 映射。只读证据表面
// （practices #20）：卡只说「这笔账划不划算」，执行门禁仍由规则引擎把关。
package dashboard

import (
	"context"
	"math"
	"sort"
	"time"

	"connectrpc.com/connect"

	dashboardv1 "arbcn/internal/dashboard/gen/arbcn/dashboard/v1"
	"arbcn/internal/fact"
	"arbcn/internal/store"
)

// oppFriction 返回 funding_hedge 摩擦 %（配置优先；0 = 默认 0.3，普通主户双 taker 已核实）。
func (s *Service) oppFriction() float64 {
	if s.OppFrictionFunding > 0 {
		return s.OppFrictionFunding
	}
	return defaultOppFrictionPct
}

// ListOppCards 机会实算卡（D-046）：对当前实时机会做确定性算账——瞬时 / 30 日均值 /
// 保本天数 / 扣摩擦净年化 / 三档判定 / 中文模板叙述。数据面：LatestFacts 取每实体瞬时值
// + QueryFacts 30d 均值。瞬时值非有限 / 无数据 → 不产卡（fail-closed 宁缺毋滥）。
func (s *Service) ListOppCards(ctx context.Context, _ *connect.Request[dashboardv1.ListOppCardsRequest]) (*connect.Response[dashboardv1.ListOppCardsResponse], error) {
	cards, err := s.oppCards(ctx, s.now())
	if err != nil {
		return nil, storeErr(err)
	}
	out := make([]*dashboardv1.OpportunityCard, 0, len(cards))
	for _, c := range cards {
		out = append(out, toOppCardProto(c))
	}
	return connect.NewResponse(&dashboardv1.ListOppCardsResponse{Cards: out}), nil
}

// oppCards 编排实算卡：LatestFacts 取每实体瞬时值 + QueryFacts 30d 均值 → 卡。
func (s *Service) oppCards(ctx context.Context, now time.Time) ([]OppCard, error) {
	from := now.Add(-oppAvgWindow)
	var cards []OppCard

	// 1. funding_hedge：每 (venue,symbol)。
	fundingLatest, err := s.st.LatestFacts(ctx, fact.KindFunding, "", "")
	if err != nil {
		return nil, err
	}
	fundingHist, err := s.st.QueryFacts(ctx, store.FactQuery{Kind: fact.KindFunding, From: from})
	if err != nil {
		return nil, err
	}
	fhist := groupByVenueSymbol(fundingHist)
	friction := s.oppFriction()
	for _, f := range fundingLatest {
		if math.IsNaN(f.Value) || math.IsInf(f.Value, 0) {
			continue
		}
		cards = append(cards, fundingCard(f.Venue, f.Symbol, f.Value, meanFacts(fhist[f.Venue+"\x00"+f.Symbol]), friction))
	}

	// 2. carry_asset：defi_rate 每 (venue,symbol)（生息，摩擦≈0）。
	defiLatest, err := s.st.LatestFacts(ctx, fact.KindDefiRate, "", "")
	if err != nil {
		return nil, err
	}
	defiHist, err := s.st.QueryFacts(ctx, store.FactQuery{Kind: fact.KindDefiRate, From: from})
	if err != nil {
		return nil, err
	}
	dhist := groupByVenueSymbol(defiHist)
	for _, f := range defiLatest {
		if math.IsNaN(f.Value) || math.IsInf(f.Value, 0) {
			continue
		}
		cards = append(cards, carryCard(f.Venue, f.Symbol, f.Value, meanFacts(dhist[f.Venue+"\x00"+f.Symbol])))
	}

	// 3. repo：reverse_repo 每 (venue,symbol) 当日时点利率。
	repoLatest, err := s.st.LatestFacts(ctx, fact.KindReverseRepo, "", "")
	if err != nil {
		return nil, err
	}
	repoHist, err := s.st.QueryFacts(ctx, store.FactQuery{Kind: fact.KindReverseRepo, From: from})
	if err != nil {
		return nil, err
	}
	rhist := groupByVenueSymbol(repoHist)
	for _, f := range repoLatest {
		if math.IsNaN(f.Value) || math.IsInf(f.Value, 0) {
			continue
		}
		cards = append(cards, repoCard(f.Venue, f.Symbol, f.Value, meanFacts(rhist[f.Venue+"\x00"+f.Symbol])))
	}

	// 稳定排序（前端确定性展示）：kind → venue → symbol。
	sort.Slice(cards, func(i, j int) bool {
		if cards[i].Kind != cards[j].Kind {
			return cards[i].Kind < cards[j].Kind
		}
		if cards[i].Venue != cards[j].Venue {
			return cards[i].Venue < cards[j].Venue
		}
		return cards[i].Symbol < cards[j].Symbol
	})
	return cards, nil
}

// toOppCardProto 映射 OppCard → proto（Avg30 NaN = 均值样本不足，前端显示「—」）。
func toOppCardProto(c OppCard) *dashboardv1.OpportunityCard {
	return &dashboardv1.OpportunityCard{
		Kind:          c.Kind,
		Venue:         c.Venue,
		Symbol:        c.Symbol,
		Inst:          c.Inst,
		Avg_30D:       c.Avg30,
		BreakEvenDays: c.BreakEvenDays,
		NetAnnualized: c.NetAnnualized,
		FrictionPct:   c.Friction,
		Rating:        c.Rating,
		Narrative:     c.Narrative,
	}
}
