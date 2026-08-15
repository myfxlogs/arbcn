package simtestnet

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"
	"time"
)

// iso8601Regex 匹配 OKX 要求的 ISO 8601 UTC 时间戳（OK-ACCESS-TIMESTAMP 头）。
var iso8601Regex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z$`)

// fakeRecorder 记录 probe 成功后的 Record 调用（探针成功登记锚点）。
type fakeRecorder struct {
	mu sync.Mutex
	m  map[string]time.Time
}

func newFakeRecorder() *fakeRecorder { return &fakeRecorder{m: map[string]time.Time{}} }
func (f *fakeRecorder) Record(name string, at time.Time) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.m[name] = at
}
func (f *fakeRecorder) got(name string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.m[name]
	return ok
}

// probeFixtureServer 模拟 binance_testnet / okx_demo 只读端点，并验证签名正确性。
// 签名错 → 返回 401（探针必须带合法签名才连通）。
func probeFixtureServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	// Binance testnet：公共 time + 签名余额。
	mux.HandleFunc("/fapi/v1/time", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"serverTime":1723708800000}`))
	})
	mux.HandleFunc("/fapi/v2/balance", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-MBX-APIKEY") != "bkey" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		q := r.URL.Query()
		wantSig := binanceSign("bsec", "recvWindow=5000&timestamp="+q.Get("timestamp"))
		if q.Get("signature") != wantSig {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	})

	// OKX demo：公共 time + 签名余额（x-simulated-trading:1）。
	mux.HandleFunc("/api/v5/public/time", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"code":"0","msg":"","data":[{"ts":"1723708800000"}]}`))
	})
	mux.HandleFunc("/api/v5/account/balance", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-simulated-trading") != "1" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if r.Header.Get("OK-ACCESS-KEY") != "okey" || r.Header.Get("OK-ACCESS-PASSPHRASE") != "opass" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		ts := r.Header.Get("OK-ACCESS-TIMESTAMP")
		// [对抗测试锚点] 2026-08-15 部署实测：OKX 要求 ISO 8601 UTC 格式，Unix 毫秒 → 50102。
		// 改回 FormatInt(UnixMilli) → 本校验必红（TestProbeRunRecordsBoth 不 Record）。
		if !iso8601Regex.MatchString(ts) {
			t.Errorf("OK-ACCESS-TIMESTAMP = %q, want ISO 8601 UTC（Unix 毫秒 → OKX 50102）", ts)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		wantSign := okxSign("osec", ts, "GET", "/api/v5/account/balance", "")
		if r.Header.Get("OK-ACCESS-SIGN") != wantSign {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"code":"0","msg":"","data":[]}`))
	})

	return httptest.NewServer(mux)
}

func testProbe(t *testing.T, srv *httptest.Server) (*Probe, *fakeRecorder) {
	t.Helper()
	cfg := Config{
		BinanceAPIKey: "bkey", BinanceSecret: "bsec",
		OKXAPIKey: "okey", OKXSecret: "osec", OKXPassphrase: "opass",
	}
	rec := newFakeRecorder()
	p, ok := NewProbe(cfg, rec)
	if !ok {
		t.Fatal("NewProbe = nil, want probe（有 key）")
	}
	p.BinanceBaseURL = srv.URL
	p.OKXBaseURL = srv.URL
	p.Now = func() time.Time { return t0 }
	return p, rec
}

// t0 固定时钟（探针时间戳确定性）。
var t0 = time.Unix(1723708800, 0)

// TestProbeRunRecordsBoth：[对抗测试锚点 §9.4 S3] 两路只读探针成功 →
// Record(sim_testnet_binance / sim_testnet_okx) 各一次。删成功后的 Record 调用 → 必红。
func TestProbeRunRecordsBoth(t *testing.T) {
	srv := probeFixtureServer(t)
	defer srv.Close()
	p, rec := testProbe(t, srv)

	if err := p.Run(context.Background()); err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !rec.got(SourceBinanceTestnet) {
		t.Error("未 Record sim_testnet_binance（成功探针必须登记）")
	}
	if !rec.got(SourceOKXDemo) {
		t.Error("未 Record sim_testnet_okx（成功探针必须登记）")
	}
}

// TestProbeSignFail：签名错误 → 401 → Run 返回错误，不 Record（登记只在成功路径）。
func TestProbeSignFail(t *testing.T) {
	srv := probeFixtureServer(t)
	defer srv.Close()
	p, rec := testProbe(t, srv)
	p.cfg.BinanceSecret = "wrong-secret" // 破坏 binance 签名

	if err := p.Run(context.Background()); err == nil {
		t.Fatal("Run(签名错) = nil, want error")
	}
	if rec.got(SourceBinanceTestnet) {
		t.Error("签名错不应 Record sim_testnet_binance")
	}
	// OKX 未受影响（独立判断）。
	if !rec.got(SourceOKXDemo) {
		t.Error("OKX 独立成功应 Record sim_testnet_okx")
	}
}

// TestNewProbeDegradesNoKeys：无 key → (nil, false)（S3 降级禁用，不阻塞其他子任务）。
func TestNewProbeDegradesNoKeys(t *testing.T) {
	if p, ok := NewProbe(Config{}, newFakeRecorder()); ok || p != nil {
		t.Fatalf("NewProbe(empty) = (%v, %v), want (nil, false)", p, ok)
	}
}
