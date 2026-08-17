// 镜像下单执行器（D-098 测试网执行层）。simtestnet 由「只读探针」扩展为「可下单」：
// PlaceOrder 对 binance_testnet / okx_demo 真实下市价单 → 回读成交 → 返回统一结果，
// 供 simapi.ConfirmSimOrder 镜像落库（best-effort，不阻断本地模拟成交，D-037 本地仍是
// PnL 大脑）。SIMULATED 隔离不变（Load 强制 SIMULATED=true 才加载 key，config.go）；
// 仍禁主网域（domains_test 把关）。只做机制验证：数量精度/深度按冒烟尺度粗略（floor
// 常见步进），交易所拒单（精度/余额）如实记录 = 有效验证输出，精确换算留后续里程碑。
package simtestnet

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// 执行 venue（store.SimOrder.Venue 值域子集：测试网/demo 可执行）。
const (
	VenueBinanceTestnet = "binance_testnet"
	VenueOKXDemo        = "okx_demo"
)

// ExecStatus 镜像执行状态（sim_order_executions.status 值域）。
const (
	ExecStatusPlaced   = "placed"   // 已下单未成交（回读仍 NEW/live）
	ExecStatusFilled   = "filled"   // 已成交（回读成交价/量）
	ExecStatusPartial  = "partial"  // 部分成交
	ExecStatusRejected = "rejected" // 交易所拒单（note 记原因）
	ExecStatusError    = "error"    // 网络/本地错误（note 记 err；镜像 best-effort 不阻断本地）
)

// ExecOrder 单腿镜像下单请求（D-098）。由调用方（simapi.ConfirmSimOrder）在本地成交前
// 组装：每 open 腿一条。Venue 决定走哪家（binance_testnet / okx_demo）；Qty 为名义
// （quote 币种模拟 USD），执行层按 RefPrice 换算交易所 base 数量；Side 为本地腿方向。
type ExecOrder struct {
	OrderID  int64   // 本地 sim_order id（对账锚点；0 = 无）
	Venue    string  // binance_testnet / okx_demo
	Symbol   string  // 基础标的（BTC / ETH / TRX）
	Side     string  // long / short（本地腿方向）
	Kind     string  // 套利类型（funding_hedge 等；记录溯源）
	Leg      string  // 腿标识（spot / perp；空 = 默认按 spot 映射 instrument）
	Qty      float64 // 名义数量（quote 币种模拟 USD）
	RefPrice float64 // 参考价（名义→base 数量换算基准）
}

// ExecResult 单腿镜像下单结果（下单 + 回读成交）。
type ExecResult struct {
	Venue           string  // binance_testnet / okx_demo
	ExchangeOrderID string  // 交易所订单号（回读；失败 = ""）
	Symbol          string  // 交易所 instrument（如 BTCUSDT / BTC-USDT-SWAP）
	Side            string  // 本地腿方向（原样回传）
	Qty             float64 // 请求 base 数量（换算后）
	FillPrice       float64 // 回读成交均价（未成交 = 0）
	FillQty         float64 // 回读已成交数量（未成交 = 0）
	Status          string  // ExecStatus*
	Note            string  // 拒单/错误原因；成功可空
}

// Executor 镜像下单执行器（binance_testnet + okx_demo）。Base 可注入（测试 httptest）；
// 空 = 默认 testnet/demo。写操作只经 key 文件（SIMULATED=true，config.go Load 已强制）。
type Executor struct {
	BinanceBaseURL string
	OKXBaseURL     string
	Client         *http.Client
	Now            func() time.Time // 0 = time.Now

	cfg Config
}

// NewExecutor 构造执行器。cfg.Empty()（无任何 key）→ (nil, false)：main 降级禁用（镜像关）。
func NewExecutor(cfg Config) (*Executor, bool) {
	if cfg.Empty() {
		return nil, false
	}
	return &Executor{
		BinanceBaseURL: binanceTestnetBase,
		OKXBaseURL:     okxDemoBase,
		cfg:            cfg,
	}, true
}

func (e *Executor) client() *http.Client {
	if e.Client != nil {
		return e.Client
	}
	return &http.Client{Timeout: 10 * time.Second}
}

func (e *Executor) now() time.Time {
	if e.Now != nil {
		return e.Now()
	}
	return time.Now()
}

func (e *Executor) binanceBase() string {
	if e.BinanceBaseURL != "" {
		return e.BinanceBaseURL
	}
	return binanceTestnetBase
}

func (e *Executor) okxBase() string {
	if e.OKXBaseURL != "" {
		return e.OKXBaseURL
	}
	return okxDemoBase
}

// PlaceOrder 下单并回读成交（镜像）。网络错误/交易所拒单都收敛进 ExecResult.Status/Note
// （调用方可直接落库）；error 仅本地入参错误（Qty/RefPrice 非法），调用方 bug 级。
func (e *Executor) PlaceOrder(ctx context.Context, o ExecOrder) (ExecResult, error) {
	if o.Venue == "" || o.Symbol == "" {
		return ExecResult{}, errors.New("simtestnet: exec order: venue/symbol required")
	}
	if o.Qty <= 0 || o.RefPrice <= 0 {
		return ExecResult{}, errors.New("simtestnet: exec order: qty/ref_price must be > 0")
	}
	switch o.Venue {
	case VenueBinanceTestnet:
		return e.placeBinance(ctx, o)
	case VenueOKXDemo:
		return e.placeOKX(ctx, o)
	default:
		return ExecResult{}, fmt.Errorf("simtestnet: exec order: unsupported venue %q", o.Venue)
	}
}

// GetOrder 回读单笔订单成交状态（对账/后续平仓镜像用）。
func (e *Executor) GetOrder(ctx context.Context, o ExecOrder, exchangeOrderID string) (ExecResult, error) {
	switch o.Venue {
	case VenueBinanceTestnet:
		return e.getBinanceOrder(ctx, o, exchangeOrderID)
	case VenueOKXDemo:
		return e.getOKXOrder(ctx, o, exchangeOrderID)
	default:
		return ExecResult{}, fmt.Errorf("simtestnet: get order: unsupported venue %q", o.Venue)
	}
}

// CancelOrder 撤单（平仓镜像前置；best-effort，失败记 Note 不 panic）。
func (e *Executor) CancelOrder(ctx context.Context, o ExecOrder, exchangeOrderID string) (ExecResult, error) {
	switch o.Venue {
	case VenueBinanceTestnet:
		return e.cancelBinance(ctx, o, exchangeOrderID)
	case VenueOKXDemo:
		return e.cancelOKX(ctx, o, exchangeOrderID)
	default:
		return ExecResult{}, fmt.Errorf("simtestnet: cancel order: unsupported venue %q", o.Venue)
	}
}

// —— Binance USDT-M 期货（testnet，仅期货：funding_hedge 只放 perp 腿，spot 腿调用方跳过）——

// placeBinance 市价单：POST /fapi/v1/order（query 签名）→ 立即回读 GET /fapi/v1/order。
// 数量 = 名义/参考价换算 base qty（floor 常见步进；精度/余额不符 → 交易所 400 拒单如实记录）。
func (e *Executor) placeBinance(ctx context.Context, o ExecOrder) (ExecResult, error) {
	inst := o.Symbol + "USDT"
	side := "BUY"
	if o.Side == "short" {
		side = "SELL"
	}
	baseQty := floorQty(o.Symbol, o.Qty/o.RefPrice)
	ts := e.now().UnixMilli()
	q := url.Values{}
	q.Set("symbol", inst)
	q.Set("side", side)
	q.Set("type", "MARKET")
	q.Set("quantity", strconv.FormatFloat(baseQty, 'f', -1, 64))
	q.Set("timestamp", strconv.FormatInt(ts, 10))
	q.Set("recvWindow", "5000")
	q.Set("signature", binanceSign(e.cfg.BinanceSecret, q.Encode()))
	hdr := http.Header{"X-MBX-APIKEY": []string{e.cfg.BinanceAPIKey}}

	body, code, err := do(ctx, e.client(), http.MethodPost, e.binanceBase()+"/fapi/v1/order?"+q.Encode(), hdr, nil)
	if err != nil {
		return ExecResult{Status: ExecStatusError, Note: "binance order: " + err.Error()}, nil
	}
	if code != http.StatusOK {
		return ExecResult{Status: ExecStatusRejected,
			Note: fmt.Sprintf("binance order: HTTP %d: %s", code, truncateBody(body))}, nil
	}
	var r binanceOrderResp
	if err := json.Unmarshal(body, &r); err != nil {
		return ExecResult{Status: ExecStatusError, Note: "binance order: parse: " + err.Error()}, nil
	}
	if r.OrderID == 0 {
		return ExecResult{Status: ExecStatusRejected, Note: "binance order: 空 orderId（拒单）"}, nil
	}
	// 下单成功 → 回读权威状态（成交价/量/终态）。
	res, err := e.getBinanceOrder(ctx, o, strconv.FormatInt(r.OrderID, 10))
	if err != nil {
		return ExecResult{Status: ExecStatusError, Note: "binance read-back: " + err.Error()}, nil
	}
	res.Symbol = inst
	return res, nil
}

// getBinanceOrder 回读 GET /fapi/v1/order（query 签名），映射状态。
func (e *Executor) getBinanceOrder(ctx context.Context, o ExecOrder, orderID string) (ExecResult, error) {
	ts := e.now().UnixMilli()
	q := url.Values{}
	q.Set("symbol", o.Symbol+"USDT")
	q.Set("orderId", orderID)
	q.Set("timestamp", strconv.FormatInt(ts, 10))
	q.Set("recvWindow", "5000")
	q.Set("signature", binanceSign(e.cfg.BinanceSecret, q.Encode()))
	hdr := http.Header{"X-MBX-APIKEY": []string{e.cfg.BinanceAPIKey}}
	body, code, err := do(ctx, e.client(), http.MethodGet, e.binanceBase()+"/fapi/v1/order?"+q.Encode(), hdr, nil)
	if err != nil {
		return ExecResult{}, err
	}
	if code != http.StatusOK {
		return ExecResult{}, fmt.Errorf("binance get order: HTTP %d: %s", code, truncateBody(body))
	}
	var r binanceOrderResp
	if err := json.Unmarshal(body, &r); err != nil {
		return ExecResult{}, fmt.Errorf("binance get order: parse: %w", err)
	}
	return binanceResult(o, r), nil
}

// binanceResult 映射 /fapi/v1/order 响应 → ExecResult（状态 + 成交价/量）。
func binanceResult(o ExecOrder, r binanceOrderResp) ExecResult {
	res := ExecResult{
		Venue: VenueBinanceTestnet, ExchangeOrderID: strconv.FormatInt(r.OrderID, 10),
		Side: o.Side, Qty: o.Qty, Note: "binance " + r.Status,
	}
	switch r.Status {
	case "FILLED":
		res.Status = ExecStatusFilled
		res.FillQty, _ = strconv.ParseFloat(r.ExecutedQty, 64)
		res.FillPrice, _ = strconv.ParseFloat(r.AvgPrice, 64)
	case "PARTIALLY_FILLED":
		res.Status = ExecStatusPartial
		res.FillQty, _ = strconv.ParseFloat(r.ExecutedQty, 64)
	case "NEW", "":
		res.Status = ExecStatusPlaced
	default: // CANCELED / REJECTED / EXPIRED
		res.Status = ExecStatusRejected
	}
	return res
}

// binanceOrderResp 是 /fapi/v1/order 响应最小面（orderId 数字、金额为字符串）。
type binanceOrderResp struct {
	OrderID     int64  `json:"orderId"`
	Status      string `json:"status"`
	ExecutedQty string `json:"executedQty"`
	AvgPrice    string `json:"avgPrice"`
}

// cancelBinance 撤单 DELETE /fapi/v1/order（query 签名）；成功 → 状态 canceled。
func (e *Executor) cancelBinance(ctx context.Context, o ExecOrder, orderID string) (ExecResult, error) {
	ts := e.now().UnixMilli()
	q := url.Values{}
	q.Set("symbol", o.Symbol+"USDT")
	q.Set("orderId", orderID)
	q.Set("timestamp", strconv.FormatInt(ts, 10))
	q.Set("recvWindow", "5000")
	q.Set("signature", binanceSign(e.cfg.BinanceSecret, q.Encode()))
	hdr := http.Header{"X-MBX-APIKEY": []string{e.cfg.BinanceAPIKey}}
	body, code, err := do(ctx, e.client(), http.MethodDelete, e.binanceBase()+"/fapi/v1/order?"+q.Encode(), hdr, nil)
	if err != nil {
		return ExecResult{Status: ExecStatusError, Note: "binance cancel: " + err.Error()}, nil
	}
	if code != http.StatusOK {
		return ExecResult{Status: ExecStatusRejected,
			Note: fmt.Sprintf("binance cancel: HTTP %d: %s", code, truncateBody(body))}, nil
	}
	return ExecResult{Venue: VenueBinanceTestnet, ExchangeOrderID: orderID, Side: o.Side,
		Status: ExecStatusRejected, Note: "binance cancel: 已撤单（成交 0）"}, nil
}

// —— OKX demo（现货 + 永续）：placeOKX / getOKXOrder / cancelOKX / okx* 见 executor_okx.go。

// floorQty 名义→base 数量换算后 floor 到常见交易步进（冒烟尺度；精度不符 → 交易所拒单
// 如实记录，不静默改单）。后续里程碑按交易所 symbol 精度规格精确换算。
func floorQty(symbol string, baseQty float64) float64 {
	step := 0.001
	switch symbol {
	case "ETH":
		step = 0.01
	case "TRX":
		step = 1
	}
	if math.IsNaN(baseQty) || math.IsInf(baseQty, 0) || baseQty < step {
		return step
	}
	return math.Floor(baseQty/step) * step
}

// do 通用 HTTP 请求（GET/POST/DELETE，JSON body 可选）。返回响应体（≤1MB）+ 状态码；
// 非 2xx 不在此报错——Binance 拒单 400 带错误体、OKX 200 带 code，状态码由调用方按所
// 在交易所语义解释。
func do(ctx context.Context, client *http.Client, method, url string, hdr http.Header, body []byte) ([]byte, int, error) {
	var rd io.Reader
	if body != nil {
		rd = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, rd)
	if err != nil {
		return nil, 0, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, vs := range hdr {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, 0, err
	}
	return b, resp.StatusCode, nil
}

// truncateBody 截断响应体（≤120 字符）供 Note 记录（防巨额错误体撑爆 note）。
func truncateBody(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 120 {
		s = s[:120] + "…"
	}
	return s
}
