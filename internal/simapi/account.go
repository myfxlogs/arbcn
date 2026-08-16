// 模拟账户快照 RPC（D-056 完整现金账本，切出本文件防 service.go 超 450 行）。
// 只读 store + latestValue（service.go 同包），零网络零密钥（D-010）。
package simapi

import (
	"context"

	"connectrpc.com/connect"

	"arbcn/internal/fact"
	simv1 "arbcn/internal/simapi/gen/arbcn/sim/v1"
	"arbcn/internal/store"
)

// GetSimAccount 模拟账户快照（D-056 完整现金账本）：现金余额 + 逐笔流水 + 净值对账。
//   - capital/cash = 账户初始本金 / 现金余额（现金账本表）；realized_pnl = Σ 全部腿已结算 pnl。
//   - unrealized_pnl = Σ open 腿 (cur−ref)×qty×dir；market_value = Σ open 腿 dir×qty×cur
//     （dir：long+1/short−1；ticker 缺失 → 该腿按 0 计，不编造）。
//   - equity = cash + market_value = capital + realized + unrealized（双恒等式交叉校验，
//     前端同时展示供业主核对「策略真赚还是理论」）。
//   - equity_rmb = 即期 USDCNH 折算（汇率缺失 = 0，前端标注 USD 原值，D-047 口径）。
//   - flows = 最近 100 条逐笔流水（ts DESC，审计账本）。
func (s *Service) GetSimAccount(ctx context.Context, _ *connect.Request[simv1.GetSimAccountRequest]) (*connect.Response[simv1.GetSimAccountResponse], error) {
	acct, err := s.st.GetSimAccount(ctx)
	if err != nil {
		return nil, storeErr(err)
	}
	// realized = Σ 全部腿 pnl（settled 已结算 + open 已结算 funding）。
	positions, err := s.st.ListSimPositions(ctx, 10000, 0)
	if err != nil {
		return nil, storeErr(err)
	}
	realized := 0.0
	for _, p := range positions {
		realized += p.PnL
	}
	// unrealized / market_value 只对 open 腿按实时价算（口径与 ListSimPositions 一致）。
	open, err := s.st.ListOpenSimPositions(ctx, "", "")
	if err != nil {
		return nil, storeErr(err)
	}
	unrealized := 0.0
	marketValue := 0.0
	for _, p := range open {
		curPrice, curOK, err := s.latestValue(ctx, fact.KindTicker, p.Venue, p.Symbol)
		if err != nil {
			return nil, storeErr(err)
		}
		if !curOK {
			continue
		}
		dir := 1.0
		if p.Side == store.SimSideShort {
			dir = -1
		}
		unrealized += (curPrice - p.RefPrice) * p.Qty * dir
		marketValue += dir * p.Qty * curPrice
	}
	equity := acct.Cash + marketValue
	fxRate, fxOK, err := s.latestValue(ctx, fact.KindFX, fxVenue, fxSymbol)
	if err != nil {
		return nil, storeErr(err)
	}
	equityRmb := 0.0
	if fxOK {
		equityRmb = equity * fxRate
	}
	flows, err := s.st.ListCashFlows(ctx, 100, 0)
	if err != nil {
		return nil, storeErr(err)
	}
	out := &simv1.GetSimAccountResponse{
		Capital:       acct.Capital,
		Cash:          acct.Cash,
		RealizedPnl:   realized,
		UnrealizedPnl: unrealized,
		MarketValue:   marketValue,
		Equity:        equity,
		FxAvailable:   fxOK,
		EquityRmb:     equityRmb,
		Flows:         make([]*simv1.CashFlow, 0, len(flows)),
	}
	for _, f := range flows {
		out.Flows = append(out.Flows, &simv1.CashFlow{
			Id:      f.ID,
			TsMs:    f.Ts.UnixMilli(),
			OrderId: f.OrderID,
			LegId:   f.LegID,
			Kind:    f.Kind,
			Amount:  f.Amount,
			Note:    f.Note,
		})
	}
	return connect.NewResponse(out), nil
}
