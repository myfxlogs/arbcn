package manual

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"arbcn/internal/fact"
	"arbcn/internal/store"
)

// fakeStore 记录 InsertFacts 调用（其余接口方法仅占位）。
type fakeStore struct {
	mu    sync.Mutex
	facts []fact.Fact
	err   error
}

func (f *fakeStore) InsertFacts(_ context.Context, fs []fact.Fact) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.err != nil {
		return f.err
	}
	f.facts = append(f.facts, fs...)
	return nil
}

func (f *fakeStore) QueryFacts(context.Context, store.FactQuery) ([]fact.Fact, error) {
	return nil, nil
}
func (f *fakeStore) UpsertRule(context.Context, store.Rule) (int64, error) { return 0, nil }
func (f *fakeStore) ListRules(context.Context) ([]store.Rule, error)       { return nil, nil }
func (f *fakeStore) GetTriggerState(context.Context, int64) (store.TriggerState, error) {
	return store.TriggerState{}, store.ErrNotFound
}
func (f *fakeStore) PutTriggerState(context.Context, store.TriggerState) error { return nil }
func (f *fakeStore) InsertAlert(context.Context, store.Alert) error            { return nil }
func (f *fakeStore) PendingAlerts(context.Context, int) ([]store.Alert, error) { return nil, nil }
func (f *fakeStore) MarkAlertDelivered(context.Context, int64) error           { return nil }
func (f *fakeStore) LatestFacts(context.Context, string, string, string) ([]fact.Fact, error) {
	return nil, nil
}
func (f *fakeStore) ListAlerts(context.Context, int, int) ([]store.Alert, error) { return nil, nil }
func (f *fakeStore) AckAlert(context.Context, int64) error                       { return nil }
func (f *fakeStore) ListUnacked(context.Context) ([]store.Alert, error)          { return nil, nil }
func (f *fakeStore) AckAll(context.Context) (int64, error)                       { return 0, nil }
func (f *fakeStore) ListTriggerStates(context.Context) ([]store.RuleState, error) {
	return nil, nil
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
func (f *fakeStore) ListSimOrders(context.Context, int, int) ([]store.SimOrder, error) {
	panic("fakeStore: ListSimOrders not used")
}
func (f *fakeStore) UpsertTestnetAccount(context.Context, store.TestnetAccount) error {
	panic("fakeStore: UpsertTestnetAccount not used")
}
func (f *fakeStore) ListTestnetAccounts(context.Context) ([]store.TestnetAccount, error) {
	panic("fakeStore: ListTestnetAccounts not used")
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
func (f *fakeStore) SettleSimPositionFunding(context.Context, int64, int64, float64) error {
	panic("fakeStore: SettleSimPositionFunding not used")
}
func (f *fakeStore) InitSimAccount(context.Context, float64) error {
	panic("fakeStore: InitSimAccount not used")
}
func (f *fakeStore) GetSimAccount(context.Context) (store.SimAccount, error) {
	panic("fakeStore: GetSimAccount not used")
}
func (f *fakeStore) ListCashFlows(context.Context, int, int) ([]store.CashFlow, error) {
	panic("fakeStore: ListCashFlows not used")
}
func (f *fakeStore) InsertEquitySnapshot(context.Context, store.EquitySnapshot) error {
	panic("fakeStore: InsertEquitySnapshot not used")
}
func (f *fakeStore) InsertSimExecution(context.Context, store.SimExecution) (int64, error) {
	panic("fakeStore: InsertSimExecution not used (D-098 only simapi)")
}
func (f *fakeStore) ListSimExecutions(context.Context, int64) ([]store.SimExecution, error) {
	panic("fakeStore: ListSimExecutions not used")
}
func (f *fakeStore) ListEquitySnapshots(context.Context, time.Time, int) ([]store.EquitySnapshot, error) {
	panic("fakeStore: ListEquitySnapshots not used")
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
func (f *fakeStore) CloseSimOrder(context.Context, int64, string, []store.SimLegClose) (int, error) {
	panic("fakeStore: CloseSimOrder not used")
}

func (f *fakeStore) count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.facts)
}

func post(t *testing.T, h http.Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/manual/fact", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// TestPostOK：最小字段 → 入库成功，Ts=now、Src=manual 默认补齐。
func TestPostOK(t *testing.T) {
	st := &fakeStore{}
	h := NewHandler(st)
	before := time.Now()
	rec := post(t, h, `{"kind":"iv","venue":"deribit","symbol":"BTC","value":34.8}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	if st.count() != 1 {
		t.Fatalf("inserted = %d, want 1", st.count())
	}
	st.mu.Lock()
	f := st.facts[0]
	st.mu.Unlock()
	if f.Kind != fact.KindIV || f.Venue != "deribit" || f.Symbol != "BTC" || f.Value != 34.8 {
		t.Errorf("fact = %+v", f)
	}
	if f.Ts.Before(before) || f.Ts.After(time.Now()) {
		t.Errorf("Ts = %v, want ~now", f.Ts)
	}
	if f.Src != "manual" {
		t.Errorf("Src = %q, want manual", f.Src)
	}
}

// TestPostWithFields：unit/ts/src 透传（ts RFC3339）。
func TestPostWithFields(t *testing.T) {
	st := &fakeStore{}
	rec := post(t, NewHandler(st),
		`{"kind":"deposit_rate","venue":"boc","symbol":"一年","value":0.95,"unit":"pct_annualized","ts":"2026-08-15T10:30:00+08:00","src":"manual: boc 柜台"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	st.mu.Lock()
	f := st.facts[0]
	st.mu.Unlock()
	if f.Unit != "pct_annualized" || f.Src != "manual: boc 柜台" {
		t.Errorf("unit/src = %q/%q", f.Unit, f.Src)
	}
	want := time.Date(2026, 8, 15, 10, 30, 0, 0, time.FixedZone("CST", 8*3600))
	if !f.Ts.Equal(want) {
		t.Errorf("Ts = %v, want %v", f.Ts, want)
	}
}

// TestPostUnitDefaultFill：unit 缺省按 kind 填充（R3-M3——空 unit 破坏 unit 感知展示）。
func TestPostUnitDefaultFill(t *testing.T) {
	st := &fakeStore{}
	rec := post(t, NewHandler(st),
		`{"kind":"funding","venue":"binance","symbol":"BTC","value":10.95}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body)
	}
	st.mu.Lock()
	f := st.facts[0]
	st.mu.Unlock()
	if f.Unit != fact.UnitPctAnnualized {
		t.Errorf("unit = %q, want %q（funding 默认年化百分数）", f.Unit, fact.UnitPctAnnualized)
	}
}

// TestPostUnitConflictRejected：显式传与 kind 口径冲突的单位 → 400（防阈值静默失效）。
func TestPostUnitConflictRejected(t *testing.T) {
	st := &fakeStore{}
	rec := post(t, NewHandler(st),
		`{"kind":"funding","venue":"binance","symbol":"BTC","value":10.95,"unit":"ratio"}`)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400（funding 单位应为 pct_annualized）", rec.Code)
	}
	if len(st.facts) != 0 {
		t.Fatalf("facts = %d, want 0（冲突单位不得入库）", len(st.facts))
	}
}

// TestPostValidation：非法输入 → 400，且不入库。
func TestPostValidation(t *testing.T) {
	cases := []struct{ name, body, want string }{
		{"bad-json", `{`, "bad JSON"},
		{"unknown-kind", `{"kind":"nope","venue":"x","symbol":"y","value":1}`, "unknown kind"},
		{"no-venue", `{"kind":"iv","symbol":"BTC","value":1}`, "venue and symbol"},
		{"no-symbol", `{"kind":"iv","venue":"deribit","value":1}`, "venue and symbol"},
		{"no-value", `{"kind":"iv","venue":"deribit","symbol":"BTC"}`, "value required"},
		{"bad-ts", `{"kind":"iv","venue":"deribit","symbol":"BTC","value":1,"ts":"2026/08/15"}`, "bad ts"},
		{"unit-conflict", `{"kind":"iv","venue":"deribit","symbol":"BTC","value":1,"unit":"days"}`, "单位应为"},
	}
	for _, tc := range cases {
		st := &fakeStore{}
		rec := post(t, NewHandler(st), tc.body)
		if rec.Code != http.StatusBadRequest {
			t.Errorf("%s: status = %d, want 400", tc.name, rec.Code)
			continue
		}
		var resp map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil || !strings.Contains(resp["error"], tc.want) {
			t.Errorf("%s: body = %s, want error containing %q", tc.name, rec.Body, tc.want)
		}
		if st.count() != 0 {
			t.Errorf("%s: inserted %d facts, want 0", tc.name, st.count())
		}
	}
}

// TestMethodNotAllowed：GET → 405。
func TestMethodNotAllowed(t *testing.T) {
	rec := httptest.NewRecorder()
	NewHandler(&fakeStore{}).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/manual/fact", nil))
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want 405", rec.Code)
	}
}

// TestNilStore：Store 未接线 → 503（依赖缺失不静默）。
func TestNilStore(t *testing.T) {
	rec := post(t, NewHandler(nil), `{"kind":"iv","venue":"x","symbol":"y","value":1}`)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
}

// TestStoreError：写库失败 → 500。
func TestStoreError(t *testing.T) {
	st := &fakeStore{err: errors.New("db down")}
	rec := post(t, NewHandler(st), `{"kind":"iv","venue":"deribit","symbol":"BTC","value":1}`)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}
}
