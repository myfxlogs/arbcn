package sim

import (
	"context"
	"errors"
	"math"
	"sort"
	"sync"
	"testing"
	"time"

	"arbcn/internal/fact"
	"arbcn/internal/store"
)

// storeStub 内嵌 nil store.Store 接口：只实现 sim 数据面（真语义），其余方法
// 运行时 nil 解引用 panic（测试误用即红）。免去 20+ 空方法样板。
// mu 保护 orders/positions/facts：settleLoop 并发测试（TestSettleLoopTicks）下
// 结算 goroutine 写、断言 goroutine 读，无锁即数据竞争（-race 必红）。
type storeStub struct {
	store.Store
	mu             sync.Mutex
	orders         []store.SimOrder
	positions      []store.SimPosition
	facts          []fact.Fact // driver 测试的行情/费率面
	nextOID        int64
	nextPID        int64
	dayNotional    float64
	orderErr       error
	insertFactsErr error // 注入 InsertFacts 失败（回填错误路径测试）
}

// snapshotPositions 返回 positions 的拷贝（并发断言读；避免与 settle goroutine 竞争）。
func (m *storeStub) snapshotPositions() []store.SimPosition {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]store.SimPosition(nil), m.positions...)
}

// —— driver 测试扩展：facts 面（LatestFacts / QueryFacts）——

// LatestFacts 返回每 (kind, venue, symbol) 最新一条事实（按 Ts 最大）。
func (m *storeStub) LatestFacts(_ context.Context, kind, venue, symbol string) ([]fact.Fact, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var best *fact.Fact
	for i := range m.facts {
		f := &m.facts[i]
		if kind != "" && f.Kind != kind {
			continue
		}
		if venue != "" && f.Venue != venue {
			continue
		}
		if symbol != "" && f.Symbol != symbol {
			continue
		}
		if best == nil || f.Ts.After(best.Ts) {
			best = f
		}
	}
	if best == nil {
		return nil, nil
	}
	return []fact.Fact{*best}, nil
}

// QueryFacts 按 FactQuery 过滤（ts 升序）。
func (m *storeStub) QueryFacts(_ context.Context, q store.FactQuery) ([]fact.Fact, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []fact.Fact
	for _, f := range m.facts {
		if q.Kind != "" && f.Kind != q.Kind {
			continue
		}
		if q.Venue != "" && f.Venue != q.Venue {
			continue
		}
		if q.Symbol != "" && f.Symbol != q.Symbol {
			continue
		}
		if !q.From.IsZero() && f.Ts.Before(q.From) {
			continue
		}
		if q.To.IsZero() == false && f.Ts.After(q.To) {
			continue
		}
		out = append(out, f)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Ts.Before(out[j].Ts) })
	return out, nil
}

// InsertFacts 批量追加到 m.facts（回填编排数据面；insertFactsErr 注入失败路径）。
func (m *storeStub) InsertFacts(_ context.Context, fs []fact.Fact) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.insertFactsErr != nil {
		return m.insertFactsErr
	}
	m.facts = append(m.facts, fs...)
	return nil
}

func (m *storeStub) InsertSimOrder(_ context.Context, o store.SimOrder) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.orderErr != nil {
		return 0, m.orderErr
	}
	m.nextOID++
	o.ID = m.nextOID
	m.orders = append(m.orders, o)
	return o.ID, nil
}
func (m *storeStub) GetSimOrder(_ context.Context, id int64) (store.SimOrder, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, o := range m.orders {
		if o.ID == id {
			return o, nil
		}
	}
	return store.SimOrder{}, store.ErrNotFound
}
func (m *storeStub) UpdateSimOrderStatus(_ context.Context, id int64, status, note string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.orders {
		if m.orders[i].ID == id {
			m.orders[i].Status = status
			if note != "" {
				m.orders[i].Note = note
			}
			return nil
		}
	}
	return nil
}
func (m *storeStub) FillSimOrder(_ context.Context, id int64, note string, legs []store.SimPosition) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	// 仿 pgstore 语义：仅 confirmed → filled + 建腿；否则拒绝（状态守卫 = 原子性测试锚点）。
	for i := range m.orders {
		if m.orders[i].ID == id {
			if m.orders[i].Status != store.SimStatusConfirmed {
				return errors.New("sim: order not confirmed")
			}
			m.orders[i].Status = store.SimStatusFilled
			if note != "" {
				m.orders[i].Note = note
			}
			for _, p := range legs {
				if p.OrderID <= 0 {
					p.OrderID = id
				}
				m.nextPID++
				p.ID = m.nextPID
				m.positions = append(m.positions, p)
			}
			return nil
		}
	}
	return store.ErrNotFound
}
func (m *storeStub) TodaySimNotional(context.Context, time.Time) (float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.dayNotional, nil
}
func (m *storeStub) InsertSimPosition(_ context.Context, p store.SimPosition) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextPID++
	p.ID = m.nextPID
	m.positions = append(m.positions, p)
	return p.ID, nil
}
func (m *storeStub) ListOpenSimPositions(_ context.Context, symbol, venue string) ([]store.SimPosition, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []store.SimPosition
	for _, p := range m.positions {
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
func (m *storeStub) SettleSimPosition(_ context.Context, id int64, addPnl float64, status string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.positions {
		if m.positions[i].ID == id {
			m.positions[i].PnL += addPnl
			m.positions[i].Status = status
			m.positions[i].UpdatedAt = t0
			return nil
		}
	}
	return nil
}

func newSim(t *testing.T) (*Simulator, *storeStub) {
	t.Helper()
	st := &storeStub{}
	return New(st, DefaultConfig()), st
}

// TestGeneratePersists：Generate 落库一条全门禁通过的对冲建议订单。
func TestGeneratePersists(t *testing.T) {
	s, st := newSim(t)
	o, err := s.Generate(context.Background(), validSignal())
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if o.ID <= 0 || o.Status != store.SimStatusSuggested || len(o.RiskFlags) != 0 {
		t.Fatalf("order = %+v, want id>0/suggested/无 flags", o)
	}
	if len(st.orders) != 1 || st.orders[0].ID != o.ID {
		t.Fatalf("orders = %+v, want 1 条 id=%d", st.orders, o.ID)
	}
}

// TestGenerateRejectedPersists：拒单也是负样本（§4"拒单不是失败"）——rejected 落库。
func TestGenerateRejectedPersists(t *testing.T) {
	s, st := newSim(t)
	sig := validSignal()
	sig.SpotPrice = 0 // 缺腿 → UNHEDGED
	o, err := s.Generate(context.Background(), sig)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if o.Status != store.SimStatusRejected || len(o.RiskFlags) == 0 || o.Note == "" {
		t.Fatalf("order = %+v, want rejected/有 flags/有 note", o)
	}
	if len(st.orders) != 1 || st.orders[0].Status != store.SimStatusRejected {
		t.Fatalf("orders = %+v, want 1 条 rejected 负样本", st.orders)
	}
}

// TestGenerateDailyOverAutoQuery：Generate 自动查询当日累计 → 超 50% → DAILY_OVER 拒单。
func TestGenerateDailyOverAutoQuery(t *testing.T) {
	s, st := newSim(t)
	st.dayNotional = 45_000 // +10_000 = 55_000 > 50_000
	o, err := s.Generate(context.Background(), validSignal())
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if o.Status != store.SimStatusRejected || !hasFlag(o, RiskDailyOver) {
		t.Fatalf("order = %+v, want rejected/DAILY_OVER", o)
	}
}

// TestConfirmAndFillHedge：confirmed → filled，建 2 腿（现货 long 非 funding + 永续 short
// funding），永续腿标定资金费率结算。
func TestConfirmAndFillHedge(t *testing.T) {
	s, st := newSim(t)
	o, err := s.Generate(context.Background(), validSignal())
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if err := st.UpdateSimOrderStatus(context.Background(), o.ID, store.SimStatusConfirmed, ""); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if err := s.ConfirmAndFill(context.Background(), o.ID); err != nil {
		t.Fatalf("ConfirmAndFill: %v", err)
	}
	filled, _ := st.GetSimOrder(context.Background(), o.ID)
	if filled.Status != store.SimStatusFilled {
		t.Fatalf("status = %q, want filled", filled.Status)
	}
	if len(st.positions) != 2 {
		t.Fatalf("positions = %d, want 2（hedge 两腿）", len(st.positions))
	}
	var spot, perp *store.SimPosition
	for i := range st.positions {
		p := &st.positions[i]
		if p.Side == store.SimSideLong {
			spot = p
		} else if p.Side == store.SimSideShort {
			perp = p
		}
	}
	if spot == nil || perp == nil {
		t.Fatalf("positions = %+v, want long+short 两腿", st.positions)
	}
	if spot.Funding || !perp.Funding {
		t.Fatalf("funding 标定错：spot.Funding=%v（want false），perp.Funding=%v（want true）",
			spot.Funding, perp.Funding)
	}
}

// TestConfirmAndFillCarry：carry 单腿（funding 生息），非 confirmed 订单拒绝成交。
func TestConfirmAndFillCarry(t *testing.T) {
	s, st := newSim(t)
	sig := validSignal()
	sig.Kind = store.SimKindCarryAsset
	sig.CarryWhite = true
	o, err := s.Generate(context.Background(), sig)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// 非 confirmed（suggested）→ 拒绝成交，防状态漂移。
	if err := s.ConfirmAndFill(context.Background(), o.ID); err == nil {
		t.Fatal("ConfirmAndFill(suggested) = nil, want error")
	}
	if len(st.positions) != 0 {
		t.Fatalf("positions = %d, want 0（未确认不得成交）", len(st.positions))
	}

	if err := st.UpdateSimOrderStatus(context.Background(), o.ID, store.SimStatusConfirmed, ""); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if err := s.ConfirmAndFill(context.Background(), o.ID); err != nil {
		t.Fatalf("ConfirmAndFill: %v", err)
	}
	if len(st.positions) != 1 || !st.positions[0].Funding || st.positions[0].Side != store.SimSideLong {
		t.Fatalf("positions = %+v, want 1 funding long 腿", st.positions)
	}
}

// TestSettleFunding：对冲持仓按 funding 周期结算——只结算 funding 腿（永续 short），
// 现货 long 腿不动。年化 10.95% → 8h 分数费率 0.0001（H1：10.95/100/1095）→ pnl += 0.0001×qty。
func TestSettleFunding(t *testing.T) {
	s, st := newSim(t)
	o, err := s.Generate(context.Background(), validSignal())
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if err := st.UpdateSimOrderStatus(context.Background(), o.ID, store.SimStatusConfirmed, ""); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if err := s.ConfirmAndFill(context.Background(), o.ID); err != nil {
		t.Fatalf("ConfirmAndFill: %v", err)
	}

	n, err := s.SettleFunding(context.Background(), "BTC", "", 10.95)
	if err != nil {
		t.Fatalf("SettleFunding: %v", err)
	}
	if n != 1 {
		t.Fatalf("settled legs = %d, want 1（仅永续 funding 腿）", n)
	}
	for i := range st.positions {
		p := &st.positions[i]
		if p.Side == store.SimSideShort {
			if math.Abs(p.PnL-1) > 1e-6 { // 0.0001 × 10000 ≈ 1
				t.Fatalf("perp pnl = %v, want 1（rate×notional，H1：点数÷100 先转分数）", p.PnL)
			}
		} else if p.PnL != 0 {
			t.Fatalf("spot pnl = %v, want 0（现货腿不结算）", p.PnL)
		}
	}
}

// TestSettleFundingSkipsNonFunding：非 funding 腿（spot）不结算，即使同 symbol。
func TestSettleFundingSkipsNonFunding(t *testing.T) {
	s, st := newSim(t)
	// 手工造两条 open 腿：一条 funding、一条非 funding。
	if _, err := st.InsertSimPosition(context.Background(), store.SimPosition{
		OrderID: 1, Ts: t0, Kind: store.SimKindCarryAsset, Venue: "sim_local", Symbol: "USDT",
		Side: store.SimSideLong, Qty: 10_000, RefPrice: 1, Funding: true, Status: store.SimPosStatusOpen,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := st.InsertSimPosition(context.Background(), store.SimPosition{
		OrderID: 1, Ts: t0, Kind: store.SimKindFundingHedge, Venue: "sim_local", Symbol: "BTC",
		Side: store.SimSideLong, Qty: 10_000, RefPrice: 60000, Funding: false, Status: store.SimPosStatusOpen,
	}); err != nil {
		t.Fatal(err)
	}
	n, err := s.SettleFunding(context.Background(), "", "", 10.95)
	if err != nil {
		t.Fatalf("SettleFunding: %v", err)
	}
	if n != 1 {
		t.Fatalf("settled = %d, want 1（只结算 funding 腿）", n)
	}
	for i := range st.positions {
		if st.positions[i].Funding && math.Abs(st.positions[i].PnL-1) > 1e-6 {
			t.Fatalf("funding leg pnl = %v, want 1", st.positions[i].PnL)
		}
		if !st.positions[i].Funding && st.positions[i].PnL != 0 {
			t.Fatalf("non-funding leg pnl = %v, want 0", st.positions[i].PnL)
		}
	}
}
