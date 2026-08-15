package defirate

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"arbcn/internal/fact"
)

// fixtureServer 以本地 httptest 服务离线 /pools 样例（testdata/），无网络可测。
func fixtureServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/pools" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, "testdata/defillama_pools.json")
	}))
}

func wantValue(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-6 {
		t.Errorf("value = %v, want %v", got, want)
	}
}

// TestDefiRatesFixture：默认 5 池（Aave/Morpho/sUSDe/BUIDL/USDY）过滤与字段逐项比对。
func TestDefiRatesFixture(t *testing.T) {
	srv := fixtureServer(t)
	defer srv.Close()
	c := NewDefiRates(Config{Pools: defaultPools, BaseURL: srv.URL})
	if c.Kind() != fact.KindDefiRate {
		t.Fatalf("Kind() = %q, want %q", c.Kind(), fact.KindDefiRate)
	}
	fs, err := c.Poll(context.Background())
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	if len(fs) != 5 {
		t.Fatalf("Poll = %d facts, want 5", len(fs))
	}
	want := []struct {
		venue, symbol string
		value         float64
	}{
		{"aave-v3", "USDC", 3.29},
		{"morpho-blue", "STEAKUSDC", 4.16},
		{"ethena-usde", "SUSDE", 4.29},
		{"blackrock-buidl", "BUIDL", 3.57},
		{"ondo-yield-assets", "USDY", 3.55},
	}
	for i, w := range want {
		f := fs[i]
		if f.Kind != fact.KindDefiRate || f.Unit != fact.UnitPctAnnualized {
			t.Errorf("facts[%d]: Kind/Unit = %q/%q", i, f.Kind, f.Unit)
		}
		if err := f.Validate(); err != nil {
			t.Errorf("facts[%d]: Validate = %v", i, err)
		}
		if f.Venue != w.venue || f.Symbol != w.symbol {
			t.Errorf("facts[%d]: venue/symbol = %q/%q, want %q/%q", i, f.Venue, f.Symbol, w.venue, w.symbol)
		}
		wantValue(t, f.Value, w.value)
		if d := time.Since(f.Ts); d < -2*time.Second || d > 2*time.Second {
			t.Errorf("facts[%d]: Ts = %v, want ~now", i, f.Ts)
		}
		if f.Src != "yields.llama.fi/pools pool="+defaultPools[i] {
			t.Errorf("facts[%d]: Src = %q", i, f.Src)
		}
	}
}

// TestDefiRatesMissingPool：配置池不在响应中 → 整个 Poll 失败（缺标的 = 漏监控窗口）。
func TestDefiRatesMissingPool(t *testing.T) {
	srv := fixtureServer(t)
	defer srv.Close()
	c := NewDefiRates(Config{Pools: []string{defaultPools[0], "00000000-0000-0000-0000-000000000000"}, BaseURL: srv.URL})
	_, err := c.Poll(context.Background())
	if err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("Poll = %v, want error containing \"missing\"", err)
	}
}

// TestDefiRatesNilApy：目标池 apy=null → 失败（无数据不静默）。
func TestDefiRatesNilApy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"status":"success","data":[{"pool":"aa70268e-4b52-42bf-a116-608b370f9501","project":"aave-v3","symbol":"USDC","apy":null}]}`)
	}))
	defer srv.Close()
	c := NewDefiRates(Config{Pools: []string{"aa70268e-4b52-42bf-a116-608b370f9501"}, BaseURL: srv.URL})
	_, err := c.Poll(context.Background())
	if err == nil || !strings.Contains(err.Error(), "nil apy") {
		t.Fatalf("Poll = %v, want error containing \"nil apy\"", err)
	}
}

// TestDefiRatesBadStatus：status 非 success → 失败。
func TestDefiRatesBadStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `{"status":"error","data":[]}`)
	}))
	defer srv.Close()
	c := NewDefiRates(Config{Pools: defaultPools, BaseURL: srv.URL})
	if _, err := c.Poll(context.Background()); err == nil {
		t.Fatal("Poll = nil, want error")
	}
}

// TestDefiRatesHTTPError：非 200 → 失败（含 status）。
func TestDefiRatesHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "boom", http.StatusBadGateway)
	}))
	defer srv.Close()
	c := NewDefiRates(Config{Pools: defaultPools, BaseURL: srv.URL})
	_, err := c.Poll(context.Background())
	if err == nil || !strings.Contains(err.Error(), "status 502") {
		t.Fatalf("Poll = %v, want error containing status 502", err)
	}
}

// TestDefiRatesNoPools：空池清单 → 不发请求、返回空。
func TestDefiRatesNoPools(t *testing.T) {
	c := NewDefiRates(Config{Pools: nil, BaseURL: "http://127.0.0.1:1"})
	fs, err := c.Poll(context.Background())
	if err != nil || len(fs) != 0 {
		t.Fatalf("Poll = %d facts, %v; want 0, nil", len(fs), err)
	}
}

// TestFromEnv：默认池 / 自定义 / 去重 / 空值回退默认。
func TestFromEnv(t *testing.T) {
	cfg := FromEnv(func(string) string { return "" })
	if len(cfg.Pools) != len(defaultPools) || cfg.BaseURL != DefaultBaseURL {
		t.Fatalf("defaults = %v %q", cfg.Pools, cfg.BaseURL)
	}
	cfg = FromEnv(func(string) string { return "a,b,b" })
	if len(cfg.Pools) != 2 || cfg.Pools[0] != "a" || cfg.Pools[1] != "b" {
		t.Fatalf("custom = %v", cfg.Pools)
	}
	cfg = FromEnv(func(string) string { return " , " })
	if len(cfg.Pools) != len(defaultPools) {
		t.Fatalf("empty-fallback = %v", cfg.Pools)
	}
}

// TestAll：命名 collector defi_rates + 默认间隔 30m。
func TestAll(t *testing.T) {
	ns := All(Config{})
	if len(ns) != 1 {
		t.Fatalf("All() = %d sources, want 1", len(ns))
	}
	n := ns[0]
	if n.Name != "defi_rates" || n.Interval != 30*time.Minute || n.Collector == nil || n.Collector.Kind() != fact.KindDefiRate {
		t.Fatalf("All()[0] = %q %s kind=%v", n.Name, n.Interval, n.Collector)
	}
}
