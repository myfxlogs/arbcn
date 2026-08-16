package simapi

import (
	"context"
	"errors"
	"math"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"

	"arbcn/internal/fact"
	"arbcn/internal/sim"
	simv1 "arbcn/internal/simapi/gen/arbcn/sim/v1"
	"arbcn/internal/store"
)

// fakeStore 是 simapi 服务测试的内存 store.Store（M3-c C3，只实装服务用到的面，
// 其余占位；sim 相关写入方法记录调用供断言——误用即红不静默）。
type fakeStore struct {
	orders        []store.SimOrder
	positions     []store.SimPosition
	accounts      []store.TestnetAccount // D-040 GetTestnetAccounts 数据面
	facts         []fact.Fact
	capital       float64 // D-056 现金账本：初始本金
	cash          float64 // D-056 现金账本：现金余额
	flows         []store.CashFlow
	nextOrderID   int64
	nextFlowIDNum int64
	snaps         []store.EquitySnapshot // D-062 判定门① 测量数据面

	accepted []acceptedCall // AcceptSimOrder 调用记录
	rejected []rejectedCall // RejectSimOrder 调用记录
}

type acceptedCall struct {
	id   int64
	note string
	legs []store.SimPosition
}

type rejectedCall struct {
	id     int64
	reason string
	flags  []string
}

func newFakeStore() *fakeStore {
	return &fakeStore{nextOrderID: 1}
}

// addOrder 追加订单并回填 id/status 默认值。
func (f *fakeStore) addOrder(o store.SimOrder) {
	if o.Status == "" {
		o.Status = store.SimStatusSuggested
	}
	o.ID = f.nextOrderID
	f.nextOrderID++
	f.orders = append(f.orders, o)
}

// addFact 追加事实（含默认单位）。
func (f *fakeStore) addFact(kind, venue, symbol string, v float64, ts time.Time) {
	f.facts = append(f.facts, fact.Fact{Kind: kind, Venue: venue, Symbol: symbol, Value: v, Ts: ts})
}

func (f *fakeStore) orderByID(id int64) (store.SimOrder, bool) {
	for _, o := range f.orders {
		if o.ID == id {
			return o, true
		}
	}
	return store.SimOrder{}, false
}

func (f *fakeStore) replaceOrder(o store.SimOrder) {
	for i := range f.orders {
		if f.orders[i].ID == o.ID {
			f.orders[i] = o
			return
		}
	}
}

// —— 服务实际调用的接口面 ——

func (f *fakeStore) GetSimOrder(_ context.Context, id int64) (store.SimOrder, error) {
	o, ok := f.orderByID(id)
	if !ok {
		return store.SimOrder{}, store.ErrNotFound
	}
	return o, nil
}

func (f *fakeStore) ListSimOrders(context.Context, int, int) ([]store.SimOrder, error) {
	out := append([]store.SimOrder(nil), f.orders...)
	sort.Slice(out, func(i, j int) bool { return out[i].Ts.After(out[j].Ts) })
	return out, nil
}

func (f *fakeStore) ListSimPositions(context.Context, int, int) ([]store.SimPosition, error) {
	return append([]store.SimPosition(nil), f.positions...), nil
}

// UpsertTestnetAccount 幂等 upsert（source 主键；D-040）。
func (f *fakeStore) UpsertTestnetAccount(_ context.Context, a store.TestnetAccount) error {
	for i := range f.accounts {
		if f.accounts[i].Source == a.Source {
			f.accounts[i] = a
			return nil
		}
	}
	f.accounts = append(f.accounts, a)
	return nil
}

// ListTestnetAccounts 按 source ASC 返回（D-040）。
func (f *fakeStore) ListTestnetAccounts(context.Context) ([]store.TestnetAccount, error) {
	out := append([]store.TestnetAccount(nil), f.accounts...)
	sort.Slice(out, func(i, j int) bool { return out[i].Source < out[j].Source })
	return out, nil
}

func (f *fakeStore) LatestFacts(_ context.Context, kind, venue, symbol string) ([]fact.Fact, error) {
	out := []fact.Fact{}
	for _, x := range f.facts {
		if kind != "" && x.Kind != kind {
			continue
		}
		if venue != "" && x.Venue != venue {
			continue
		}
		if symbol != "" && x.Symbol != symbol {
			continue
		}
		out = append(out, x)
	}
	// DISTINCT ON (kind,venue,symbol) ts DESC 语义：只回每键最新一条。
	sort.Slice(out, func(i, j int) bool { return out[i].Ts.After(out[j].Ts) })
	seen := map[string]bool{}
	dedup := []fact.Fact{}
	for _, x := range out {
		k := x.Kind + "\x00" + x.Venue + "\x00" + x.Symbol
		if seen[k] {
			continue
		}
		seen[k] = true
		dedup = append(dedup, x)
	}
	return dedup, nil
}

func (f *fakeStore) AcceptSimOrder(_ context.Context, id int64, note string, legs []store.SimPosition) error {
	f.accepted = append(f.accepted, acceptedCall{id: id, note: note, legs: legs})
	o, ok := f.orderByID(id)
	if !ok || o.Status != store.SimStatusSuggested {
		return errors.New("fakeStore: AcceptSimOrder 守卫（status != suggested）")
	}
	o.Status = store.SimStatusFilled
	o.Note = note
	f.replaceOrder(o)
	return nil
}

func (f *fakeStore) RejectSimOrder(_ context.Context, id int64, reason string, flags ...string) error {
	f.rejected = append(f.rejected, rejectedCall{id: id, reason: reason, flags: flags})
	o, ok := f.orderByID(id)
	if !ok || o.Status != store.SimStatusSuggested {
		return errors.New("fakeStore: RejectSimOrder 守卫（status != suggested）")
	}
	o.Status = store.SimStatusRejected
	o.Note = reason
	for _, fl := range flags {
		if !slices.Contains(o.RiskFlags, fl) {
			o.RiskFlags = append(o.RiskFlags, fl)
		}
	}
	f.replaceOrder(o)
	return nil
}

// CloseSimOrder 平仓（D-055，服务测试真语义，与 pgstore 守卫一致）：订单必须 filled
// （否则 ErrNotFound）→ 逐腿 pnl += AddPnl + settled（腿必须属于本单且 open，任一 miss
// 回滚）→ 订单 closed + note 覆盖（非空时）。返回平掉腿数。
func (f *fakeStore) CloseSimOrder(_ context.Context, orderID int64, note string, closes []store.SimLegClose) (int, error) {
	if len(closes) == 0 {
		return 0, errors.New("fakeStore: CloseSimOrder: closes required")
	}
	o, ok := f.orderByID(orderID)
	if !ok || o.Status != store.SimStatusFilled {
		return 0, store.ErrNotFound
	}
	n := 0
	for _, c := range closes {
		idx := -1
		for i := range f.positions {
			if f.positions[i].ID == c.ID && f.positions[i].OrderID == orderID && f.positions[i].Status == store.SimPosStatusOpen {
				idx = i
				break
			}
		}
		if idx < 0 {
			return 0, errors.New("fakeStore: CloseSimOrder: leg 非 open/不属于本单（回滚，防半仓 D-019）")
		}
		f.positions[idx].PnL += c.AddPnl
		f.positions[idx].Status = store.SimPosStatusSettled
		// D-056：平仓现金流入账（与 pgstore 同语义）。
		f.cash += c.CashDelta
		f.flows = append(f.flows, store.CashFlow{ID: f.nextFlowID(), Ts: t0,
			OrderID: orderID, LegID: c.ID, Kind: store.CashKindClose, Amount: c.CashDelta})
		n++
	}
	o.Status = store.SimStatusClosed
	if note != "" {
		o.Note = note
	}
	f.replaceOrder(o)
	return n, nil
}

// —— 其余接口占位（未用即红） ——

func (f *fakeStore) InsertFacts(context.Context, []fact.Fact) error {
	panic("fakeStore: InsertFacts not used")
}
// QueryFacts 按 kind/From/limit 过滤（ts 升序）——GetPerformanceReport 环境条件
// 数据面（funding 历史）真语义；其余测试不经过（此前 panic 占位）。
func (f *fakeStore) QueryFacts(_ context.Context, q store.FactQuery) ([]fact.Fact, error) {
	out := []fact.Fact{}
	for _, ft := range f.facts {
		if q.Kind != "" && ft.Kind != q.Kind {
			continue
		}
		if !q.From.IsZero() && ft.Ts.Before(q.From) {
			continue
		}
		out = append(out, ft)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Ts.Before(out[j].Ts) })
	if q.Limit > 0 && q.Limit < len(out) {
		out = out[:q.Limit]
	}
	return out, nil
}
func (f *fakeStore) UpsertRule(context.Context, store.Rule) (int64, error) {
	panic("fakeStore: UpsertRule not used")
}
func (f *fakeStore) ListRules(context.Context) ([]store.Rule, error) {
	panic("fakeStore: ListRules not used")
}
func (f *fakeStore) GetTriggerState(context.Context, int64) (store.TriggerState, error) {
	panic("fakeStore: GetTriggerState not used")
}
func (f *fakeStore) PutTriggerState(context.Context, store.TriggerState) error {
	panic("fakeStore: PutTriggerState not used")
}
func (f *fakeStore) InsertAlert(context.Context, store.Alert) error {
	panic("fakeStore: InsertAlert not used")
}
func (f *fakeStore) PendingAlerts(context.Context, int) ([]store.Alert, error) {
	panic("fakeStore: PendingAlerts not used")
}
func (f *fakeStore) MarkAlertDelivered(context.Context, int64) error {
	panic("fakeStore: MarkAlertDelivered not used")
}
func (f *fakeStore) ListAlerts(context.Context, int, int) ([]store.Alert, error) {
	panic("fakeStore: ListAlerts not used")
}
func (f *fakeStore) AckAlert(context.Context, int64) error {
	panic("fakeStore: AckAlert not used")
}
func (f *fakeStore) ListTriggerStates(context.Context) ([]store.RuleState, error) {
	panic("fakeStore: ListTriggerStates not used")
}
func (f *fakeStore) ListUnacked(context.Context) ([]store.Alert, error) {
	panic("fakeStore: ListUnacked not used")
}
func (f *fakeStore) AckAll(context.Context) (int64, error) {
	panic("fakeStore: AckAll not used")
}
func (f *fakeStore) InsertLedgerEntry(context.Context, store.LedgerEntry) (int64, error) {
	panic("fakeStore: InsertLedgerEntry not used")
}
func (f *fakeStore) ListLedgerEntries(context.Context, int, int) ([]store.LedgerEntry, error) {
	panic("fakeStore: ListLedgerEntries not used")
}
func (f *fakeStore) LedgerSummary(context.Context) ([]store.TierSummary, error) {
	panic("fakeStore: LedgerSummary not used")
}
func (f *fakeStore) InsertSimOrder(context.Context, store.SimOrder) (int64, error) {
	panic("fakeStore: InsertSimOrder not used")
}
func (f *fakeStore) UpdateSimOrderStatus(context.Context, int64, string, string) error {
	panic("fakeStore: UpdateSimOrderStatus not used")
}
func (f *fakeStore) FillSimOrder(context.Context, int64, string, []store.SimPosition) error {
	panic("fakeStore: FillSimOrder not used")
}
func (f *fakeStore) TodaySimNotional(context.Context, time.Time) (float64, error) {
	panic("fakeStore: TodaySimNotional not used")
}
func (f *fakeStore) InsertSimPosition(context.Context, store.SimPosition) (int64, error) {
	panic("fakeStore: InsertSimPosition not used")
}
// ListOpenSimPositions 返回 open 腿（symbol/venue 空 = 不限；CloseSimOrder 数据面）。
func (f *fakeStore) ListOpenSimPositions(_ context.Context, symbol, venue string) ([]store.SimPosition, error) {
	out := []store.SimPosition{}
	for _, p := range f.positions {
		if p.Status != store.SimPosStatusOpen {
			continue
		}
		if symbol != "" && p.Symbol != symbol {
			continue
		}
		if venue != "" && p.Venue != venue {
			continue
		}
		out = append(out, p)
	}
	return out, nil
}
// nextFlowID 自增流水 id（fake 内存版）。
func (f *fakeStore) nextFlowID() int64 {
	f.nextFlowIDNum++
	return f.nextFlowIDNum
}

// SettleSimPositionFunding 资金费入账（D-056 真语义，与 pgstore 一致）：腿 pnl += addPnl
// + funding 现金流 + 现金余额 += addPnl。
func (f *fakeStore) SettleSimPositionFunding(_ context.Context, id, orderID int64, addPnl float64) error {
	for i := range f.positions {
		if f.positions[i].ID == id {
			f.positions[i].PnL += addPnl
			f.positions[i].UpdatedAt = t0
			f.cash += addPnl
			f.flows = append(f.flows, store.CashFlow{ID: f.nextFlowID(), Ts: t0,
				OrderID: orderID, LegID: id, Kind: store.CashKindFunding, Amount: addPnl})
			return nil
		}
	}
	return nil
}

// InitSimAccount 首启入金（幂等，重启不重置）。
func (f *fakeStore) InitSimAccount(_ context.Context, capital float64) error {
	if capital <= 0 {
		return errors.New("fakeStore: InitSimAccount: capital must be > 0")
	}
	if f.capital == 0 && len(f.flows) == 0 {
		f.capital = capital
		f.cash = capital
		f.flows = append(f.flows, store.CashFlow{ID: f.nextFlowID(), Ts: t0,
			Kind: store.CashKindCapitalIn, Amount: capital})
	}
	return nil
}

func (f *fakeStore) GetSimAccount(context.Context) (store.SimAccount, error) {
	return store.SimAccount{Capital: f.capital, Cash: f.cash, UpdatedAt: t0}, nil
}

// ListCashFlows 按 id DESC 分页返回（审计账本）。
func (f *fakeStore) ListCashFlows(_ context.Context, limit, offset int) ([]store.CashFlow, error) {
	out := append([]store.CashFlow(nil), f.flows...)
	sort.Slice(out, func(i, j int) bool { return out[i].ID > out[j].ID })
	if offset < 0 {
		offset = 0
	}
	if offset >= len(out) {
		return []store.CashFlow{}, nil
	}
	out = out[offset:]
	if limit > 0 && limit < len(out) {
		out = out[:limit]
	}
	return out, nil
}

// InsertEquitySnapshot 落一份快照（D-062 判定门① 数据面）。
func (f *fakeStore) InsertEquitySnapshot(_ context.Context, s store.EquitySnapshot) error {
	f.snaps = append(f.snaps, s)
	return nil
}

// ListEquitySnapshots 按 ts ASC 返回 [since, +∞) 内快照（TWR 链乘顺序）。
func (f *fakeStore) ListEquitySnapshots(_ context.Context, since time.Time, limit int) ([]store.EquitySnapshot, error) {
	out := []store.EquitySnapshot{}
	for _, s := range f.snaps {
		if !since.IsZero() && s.Ts.Before(since) {
			continue
		}
		out = append(out, s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Ts.Before(out[j].Ts) })
	if limit > 0 && limit < len(out) {
		out = out[:limit]
	}
	return out, nil
}
func (f *fakeStore) ListKnowledgeEntries(context.Context) ([]store.KnowledgeEntry, error) {
	return nil, nil
}
func (f *fakeStore) UpsertKnowledgeEntry(context.Context, store.KnowledgeEntry) (int64, error) {
	panic("fakeStore: UpsertKnowledgeEntry not used")
}

func (f *fakeStore) ReviewKnowledgeEntry(context.Context, string, string, string, string, string) error {
	panic("fakeStore: ReviewKnowledgeEntry not used")
}

// t0 / t0Facts：服务测试统一锚定时钟（practices #10：时钟注入覆盖确认成交腿时间戳）。
var t0 = time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)

// service 构造注入固定时钟的 SimService + 直连调用（不经 HTTP，覆盖全部 RPC 逻辑面）。
func service(st store.Store, cfg sim.Config) *Service {
	s := NewService(st, cfg)
	s.Now = func() time.Time { return t0 }
	return s
}

// fundingOrder 返回一条典型的 funding_hedge 建议订单（ref=100 / spread=5，二次门禁通过基线）。
func fundingOrder(id int64) store.SimOrder {
	return store.SimOrder{ID: id, Ts: t0.Add(-time.Hour), SrcRule: "funding_warn",
		Kind: store.SimKindFundingHedge, Venue: "binance", Symbol: "BTC",
		Side: store.SimSideHedge, Qty: 10000, RefPrice: 100, ExpectedSpread: 5,
		Status: store.SimStatusSuggested}
}

func confirmReq(id int64) *connect.Request[simv1.ConfirmSimOrderRequest] {
	return connect.NewRequest(&simv1.ConfirmSimOrderRequest{Id: id})
}

// repoOrder 返回典型 repo 建议订单（D-039：ref=面值 100 恒定 / spread=当日逆回购利率）。
func repoOrder(id int64) store.SimOrder {
	return store.SimOrder{ID: id, Ts: t0.Add(-time.Hour), SrcRule: "reverse_repo_timing",
		Kind: store.SimKindRepo, Venue: "domestic", Symbol: "GC001",
		Side: store.SimSideLong, Qty: 10000, RefPrice: 100, ExpectedSpread: 5,
		Status: store.SimStatusSuggested}
}

// carryOrder 返回典型 carry_asset 建议订单（D-039：ref=稳定币面值锚 1.0 / spread=生息年化）。
func carryOrder(id int64) store.SimOrder {
	return store.SimOrder{ID: id, Ts: t0.Add(-time.Hour), SrcRule: "defi_large_tier_change",
		Kind: store.SimKindCarryAsset, Venue: "sim_local", Symbol: "USDT",
		Side: store.SimSideLong, Qty: 10000, RefPrice: 1.0, ExpectedSpread: 5,
		Status: store.SimStatusSuggested}
}

// TestListSimOrdersEmptyAndStatusFilter：空库返回 [] 不报错；status 过滤生效。
func TestListSimOrdersEmptyAndStatusFilter(t *testing.T) {
	st := newFakeStore()
	s := service(st, sim.Config{})

	// 空库 → []（非 nil）。
	resp, err := s.ListSimOrders(context.Background(), connect.NewRequest(&simv1.ListSimOrdersRequest{}))
	if err != nil {
		t.Fatalf("ListSimOrders(empty): %v", err)
	}
	if resp.Msg.Orders == nil || len(resp.Msg.Orders) != 0 {
		t.Fatalf("orders = %#v, want 空 []", resp.Msg.Orders)
	}

	// 两单：suggested + rejected；status=suggested → 只回 suggested。
	st.addOrder(fundingOrder(0))
	r := fundingOrder(0)
	r.Status = store.SimStatusRejected
	st.addOrder(r)

	all, err := s.ListSimOrders(context.Background(), connect.NewRequest(&simv1.ListSimOrdersRequest{}))
	if err != nil || len(all.Msg.Orders) != 2 {
		t.Fatalf("ListSimOrders(all) = %d, %v, want 2", len(all.Msg.Orders), err)
	}
	sug, err := s.ListSimOrders(context.Background(),
		connect.NewRequest(&simv1.ListSimOrdersRequest{Status: store.SimStatusSuggested}))
	if err != nil || len(sug.Msg.Orders) != 1 || sug.Msg.Orders[0].Status != store.SimStatusSuggested {
		t.Fatalf("ListSimOrders(suggested) = %d orders, %v, want 1 suggested", len(sug.Msg.Orders), err)
	}
}

// TestConfirmSimOrderAccept：二次门禁通过 → 原子成交（accepted=true，订单 filled，
// legs 建腿口径与 sim.BuildLegs 一致、时间戳用注入时钟）。
func TestConfirmSimOrderAccept(t *testing.T) {
	st := newFakeStore()
	st.addOrder(fundingOrder(0))
	st.addFact(fact.KindTicker, "binance", "BTC", 100, t0.Add(-time.Minute))
	st.addFact(fact.KindFunding, "binance", "BTC", 5, t0.Add(-time.Minute))
	s := service(st, sim.Config{})

	resp, err := s.ConfirmSimOrder(context.Background(), confirmReq(1))
	if err != nil {
		t.Fatalf("ConfirmSimOrder(accept): %v", err)
	}
	if !resp.Msg.Accepted || resp.Msg.Order.Status != store.SimStatusFilled {
		t.Fatalf("accepted = %v, status = %q, want true/filled", resp.Msg.Accepted, resp.Msg.Order.Status)
	}
	if len(st.accepted) != 1 || len(st.rejected) != 0 {
		t.Fatalf("accepted = %d, rejected = %d, want 1/0", len(st.accepted), len(st.rejected))
	}
	// 建腿时间戳 = 注入时钟（practices #10）。
	call := st.accepted[0]
	if call.id != 1 || len(call.legs) != 2 {
		t.Fatalf("AcceptSimOrder call = %+v, want id=1 legs=2", call)
	}
	if !call.legs[0].Ts.Equal(t0) || !call.legs[1].Ts.Equal(t0) {
		t.Fatalf("leg Ts = %v/%v, want 注入时钟 %v", call.legs[0].Ts, call.legs[1].Ts, t0)
	}
	// funding_hedge = 现货 long 非 funding + 永续 short funding（与 sim.BuildLegs 口径一致）。
	if call.legs[0].Side != store.SimSideLong || call.legs[0].Funding || call.legs[0].OrderID != 1 {
		t.Errorf("leg0 = %+v, want long 非 funding OrderID=1", call.legs[0])
	}
	if call.legs[1].Side != store.SimSideShort || !call.legs[1].Funding {
		t.Errorf("leg1 = %+v, want short funding", call.legs[1])
	}
}

// TestConfirmSimOrderNonSuggested：非 suggested（filled/rejected/confirmed）→
// FailedPrecondition，且不产生任何写（防重复确认）。
func TestConfirmSimOrderNonSuggested(t *testing.T) {
	for _, status := range []string{store.SimStatusFilled, store.SimStatusRejected, store.SimStatusConfirmed} {
		st := newFakeStore()
		o := fundingOrder(0)
		o.Status = status
		st.addOrder(o)
		s := service(st, sim.Config{})

		_, err := s.ConfirmSimOrder(context.Background(), confirmReq(1))
		if err == nil {
			t.Fatalf("%s: ConfirmSimOrder = nil, want error", status)
		}
		if connect.CodeOf(err) != connect.CodeFailedPrecondition {
			t.Fatalf("%s: code = %v, want FailedPrecondition", status, connect.CodeOf(err))
		}
		if len(st.accepted)+len(st.rejected) != 0 {
			t.Fatalf("%s: accepted+rejected = %d, want 0（不得写入）", status, len(st.accepted)+len(st.rejected))
		}
	}
}

// TestConfirmSimOrderInvalidId：id ≤ 0 → InvalidArgument。
func TestConfirmSimOrderInvalidId(t *testing.T) {
	s := service(newFakeStore(), sim.Config{})
	_, err := s.ConfirmSimOrder(context.Background(), confirmReq(0))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want InvalidArgument", connect.CodeOf(err))
	}
}

// TestConfirmSimOrderUnknownId：未知 id → 存储错误（Unavailable，与 dashboard 同口径）。
func TestConfirmSimOrderUnknownId(t *testing.T) {
	s := service(newFakeStore(), sim.Config{})
	_, err := s.ConfirmSimOrder(context.Background(), confirmReq(999))
	if connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatalf("code = %v, want Unavailable", connect.CodeOf(err))
	}
}

// TestConfirmSimOrderDriftReject：确认时刻行情漂移 > 2% → 拒单（accepted=false，
// RejectSimOrder 带 SPREAD_DRIFT flag，订单 rejected，note 含原因）。
func TestConfirmSimOrderDriftReject(t *testing.T) {
	st := newFakeStore()
	st.addOrder(fundingOrder(0))
	st.addFact(fact.KindTicker, "binance", "BTC", 105, t0.Add(-time.Minute)) // +5% > 2%
	st.addFact(fact.KindFunding, "binance", "BTC", 5, t0.Add(-time.Minute))
	s := service(st, sim.Config{})

	resp, err := s.ConfirmSimOrder(context.Background(), confirmReq(1))
	if err != nil {
		t.Fatalf("ConfirmSimOrder(drift): %v", err)
	}
	if resp.Msg.Accepted {
		t.Fatal("accepted = true, want false（漂移拒单）")
	}
	if resp.Msg.Order.Status != store.SimStatusRejected {
		t.Fatalf("status = %q, want rejected", resp.Msg.Order.Status)
	}
	if len(st.rejected) != 1 || len(st.accepted) != 0 {
		t.Fatalf("accepted = %d, rejected = %d, want 0/1", len(st.accepted), len(st.rejected))
	}
	call := st.rejected[0]
	if !slices.Contains(call.flags, sim.RiskSpreadDrift) {
		t.Fatalf("flags = %v, want 含 SPREAD_DRIFT", call.flags)
	}
	if !strings.Contains(call.reason, "SPREAD_DRIFT") {
		t.Fatalf("reason = %q, want 含 SPREAD_DRIFT", call.reason)
	}
}

// TestConfirmSimOrderFailClosedNoData：确认时刻查不到 ticker/funding → fail-closed
// 拒单（§10.3 无数据不确认；负样本保留）。
func TestConfirmSimOrderFailClosedNoData(t *testing.T) {
	for _, missing := range []string{"ticker", "funding"} {
		st := newFakeStore()
		st.addOrder(fundingOrder(0))
		if missing != "ticker" {
			st.addFact(fact.KindTicker, "binance", "BTC", 100, t0.Add(-time.Minute))
		}
		if missing != "funding" {
			st.addFact(fact.KindFunding, "binance", "BTC", 5, t0.Add(-time.Minute))
		}
		s := service(st, sim.Config{})

		resp, err := s.ConfirmSimOrder(context.Background(), confirmReq(1))
		if err != nil {
			t.Fatalf("%s: ConfirmSimOrder: %v", missing, err)
		}
		if resp.Msg.Accepted || resp.Msg.Order.Status != store.SimStatusRejected {
			t.Fatalf("%s: accepted = %v, status = %q, want false/rejected", missing, resp.Msg.Accepted, resp.Msg.Order.Status)
		}
		if len(st.rejected) != 1 {
			t.Fatalf("%s: rejected = %d, want 1", missing, len(st.rejected))
		}
		if !strings.Contains(st.rejected[0].reason, "fail-closed") {
			t.Fatalf("%s: reason = %q, want 含 fail-closed", missing, st.rejected[0].reason)
		}
	}
}

// closeSeed 构造 filled 订单 + funding_hedge 两 open 腿（short 已结算 funding 30）。
// 返回 fakeStore 供 CloseSimOrder 各用例复用。
func closeSeed() *fakeStore {
	st := newFakeStore()
	o := fundingOrder(0)
	o.Status = store.SimStatusFilled
	st.addOrder(o)
	st.positions = append(st.positions,
		store.SimPosition{ID: 10, OrderID: 1, Ts: t0, Kind: store.SimKindFundingHedge,
			Venue: "binance", Symbol: "BTC", Side: store.SimSideLong, Qty: 100, RefPrice: 100,
			Funding: false, PnL: 0, Status: store.SimPosStatusOpen},
		store.SimPosition{ID: 11, OrderID: 1, Ts: t0, Kind: store.SimKindFundingHedge,
			Venue: "binance", Symbol: "BTC", Side: store.SimSideShort, Qty: 100, RefPrice: 100,
			Funding: true, PnL: 30, Status: store.SimPosStatusOpen},
	)
	return st
}

func closeReq(id int64) *connect.Request[simv1.CloseSimOrderRequest] {
	return connect.NewRequest(&simv1.CloseSimOrderRequest{Id: id, Note: "人工平仓"})
}

// TestCloseSimOrder：filled 订单 + open 腿 → 按当前价结算浮动 → 整单平。订单 closed、
// 两腿 settled；realized = 已结算 funding + 浮动合计；realized_rmb = 即期折算。
// funding_hedge 对冲净值 = 价格浮动两腿抵消（long +1000 / short -1000）+ funding 30。
func TestCloseSimOrder(t *testing.T) {
	st := closeSeed()
	st.addFact(fact.KindTicker, "binance", "BTC", 110, t0.Add(-time.Minute)) // +10% vs ref 100
	st.addFact(fact.KindFX, fxVenue, fxSymbol, 7.2, t0.Add(-time.Minute))
	s := service(st, sim.Config{})

	resp, err := s.CloseSimOrder(context.Background(), closeReq(1))
	if err != nil {
		t.Fatalf("CloseSimOrder: %v", err)
	}
	if resp.Msg.OrderId != 1 || resp.Msg.ClosedLegs != 2 {
		t.Fatalf("resp = %+v, want order 1 closed_legs=2", resp.Msg)
	}
	// long: 0 + (110-100)*100*+1 = 1000；short: 30 + (110-100)*100*-1 = -970。
	if resp.Msg.RealizedPnl != 30 {
		t.Fatalf("realized_pnl = %v, want 30（浮动对冲抵消 + funding 30）", resp.Msg.RealizedPnl)
	}
	if resp.Msg.RealizedRmb != 30*7.2 {
		t.Fatalf("realized_rmb = %v, want %v", resp.Msg.RealizedRmb, 30*7.2)
	}
	got, _ := st.orderByID(1)
	if got.Status != store.SimStatusClosed || got.Note != "人工平仓" {
		t.Fatalf("order = %+v, want closed + note 人工平仓", got)
	}
	byID := map[int64]store.SimPosition{}
	for _, p := range st.positions {
		byID[p.ID] = p
	}
	if byID[10].Status != store.SimPosStatusSettled || byID[10].PnL != 1000 {
		t.Fatalf("long = %+v, want settled pnl=1000", byID[10])
	}
	if byID[11].Status != store.SimPosStatusSettled || byID[11].PnL != -970 {
		t.Fatalf("short = %+v, want settled pnl=-970", byID[11])
	}
	// 平仓后再平 → 订单非 filled → FailedPrecondition（防重复平）。
	_, err = s.CloseSimOrder(context.Background(), closeReq(1))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("second close code = %v, want FailedPrecondition", connect.CodeOf(err))
	}
}

// TestCloseSimOrderTickerMissing：ticker 查不到 → 浮动 add=0（宁缺毋滥不编造），
// realized 仅含已结算 funding；fx 缺失 → realized_rmb=0（前端标 USD 原值，D-047）。
func TestCloseSimOrderTickerMissing(t *testing.T) {
	st := closeSeed()
	s := service(st, sim.Config{}) // 无 ticker / 无 fx 事实

	resp, err := s.CloseSimOrder(context.Background(), closeReq(1))
	if err != nil {
		t.Fatalf("CloseSimOrder(no ticker): %v", err)
	}
	if resp.Msg.RealizedPnl != 30 || resp.Msg.ClosedLegs != 2 {
		t.Fatalf("resp = %+v, want realized=30（仅 funding）/legs=2", resp.Msg)
	}
	if resp.Msg.RealizedRmb != 0 {
		t.Fatalf("realized_rmb = %v, want 0（fx 缺失）", resp.Msg.RealizedRmb)
	}
}

// TestCloseSimOrderNonFilled：非 filled（suggested/rejected/closed）→ FailedPrecondition，
// 不产生任何写。
func TestCloseSimOrderNonFilled(t *testing.T) {
	for _, status := range []string{store.SimStatusSuggested, store.SimStatusRejected, store.SimStatusClosed} {
		st := newFakeStore()
		o := fundingOrder(0)
		o.Status = status
		st.addOrder(o)
		s := service(st, sim.Config{})

		_, err := s.CloseSimOrder(context.Background(), closeReq(1))
		if connect.CodeOf(err) != connect.CodeFailedPrecondition {
			t.Fatalf("%s: code = %v, want FailedPrecondition", status, connect.CodeOf(err))
		}
		if len(st.positions) != 0 {
			t.Fatalf("%s: positions written = %d, want 0（不得写入）", status, len(st.positions))
		}
	}
}

// TestCloseSimOrderUnknownId：未知订单 → GetSimOrder ErrNotFound → Unavailable。
func TestCloseSimOrderUnknownId(t *testing.T) {
	s := service(newFakeStore(), sim.Config{})
	_, err := s.CloseSimOrder(context.Background(), closeReq(999))
	if connect.CodeOf(err) != connect.CodeUnavailable {
		t.Fatalf("code = %v, want Unavailable", connect.CodeOf(err))
	}
}

// TestCloseSimOrderNoOpenLegs：filled 订单但无 open 腿（已无持仓）→ InvalidArgument。
func TestCloseSimOrderNoOpenLegs(t *testing.T) {
	st := newFakeStore()
	o := fundingOrder(0)
	o.Status = store.SimStatusFilled
	st.addOrder(o)
	s := service(st, sim.Config{})

	_, err := s.CloseSimOrder(context.Background(), closeReq(1))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want InvalidArgument（无可平腿）", connect.CodeOf(err))
	}
}

// TestCloseSimOrderInvalidId：id ≤ 0 → InvalidArgument。
func TestCloseSimOrderInvalidId(t *testing.T) {
	s := service(newFakeStore(), sim.Config{})
	_, err := s.CloseSimOrder(context.Background(), closeReq(0))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("code = %v, want InvalidArgument", connect.CodeOf(err))
	}
}

// accountSeed 构造现金账本对账种子：入金 100k + open 对冲两腿
// （long 100@100 / short 100@100）→ 资金费结算 short +30（入现金账本），ticker=110。
// 期望：realized=30；unrealized=0（对冲浮动抵消）；market_value=0（11000−11000）；
// equity = cash(100030) + 0 = 100030 = capital(100k)+realized(30)+unrealized(0)。
func accountSeed(t *testing.T) *fakeStore {
	t.Helper()
	st := newFakeStore()
	o := fundingOrder(0)
	o.Status = store.SimStatusFilled
	st.addOrder(o) // id=1，与腿 OrderID 对应（CloseSimOrder 需 filled 订单）
	if err := st.InitSimAccount(context.Background(), 100_000); err != nil {
		t.Fatalf("InitSimAccount: %v", err)
	}
	st.positions = append(st.positions,
		store.SimPosition{ID: 10, OrderID: 1, Ts: t0, Kind: store.SimKindFundingHedge,
			Venue: "binance", Symbol: "BTC", Side: store.SimSideLong, Qty: 100, RefPrice: 100,
			Funding: false, PnL: 0, Status: store.SimPosStatusOpen},
		store.SimPosition{ID: 11, OrderID: 1, Ts: t0, Kind: store.SimKindFundingHedge,
			Venue: "binance", Symbol: "BTC", Side: store.SimSideShort, Qty: 100, RefPrice: 100,
			Funding: true, PnL: 0, Status: store.SimPosStatusOpen},
	)
	if err := st.SettleSimPositionFunding(context.Background(), 11, 1, 30); err != nil {
		t.Fatalf("SettleSimPositionFunding: %v", err)
	}
	st.addFact(fact.KindTicker, "binance", "BTC", 110, t0.Add(-time.Minute))
	st.addFact(fact.KindFX, fxVenue, fxSymbol, 7.2, t0.Add(-time.Minute))
	return st
}

// TestGetSimAccount：现金账本对账——双恒等式交叉校验（D-056）。
func TestGetSimAccount(t *testing.T) {
	st := accountSeed(t)
	s := service(st, sim.Config{})

	resp, err := s.GetSimAccount(context.Background(), connect.NewRequest(&simv1.GetSimAccountRequest{}))
	if err != nil {
		t.Fatalf("GetSimAccount: %v", err)
	}
	m := resp.Msg
	if m.Capital != 100_000 || m.Cash != 100_030 {
		t.Fatalf("capital/cash = %v/%v, want 100000/100030（入金 100k + funding 30）", m.Capital, m.Cash)
	}
	if m.RealizedPnl != 30 {
		t.Fatalf("realized = %v, want 30（Σ 腿 pnl）", m.RealizedPnl)
	}
	if m.UnrealizedPnl != 0 {
		t.Fatalf("unrealized = %v, want 0（对冲浮动抵消）", m.UnrealizedPnl)
	}
	if m.MarketValue != 0 {
		t.Fatalf("market_value = %v, want 0（11000−11000）", m.MarketValue)
	}
	if m.Equity != 100_030 {
		t.Fatalf("equity = %v, want 100030（cash + market_value）", m.Equity)
	}
	if !m.FxAvailable || m.EquityRmb != 100_030*7.2 {
		t.Fatalf("fx = %v/rmb = %v, want true/720216", m.FxAvailable, m.EquityRmb)
	}
	// 双恒等式：equity = cash + market_value = capital + realized + unrealized。
	if got := m.Capital + m.RealizedPnl + m.UnrealizedPnl; got != m.Equity {
		t.Fatalf("capital+realized+unrealized = %v ≠ equity %v（对账断裂）", got, m.Equity)
	}
	// 流水：capital_in + funding 两条，id DESC（funding 在前）。
	if len(m.Flows) != 2 {
		t.Fatalf("flows = %d, want 2（capital_in + funding）", len(m.Flows))
	}
	if m.Flows[0].Kind != store.CashKindFunding || m.Flows[0].Amount != 30 {
		t.Fatalf("flows[0] = %+v, want funding +30（id DESC）", m.Flows[0])
	}
	if m.Flows[1].Kind != store.CashKindCapitalIn || m.Flows[1].Amount != 100_000 {
		t.Fatalf("flows[1] = %+v, want capital_in +100000", m.Flows[1])
	}
}

// TestGetSimAccountEmpty：账户未初始化（InitSimAccount 未跑）→ 零值兜底，不报错。
func TestGetSimAccountEmpty(t *testing.T) {
	s := service(newFakeStore(), sim.Config{})
	resp, err := s.GetSimAccount(context.Background(), connect.NewRequest(&simv1.GetSimAccountRequest{}))
	if err != nil {
		t.Fatalf("GetSimAccount: %v", err)
	}
	if resp.Msg.Capital != 0 || resp.Msg.Cash != 0 || resp.Msg.Equity != 0 {
		t.Fatalf("resp = %+v, want 全零兜底（未入金）", resp.Msg)
	}
}

// TestGetSimAccountCloseReconcile：平仓后现金账本与腿 pnl 对账——close 现金流
// （long +qty×cur / short −qty×cur）入账后，equity 恒等于 capital + realized。
func TestGetSimAccountCloseReconcile(t *testing.T) {
	st := accountSeed(t)
	s := service(st, sim.Config{})
	// 平仓（ticker 110）：long 浮动 +1000、short −1000；close 现金流 long +11000 / short −11000。
	if _, err := s.CloseSimOrder(context.Background(), closeReq(1)); err != nil {
		t.Fatalf("CloseSimOrder: %v", err)
	}
	resp, err := s.GetSimAccount(context.Background(), connect.NewRequest(&simv1.GetSimAccountRequest{}))
	if err != nil {
		t.Fatalf("GetSimAccount: %v", err)
	}
	m := resp.Msg
	// cash = 100030 + (11000 − 11000) = 100030；realized = 30 + 0（浮动抵消）。
	if m.Cash != 100_030 {
		t.Fatalf("cash = %v, want 100030（close 净现金流 0）", m.Cash)
	}
	if m.RealizedPnl != 30 || m.UnrealizedPnl != 0 || m.MarketValue != 0 {
		t.Fatalf("realized/unrealized/mv = %v/%v/%v, want 30/0/0（全平后无敞口）", m.RealizedPnl, m.UnrealizedPnl, m.MarketValue)
	}
	if got := m.Capital + m.RealizedPnl + m.UnrealizedPnl; got != m.Equity {
		t.Fatalf("capital+realized+unrealized = %v ≠ equity %v（对账断裂）", got, m.Equity)
	}
	// 流水含 close 两条：funding +30 之后是 close 11000 / −11000（id DESC：最新在前）。
	if len(m.Flows) != 4 {
		t.Fatalf("flows = %d, want 4（capital_in + funding + close×2）", len(m.Flows))
	}
	if m.Flows[0].Kind != store.CashKindClose || m.Flows[0].Amount != -11_000 {
		t.Fatalf("flows[0] = %+v, want close −11000（short 腿买回）", m.Flows[0])
	}
	if m.Flows[1].Kind != store.CashKindClose || m.Flows[1].Amount != 11_000 {
		t.Fatalf("flows[1] = %+v, want close +11000（long 腿卖出）", m.Flows[1])
	}
}

// TestConfirmSimOrderSpreadReject：预期年化变化 > 20% → 拒单（第二门独立触发）。
func TestConfirmSimOrderSpreadReject(t *testing.T) {
	st := newFakeStore()
	st.addOrder(fundingOrder(0))
	st.addFact(fact.KindTicker, "binance", "BTC", 100, t0.Add(-time.Minute))
	st.addFact(fact.KindFunding, "binance", "BTC", 6.5, t0.Add(-time.Minute)) // +30% > 20%
	s := service(st, sim.Config{})

	resp, err := s.ConfirmSimOrder(context.Background(), confirmReq(1))
	if err != nil {
		t.Fatalf("ConfirmSimOrder(spread): %v", err)
	}
	if resp.Msg.Accepted {
		t.Fatal("accepted = true, want false（价差漂移拒单）")
	}
	if len(st.rejected) != 1 || !slices.Contains(st.rejected[0].flags, sim.RiskSpreadDrift) {
		t.Fatalf("rejected = %+v, want SPREAD_DRIFT 拒单", st.rejected)
	}
}

// —— D-039 kind 分派数据面：repo/carry 确认流（M3-c 验收发现：spec §10.3 硬编码
// ticker/funding 双查让 repo/carry 恒拒。修复后按 kind 选权威源，本组测试为回归锚点——
// 删 kind 分派（回到硬编码双查）→ TestConfirmSimOrderRepoAccept / TestConfirmSimOrderCarryAccept 必红。——

// TestConfirmSimOrderRepoAccept：repo 确认 → 查 reverse_repo 当日利率（非 ticker/funding），
// ref=面值 100 漂移恒 0；利率未变 → 通过（accepted=true，杜绝"repo 恒拒"）。
func TestConfirmSimOrderRepoAccept(t *testing.T) {
	st := newFakeStore()
	st.addOrder(repoOrder(0))
	st.addFact(fact.KindReverseRepo, "", "", 5, t0.Add(-time.Minute)) // 当日回购利率同生成时
	s := service(st, sim.Config{})

	resp, err := s.ConfirmSimOrder(context.Background(), confirmReq(1))
	if err != nil {
		t.Fatalf("ConfirmSimOrder(repo accept): %v", err)
	}
	if !resp.Msg.Accepted || resp.Msg.Order.Status != store.SimStatusFilled {
		t.Fatalf("accepted = %v, status = %q, want true/filled（repo 不得恒拒）", resp.Msg.Accepted, resp.Msg.Order.Status)
	}
	if len(st.accepted) != 1 || len(st.rejected) != 0 {
		t.Fatalf("accepted = %d, rejected = %d, want 1/0", len(st.accepted), len(st.rejected))
	}
}

// TestConfirmSimOrderRepoReject：确认时点回购利率变化 >20%（5 → 6.5）→ SPREAD_DRIFT 拒。
// repo 的真实漂移风险是利率变化（锁定时点利率），面值漂移恒 0。
func TestConfirmSimOrderRepoReject(t *testing.T) {
	st := newFakeStore()
	st.addOrder(repoOrder(0))
	st.addFact(fact.KindReverseRepo, "", "", 6.5, t0.Add(-time.Minute)) // +30% > 20%
	s := service(st, sim.Config{})

	resp, err := s.ConfirmSimOrder(context.Background(), confirmReq(1))
	if err != nil {
		t.Fatalf("ConfirmSimOrder(repo reject): %v", err)
	}
	if resp.Msg.Accepted || resp.Msg.Order.Status != store.SimStatusRejected {
		t.Fatalf("accepted = %v, status = %q, want false/rejected", resp.Msg.Accepted, resp.Msg.Order.Status)
	}
	if len(st.rejected) != 1 || !slices.Contains(st.rejected[0].flags, sim.RiskSpreadDrift) {
		t.Fatalf("rejected = %+v, want SPREAD_DRIFT 拒单", st.rejected)
	}
}

// TestConfirmSimOrderRepoFailClosed：确认时点查不到 reverse_repo → fail-closed 拒
// （与生成侧 repoSignal「无事实不建单」同口径，宁缺毋滥）。
func TestConfirmSimOrderRepoFailClosed(t *testing.T) {
	st := newFakeStore()
	st.addOrder(repoOrder(0))
	s := service(st, sim.Config{})

	resp, err := s.ConfirmSimOrder(context.Background(), confirmReq(1))
	if err != nil {
		t.Fatalf("ConfirmSimOrder(repo fail-closed): %v", err)
	}
	if resp.Msg.Accepted || resp.Msg.Order.Status != store.SimStatusRejected {
		t.Fatalf("accepted = %v, status = %q, want false/rejected（fail-closed）", resp.Msg.Accepted, resp.Msg.Order.Status)
	}
	if len(st.rejected) != 1 || !strings.Contains(st.rejected[0].reason, "fail-closed") {
		t.Fatalf("rejected = %+v, want fail-closed 拒单", st.rejected)
	}
}

// TestConfirmSimOrderCarryAccept：carry 确认 → 查 defi_rate 生息年化；无 ticker（稳定币
// 无现价行情）→ ref=面值锚 1.0 漂移恒 0，只查年化；未变 → 通过（杜绝"carry 恒拒"）。
func TestConfirmSimOrderCarryAccept(t *testing.T) {
	st := newFakeStore()
	st.addOrder(carryOrder(0))
	st.addFact(fact.KindDefiRate, "sim_local", "USDT", 5, t0.Add(-time.Minute))
	s := service(st, sim.Config{})

	resp, err := s.ConfirmSimOrder(context.Background(), confirmReq(1))
	if err != nil {
		t.Fatalf("ConfirmSimOrder(carry accept): %v", err)
	}
	if !resp.Msg.Accepted || resp.Msg.Order.Status != store.SimStatusFilled {
		t.Fatalf("accepted = %v, status = %q, want true/filled（carry 不得恒拒）", resp.Msg.Accepted, resp.Msg.Order.Status)
	}
	if len(st.accepted) != 1 || len(st.rejected) != 0 {
		t.Fatalf("accepted = %d, rejected = %d, want 1/0", len(st.accepted), len(st.rejected))
	}
}

// TestConfirmSimOrderCarryTickerDrift：carry 有 ticker 时 ref 检查生效——稳定币现价
// 1.0 → 1.05（+5% > 2%）→ SPREAD_DRIFT 拒（ticker 数据存在则价格漂移不能放过）。
func TestConfirmSimOrderCarryTickerDrift(t *testing.T) {
	st := newFakeStore()
	st.addOrder(carryOrder(0))
	st.addFact(fact.KindDefiRate, "sim_local", "USDT", 5, t0.Add(-time.Minute))
	st.addFact(fact.KindTicker, "sim_local", "USDT", 1.05, t0.Add(-time.Minute)) // +5%
	s := service(st, sim.Config{})

	resp, err := s.ConfirmSimOrder(context.Background(), confirmReq(1))
	if err != nil {
		t.Fatalf("ConfirmSimOrder(carry ticker drift): %v", err)
	}
	if resp.Msg.Accepted || resp.Msg.Order.Status != store.SimStatusRejected {
		t.Fatalf("accepted = %v, status = %q, want false/rejected", resp.Msg.Accepted, resp.Msg.Order.Status)
	}
	if len(st.rejected) != 1 || !slices.Contains(st.rejected[0].flags, sim.RiskSpreadDrift) {
		t.Fatalf("rejected = %+v, want SPREAD_DRIFT 拒单", st.rejected)
	}
}

// TestConfirmSimOrderCarrySpreadReject：生息年化变化 >20%（5 → 6.5）→ SPREAD_DRIFT 拒
// （无 ticker 时 ref 检查跳过，spread 年化检查仍独立触发）。
func TestConfirmSimOrderCarrySpreadReject(t *testing.T) {
	st := newFakeStore()
	st.addOrder(carryOrder(0))
	st.addFact(fact.KindDefiRate, "sim_local", "USDT", 6.5, t0.Add(-time.Minute)) // +30%
	s := service(st, sim.Config{})

	resp, err := s.ConfirmSimOrder(context.Background(), confirmReq(1))
	if err != nil {
		t.Fatalf("ConfirmSimOrder(carry spread reject): %v", err)
	}
	if resp.Msg.Accepted || resp.Msg.Order.Status != store.SimStatusRejected {
		t.Fatalf("accepted = %v, status = %q, want false/rejected", resp.Msg.Accepted, resp.Msg.Order.Status)
	}
	if len(st.rejected) != 1 || !slices.Contains(st.rejected[0].flags, sim.RiskSpreadDrift) {
		t.Fatalf("rejected = %+v, want SPREAD_DRIFT 拒单", st.rejected)
	}
}

// TestConfirmSimOrderCarryFailClosed：确认时点查不到 defi_rate → fail-closed 拒
// （生息年化是 carry 权威源，宁缺毋滥）。
func TestConfirmSimOrderCarryFailClosed(t *testing.T) {
	st := newFakeStore()
	st.addOrder(carryOrder(0))
	s := service(st, sim.Config{})

	resp, err := s.ConfirmSimOrder(context.Background(), confirmReq(1))
	if err != nil {
		t.Fatalf("ConfirmSimOrder(carry fail-closed): %v", err)
	}
	if resp.Msg.Accepted || resp.Msg.Order.Status != store.SimStatusRejected {
		t.Fatalf("accepted = %v, status = %q, want false/rejected（fail-closed）", resp.Msg.Accepted, resp.Msg.Order.Status)
	}
	if len(st.rejected) != 1 || !strings.Contains(st.rejected[0].reason, "fail-closed") {
		t.Fatalf("rejected = %+v, want fail-closed 拒单", st.rejected)
	}
}

// TestListSimPositionsRMBConversion：pnl_rmb = pnl × 即期 USDCNH；汇率缺失 → 0。
func TestListSimPositionsRMBConversion(t *testing.T) {
	// 有汇率：pnl=100 × 7.25 → 725。
	st := newFakeStore()
	st.positions = []store.SimPosition{
		{ID: 1, OrderID: 1, Ts: t0, Kind: store.SimKindFundingHedge, Venue: "binance",
			Symbol: "BTC", Side: store.SimSideLong, Qty: 10000, RefPrice: 100, Funding: true, PnL: 100},
	}
	st.addFact(fact.KindFX, fxVenue, fxSymbol, 7.25, t0.Add(-time.Minute))
	s := service(st, sim.Config{})

	resp, err := s.ListSimPositions(context.Background(), connect.NewRequest(&simv1.ListSimPositionsRequest{}))
	if err != nil || len(resp.Msg.Positions) != 1 {
		t.Fatalf("ListSimPositions = %d, %v, want 1", len(resp.Msg.Positions), err)
	}
	if got := resp.Msg.Positions[0].PnlRmb; got != 725 {
		t.Fatalf("pnl_rmb = %v, want 725（100×7.25）", got)
	}

	// 无汇率 → pnl_rmb=0（前端标「USD 原值」，H1/R6#1 刻度线：绝对金额用即期）。
	st2 := newFakeStore()
	st2.positions = append([]store.SimPosition(nil), st.positions...)
	s2 := service(st2, sim.Config{})
	resp2, err := s2.ListSimPositions(context.Background(), connect.NewRequest(&simv1.ListSimPositionsRequest{}))
	if err != nil || len(resp2.Msg.Positions) != 1 {
		t.Fatalf("ListSimPositions(no fx) = %d, %v, want 1", len(resp2.Msg.Positions), err)
	}
	if got := resp2.Msg.Positions[0].PnlRmb; got != 0 {
		t.Fatalf("pnl_rmb = %v, want 0（无汇率）", got)
	}
}

// TestListSimPositionsRealtime：实时数值字段（对话 #57 需求 3）。
//   - cur_price = ticker 最新；expected_ann = 生息腿当前 funding 年化（现货腿 = 0）；
//   - unrealized_pnl = (cur-ref) × qty × 方向（short=-1）；
//   - ticker 缺失 → cur_price=0 + unrealized=0（不编造浮动）。
// [对抗测试锚点] 删除 ListSimPositions 里 unrealized 计算 / cur_price 查询 → 本测试必红。
func TestListSimPositionsRealtime(t *testing.T) {
	// 永续空腿：ref=100, qty=10000；ticker=105 → 未实现 = (105-100)×10000×(-1) = -50000；
	// funding=6.6 → expected_ann=6.6。现货多腿：expected_ann=0（非生息腿），unrealized 按 long 方向。
	st := newFakeStore()
	st.positions = []store.SimPosition{
		{ID: 1, OrderID: 1, Ts: t0, Kind: store.SimKindFundingHedge, Venue: "binance",
			Symbol: "BTC", Side: store.SimSideShort, Qty: 10000, RefPrice: 100, Funding: true, PnL: 50},
		{ID: 2, OrderID: 1, Ts: t0, Kind: store.SimKindFundingHedge, Venue: "binance",
			Symbol: "BTC", Side: store.SimSideLong, Qty: 10000, RefPrice: 100, Funding: false, PnL: 0},
	}
	st.addFact(fact.KindTicker, "binance", "BTC", 105, t0.Add(-time.Minute))
	st.addFact(fact.KindFunding, "binance", "BTC", 6.6, t0.Add(-time.Minute))
	s := service(st, sim.Config{})

	resp, err := s.ListSimPositions(context.Background(), connect.NewRequest(&simv1.ListSimPositionsRequest{}))
	if err != nil || len(resp.Msg.Positions) != 2 {
		t.Fatalf("ListSimPositions = %d, %v, want 2", len(resp.Msg.Positions), err)
	}
	short, long := resp.Msg.Positions[0], resp.Msg.Positions[1] // 顺序 = fakeStore 原样
	if short.CurPrice != 105 {
		t.Fatalf("short cur_price = %v, want 105", short.CurPrice)
	}
	if short.ExpectedAnn != 6.6 {
		t.Fatalf("short expected_ann = %v, want 6.6（funding 生息腿）", short.ExpectedAnn)
	}
	if short.UnrealizedPnl != -50000 {
		t.Fatalf("short unrealized_pnl = %v, want -50000（(105-100)×10000×-1）", short.UnrealizedPnl)
	}
	if long.CurPrice != 105 {
		t.Fatalf("long cur_price = %v, want 105", long.CurPrice)
	}
	if long.ExpectedAnn != 0 {
		t.Fatalf("long expected_ann = %v, want 0（现货腿非生息）", long.ExpectedAnn)
	}
	if long.UnrealizedPnl != 50000 {
		t.Fatalf("long unrealized_pnl = %v, want +50000（(105-100)×10000×+1）", long.UnrealizedPnl)
	}

	// ticker 缺失 → cur_price=0 + unrealized=0（不编造浮动，宁缺毋滥）。
	st2 := newFakeStore()
	st2.positions = append([]store.SimPosition(nil), st.positions...)
	st2.addFact(fact.KindFunding, "binance", "BTC", 6.6, t0.Add(-time.Minute))
	s2 := service(st2, sim.Config{})
	resp2, err := s2.ListSimPositions(context.Background(), connect.NewRequest(&simv1.ListSimPositionsRequest{}))
	if err != nil || len(resp2.Msg.Positions) != 2 {
		t.Fatalf("ListSimPositions(no ticker) = %d, %v, want 2", len(resp2.Msg.Positions), err)
	}
	if got := resp2.Msg.Positions[0].UnrealizedPnl; got != 0 {
		t.Fatalf("unrealized_pnl(no ticker) = %v, want 0（行情缺失不编造浮动）", got)
	}
}

// TestGetSimReport：未启用（路径空 / HistoryDays=0）/ 文件不存在 / 存在 三态。
func TestGetSimReport(t *testing.T) {
	t.Run("disabled-empty-path", func(t *testing.T) {
		s := service(newFakeStore(), sim.Config{})
		resp, err := s.GetSimReport(context.Background(), connect.NewRequest(&simv1.GetSimReportRequest{}))
		if err != nil || resp.Msg.Exists {
			t.Fatalf("exists = %v, %v, want false", resp.Msg.Exists, err)
		}
		if resp.Msg.Note == "" {
			t.Fatal("note 为空，want 未启用说明")
		}
	})
	t.Run("disabled-zero-history", func(t *testing.T) {
		s := service(newFakeStore(), sim.Config{ReportPath: "x.md", HistoryDays: 0})
		resp, err := s.GetSimReport(context.Background(), connect.NewRequest(&simv1.GetSimReportRequest{}))
		if err != nil || resp.Msg.Exists {
			t.Fatalf("exists = %v, %v, want false", resp.Msg.Exists, err)
		}
	})
	t.Run("missing-file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "nope.md")
		s := service(newFakeStore(), sim.Config{ReportPath: path, HistoryDays: 7})
		resp, err := s.GetSimReport(context.Background(), connect.NewRequest(&simv1.GetSimReportRequest{}))
		if err != nil || resp.Msg.Exists {
			t.Fatalf("exists = %v, %v, want false", resp.Msg.Exists, err)
		}
		if resp.Msg.Note == "" || resp.Msg.Markdown != "" {
			t.Fatalf("note = %q, markdown = %q, want 未生成说明", resp.Msg.Note, resp.Msg.Markdown)
		}
	})
	t.Run("exists", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "sim_report.md")
		if err := os.WriteFile(path, []byte("# 模拟盘周报\n"), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
		s := service(newFakeStore(), sim.Config{ReportPath: path, HistoryDays: 7})
		resp, err := s.GetSimReport(context.Background(), connect.NewRequest(&simv1.GetSimReportRequest{}))
		if err != nil || !resp.Msg.Exists {
			t.Fatalf("exists = %v, %v, want true", resp.Msg.Exists, err)
		}
		if resp.Msg.Markdown != "# 模拟盘周报\n" {
			t.Fatalf("markdown = %q, want 文件内容", resp.Msg.Markdown)
		}
	})
}

// TestGetTestnetAccounts：D-040 测试网账户区数据面——空库返回 [] 不报错；
// 有快照按 source ASC 返回，details 与 equity_usd 口径原样透传。
func TestGetTestnetAccounts(t *testing.T) {
	st := newFakeStore()
	s := service(st, sim.Config{})

	// 空库 → []（非 nil）。
	resp, err := s.GetTestnetAccounts(context.Background(), connect.NewRequest(&simv1.GetTestnetAccountsRequest{}))
	if err != nil {
		t.Fatalf("GetTestnetAccounts(empty): %v", err)
	}
	if resp.Msg.Accounts == nil || len(resp.Msg.Accounts) != 0 {
		t.Fatalf("accounts = %#v, want 空 []", resp.Msg.Accounts)
	}

	// 两路快照（okx 先插 → 返回仍按 source ASC = binance 在前）。
	_ = st.UpsertTestnetAccount(context.Background(), store.TestnetAccount{
		Source: "sim_testnet_okx", AccountAlias: "", EquityUSD: 65000,
		Details: []store.TestnetAccountDetail{
			{Asset: "USDT", Balance: "5000", EquityUSD: 5000},
			{Asset: "BTC", Balance: "1", EquityUSD: 60000},
		},
		UpdatedAt: t0,
	})
	_ = st.UpsertTestnetAccount(context.Background(), store.TestnetAccount{
		Source: "sim_testnet_binance", AccountAlias: "s_testnet", EquityUSD: 10000,
		Details: []store.TestnetAccountDetail{
			{Asset: "BTC", Balance: "0.01000000", EquityUSD: 0},
			{Asset: "USDT", Balance: "5000.00000000", EquityUSD: 5000},
		},
		UpdatedAt: t0,
	})

	resp, err = s.GetTestnetAccounts(context.Background(), connect.NewRequest(&simv1.GetTestnetAccountsRequest{}))
	if err != nil {
		t.Fatalf("GetTestnetAccounts: %v", err)
	}
	if len(resp.Msg.Accounts) != 2 {
		t.Fatalf("accounts = %d, want 2", len(resp.Msg.Accounts))
	}
	bin, okx := resp.Msg.Accounts[0], resp.Msg.Accounts[1]
	if bin.Source != "sim_testnet_binance" || okx.Source != "sim_testnet_okx" {
		t.Fatalf("source 顺序 = %q, %q, want binance, okx（ASC）", bin.Source, okx.Source)
	}
	if bin.AccountAlias != "s_testnet" || bin.EquityUsd != 10000 {
		t.Errorf("binance = (%q, %v), want (s_testnet, 10000)", bin.AccountAlias, bin.EquityUsd)
	}
	if okx.EquityUsd != 65000 || len(okx.Details) != 2 {
		t.Errorf("okx = (%v, %d details), want (65000, 2)", okx.EquityUsd, len(okx.Details))
	}
	if bin.UpdatedAtMs != t0.UnixMilli() {
		t.Errorf("binance UpdatedAtMs = %d, want %d（updated_at 毫秒透传）", bin.UpdatedAtMs, t0.UnixMilli())
	}
	if okx.Details[0].Asset != "USDT" || okx.Details[0].Balance != "5000" || okx.Details[0].EquityUsd != 5000 {
		t.Errorf("okx detail[0] = %+v, want USDT 5000/5000", okx.Details[0])
	}
}

// gateSnaps 造 N 份均匀间隔快照：首 ts=start、末 ts=start+days*24h，equity 从 from
// 线性到 to，cash=equity（无持仓，D-063 恒等式成立）。n=1 → 单快照。
func gateSnaps(start time.Time, days float64, n int, from, to float64) []store.EquitySnapshot {
	out := make([]store.EquitySnapshot, n)
	span := time.Duration(days * 24 * float64(time.Hour))
	for i := 0; i < n; i++ {
		frac := 0.0
		if n > 1 {
			frac = float64(i) / float64(n-1)
		}
		e := from + (to-from)*frac
		out[i] = store.EquitySnapshot{Ts: start.Add(time.Duration(frac * float64(span))), Equity: e, Cash: e, Realized: to - from}
	}
	return out
}

// gateStore 造判定门① 测试 store：满窗快照 + 首启 capital_in（Ts 落在首快照前，
// TWR 走简单期初期末路径）+ 本金。
func gateStore(start time.Time, days float64, n int, from, to float64, orders ...store.SimOrder) *fakeStore {
	st := &fakeStore{snaps: gateSnaps(start, days, n, from, to)}
	st.capital, st.cash = from, to
	st.flows = []store.CashFlow{{ID: 1, Ts: start, Kind: store.CashKindCapitalIn, Amount: from}}
	st.orders = orders
	return st
}

// TestGetPerformanceReport 判定门① RPC 端到端（D-062 + D-063 可信度层）：
// PENDING（快照不足）/ PASS（满窗 30 天覆盖判定线 + 覆盖率字段）/ DATA_ANOMALY
// （覆盖不足 → 判定不采信；恒等式破坏 → 判定不采信）/ ENV_NO_WINDOW（零成交）。
func TestGetPerformanceReport(t *testing.T) {
	ctx := context.Background()
	start := t0.Add(-30 * 24 * time.Hour) // 30 天窗口起点
	req := connect.NewRequest(&simv1.GetPerformanceReportRequest{})

	t.Run("pending_insufficient_snaps", func(t *testing.T) {
		resp, err := service(gateStore(start, 30, 1, 100, 100), sim.DefaultConfig()).GetPerformanceReport(ctx, req)
		if err != nil {
			t.Fatalf("GetPerformanceReport: %v", err)
		}
		if resp.Msg.Status != sim.GatePending {
			t.Fatalf("status = %q, want pending", resp.Msg.Status)
		}
		if resp.Msg.SnapshotCount != 1 {
			t.Errorf("snapshot_count = %d, want 1", resp.Msg.SnapshotCount)
		}
	})

	t.Run("pass_full_coverage", func(t *testing.T) {
		// 91 快照均匀跨 30 天（8h 间隔）→ 窗口 30 天满、覆盖率 91/90 截 1。
		// 30 天 +4.5% → TWR 年化远超判定线 4.0% → PASS。一单成交。
		st := gateStore(start, 30, 91, 100, 104.5,
			store.SimOrder{ID: 1, Ts: t0.Add(-24 * time.Hour), Status: store.SimStatusFilled})
		resp, err := service(st, sim.DefaultConfig()).GetPerformanceReport(ctx, req)
		if err != nil {
			t.Fatalf("GetPerformanceReport: %v", err)
		}
		m := resp.Msg
		if m.Status != sim.GatePass {
			t.Fatalf("status = %q (%s), want pass", m.Status, m.StatusNote)
		}
		if m.SnapshotCoverage < 0.99 || m.ExpectedSnapshots != 90 {
			t.Errorf("coverage/expected = %v/%d, want ~1/90", m.SnapshotCoverage, m.ExpectedSnapshots)
		}
		// 单位锁定：RPC 返回百分点点数（30 天 +4.5% → 年化 ≈ 70.8，不是小数 0.708）。
		// 防「判定门① 自己骗人」的单位错配回潮（gate 用 4.0 阈值比小数 0.7 永远 FAIL）。
		if math.Abs(m.TwrAnnualized-70.8368) > 0.01 || math.Abs(m.MwrAnnualized-70.8368) > 0.01 {
			t.Errorf("twr/mwr = %v/%v, want ≈70.84（百分点点数）", m.TwrAnnualized, m.MwrAnnualized)
		}
	})

	t.Run("data_anomaly_low_coverage", func(t *testing.T) {
		// 30 天窗口仅 20 快照（20/90 ≈ 22% < 90%）→ 判定不采信。
		resp, err := service(gateStore(start, 30, 20, 100, 104.5,
			store.SimOrder{ID: 1, Ts: t0.Add(-24 * time.Hour), Status: store.SimStatusFilled}), sim.DefaultConfig()).GetPerformanceReport(ctx, req)
		if err != nil {
			t.Fatalf("GetPerformanceReport: %v", err)
		}
		if resp.Msg.Status != sim.GateDataAnomaly {
			t.Fatalf("status = %q, want data_anomaly（覆盖不足判定不采信）", resp.Msg.Status)
		}
		if !strings.Contains(resp.Msg.StatusNote, "覆盖率") {
			t.Errorf("note = %q, want 含覆盖率", resp.Msg.StatusNote)
		}
	})

	t.Run("data_anomaly_integrity", func(t *testing.T) {
		// 恒等式破坏（快照 equity ≠ cash + market_value）→ 判定不采信。
		st := gateStore(start, 30, 91, 100, 104.5,
			store.SimOrder{ID: 1, Ts: t0.Add(-24 * time.Hour), Status: store.SimStatusFilled})
		st.snaps[45].Equity = 999999
		resp, err := service(st, sim.DefaultConfig()).GetPerformanceReport(ctx, req)
		if err != nil {
			t.Fatalf("GetPerformanceReport: %v", err)
		}
		if resp.Msg.Status != sim.GateDataAnomaly {
			t.Fatalf("status = %q, want data_anomaly（恒等式破坏判定不采信）", resp.Msg.Status)
		}
	})

	t.Run("env_no_window_zero_orders", func(t *testing.T) {
		resp, err := service(gateStore(start, 30, 91, 100, 100), sim.DefaultConfig()).GetPerformanceReport(ctx, req)
		if err != nil {
			t.Fatalf("GetPerformanceReport: %v", err)
		}
		if resp.Msg.Status != sim.GateEnvNoWindow {
			t.Fatalf("status = %q (%s), want env_no_window", resp.Msg.Status, resp.Msg.StatusNote)
		}
	})
}
