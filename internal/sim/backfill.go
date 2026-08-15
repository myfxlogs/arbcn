package sim

import (
	"context"
	"fmt"
	"time"

	"arbcn/internal/store"
)

// Simulator 是 M3-a 本地模拟盘驱动（04-m3-spec §3.2）。
// 纯本地回填：SignalToOrder 生成 → 确认 → 按 ref_price 全额即时成交 → 建 sim_positions；
// funding 周期结算 pnl = rate × notional。无任何外部连接（零密钥，D-010）。
// M3-a 不接 testnet（M3-b 才引入）；成交 = 全额即时、忽略滑点/深度（模拟假设，§3.2）。
type Simulator struct {
	st  store.Store
	cfg Config
	// Now 为测试注入时钟（TodaySimNotional 日界 / 订单时间戳）；0 = time.Now。
	Now func() time.Time
}

// New 构造本地模拟盘驱动。
func New(st store.Store, cfg Config) *Simulator {
	return &Simulator{st: st, cfg: cfg}
}

func (s *Simulator) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// Generate 生成并落库一条建议订单（SignalToOrder → InsertSimOrder）。
// DAILY_OVER 数据面自动查询：sig.DayNotional 未显式给时按订单时刻当日累计回填。
// 返回落库订单（含 id 与 risk_flags/status）。
func (s *Simulator) Generate(ctx context.Context, sig Signal) (store.SimOrder, error) {
	if sig.Ts.IsZero() {
		sig.Ts = s.now()
	}
	if sig.DayNotional == 0 {
		day, err := s.st.TodaySimNotional(ctx, sig.Ts)
		if err != nil {
			return store.SimOrder{}, fmt.Errorf("sim: today notional: %w", err)
		}
		sig.DayNotional = day
	}
	o := SignalToOrder(sig, s.cfg)
	id, err := s.st.InsertSimOrder(ctx, o)
	if err != nil {
		return store.SimOrder{}, err
	}
	o.ID = id
	return o, nil
}

// ConfirmAndFill 确认并即时成交：confirmed → filled（按 ref_price 全额成交，模拟假设）。
// 成交后建 sim_positions 腿：funding_hedge = 两行（现货 long + 永续 short，永续腿标定
// funding），carry/repo = 一行（funding 生息腿）。订单非 confirmed → 错误（防状态漂移）。
func (s *Simulator) ConfirmAndFill(ctx context.Context, orderID int64) error {
	o, err := s.st.GetSimOrder(ctx, orderID)
	if err != nil {
		return err
	}
	if o.Status != store.SimStatusConfirmed {
		return fmt.Errorf("sim: order %d status = %q, want confirmed", orderID, o.Status)
	}

	now := s.now()
	legs := BuildLegs(o, now)
	// 原子成交（M3-a 复审 M1）：订单状态与建腿在 store 单事务内完成——任一失败整体
	// 回滚，不留"filled 但缺腿"的半对冲裸敞口（D-019）。前置 status 检查只为报错更清，
	// 事务内 RowsAffected 守卫才是权威（并发/状态漂移也在 store 层拦截）。
	return s.st.FillSimOrder(ctx, orderID, "本地模拟即时成交 @ ref_price（忽略滑点/深度）", legs)
}

// BuildLegs 组装订单成交腿（04-m3-spec §3.2 / §10.4 C3 共享建腿逻辑）：
// funding_hedge = 两行（现货 long 非 funding + 永续 short funding），carry/repo = 一行
// （funding 生息腿）。ConfirmAndFill（M3-b）与 simapi.ConfirmSimOrder（M3-c 人工流）
// 共用，保证两侧建腿口径一致。
func BuildLegs(o store.SimOrder, now time.Time) []store.SimPosition {
	legs := []store.SimPosition{}
	switch o.Kind {
	case store.SimKindFundingHedge:
		legs = append(legs,
			store.SimPosition{OrderID: o.ID, Ts: now, Kind: o.Kind, Venue: o.Venue,
				Symbol: o.Symbol, Side: store.SimSideLong, Qty: o.Qty, RefPrice: o.RefPrice,
				Funding: false, Status: store.SimPosStatusOpen},
			store.SimPosition{OrderID: o.ID, Ts: now, Kind: o.Kind, Venue: o.Venue,
				Symbol: o.Symbol, Side: store.SimSideShort, Qty: o.Qty, RefPrice: o.RefPrice,
				Funding: true, Status: store.SimPosStatusOpen},
		)
	default: // carry_asset / repo：单腿生息。
		legs = append(legs, store.SimPosition{OrderID: o.ID, Ts: now, Kind: o.Kind, Venue: o.Venue,
			Symbol: o.Symbol, Side: store.SimSideLong, Qty: o.Qty, RefPrice: o.RefPrice,
			Funding: true, Status: store.SimPosStatusOpen})
	}
	return legs
}

// SettleFunding 按 funding 周期结算：对 (symbol, venue) 下全部 open 且 funding 的
// 持仓腿，pnl += SettleFundingPnl(Per8hRate(annualized), qty)，置 updated_at。
// 返回结算腿数。annualized 为年化资金费率（%）；调用方从行情/快照供给。
// M3-b §9.3：venue 维度避免 BTC@binance 与 BTC@okx 互相污染（错 rate / 串 venue → 必红）。
func (s *Simulator) SettleFunding(ctx context.Context, symbol, venue string, annualized float64) (int, error) {
	legs, err := s.st.ListOpenSimPositions(ctx, symbol, venue)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, l := range legs {
		if !l.Funding {
			continue
		}
		add := SettleFundingPnl(Per8hRate(annualized), l.Qty)
		if err := s.st.SettleSimPosition(ctx, l.ID, add, store.SimPosStatusOpen); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}
