// 事实快照 + RMB 折算 RPC（M2-b §4/§5）：把 USD 计价事实 × 当日 USDCNH
// 折算为 RMB 净收益视角（服务端在展示层折算，原始事实不污染，02 §8）。
package dashboard

import (
	"context"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	dashboardv1 "arbcn/internal/dashboard/gen/arbcn/dashboard/v1"
	"arbcn/internal/fact"
	"arbcn/internal/rmb"
	"arbcn/internal/store"
)

// ListFacts 事实快照 + RMB 折算（M2-b §4/§5 机器可读投影）。
// 数据面：LatestFacts 全量 → 排除 heartbeat 内部遥测（与 exporter skipKinds 同语义，
// 快照 = 市场事实）→ 最新 fx（kind=fx 最新值）→ 30d fx 序列年化升值 →
// rmb.Convert 折算。原始事实不污染（02 §8）；汇率缺失 → USD 原值 + "汇率不可用"标记。
func (s *Service) ListFacts(ctx context.Context, req *connect.Request[dashboardv1.ListFactsRequest]) (*connect.Response[dashboardv1.ListFactsResponse], error) {
	facts, err := s.st.LatestFacts(ctx, req.Msg.Kind, req.Msg.Venue, req.Msg.Symbol)
	if err != nil {
		return nil, storeErr(err)
	}
	facts = excludeHeartbeat(facts)
	fx, err := s.latestFX(ctx)
	if err != nil {
		return nil, storeErr(err)
	}
	appreciation, err := s.fxAppreciation(ctx)
	if err != nil {
		return nil, storeErr(err)
	}
	converted := rmb.Convert(facts, fx, appreciation)

	out := make([]*dashboardv1.FactRmb, 0, len(converted))
	for _, c := range converted {
		out = append(out, toFactRmb(c))
	}
	resp := &dashboardv1.ListFactsResponse{Facts: out}
	if fx != nil {
		resp.FxRate = fx.Value
		resp.FxAvailable = true
		resp.FxTs = timestamppb.New(fx.Ts)
	}
	return connect.NewResponse(resp), nil
}

// excludeHeartbeat 过滤内部遥测事实（heartbeat 非市场事实，与 exporter skipKinds 一致）。
// 防御性重排：保持输入顺序（stable），不原地改写调用方切片。
func excludeHeartbeat(facts []fact.Fact) []fact.Fact {
	out := make([]fact.Fact, 0, len(facts))
	for _, f := range facts {
		if f.Kind != fact.KindHeartbeat {
			out = append(out, f)
		}
	}
	return out
}

// latestFX 返回最新 fx fact（kind=fx 全部键取 ts 最大）；无记录 = nil（汇率缺失）。
func (s *Service) latestFX(ctx context.Context) (*fact.Fact, error) {
	fxs, err := s.st.LatestFacts(ctx, fact.KindFX, "", "")
	if err != nil {
		return nil, err
	}
	var best *fact.Fact
	for i := range fxs {
		if best == nil || fxs[i].Ts.After(best.Ts) {
			best = &fxs[i]
		}
	}
	return best, nil
}

// fxAppreciation 计算 30d 年化人民币升值率（rmb.TrendDays 窗口）；序列不足 → 0。
func (s *Service) fxAppreciation(ctx context.Context) (float64, error) {
	series, err := s.st.QueryFacts(ctx, store.FactQuery{
		Kind: fact.KindFX,
		From: rmb.SeriesWindow(s.now()),
	})
	if err != nil {
		return 0, err
	}
	return rmb.AnnualizedRMBAppreciation(series), nil
}

// toFactRmb 映射 rmb.Converted → proto。
func toFactRmb(c rmb.Converted) *dashboardv1.FactRmb {
	return &dashboardv1.FactRmb{
		Kind:        c.Fact.Kind,
		Venue:       c.Fact.Venue,
		Symbol:      c.Fact.Symbol,
		Value:       c.Fact.Value,
		Unit:        c.Fact.Unit,
		Ts:          timestamppb.New(c.Fact.Ts),
		Src:         c.Fact.Src,
		RmbValue:    c.RMBValue,
		FxRate:      c.FXRate,
		FxAvailable: c.FXAvailable,
		Covered:     c.Covered,
	}
}
