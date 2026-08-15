package fx

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

// fixtureServer 服务离线报价脚本（testdata/），并校验 Referer 头（新浪 hq 硬要求）。
func fixtureServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/list="+codeUSDCNH { // 新浪 hq 的 list= 在路径上（非查询参数）
			http.NotFound(w, r)
			return
		}
		if r.Referer() != DefaultReferer {
			t.Errorf("Referer = %q, want %q", r.Referer(), DefaultReferer)
		}
		http.ServeFile(w, r, "testdata/sina_fx_usdcnh.txt")
	}))
}

// TestFXFixture：最新价 6.7443 + Ts 取自报价日期时间（CST）。
func TestFXFixture(t *testing.T) {
	srv := fixtureServer(t)
	defer srv.Close()
	c := NewFX(Config{BaseURL: srv.URL})
	if c.Kind() != fact.KindFX {
		t.Fatalf("Kind() = %q, want %q", c.Kind(), fact.KindFX)
	}
	fs, err := c.Poll(context.Background())
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	if len(fs) != 1 {
		t.Fatalf("Poll = %d facts, want 1", len(fs))
	}
	f := fs[0]
	if err := f.Validate(); err != nil {
		t.Fatalf("Validate = %v", err)
	}
	if f.Venue != VenueSina || f.Symbol != "USDCNH" || f.Unit != fact.UnitPrice {
		t.Errorf("venue/symbol/unit = %q/%q/%q", f.Venue, f.Symbol, f.Unit)
	}
	if want := 6.7443; f.Value != want {
		t.Errorf("Value = %v, want %v", f.Value, want)
	}
	wantTs := time.Date(2026, 8, 15, 4, 59, 44, 0, time.FixedZone("CST", 8*3600))
	if !f.Ts.Equal(wantTs) {
		t.Errorf("Ts = %v, want %v", f.Ts, wantTs)
	}
	if f.Src != "hq.sinajs.cn/list=fx_susdcnh" {
		t.Errorf("Src = %q", f.Src)
	}
}

// TestFXBadBody：引号载荷缺失/字段不足/价非法 → 失败（含原因）。
func TestFXBadBody(t *testing.T) {
	cases := []struct{ name, body, want string }{
		{"no-quotes", `var hq_str_fx_susdcnh=oops;`, "no quoted payload"},
		{"short", `var hq_str_fx_susdcnh="a,b,c";`, "fields"},
		{"bad-price", `var hq_str_fx_susdcnh="00:00:00,1,2,abc,4,5,6,7,8,9,10,11,12,13,14,15,16,2026-08-15";`, "bad price"},
		{"non-positive", `var hq_str_fx_susdcnh="00:00:00,1,2,0,4,5,6,7,8,9,10,11,12,13,14,15,16,2026-08-15";`, "bad price"},
	}
	for _, tc := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			fmt.Fprint(w, tc.body)
		}))
		c := NewFX(Config{BaseURL: srv.URL})
		_, err := c.Poll(context.Background())
		srv.Close()
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("%s: Poll = %v, want error containing %q", tc.name, err, tc.want)
		}
	}
}

// TestFXTsFallback：日期字段非法 → Ts 回退本地时间（时间戳异常不阻断行情）。
func TestFXTsFallback(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprint(w, `var hq_str_fx_susdcnh="00:00:00,1,2,6.5,4,5,6,7,8,9,10,11,12,13,14,15,16,not-a-date";`)
	}))
	defer srv.Close()
	fs, err := NewFX(Config{BaseURL: srv.URL}).Poll(context.Background())
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	if d := time.Since(fs[0].Ts); d < -2*time.Second || d > 2*time.Second {
		t.Errorf("Ts = %v, want ~now", fs[0].Ts)
	}
}

// TestFXHTTPError：非 200 → 失败（含 status）。
func TestFXHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer srv.Close()
	_, err := NewFX(Config{BaseURL: srv.URL}).Poll(context.Background())
	if err == nil || !strings.Contains(err.Error(), "status 403") {
		t.Fatalf("Poll = %v, want error containing status 403", err)
	}
}

// TestFromEnv：默认基址/Referer + 覆盖。
func TestFromEnv(t *testing.T) {
	cfg := FromEnv(func(string) string { return "" })
	if cfg.BaseURL != DefaultBaseURL || cfg.Referer != DefaultReferer {
		t.Fatalf("defaults = %q/%q", cfg.BaseURL, cfg.Referer)
	}
	cfg = FromEnv(func(string) string { return "http://proxy.local" })
	if cfg.BaseURL != "http://proxy.local" {
		t.Fatalf("override = %q", cfg.BaseURL)
	}
}

// TestAll：命名 collector fx + 默认间隔 5m。
func TestAll(t *testing.T) {
	ns := All(Config{})
	if len(ns) != 1 {
		t.Fatalf("All() = %d sources, want 1", len(ns))
	}
	if ns[0].Name != "fx" || ns[0].Interval != 5*time.Minute || ns[0].Collector == nil || ns[0].Collector.Kind() != fact.KindFX {
		t.Fatalf("All()[0] = %q %s", ns[0].Name, ns[0].Interval)
	}
}
