package exchange

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"arbcn/internal/collect"
)

// 错误路径测试：缺币种 / OKX 业务码 / HTTP 状态 / 无符号配置。
// 口径：任一目标币种异常 → 整个 Poll 失败（缺币种 = 漏监控窗口，不静默降级）。

// TestBinanceFundingMissingSymbol：请求币种不在 premiumIndex/fundingInfo 中 → 失败。
func TestBinanceFundingMissingSymbol(t *testing.T) {
	srv := fixtureServer(t)
	defer srv.Close()
	c := NewBinanceFunding(Config{Symbols: []string{"BTC", "LTC"}, BinanceBaseURL: srv.URL})
	_, err := c.Poll(context.Background())
	if err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("Poll = %v, want error containing \"missing\"", err)
	}
}

// TestOKXTickerMissingSymbol：目标 instrument 不在响应中 → 失败。
func TestOKXTickerMissingSymbol(t *testing.T) {
	srv := fixtureServer(t)
	defer srv.Close()
	c := NewOKXTicker(Config{Symbols: []string{"BTC", "LTC"}, OKXBaseURL: srv.URL})
	_, err := c.Poll(context.Background())
	if err == nil || !strings.Contains(err.Error(), "missing") {
		t.Fatalf("Poll = %v, want error containing \"missing\"", err)
	}
}

// TestOKXBusinessError：HTTP 200 但业务码非 0 → 错误透出（含 code）。
func TestOKXBusinessError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"code":"51000","msg":"boom","data":[]}`))
	}))
	defer srv.Close()
	c := NewOKXTicker(Config{Symbols: []string{"BTC"}, OKXBaseURL: srv.URL})
	_, err := c.Poll(context.Background())
	if err == nil || !strings.Contains(err.Error(), "code=51000") {
		t.Fatalf("Poll = %v, want error containing code=51000", err)
	}
}

// TestHTTPStatusError：非 200 响应 → 错误透出（含 status）。
func TestHTTPStatusError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer srv.Close()
	c := NewBinanceTicker(Config{Symbols: []string{"BTC"}, BinanceBaseURL: srv.URL})
	_, err := c.Poll(context.Background())
	if err == nil || !strings.Contains(err.Error(), "status 403") {
		t.Fatalf("Poll = %v, want error containing status 403", err)
	}
}

// TestPollNoSymbols：无符号配置 → 不发请求、返回空（FromEnv 常态下不会产生此配置）。
func TestPollNoSymbols(t *testing.T) {
	srv := fixtureServer(t)
	defer srv.Close()
	cfg := Config{BinanceBaseURL: srv.URL, OKXBaseURL: srv.URL}
	cs := []collect.Collector{NewBinanceFunding(cfg), NewBinanceTicker(cfg), NewOKXFunding(cfg), NewOKXTicker(cfg)}
	for _, c := range cs {
		fs, err := c.Poll(context.Background())
		if err != nil || len(fs) != 0 {
			t.Errorf("%T: Poll = %d facts, %v; want 0, nil", c, len(fs), err)
		}
	}
}
