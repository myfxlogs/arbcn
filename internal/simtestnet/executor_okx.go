// OKX demo 下单执行（D-098 测试网执行层，executor.go 的 OKX 半边）。
// OKX demo 有现货 + 永续（funding_hedge 双腿都可放，完整 delta 中性）：
// instId 按腿映射（spot=BTC-USDT / perp=BTC-USDT-SWAP）；sz 为 base 数量
// （swap = 张数 = baseQty/合约面值）。签名 = body 参与 HMAC + x-simulated-trading:1。
package simtestnet

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
)

// placeOKX 市价单：POST /api/v5/trade/order（body 签名 + x-simulated-trading:1）→ 回读
// GET /api/v5/trade/order。
func (e *Executor) placeOKX(ctx context.Context, o ExecOrder) (ExecResult, error) {
	inst := okxInst(o.Symbol, o.Leg)
	side := "buy"
	if o.Side == "short" {
		side = "sell"
	}
	tdMode := "cash"
	sz := strconv.FormatFloat(floorQty(o.Symbol, o.Qty/o.RefPrice), 'f', -1, 64)
	if o.Leg == "perp" {
		tdMode = "cross"
		sz = strconv.FormatFloat(o.Qty/o.RefPrice/okxContractSize(o.Symbol), 'f', -1, 64)
	}
	payload, _ := json.Marshal(map[string]string{
		"instId": inst, "tdMode": tdMode, "side": side, "ordType": "market", "sz": sz,
	})
	ts := e.now().UTC().Format("2006-01-02T15:04:05.000Z")
	hdr := okxHeaders(e.cfg, ts, "POST", "/api/v5/trade/order", string(payload))

	body, code, err := do(ctx, e.client(), http.MethodPost, e.okxBase()+"/api/v5/trade/order", hdr, payload)
	if err != nil {
		return ExecResult{Status: ExecStatusError, Note: "okx order: " + err.Error()}, nil
	}
	if code != http.StatusOK {
		return ExecResult{Status: ExecStatusRejected,
			Note: fmt.Sprintf("okx order: HTTP %d: %s", code, truncateBody(body))}, nil
	}
	var r okxOrderResp
	if err := json.Unmarshal(body, &r); err != nil {
		return ExecResult{Status: ExecStatusError, Note: "okx order: parse: " + err.Error()}, nil
	}
	if r.Code != "0" {
		return ExecResult{Status: ExecStatusRejected, Note: fmt.Sprintf("okx order: code=%s msg=%s", r.Code, r.Msg)}, nil
	}
	if len(r.Data) == 0 {
		return ExecResult{Status: ExecStatusRejected, Note: "okx order: 空 data（拒单）"}, nil
	}
	d := r.Data[0]
	if d.SCode != "" && d.SCode != "0" {
		return ExecResult{Status: ExecStatusRejected, Note: fmt.Sprintf("okx order: sCode=%s sMsg=%s", d.SCode, d.SMsg)}, nil
	}
	res, err := e.getOKXOrder(ctx, o, d.OrdID)
	if err != nil {
		return ExecResult{Status: ExecStatusError, Note: "okx read-back: " + err.Error()}, nil
	}
	res.Symbol = inst
	return res, nil
}

// getOKXOrder 回读 GET /api/v5/trade/order（body 签名），映射状态。
func (e *Executor) getOKXOrder(ctx context.Context, o ExecOrder, orderID string) (ExecResult, error) {
	path := "/api/v5/trade/order"
	ts := e.now().UTC().Format("2006-01-02T15:04:05.000Z")
	hdr := okxHeaders(e.cfg, ts, "GET", path, "")
	u := e.okxBase() + path + "?" + url.Values{"instId": {okxInst(o.Symbol, o.Leg)}, "ordId": {orderID}}.Encode()
	body, code, err := do(ctx, e.client(), http.MethodGet, u, hdr, nil)
	if err != nil {
		return ExecResult{}, err
	}
	if code != http.StatusOK {
		return ExecResult{}, fmt.Errorf("okx get order: HTTP %d: %s", code, truncateBody(body))
	}
	var r okxOrderDetail
	if err := json.Unmarshal(body, &r); err != nil {
		return ExecResult{}, fmt.Errorf("okx get order: parse: %w", err)
	}
	if r.Code != "0" || len(r.Data) == 0 {
		return ExecResult{}, fmt.Errorf("okx get order: code=%s data=%d", r.Code, len(r.Data))
	}
	d := r.Data[0]
	res := ExecResult{
		Venue: VenueOKXDemo, ExchangeOrderID: orderID, Side: o.Side, Qty: o.Qty,
		Note: "okx " + d.State,
	}
	switch d.State {
	case "filled":
		res.Status = ExecStatusFilled
		res.FillQty, _ = strconv.ParseFloat(d.AccFillSz, 64)
		res.FillPrice, _ = strconv.ParseFloat(d.AvgPx, 64)
	case "partially_filled":
		res.Status = ExecStatusPartial
		res.FillQty, _ = strconv.ParseFloat(d.AccFillSz, 64)
	case "live", "":
		res.Status = ExecStatusPlaced
	default: // canceled
		res.Status = ExecStatusRejected
	}
	return res, nil
}

// cancelOKX 撤单 POST /api/v5/trade/cancel-order（body 签名）；成功 → 状态 canceled。
func (e *Executor) cancelOKX(ctx context.Context, o ExecOrder, orderID string) (ExecResult, error) {
	inst := okxInst(o.Symbol, o.Leg)
	payload, _ := json.Marshal(map[string]string{"instId": inst, "ordId": orderID})
	path := "/api/v5/trade/cancel-order"
	ts := e.now().UTC().Format("2006-01-02T15:04:05.000Z")
	hdr := okxHeaders(e.cfg, ts, "POST", path, string(payload))
	body, code, err := do(ctx, e.client(), http.MethodPost, e.okxBase()+path, hdr, payload)
	if err != nil {
		return ExecResult{Status: ExecStatusError, Note: "okx cancel: " + err.Error()}, nil
	}
	if code != http.StatusOK {
		return ExecResult{Status: ExecStatusRejected,
			Note: fmt.Sprintf("okx cancel: HTTP %d: %s", code, truncateBody(body))}, nil
	}
	var r okxOrderResp
	if err := json.Unmarshal(body, &r); err != nil {
		return ExecResult{Status: ExecStatusError, Note: "okx cancel: parse: " + err.Error()}, nil
	}
	if r.Code != "0" {
		return ExecResult{Status: ExecStatusRejected, Note: fmt.Sprintf("okx cancel: code=%s msg=%s", r.Code, r.Msg)}, nil
	}
	return ExecResult{Venue: VenueOKXDemo, ExchangeOrderID: orderID, Side: o.Side,
		Status: ExecStatusRejected, Note: "okx cancel: 已撤单（成交 0）"}, nil
}

// okxOrderResp / okxOrderDetail 是 OKX 下单/回读响应最小面。
type okxOrderResp struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		OrdID string `json:"ordId"`
		SCode string `json:"sCode"`
		SMsg  string `json:"sMsg"`
	} `json:"data"`
}

type okxOrderDetail struct {
	Code string `json:"code"`
	Data []struct {
		State     string `json:"state"`
		AvgPx     string `json:"avgPx"`
		AccFillSz string `json:"accFillSz"`
	} `json:"data"`
}

// okxHeaders 组装 OKX 签名头（ISO 8601 UTC 时间戳，probe.go:159 部署教训；demo 必须带
// x-simulated-trading:1）。与 probeOKX 同签名方式，POST body 参与 HMAC。
func okxHeaders(cfg Config, ts, method, path, body string) http.Header {
	return http.Header{
		"OK-ACCESS-KEY":        []string{cfg.OKXAPIKey},
		"OK-ACCESS-SIGN":       []string{okxSign(cfg.OKXSecret, ts, method, path, body)},
		"OK-ACCESS-TIMESTAMP":  []string{ts},
		"OK-ACCESS-PASSPHRASE": []string{cfg.OKXPassphrase},
		"x-simulated-trading":  []string{"1"},
	}
}

// okxInst OKX instrument：perp(swap) = BTC-USDT-SWAP；spot = BTC-USDT。
func okxInst(symbol, leg string) string {
	if leg == "perp" {
		return symbol + "-USDT-SWAP"
	}
	return symbol + "-USDT"
}

// okxContractSize OKX swap 每张合约的 base 数量（面值；BTC-USDT-SWAP = 0.01 BTC 等）。
func okxContractSize(symbol string) float64 {
	switch symbol {
	case "BTC":
		return 0.01
	case "ETH":
		return 0.1
	default:
		return 1
	}
}
