package dashboard

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sort"
	"testing"
	"time"

	"arbcn/internal/dashboard/gen/arbcn/dashboard/v1/dashboardv1connect"
	"arbcn/internal/fact"
	"arbcn/internal/store"
)

// t0 是测试基准时钟。
var t0 = time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)

// fakeStore：内存版 store.Store（dashboard 读取路径有真语义；
// 写路径本包不经过，误用即红）。
type fakeStore struct {
	facts     []fact.Fact
	alerts    []store.Alert
	states    []store.RuleState
	ledger    []store.LedgerEntry
	orders    []store.SimOrder
	knowledge []store.KnowledgeEntry
	nextID    int64 // 台账自增 id（fake 内存版）
	err       error // 注入存储层故障
}

func (f *fakeStore) LatestFacts(_ context.Context, kind, venue, symbol string) ([]fact.Fact, error) {
	if f.err != nil {
		return nil, f.err
	}
	latest := map[string]fact.Fact{}
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
		k := x.Kind + "\x00" + x.Venue + "\x00" + x.Symbol
		if cur, ok := latest[k]; !ok || x.Ts.After(cur.Ts) {
			latest[k] = x
		}
	}
	out := make([]fact.Fact, 0, len(latest))
	for _, v := range latest {
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		if a.Venue != b.Venue {
			return a.Venue < b.Venue
		}
		return a.Symbol < b.Symbol
	})
	return out, nil
}

func (f *fakeStore) ListAlerts(_ context.Context, limit, offset int) ([]store.Alert, error) {
	if f.err != nil {
		return nil, f.err
	}
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	out := append([]store.Alert(nil), f.alerts...)
	sort.Slice(out, func(i, j int) bool {
		if !out[i].Ts.Equal(out[j].Ts) {
			return out[i].Ts.After(out[j].Ts)
		}
		return out[i].ID > out[j].ID
	})
	if offset >= len(out) {
		return nil, nil
	}
	end := min(offset+limit, len(out))
	return out[offset:end], nil
}

func (f *fakeStore) AckAlert(_ context.Context, id int64) error {
	if f.err != nil {
		return f.err
	}
	for i := range f.alerts {
		if f.alerts[i].ID == id {
			f.alerts[i].Acked = true
		}
	}
	return nil
}

func (f *fakeStore) ListUnacked(_ context.Context) ([]store.Alert, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := []store.Alert{}
	for _, a := range f.alerts {
		if !a.Acked {
			out = append(out, a)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].Ts.Equal(out[j].Ts) {
			return out[i].Ts.After(out[j].Ts)
		}
		return out[i].ID > out[j].ID
	})
	return out, nil
}

func (f *fakeStore) AckAll(_ context.Context) (int64, error) {
	if f.err != nil {
		return 0, f.err
	}
	n := int64(0)
	for i := range f.alerts {
		if !f.alerts[i].Acked {
			f.alerts[i].Acked = true
			n++
		}
	}
	return n, nil
}

func (f *fakeStore) ListTriggerStates(context.Context) ([]store.RuleState, error) {
	if f.err != nil {
		return nil, f.err
	}
	return append([]store.RuleState(nil), f.states...), nil
}

// —— 台账路径（M2-b §6：AddLedgerEntry/ListLedgerEntries/LedgerSummary 真语义）——

func (f *fakeStore) InsertLedgerEntry(_ context.Context, e store.LedgerEntry) (int64, error) {
	if f.err != nil {
		return 0, f.err
	}
	f.nextID++
	e.ID = f.nextID
	f.ledger = append(f.ledger, e)
	return e.ID, nil
}

func (f *fakeStore) ListLedgerEntries(_ context.Context, limit, offset int) ([]store.LedgerEntry, error) {
	if f.err != nil {
		return nil, f.err
	}
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	out := append([]store.LedgerEntry(nil), f.ledger...)
	sort.Slice(out, func(i, j int) bool {
		if !out[i].Date.Equal(out[j].Date) {
			return out[i].Date.After(out[j].Date)
		}
		return out[i].ID > out[j].ID
	})
	if offset >= len(out) {
		return nil, nil
	}
	end := min(offset+limit, len(out))
	return out[offset:end], nil
}

func (f *fakeStore) LedgerSummary(_ context.Context) ([]store.TierSummary, error) {
	if f.err != nil {
		return nil, f.err
	}
	agg := map[string]*store.TierSummary{}
	order := []string{}
	for _, e := range f.ledger {
		if _, ok := agg[e.Tier]; !ok {
			agg[e.Tier] = &store.TierSummary{Tier: e.Tier}
			order = append(order, e.Tier)
		}
		s := agg[e.Tier]
		s.EntryCount++
		s.Net += e.Amount
		if e.Amount > 0 {
			s.Inflow += e.Amount
		} else {
			s.Outflow += -e.Amount
		}
	}
	sort.Strings(order)
	out := make([]store.TierSummary, 0, len(order))
	for _, t := range order {
		out = append(out, *agg[t])
	}
	return out, nil
}

// —— M3-a 模拟盘面（04-m3-spec §3）：dashboard 服务不经过（RPC 延后 M3-c），误用即红 ——

func (f *fakeStore) InsertSimOrder(context.Context, store.SimOrder) (int64, error) {
	panic("fakeStore: InsertSimOrder not used")
}
func (f *fakeStore) ListSimOrders(_ context.Context, limit, offset int) ([]store.SimOrder, error) {
	if f.err != nil {
		return nil, f.err
	}
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	out := append([]store.SimOrder(nil), f.orders...)
	sort.Slice(out, func(i, j int) bool {
		if !out[i].Ts.Equal(out[j].Ts) {
			return out[i].Ts.After(out[j].Ts)
		}
		return out[i].ID > out[j].ID
	})
	if offset >= len(out) {
		return nil, nil
	}
	end := min(offset+limit, len(out))
	return out[offset:end], nil
}
func (f *fakeStore) GetSimOrder(context.Context, int64) (store.SimOrder, error) {
	panic("fakeStore: GetSimOrder not used")
}
func (f *fakeStore) UpdateSimOrderStatus(context.Context, int64, string, string) error {
	panic("fakeStore: UpdateSimOrderStatus not used")
}
func (f *fakeStore) FillSimOrder(context.Context, int64, string, []store.SimPosition) error {
	panic("fakeStore: FillSimOrder not used")
}
func (f *fakeStore) AcceptSimOrder(context.Context, int64, string, []store.SimPosition) error {
	panic("fakeStore: AcceptSimOrder not used")
}
func (f *fakeStore) RejectSimOrder(context.Context, int64, string, ...string) error {
	panic("fakeStore: RejectSimOrder not used")
}
func (f *fakeStore) TodaySimNotional(context.Context, time.Time) (float64, error) {
	panic("fakeStore: TodaySimNotional not used")
}
func (f *fakeStore) InsertSimPosition(context.Context, store.SimPosition) (int64, error) {
	panic("fakeStore: InsertSimPosition not used")
}
func (f *fakeStore) ListSimPositions(context.Context, int, int) ([]store.SimPosition, error) {
	panic("fakeStore: ListSimPositions not used")
}
func (f *fakeStore) ListOpenSimPositions(context.Context, string, string) ([]store.SimPosition, error) {
	panic("fakeStore: ListOpenSimPositions not used")
}
func (f *fakeStore) UpsertTestnetAccount(context.Context, store.TestnetAccount) error {
	panic("fakeStore: UpsertTestnetAccount not used")
}
func (f *fakeStore) ListTestnetAccounts(context.Context) ([]store.TestnetAccount, error) {
	panic("fakeStore: ListTestnetAccounts not used")
}
func (f *fakeStore) SettleSimPosition(context.Context, int64, float64, string) error {
	panic("fakeStore: SettleSimPosition not used")
}

// —— D-046 经验库（知识匹配/浏览 RPC 有真语义）——

func (f *fakeStore) ListKnowledgeEntries(context.Context) ([]store.KnowledgeEntry, error) {
	if f.err != nil {
		return nil, f.err
	}
	out := append([]store.KnowledgeEntry(nil), f.knowledge...)
	sort.Slice(out, func(i, j int) bool { return out[i].Signature < out[j].Signature })
	return out, nil
}

func (f *fakeStore) UpsertKnowledgeEntry(context.Context, store.KnowledgeEntry) (int64, error) {
	panic("fakeStore: UpsertKnowledgeEntry not used")
}

// ReviewKnowledgeEntry 复核（D-054 服务测试真语义）：写 validated_at=now + status +
// 可选 verdict + note（verdict 空 = 保留原判定）。
func (f *fakeStore) ReviewKnowledgeEntry(_ context.Context, signature, status, verdict, note string) error {
	if f.err != nil {
		return f.err
	}
	for i := range f.knowledge {
		if f.knowledge[i].Signature == signature {
			now := time.Now().UTC()
			f.knowledge[i].ValidatedAt = &now
			f.knowledge[i].Status = status
			if verdict != "" {
				f.knowledge[i].Verdict = verdict
			}
			f.knowledge[i].ValidationNote = note
			return nil
		}
	}
	return store.ErrNotFound
}

// CloseSimOrder 平仓：dashboard 服务不经过（只读 + ack + knowledge 复核），误用即红。
func (f *fakeStore) CloseSimOrder(context.Context, int64, string, []store.SimLegClose) (int, error) {
	panic("fakeStore: CloseSimOrder not used")
}

// —— 其余写路径：dashboard 服务不经过（只读 + ack），误用即红 ——

func (f *fakeStore) InsertFacts(context.Context, []fact.Fact) error {
	panic("fakeStore: InsertFacts not used")
}
func (f *fakeStore) QueryFacts(_ context.Context, q store.FactQuery) ([]fact.Fact, error) {
	if f.err != nil {
		return nil, f.err
	}
	to := q.To
	if to.IsZero() {
		to = time.Now()
	}
	out := []fact.Fact{}
	for _, x := range f.facts {
		if q.Kind != "" && x.Kind != q.Kind {
			continue
		}
		if q.Venue != "" && x.Venue != q.Venue {
			continue
		}
		if q.Symbol != "" && x.Symbol != q.Symbol {
			continue
		}
		if x.Ts.Before(q.From) || !x.Ts.Before(to) {
			continue
		}
		out = append(out, x)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Ts.Before(out[j].Ts) })
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

// fakePinger 满足 httpapi.Pinger。
type fakePinger struct{ err error }

func (p fakePinger) Ping(context.Context) error { return p.err }

// newTestServer 起 httptest 服务并返回 Connect 客户端（真传输，非 mock handler）。
func newTestServer(t *testing.T, svc *Service) dashboardv1connect.DashboardServiceClient {
	t.Helper()
	path, handler := svc.Handler()
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return dashboardv1connect.NewDashboardServiceClient(srv.Client(), srv.URL)
}
