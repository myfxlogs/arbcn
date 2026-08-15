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
	facts  []fact.Fact
	alerts []store.Alert
	states []store.RuleState
	err    error // 注入存储层故障
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

func (f *fakeStore) ListTriggerStates(context.Context) ([]store.RuleState, error) {
	if f.err != nil {
		return nil, f.err
	}
	return append([]store.RuleState(nil), f.states...), nil
}

// —— 写路径：dashboard 服务不经过（只读 + ack），误用即红 ——

func (f *fakeStore) InsertFacts(context.Context, []fact.Fact) error {
	panic("fakeStore: InsertFacts not used")
}
func (f *fakeStore) QueryFacts(context.Context, store.FactQuery) ([]fact.Fact, error) {
	panic("fakeStore: QueryFacts not used")
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
