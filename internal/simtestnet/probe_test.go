package simtestnet

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sync"
	"testing"
	"time"

	"arbcn/internal/store"
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
		// D-040 余额解析锚点：真实 testnet 虚拟资金形态（BTC + 稳定币）。
		w.Write([]byte(`[
			{"accountAlias":"s_testnet","asset":"BTC","balance":"0.01000000","crossWalletBalance":"0.01000000","crossUnPnl":"0.00000000","availableBalance":"0.01000000","maxWithdrawAmount":"0.01000000","marginAvailable":true,"updateTime":0},
			{"accountAlias":"s_testnet","asset":"USDT","balance":"5000.00000000","crossWalletBalance":"5000.00000000","crossUnPnl":"0.00000000","availableBalance":"5000.00000000","maxWithdrawAmount":"5000.00000000","marginAvailable":true,"updateTime":0},
			{"accountAlias":"s_testnet","asset":"USDC","balance":"5000.00000000","crossWalletBalance":"5000.00000000","crossUnPnl":"0.00000000","availableBalance":"5000.00000000","maxWithdrawAmount":"5000.00000000","marginAvailable":true,"updateTime":0}
		]`))
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
		// D-040 余额解析锚点：OKX demo 虚拟资金（USDT 5000 + BTC 1，totalEq 精确折算）。
		w.Write([]byte(`{"code":"0","msg":"","data":[{"adjEq":"","balData":[],"details":[{"availBal":"","availEq":"","cashBal":"5000","ccy":"USDT","eq":"5000","eqUsd":"5000","frozenBal":"0","interest":"","upl":"","uplLiab":""},{"availBal":"","availEq":"","cashBal":"1","ccy":"BTC","eq":"1","eqUsd":"60000","frozenBal":"0","interest":"","upl":"","uplLiab":""}],"imr":"","isoEq":"","mgnRatio":"","mmr":"","notionalUsd":"","ordFroz":"","totalEq":"65000","ts":"1723708800000","uTime":"1723708800000"}]}`))
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

	if _, err := p.Run(context.Background()); err != nil {
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

	if _, err := p.Run(context.Background()); err == nil {
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

// TestProbeRunReturnsSnapshots：[对抗测试锚点 D-040] 探针成功 → Run 返回两路余额快照
// （测试网账户区数据面）。删余额解析 → 必红。
//
// binance：accountAlias=s_testnet；equity_usd = 稳定币合计（USDT 5000 + USDC 5000 = 10000，
// 近似口径）；BTC 无行情折算 → EquityUSD=0。okx：equity_usd = totalEq 65000（精确）；
// details 带 eqUsd。
func TestProbeRunReturnsSnapshots(t *testing.T) {
	srv := probeFixtureServer(t)
	defer srv.Close()
	p, _ := testProbe(t, srv)

	accts, err := p.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if len(accts) != 2 {
		t.Fatalf("Run 返回 %d 个账户快照, want 2（binance + okx）", len(accts))
	}

	bin, ok := findAccount(accts, SourceBinanceTestnet)
	if !ok {
		t.Fatal("缺少 sim_testnet_binance 快照")
	}
	if bin.AccountAlias != "s_testnet" {
		t.Errorf("binance AccountAlias = %q, want s_testnet", bin.AccountAlias)
	}
	if bin.EquityUSD != 10000 {
		t.Errorf("binance EquityUSD = %v, want 10000（USDT 5000 + USDC 5000 稳定币合计）", bin.EquityUSD)
	}
	if len(bin.Details) != 3 {
		t.Fatalf("binance details = %d 项, want 3（BTC/USDT/USDC）", len(bin.Details))
	}
	for _, d := range bin.Details {
		if d.Asset == "BTC" && d.EquityUSD != 0 {
			t.Errorf("binance BTC EquityUSD = %v, want 0（非稳定币无行情折算）", d.EquityUSD)
		}
		if d.Asset == "USDT" && (d.Balance != "5000.00000000" || d.EquityUSD != 5000) {
			t.Errorf("binance USDT detail = (%s, %v), want (5000.00000000, 5000)", d.Balance, d.EquityUSD)
		}
	}

	okx, ok := findAccount(accts, SourceOKXDemo)
	if !ok {
		t.Fatal("缺少 sim_testnet_okx 快照")
	}
	if okx.EquityUSD != 65000 {
		t.Errorf("okx EquityUSD = %v, want 65000（totalEq 精确折算）", okx.EquityUSD)
	}
	if len(okx.Details) != 2 {
		t.Fatalf("okx details = %d 项, want 2（USDT/BTC）", len(okx.Details))
	}
	for _, d := range okx.Details {
		if d.Asset == "USDT" && (d.Balance != "5000" || d.EquityUSD != 5000) {
			t.Errorf("okx USDT detail = (%s, %v), want (5000, 5000)", d.Balance, d.EquityUSD)
		}
		if d.Asset == "BTC" && d.EquityUSD != 60000 {
			t.Errorf("okx BTC EquityUSD = %v, want 60000（eqUsd）", d.EquityUSD)
		}
	}
}

// findAccount 按 source 找快照。
func findAccount(accts []store.TestnetAccount, source string) (store.TestnetAccount, bool) {
	for _, a := range accts {
		if a.Source == source {
			return a, true
		}
	}
	return store.TestnetAccount{}, false
}

// TestNewProbeDegradesNoKeys：无 key → (nil, false)（S3 降级禁用，不阻塞其他子任务）。
func TestNewProbeDegradesNoKeys(t *testing.T) {
	if p, ok := NewProbe(Config{}, newFakeRecorder()); ok || p != nil {
		t.Fatalf("NewProbe(empty) = (%v, %v), want (nil, false)", p, ok)
	}
}
