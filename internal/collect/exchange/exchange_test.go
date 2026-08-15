package exchange

import (
	"context"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"arbcn/internal/collect"
	"arbcn/internal/fact"
)

// fixtureServer 以本地 httptest 服务离线样例响应（testdata/*.json），无网络可测。
func fixtureServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()
	mux.HandleFunc("/fapi/v1/fundingInfo", serveJSON("testdata/binance_fundingInfo.json"))
	mux.HandleFunc("/fapi/v1/premiumIndex", serveJSON("testdata/binance_premiumIndex.json"))
	mux.HandleFunc("/fapi/v1/ticker/price", serveJSON("testdata/binance_ticker_price.json"))
	okxFunding := map[string]string{
		"BTC-USDT-SWAP": "testdata/okx_funding_btc.json",
		"ETH-USDT-SWAP": "testdata/okx_funding_eth.json",
		"TRX-USDT-SWAP": "testdata/okx_funding_trx.json",
	}
	mux.HandleFunc("/api/v5/public/funding-rate", func(w http.ResponseWriter, r *http.Request) {
		f, ok := okxFunding[r.URL.Query().Get("instId")]
		if !ok {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, f)
	})
	mux.HandleFunc("/api/v5/market/tickers", serveJSON("testdata/okx_tickers.json"))
	return httptest.NewServer(mux)
}

func serveJSON(path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, path)
	}
}

func wantValue(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-6 {
		t.Errorf("value = %v, want %v", got, want)
	}
}

// checkFacts 对一组事实做公共断言（Kind/Unit/Validate）。
func checkFacts(t *testing.T, fs []fact.Fact, kind, unit string, n int) {
	t.Helper()
	if len(fs) != n {
		t.Fatalf("got %d facts, want %d", len(fs), n)
	}
	for _, f := range fs {
		if f.Kind != kind {
			t.Errorf("fact %s/%s: Kind = %q, want %q", f.Venue, f.Symbol, f.Kind, kind)
		}
		if f.Unit != unit {
			t.Errorf("fact %s/%s: Unit = %q, want %q", f.Venue, f.Symbol, f.Unit, unit)
		}
		if err := f.Validate(); err != nil {
			t.Errorf("fact %s/%s: Validate = %v", f.Venue, f.Symbol, err)
		}
	}
}

// TestBinanceFundingFixture：离线 fixture 年化折算（8h ×1095 / 4h ×2190，负费率保留）+ Src 含原始值口径。
func TestBinanceFundingFixture(t *testing.T) {
	srv := fixtureServer(t)
	defer srv.Close()
	c := NewBinanceFunding(Config{Symbols: []string{"BTC", "ETH", "TRX"}, BinanceBaseURL: srv.URL})
	if c.Kind() != fact.KindFunding {
		t.Fatalf("Kind() = %q, want %q", c.Kind(), fact.KindFunding)
	}
	fs, err := c.Poll(context.Background())
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	checkFacts(t, fs, fact.KindFunding, fact.UnitPctAnnualized, 3)
	for _, f := range fs {
		if f.Venue != VenueBinance {
			t.Errorf("Venue = %q, want %q", f.Venue, VenueBinance)
		}
		if !f.Ts.Equal(time.UnixMilli(1755288000000)) {
			t.Errorf("%s: Ts = %v, want fixture time", f.Symbol, f.Ts)
		}
	}
	wantVal, wantSrc := []float64{10.95, 8.76, -4.38},
		[]string{"fapi/v1/premiumIndex rate=0.00010000 per8h",
			"fapi/v1/premiumIndex rate=0.00008000 per8h",
			"fapi/v1/premiumIndex rate=-0.00002000 per4h"}
	for i, wv := range wantVal {
		wantValue(t, fs[i].Value, wv)
	}
	if fs[2].Symbol != "TRX" {
		t.Errorf("Symbol[2] = %q, want TRX", fs[2].Symbol)
	}
	for i, ws := range wantSrc {
		if fs[i].Src != ws {
			t.Errorf("Src[%d] = %q, want %q", i, fs[i].Src, ws)
		}
	}
}

// TestBinanceTickerFixture：全 symbol 最新价本地过滤，价格与时间戳逐项比对。
func TestBinanceTickerFixture(t *testing.T) {
	srv := fixtureServer(t)
	defer srv.Close()
	c := NewBinanceTicker(Config{Symbols: []string{"BTC", "ETH", "TRX"}, BinanceBaseURL: srv.URL})
	if c.Kind() != fact.KindTicker {
		t.Fatalf("Kind() = %q, want %q", c.Kind(), fact.KindTicker)
	}
	fs, err := c.Poll(context.Background())
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	checkFacts(t, fs, fact.KindTicker, fact.UnitPrice, 3)
	wantVal := []float64{65012.30, 3123.40, 0.2461}
	for i, f := range fs {
		wantValue(t, f.Value, wantVal[i])
		if f.Src != "fapi/v1/ticker/price" {
			t.Errorf("Src[%d] = %q", i, f.Src)
		}
		if !f.Ts.Equal(time.UnixMilli(1755288000000)) {
			t.Errorf("%s: Ts = %v, want fixture time", f.Symbol, f.Ts)
		}
	}
}

// TestOKXFundingFixture：结算时间戳对推断频率（8h/4h）→ 年化；TRX 负费率保留。
func TestOKXFundingFixture(t *testing.T) {
	srv := fixtureServer(t)
	defer srv.Close()
	c := NewOKXFunding(Config{Symbols: []string{"BTC", "ETH", "TRX"}, OKXBaseURL: srv.URL})
	fs, err := c.Poll(context.Background())
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	checkFacts(t, fs, fact.KindFunding, fact.UnitPctAnnualized, 3)
	wantVal, wantSrc := []float64{10.95, 10.95, -3.285},
		[]string{"api/v5/public/funding-rate rate=0.00010000 per8h",
			"api/v5/public/funding-rate rate=0.00005000 per4h",
			"api/v5/public/funding-rate rate=-0.00003000 per8h"}
	for i, wv := range wantVal {
		wantValue(t, fs[i].Value, wv)
		if fs[i].Venue != VenueOKX {
			t.Errorf("Venue[%d] = %q, want %q", i, fs[i].Venue, VenueOKX)
		}
		if fs[i].Src != wantSrc[i] {
			t.Errorf("Src[%d] = %q, want %q", i, fs[i].Src, wantSrc[i])
		}
		if d := time.Since(fs[i].Ts); d < -2*time.Second || d > 2*time.Second {
			t.Errorf("Ts[%d] = %v, want ~now", i, fs[i].Ts)
		}
	}
}

// TestOKXTickerFixture：全部 SWAP ticker 本地过滤，含非目标 instrument（过滤正确性）。
func TestOKXTickerFixture(t *testing.T) {
	srv := fixtureServer(t)
	defer srv.Close()
	c := NewOKXTicker(Config{Symbols: []string{"BTC", "ETH", "TRX"}, OKXBaseURL: srv.URL})
	fs, err := c.Poll(context.Background())
	if err != nil {
		t.Fatalf("Poll: %v", err)
	}
	checkFacts(t, fs, fact.KindTicker, fact.UnitPrice, 3)
	wantVal := []float64{65012.4, 3123.5, 0.2462}
	for i, f := range fs {
		wantValue(t, f.Value, wantVal[i])
		if !f.Ts.Equal(time.UnixMilli(1755288000000)) {
			t.Errorf("%s: Ts = %v, want fixture time", f.Symbol, f.Ts)
		}
	}
}

// TestFromEnv：默认币种 / 自定义 / 大小写与空格归一 / 空值回退默认。
func TestFromEnv(t *testing.T) {
	cases := []struct {
		name string
		env  string
		want []string
	}{
		{"default", "", []string{"BTC", "ETH", "TRX"}},
		{"custom", "BTC,TRX", []string{"BTC", "TRX"}},
		{"normalize", " btc , eth ", []string{"BTC", "ETH"}},
		{"empty-fallback", " , ", []string{"BTC", "ETH", "TRX"}},
	}
	for _, tc := range cases {
		cfg := FromEnv(func(string) string { return tc.env })
		if len(cfg.Symbols) != len(tc.want) {
			t.Errorf("%s: Symbols = %v, want %v", tc.name, cfg.Symbols, tc.want)
			continue
		}
		for i := range tc.want {
			if cfg.Symbols[i] != tc.want[i] {
				t.Errorf("%s: Symbols = %v, want %v", tc.name, cfg.Symbols, tc.want)
				break
			}
		}
	}
	if cfg := FromEnv(func(string) string { return "" }); cfg.BinanceBaseURL != "https://fapi.binance.com" || cfg.OKXBaseURL != "https://www.okx.com" {
		t.Errorf("default base URLs = %q/%q", cfg.BinanceBaseURL, cfg.OKXBaseURL)
	}
}

// TestAll：四个命名 collector（binance/okx × funding/ticker）+ 默认间隔。
func TestAll(t *testing.T) {
	ns := All(Config{Symbols: []string{"BTC"}})
	want := map[string]struct {
		kind string
		iv   time.Duration
	}{
		"binance_funding": {fact.KindFunding, 5 * time.Minute},
		"binance_ticker":  {fact.KindTicker, time.Minute},
		"okx_funding":     {fact.KindFunding, 5 * time.Minute},
		"okx_ticker":      {fact.KindTicker, time.Minute},
	}
	if len(ns) != len(want) {
		t.Fatalf("All() = %d sources, want %d", len(ns), len(want))
	}
	for _, n := range ns {
		w, ok := want[n.Name]
		if !ok {
			t.Errorf("unexpected source %q", n.Name)
			continue
		}
		if n.Collector == nil || n.Collector.Kind() != w.kind || n.Interval != w.iv {
			t.Errorf("%s = kind/iv (%s/%s), want (%s/%s)", n.Name, kindOf(n), n.Interval, w.kind, w.iv)
		}
	}
}

func kindOf(n collect.Named) string {
	if n.Collector == nil {
		return "<nil>"
	}
	return n.Collector.Kind()
}
