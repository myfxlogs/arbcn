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

// TestPostValidation：非法输入 → 400，且不入库。
func TestPostValidation(t *testing.T) {
	cases := []struct{ name, body, want string }{
		{"bad-json", `{`, "bad JSON"},
		{"unknown-kind", `{"kind":"nope","venue":"x","symbol":"y","value":1}`, "unknown kind"},
		{"no-venue", `{"kind":"iv","symbol":"BTC","value":1}`, "venue and symbol"},
		{"no-symbol", `{"kind":"iv","venue":"deribit","value":1}`, "venue and symbol"},
		{"no-value", `{"kind":"iv","venue":"deribit","symbol":"BTC"}`, "value required"},
		{"bad-ts", `{"kind":"iv","venue":"deribit","symbol":"BTC","value":1,"ts":"2026/08/15"}`, "bad ts"},
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
