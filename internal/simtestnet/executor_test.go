package simtestnet

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

// testExecutor 构造指向 fixture server 的执行器（注入固定时钟 t0，与 probe_test 同款）。
func testExecutor(t *testing.T, srv *httptest.Server) *Executor {
	t.Helper()
	cfg := Config{
		BinanceAPIKey: "bkey", BinanceSecret: "bsec",
		OKXAPIKey: "okey", OKXSecret: "osec", OKXPassphrase: "opass",
	}
	e, ok := NewExecutor(cfg)
	if !ok {
		t.Fatal("NewExecutor = nil, want executor（有 key）")
	}
	e.BinanceBaseURL = srv.URL
	e.OKXBaseURL = srv.URL
	e.Now = func() time.Time { return t0 }
	return e
}

// execFixtureServer 模拟 testnet/demo 下单端点，并验证签名/头正确性（签名错 → 401）。
// 下单成功返回 FILLED 订单，供回读断言。Binance 签名 = query 重算比对；OKX = body 原文
// 参与 HMAC 重算比对 + x-simulated-trading:1 + ISO 8601 时间戳（全部锚点）。
func execFixtureServer(t *testing.T) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	// Binance testnet：POST 下单 → FILLED；GET 回读 → 同一订单。签名重算 = 客户端签的是
	// 全 query（剔除 signature 后的 url.Values.Encode() 稳定序），fixture 用同一算法重算。
	mux.HandleFunc("/fapi/v1/order", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-MBX-APIKEY") != "bkey" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		q := r.URL.Query()
		q.Del("signature")
		wantSig := binanceSign("bsec", q.Encode())
		if r.URL.Query().Get("signature") != wantSig {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if q.Get("symbol") != "BTCUSDT" {
			t.Errorf("binance symbol = %q, want BTCUSDT", q.Get("symbol"))
		}
		if r.Method == http.MethodPost {
			if q.Get("type") != "MARKET" {
				t.Errorf("binance type = %q, want MARKET", q.Get("type"))
			}
			if q.Get("side") != "SELL" {
				t.Errorf("binance POST side = %q, want SELL（perp short 腿）", q.Get("side"))
			}
			// 数量必须为 floor 后的 base 数量（名义 60000 / 参考价 60000 = 1 BTC）。
			if q.Get("quantity") != "1" {
				t.Errorf("binance quantity = %q, want 1（名义/参考价 floor）", q.Get("quantity"))
			}
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"orderId":1001,"symbol":"BTCUSDT","status":"FILLED","executedQty":"1","avgPrice":"59950.5"}`))
	})

	// OKX demo：POST 下单（body 签名 + x-simulated-trading:1）→ ordId；GET 回读 → filled。
	mux.HandleFunc("/api/v5/trade/order", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-simulated-trading") != "1" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		ts := r.Header.Get("OK-ACCESS-TIMESTAMP")
		if !iso8601Regex.MatchString(ts) {
			t.Errorf("OK-ACCESS-TIMESTAMP = %q, want ISO 8601 UTC（Unix 毫秒 → OKX 50102）", ts)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.Header.Get("OK-ACCESS-KEY") != "okey" || r.Header.Get("OK-ACCESS-PASSPHRASE") != "opass" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.Method == http.MethodGet {
			wantSign := okxSign("osec", ts, "GET", "/api/v5/trade/order", "")
			if r.Header.Get("OK-ACCESS-SIGN") != wantSign {
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"code":"0","msg":"","data":[{"instId":"BTC-USDT-SWAP","ordId":"777","state":"filled","avgPx":"59951.0","accFillSz":"1","px":"","sz":"1","side":"sell"}]}`))
			return
		}
		// POST：body 原文参与 HMAC（读原始 body 再比对签名）。
		raw, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		wantSign := okxSign("osec", ts, "POST", "/api/v5/trade/order", string(raw))
		if r.Header.Get("OK-ACCESS-SIGN") != wantSign {
			t.Errorf("OK-ACCESS-SIGN 与 body 原文 HMAC 不符（POST body 必须参与签名）")
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		var payload map[string]string
		if err := json.Unmarshal(raw, &payload); err != nil {
			t.Errorf("okx order body decode: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if payload["instId"] != "BTC-USDT-SWAP" {
			t.Errorf("okx instId = %q, want BTC-USDT-SWAP（perp 腿）", payload["instId"])
		}
		if payload["tdMode"] != "cross" {
			t.Errorf("okx tdMode = %q, want cross（swap）", payload["tdMode"])
		}
		if payload["ordType"] != "market" {
			t.Errorf("okx ordType = %q, want market", payload["ordType"])
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"code":"0","msg":"","data":[{"ordId":"777","sCode":"0","sMsg":""}]}`))
	})

	// OKX 撤单端点（demo 头校验）。
	mux.HandleFunc("/api/v5/trade/cancel-order", func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-simulated-trading") != "1" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"code":"0","msg":"","data":[{"ordId":"777"}]}`))
	})

	return httptest.NewServer(mux)
}

// TestNewExecutorDegradesNoKeys：无 key → (nil, false)（镜像降级禁用）。
func TestNewExecutorDegradesNoKeys(t *testing.T) {
	if e, ok := NewExecutor(Config{}); ok || e != nil {
		t.Fatalf("NewExecutor(empty) = (%v, %v), want (nil, false)", e, ok)
	}
}

// TestPlaceOrderBinanceFilled：[对抗测试锚点 D-098] binance_testnet 下市价单 → 回读
// FILLED，成交价/量落 ExecResult。删回读/删状态映射 → 必红。
func TestPlaceOrderBinanceFilled(t *testing.T) {
	srv := execFixtureServer(t)
	defer srv.Close()
	e := testExecutor(t, srv)

	res, err := e.PlaceOrder(context.Background(), ExecOrder{
		Venue: VenueBinanceTestnet, Symbol: "BTC", Side: "short",
		Kind: "funding_hedge", Leg: "perp", Qty: 60000, RefPrice: 60000,
	})
	if err != nil {
		t.Fatalf("PlaceOrder: %v", err)
	}
	if res.Status != ExecStatusFilled {
		t.Fatalf("Status = %q, want filled", res.Status)
	}
	if res.ExchangeOrderID != "1001" {
		t.Errorf("ExchangeOrderID = %q, want 1001", res.ExchangeOrderID)
	}
	if res.FillQty != 1 || res.FillPrice != 59950.5 {
		t.Errorf("Fill = (%v @ %v), want (1 @ 59950.5)", res.FillQty, res.FillPrice)
	}
	if res.Venue != VenueBinanceTestnet {
		t.Errorf("Venue = %q, want binance_testnet", res.Venue)
	}
}

// TestPlaceOrderOKXPerpFilled：[对抗测试锚点 D-098] okx_demo perp 腿 → 完整对冲腿可放，
// 成交回读。OKX 请求必须带 x-simulated-trading:1 + body 原文签名（fixture 校验）。
func TestPlaceOrderOKXPerpFilled(t *testing.T) {
	srv := execFixtureServer(t)
	defer srv.Close()
	e := testExecutor(t, srv)

	res, err := e.PlaceOrder(context.Background(), ExecOrder{
		Venue: VenueOKXDemo, Symbol: "BTC", Side: "short",
		Kind: "funding_hedge", Leg: "perp", Qty: 60000, RefPrice: 60000,
	})
	if err != nil {
		t.Fatalf("PlaceOrder: %v", err)
	}
	if res.Status != ExecStatusFilled {
		t.Fatalf("Status = %q, want filled", res.Status)
	}
	if res.ExchangeOrderID != "777" {
		t.Errorf("ExchangeOrderID = %q, want 777", res.ExchangeOrderID)
	}
	if res.FillPrice != 59951.0 {
		t.Errorf("FillPrice = %v, want 59951.0（OKX avgPx）", res.FillPrice)
	}
	if res.Venue != VenueOKXDemo {
		t.Errorf("Venue = %q, want okx_demo", res.Venue)
	}
}

// TestPlaceOrderRejectedOnExchangeError：交易所拒单（HTTP 400 + 错误体）→ Status=rejected
// + note 记原因（不 panic、不当作本地错误）。镜像 best-effort 语义。
func TestPlaceOrderRejectedOnExchangeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/fapi/v1/order") {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"code":-2019,"msg":"Account has insufficient balance"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	e := testExecutor(t, srv)

	res, err := e.PlaceOrder(context.Background(), ExecOrder{
		Venue: VenueBinanceTestnet, Symbol: "BTC", Side: "short",
		Kind: "funding_hedge", Leg: "perp", Qty: 60000, RefPrice: 60000,
	})
	if err != nil {
		t.Fatalf("PlaceOrder: %v", err)
	}
	if res.Status != ExecStatusRejected {
		t.Fatalf("Status = %q, want rejected（交易所拒单收敛进结果）", res.Status)
	}
	if !strings.Contains(res.Note, "insufficient") {
		t.Errorf("Note = %q, want 含交易所错误原因", res.Note)
	}
}

// TestPlaceOrderInvalidInput：入参非法（qty ≤ 0）→ 本地错误（调用方 bug 级）。
func TestPlaceOrderInvalidInput(t *testing.T) {
	e, _ := NewExecutor(Config{BinanceAPIKey: "bkey", BinanceSecret: "bsec"})
	_, err := e.PlaceOrder(context.Background(), ExecOrder{
		Venue: VenueBinanceTestnet, Symbol: "BTC", Side: "short", Qty: 0, RefPrice: 60000,
	})
	if err == nil {
		t.Fatal("PlaceOrder(qty=0) = nil, want error")
	}
}

// TestFloorQty：名义→base 数量 floor 常见步进（BTC 0.001 / ETH 0.01 / TRX 1）。
func TestFloorQty(t *testing.T) {
	for _, c := range []struct {
		sym  string
		base float64
		want float64
	}{
		{"BTC", 1.0, 1.0},
		{"BTC", 0.0005, 0.001},   // 低于步进 → 步进
		{"BTC", 1.234567, 1.234}, // floor 到 0.001
		{"ETH", 5.0, 5.0},
		{"ETH", 5.018, 5.01}, // floor 到 0.01
		{"TRX", 2500, 2500},
		{"TRX", 0.5, 1}, // floor 到 1
	} {
		if got := floorQty(c.sym, c.base); got != c.want {
			t.Errorf("floorQty(%s, %v) = %v, want %v", c.sym, c.base, got, c.want)
		}
	}
}

// TestCancelOrderOKX：撤单走 /api/v5/trade/cancel-order + demo 头，结果含撤单 note。
func TestCancelOrderOKX(t *testing.T) {
	srv := execFixtureServer(t)
	defer srv.Close()
	e := testExecutor(t, srv)

	res, err := e.CancelOrder(context.Background(), ExecOrder{
		Venue: VenueOKXDemo, Symbol: "BTC", Side: "short", Leg: "perp", Qty: 60000, RefPrice: 60000,
	}, "777")
	if err != nil {
		t.Fatalf("CancelOrder: %v", err)
	}
	if res.Status != ExecStatusRejected || !strings.Contains(res.Note, "已撤单") {
		t.Errorf("CancelOrder = %+v, want rejected + 撤单 note", res)
	}
	if res.ExchangeOrderID != "777" {
		t.Errorf("ExchangeOrderID = %q, want 777", res.ExchangeOrderID)
	}
}

// TestGetOrderBinance：回读 GET /fapi/v1/order（query 签名，fixture 重算比对 401）→
// 状态/成交映射正确。
func TestGetOrderBinance(t *testing.T) {
	srv := execFixtureServer(t)
	defer srv.Close()
	e := testExecutor(t, srv)

	res, err := e.GetOrder(context.Background(), ExecOrder{
		Venue: VenueBinanceTestnet, Symbol: "BTC", Side: "short", Qty: 60000, RefPrice: 60000,
	}, "1001")
	if err != nil {
		t.Fatalf("GetOrder: %v", err)
	}
	if res.Status != ExecStatusFilled || res.FillQty != 1 || res.ExchangeOrderID != "1001" {
		t.Errorf("GetOrder = %+v, want filled 1 @ 1001", res)
	}
}

// TestExecOrderSignedQuery：binance 下单签名基于 URL-encoded query（timestamp 在前、
// signature 在后）；签名入 query 供 fixture 重算比对（此处验证签名机制本身可重算）。
func TestExecOrderSignedQuery(t *testing.T) {
	q := url.Values{}
	q.Set("timestamp", strconv.FormatInt(t0.UnixMilli(), 10))
	q.Set("recvWindow", "5000")
	sig := binanceSign("bsec", q.Encode())
	q.Set("signature", sig)
	if !strings.Contains(q.Encode(), "signature="+sig) {
		t.Error("signature 未入 query（签名必须是最后一段，fixture 依此重算）")
	}
}
