package rule

import (
	"context"
	"sort"
	"sync"
	"testing"
	"time"

	"arbcn/internal/fact"
	"arbcn/internal/store"
)

// t0 是测试基准时钟（所有合成 fact 的锚点）。
var t0 = time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)

// fct 组装合成 fact（Ts = t0 + off）。
func fct(kind, venue, symbol string, v float64, off time.Duration) fact.Fact {
	return fact.Fact{Kind: kind, Venue: venue, Symbol: symbol, Value: v, Ts: t0.Add(off), Src: "test"}
}

// fakeStore 是内存版 store.Store（线程安全：Engine.Run 并发使用）。
type fakeStore struct {
	mu      sync.Mutex
	rules   []store.Rule
	facts   []fact.Fact
	state   map[int64]store.TriggerState
	alerts  []store.Alert
	queries int
}

func newFakeStore(rules []store.Rule, facts []fact.Fact) *fakeStore {
	for i := range rules {
		rules[i].ID = int64(i + 1)
	}
	return &fakeStore{rules: rules, facts: facts, state: map[int64]store.TriggerState{}}
}

func (f *fakeStore) InsertFacts(_ context.Context, fs []fact.Fact) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.facts = append(f.facts, fs...)
	return nil
}

// QueryFacts 按 FactQuery 过滤（ts 升序，与 pgstore 一致）。
func (f *fakeStore) QueryFacts(_ context.Context, q store.FactQuery) ([]fact.Fact, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.queries++
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
		if !q.From.IsZero() && x.Ts.Before(q.From) {
			continue
		}
		if !q.To.IsZero() && !x.Ts.Before(q.To) {
			continue
		}
		out = append(out, x)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Ts.Before(out[j].Ts) })
	return out, nil
}

func (f *fakeStore) UpsertRule(_ context.Context, r store.Rule) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := range f.rules {
		if f.rules[i].Name == r.Name {
			r.ID = f.rules[i].ID
			f.rules[i] = r
			return r.ID, nil
		}
	}
	r.ID = int64(len(f.rules) + 1)
	f.rules = append(f.rules, r)
	return r.ID, nil
}

func (f *fakeStore) ListRules(context.Context) ([]store.Rule, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]store.Rule(nil), f.rules...), nil
}

func (f *fakeStore) GetTriggerState(_ context.Context, ruleID int64) (store.TriggerState, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.state[ruleID]
	if !ok {
		return store.TriggerState{}, store.ErrNotFound
	}
	return s, nil
}

func (f *fakeStore) PutTriggerState(_ context.Context, s store.TriggerState) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.state[s.RuleID] = s
	return nil
}

func (f *fakeStore) InsertAlert(_ context.Context, a store.Alert) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.alerts = append(f.alerts, a)
	return nil
}

// PendingAlerts / MarkAlertDelivered（M1-f Store 扩展；rule 测试不涉及投递，
// 实现对齐接口语义）。
func (f *fakeStore) PendingAlerts(_ context.Context, limit int) ([]store.Alert, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := []store.Alert{}
	for _, a := range f.alerts {
		if a.Delivered {
			continue
		}
		out = append(out, a)
		if limit > 0 && len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (f *fakeStore) MarkAlertDelivered(_ context.Context, id int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for i := range f.alerts {
		if f.alerts[i].ID == id {
			f.alerts[i].Delivered = true
		}
	}
	return nil
}

// LatestFacts / ListAlerts / AckAlert / ListTriggerStates（M1-g Store 扩展；
// rule 测试不经过仪表盘读取路径：误用即红，不静默）。
func (f *fakeStore) LatestFacts(context.Context, string, string, string) ([]fact.Fact, error) {
	panic("fakeStore: LatestFacts not used")
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

// —— M2-b 台账面（M2-b §6）：rule 测试不经过台账，误用即红 ——
func (f *fakeStore) InsertLedgerEntry(context.Context, store.LedgerEntry) (int64, error) {
	panic("fakeStore: InsertLedgerEntry not used")
}
func (f *fakeStore) ListLedgerEntries(context.Context, int, int) ([]store.LedgerEntry, error) {
	panic("fakeStore: ListLedgerEntries not used")
}
func (f *fakeStore) LedgerSummary(context.Context) ([]store.TierSummary, error) {
	panic("fakeStore: LedgerSummary not used")
}

// —— M3-a 模拟盘面（04-m3-spec §3）：rule 测试不经过，误用即红 ——
func (f *fakeStore) InsertSimOrder(context.Context, store.SimOrder) (int64, error) {
	panic("fakeStore: InsertSimOrder not used")
}
func (f *fakeStore) ListSimOrders(context.Context, int, int) ([]store.SimOrder, error) {
	panic("fakeStore: ListSimOrders not used")
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

func (f *fakeStore) alertsCopy() []store.Alert {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]store.Alert(nil), f.alerts...)
}

func (f *fakeStore) queryCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.queries
}

// waitFor 轮询等待条件成立（collect 包同款）。
func waitFor(t *testing.T, d time.Duration, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(d)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("condition not met in time")
}
