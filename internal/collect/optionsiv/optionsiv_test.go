package optionsiv

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"arbcn/internal/fact"
)

// fixtureServer 按 currency 参数服务离线 DVOL 日线（testdata/），无网络可测。
func fixtureServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v2/public/get_volatility_index_data" {
			http.NotFound(w, r)
			return
		}
		switch r.URL.Query().Get("currency") {
		case "BTC":
			http.ServeFile(w, r, "testdata/deribit_iv_btc.json")
		case "ETH":
			http.ServeFile(w, r, "testdata/deribit_iv_eth.json")
		default:
			http.NotFound(w, r)
		}
	}))
}

// TestIVFixture：取最近一根 close = 当前 DVOL，ts 取该根时间戳。
func TestIVFixture(t *testing.T) {
	srv := fixtureServer(t)
	defer srv.Close()
	c := NewDeribitIV(Config{Currencies: DefaultCurrencies, BaseURL: srv.URL})
	if c.Kind() != fact.KindIV {
		t.Fatalf("Kind() = %q, want %q", c.Kind(), fact.KindIV)
	}
	fs, err := c.Poll(context.Background())
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	if len(fs) != 2 {
		t.Fatalf("Poll = %d facts, want 2", len(fs))
	}
	want := []struct {
		symbol string
		value  float64
	}{{"BTC", 34.82}, {"ETH", 45.21}}
	for i, w := range want {
		f := fs[i]
		if err := f.Validate(); err != nil {
			t.Errorf("facts[%d]: Validate = %v", i, err)
		}
		if f.Venue != VenueDeribit || f.Unit != fact.UnitPct || f.Symbol != w.symbol {
			t.Errorf("facts[%d]: venue/unit/symbol = %q/%q/%q", i, f.Venue, f.Unit, f.Symbol)
		}
		if f.Value != w.value {
			t.Errorf("facts[%d]: Value = %v, want %v", i, f.Value, w.value)
		}
		if !f.Ts.Equal(time.UnixMilli(1786579200000)) {
			t.Errorf("facts[%d]: Ts = %v, want fixture time", i, f.Ts)
		}
		if f.Src != "api/v2/public/get_volatility_index_data DVOL" {
			t.Errorf("facts[%d]: Src = %q", i, f.Src)
		}
	}
}

// TestIVEmptyData / TestIVRPCError / TestIVBadRow / TestIVHTTPError：各错误路径。
func TestIVErrors(t *testing.T) {
	cases := []struct{ name, body, want string }{
		{"empty", `{"result":{"data":[]}}`, "empty data"},
		{"rpc-error", `{"error":{"code":-32602,"message":"Invalid params"}}`, "Invalid params"},
		{"bad-row", `{"result":{"data":[[1786579200000,34.74]]}}`, "bad row length"},
		{"no-result", `{"jsonrpc":"2.0"}`, "empty data"},
	}
	for _, tc := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprint(w, tc.body)
		}))
		c := NewDeribitIV(Config{Currencies: []string{"BTC"}, BaseURL: srv.URL})
		_, err := c.Poll(context.Background())
		srv.Close()
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: Poll = %v, want error containing %q", tc.name, err, tc.want)
		}
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "blocked", http.StatusForbidden)
	}))
	defer srv.Close()
	_, err := NewDeribitIV(Config{Currencies: []string{"BTC"}, BaseURL: srv.URL}).Poll(context.Background())
	if err == nil || !strings.Contains(err.Error(), "status 403") {
		t.Errorf("http: Poll = %v, want error containing status 403", err)
	}
}

// TestIVNoCurrencies：空币种清单 → 不发请求、返回空。
func TestIVNoCurrencies(t *testing.T) {
	c := NewDeribitIV(Config{BaseURL: "http://127.0.0.1:1"})
	fs, err := c.Poll(context.Background())
	if err != nil || len(fs) != 0 {
		t.Fatalf("Poll = %d facts, %v; want 0, nil", len(fs), err)
	}
}

// TestFromEnv：默认币种 / 自定义归一。
func TestFromEnv(t *testing.T) {
	cfg := FromEnv(func(string) string { return "" })
	if len(cfg.Currencies) != 2 || cfg.BaseURL != DefaultBaseURL {
		t.Fatalf("defaults = %v %q", cfg.Currencies, cfg.BaseURL)
	}
	cfg = FromEnv(func(string) string { return " btc , ETH " })
	if len(cfg.Currencies) != 2 || cfg.Currencies[0] != "BTC" {
		t.Fatalf("custom = %v", cfg.Currencies)
	}
}

// TestAll：命名 collector deribit_iv + 默认间隔 30m。
func TestAll(t *testing.T) {
	ns := All(Config{})
	if len(ns) != 1 {
		t.Fatalf("All() = %d sources, want 1", len(ns))
	}
	if ns[0].Name != "deribit_iv" || ns[0].Interval != 30*time.Minute || ns[0].Collector == nil || ns[0].Collector.Kind() != fact.KindIV {
		t.Fatalf("All()[0] = %q %s", ns[0].Name, ns[0].Interval)
	}
}
