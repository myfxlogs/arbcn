package alert

import (
	"context"
	"sync"
	"testing"
	"time"

	"arbcn/internal/fact"
	"arbcn/internal/store"
)

// memStore：内存版 store.Store（alerter/heartbeat 测试用）。alerts 投递状态机
// 与 pgstore 同语义（ts 升序、delivered 过滤）；测试不用的方法 panic（误用即红）。
type memStore struct {
	mu        sync.Mutex
	facts     []fact.Fact
	alerts    []store.Alert
	nextID    int64
	pendErr   error
	pendCalls int
}

func newMemStore() *memStore { return &memStore{} }

func (m *memStore) setPendErr(err error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pendErr = err
}

func (m *memStore) InsertFacts(_ context.Context, fs []fact.Fact) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.facts = append(m.facts, fs...)
	return nil
}

func (m *memStore) InsertAlert(_ context.Context, a store.Alert) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nextID++
	a.ID = m.nextID
	if a.Ts.IsZero() {
		a.Ts = time.Now()
	}
	m.alerts = append(m.alerts, a)
	return nil
}

func (m *memStore) PendingAlerts(_ context.Context, limit int) ([]store.Alert, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.pendCalls++
	if m.pendErr != nil {
		return nil, m.pendErr
	}
	if limit <= 0 {
		limit = 100
	}
	out := []store.Alert{}
	for _, a := range m.alerts {
		if a.Delivered {
			continue
		}
		out = append(out, a)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (m *memStore) MarkAlertDelivered(_ context.Context, id int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.alerts {
		if m.alerts[i].ID == id {
			m.alerts[i].Delivered = true
		}
	}
	return nil
}

func (m *memStore) factsCopy() []fact.Fact {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]fact.Fact(nil), m.facts...)
}

func (m *memStore) deliveredCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	n := 0
	for _, a := range m.alerts {
		if a.Delivered {
			n++
		}
	}
	return n
}

func (m *memStore) pendCallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.pendCalls
}

// 以下方法本包测试不经过：误用即 panic（失败要响，不静默）。
func (m *memStore) QueryFacts(context.Context, store.FactQuery) ([]fact.Fact, error) {
	panic("memStore: QueryFacts not used")
}

func (m *memStore) UpsertRule(context.Context, store.Rule) (int64, error) {
	panic("memStore: UpsertRule not used")
}

func (m *memStore) ListRules(context.Context) ([]store.Rule, error) {
	panic("memStore: ListRules not used")
}

func (m *memStore) GetTriggerState(context.Context, int64) (store.TriggerState, error) {
	panic("memStore: GetTriggerState not used")
}

func (m *memStore) PutTriggerState(context.Context, store.TriggerState) error {
	panic("memStore: PutTriggerState not used")
}

func (m *memStore) LatestFacts(context.Context, string, string, string) ([]fact.Fact, error) {
	panic("memStore: LatestFacts not used")
}

func (m *memStore) ListAlerts(context.Context, int, int) ([]store.Alert, error) {
	panic("memStore: ListAlerts not used")
}

func (m *memStore) AckAlert(context.Context, int64) error {
	panic("memStore: AckAlert not used")
}

func (m *memStore) ListTriggerStates(context.Context) ([]store.RuleState, error) {
	panic("memStore: ListTriggerStates not used")
}

func (m *memStore) ListUnacked(context.Context) ([]store.Alert, error) {
	panic("memStore: ListUnacked not used")
}

func (m *memStore) AckAll(context.Context) (int64, error) {
	panic("memStore: AckAll not used")
}

// —— M2-b 台账面（M2-b §6）：alert 测试不经过台账，误用即红 ——
func (m *memStore) InsertLedgerEntry(context.Context, store.LedgerEntry) (int64, error) {
	panic("memStore: InsertLedgerEntry not used")
}
func (m *memStore) ListLedgerEntries(context.Context, int, int) ([]store.LedgerEntry, error) {
	panic("memStore: ListLedgerEntries not used")
}
func (m *memStore) LedgerSummary(context.Context) ([]store.TierSummary, error) {
	panic("memStore: LedgerSummary not used")
}

// —— M3-a 模拟盘面（04-m3-spec §3）：alert 测试不经过，误用即红 ——
func (m *memStore) InsertSimOrder(context.Context, store.SimOrder) (int64, error) {
	panic("memStore: InsertSimOrder not used")
}
func (m *memStore) ListSimOrders(context.Context, int, int) ([]store.SimOrder, error) {
	panic("memStore: ListSimOrders not used")
}
func (m *memStore) GetSimOrder(context.Context, int64) (store.SimOrder, error) {
	panic("memStore: GetSimOrder not used")
}
func (m *memStore) UpdateSimOrderStatus(context.Context, int64, string, string) error {
	panic("memStore: UpdateSimOrderStatus not used")
}
func (m *memStore) FillSimOrder(context.Context, int64, string, []store.SimPosition) error {
	panic("memStore: FillSimOrder not used")
}
func (m *memStore) TodaySimNotional(context.Context, time.Time) (float64, error) {
	panic("memStore: TodaySimNotional not used")
}
func (m *memStore) InsertSimPosition(context.Context, store.SimPosition) (int64, error) {
	panic("memStore: InsertSimPosition not used")
}
func (m *memStore) ListSimPositions(context.Context, int, int) ([]store.SimPosition, error) {
	panic("memStore: ListSimPositions not used")
}
func (m *memStore) ListOpenSimPositions(context.Context, string, string) ([]store.SimPosition, error) {
	panic("memStore: ListOpenSimPositions not used")
}
func (m *memStore) SettleSimPosition(context.Context, int64, float64, string) error {
	panic("memStore: SettleSimPosition not used")
}

// fakeClock 测试注入时钟。
type fakeClock struct{ t time.Time }

func (c *fakeClock) now() time.Time { return c.t }

// waitFor 轮询等待条件成立（collect/rule 包同款）。
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
