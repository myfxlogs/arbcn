// 只读探针（04-m3-spec §9.4 S3，零下单路径）。
// 对 binance_testnet / okx_demo 做公共行情 + 账户只读查询，验证 key 连通：
// 成功 → alert.Heartbeat.Record("sim_testnet_binance"/"sim_testnet_okx", now)。
// 失败仅返回错误（settle 循环 warn 不退出，D-032 同口径）。无任何下单/挂单端点。
package simtestnet

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

// 探针默认域（binance USDT-M futures testnet / OKX demo 同 host + x-simulated-trading）。
const (
	binanceTestnetBase = "https://testnet.binancefuture.com"
	okxDemoBase        = "https://www.okx.com"
)

// 探针源名（Heartbeat Record 键；出现在 ListSourceHealth）。
const (
	SourceBinanceTestnet = "sim_testnet_binance"
	SourceOKXDemo        = "sim_testnet_okx"
)

// RecordRecorder 是探针成功后的登记口（alert.Heartbeat.Record 满足；测试可注入 fake）。
type RecordRecorder interface {
	Record(name string, at time.Time)
}

// Probe 是 testnet 只读连通性探针。Base 可注入（测试 httptest）；空 = 默认 testnet/demo。
type Probe struct {
	BinanceBaseURL string // 默认 https://testnet.binancefuture.com
	OKXBaseURL     string // 默认 https://www.okx.com
	Client         *http.Client
	HB             RecordRecorder // 非 nil：成功时 Record
	Now            func() time.Time // 0 = time.Now

	cfg Config
}

// NewProbe 构造探针。cfg.Empty()（无任何 key）→ (nil, false)：main 降级禁用探针。
func NewProbe(cfg Config, hb RecordRecorder) (*Probe, bool) {
	if cfg.Empty() {
		return nil, false
	}
	return &Probe{
		BinanceBaseURL: binanceTestnetBase,
		OKXBaseURL:     okxDemoBase,
		HB:             hb,
		cfg:            cfg,
	}, true
}

func (p *Probe) client() *http.Client {
	if p.Client != nil {
		return p.Client
	}
	return &http.Client{Timeout: 10 * time.Second}
}

func (p *Probe) now() time.Time {
	if p.Now != nil {
		return p.Now()
	}
	return time.Now()
}

func (p *Probe) binanceBase() string {
	if p.BinanceBaseURL != "" {
		return p.BinanceBaseURL
	}
	return binanceTestnetBase
}

func (p *Probe) okxBase() string {
	if p.OKXBaseURL != "" {
		return p.OKXBaseURL
	}
	return okxDemoBase
}

// Run 执行 binance + okx 两路只读探针；每路成功 → hb.Record。一路失败不影响另一路
// （各自独立判断）；全部失败返回聚合错误。零下单端点。
func (p *Probe) Run(ctx context.Context) error {
	var errs []error
	if p.cfg.BinanceAPIKey != "" {
		if err := p.probeBinance(ctx); err != nil {
			errs = append(errs, fmt.Errorf("simtestnet binance: %w", err))
		}
	}
	if p.cfg.OKXAPIKey != "" {
		if err := p.probeOKX(ctx); err != nil {
			errs = append(errs, fmt.Errorf("simtestnet okx: %w", err))
		}
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// probeBinance：公共行情 /fapi/v1/time + 账户只读 /fapi/v2/balance（HMAC-SHA256 签名）。
func (p *Probe) probeBinance(ctx context.Context) error {
	client := p.client()
	base := p.binanceBase()

	// 公共行情（无签名）。
	if _, err := p.get(ctx, client, base+"/fapi/v1/time", nil); err != nil {
		return fmt.Errorf("public time: %w", err)
	}
	// 账户只读（签名；仅查余额，不建单）。TS + recvWindow 防重放。
	ts := p.now().UnixMilli()
	q := url.Values{}
	q.Set("timestamp", strconv.FormatInt(ts, 10))
	q.Set("recvWindow", "5000")
	sig := binanceSign(p.cfg.BinanceSecret, q.Encode())
	q.Set("signature", sig)
	hdr := http.Header{"X-MBX-APIKEY": []string{p.cfg.BinanceAPIKey}}
	if _, err := p.get(ctx, client, base+"/fapi/v2/balance?"+q.Encode(), hdr); err != nil {
		return fmt.Errorf("account balance: %w", err)
	}
	if p.HB != nil {
		p.HB.Record(SourceBinanceTestnet, p.now())
	}
	return nil
}

// probeOKX：公共行情 /api/v5/public/time + 账户只读 /api/v5/account/balance
// （x-simulated-trading:1 + OK 签名头）。
func (p *Probe) probeOKX(ctx context.Context) error {
	client := p.client()
	base := p.okxBase()

	// 公共行情（无签名）。
	if _, err := p.get(ctx, client, base+"/api/v5/public/time", nil); err != nil {
		return fmt.Errorf("public time: %w", err)
	}
	// 账户只读（签名；仅查余额，零下单）。模拟盘由 x-simulated-trading:1 头声明。
	ts := strconv.FormatInt(p.now().UnixMilli(), 10)
	sign := okxSign(p.cfg.OKXSecret, ts, "GET", "/api/v5/account/balance", "")
	hdr := http.Header{
		"OK-ACCESS-KEY":        []string{p.cfg.OKXAPIKey},
		"OK-ACCESS-SIGN":       []string{sign},
		"OK-ACCESS-TIMESTAMP":  []string{ts},
		"OK-ACCESS-PASSPHRASE": []string{p.cfg.OKXPassphrase},
		"x-simulated-trading":  []string{"1"},
	}
	body, err := p.get(ctx, client, base+"/api/v5/account/balance", hdr)
	if err != nil {
		return fmt.Errorf("account balance: %w", err)
	}
	if err := parseOKXResp(body); err != nil {
		return fmt.Errorf("account balance: %w", err)
	}
	if p.HB != nil {
		p.HB.Record(SourceOKXDemo, p.now())
	}
	return nil
}

// get 发 GET 并返回响应体（≤1MB）。只验证 HTTP 200 可达；body 供调用方按需解析。
func (p *Probe) get(ctx context.Context, client *http.Client, url string, hdr http.Header) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	for k, vs := range hdr {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}
	return body, nil
}

// binanceSign HMAC-SHA256（Binance 签名 = hex(hmac_sha256(secret, queryString))）。
func binanceSign(secret, query string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(query))
	return hex.EncodeToString(mac.Sum(nil))
}

// okxSign OKX 签名 = base64(hmac_sha256(secret, ts+method+path+body))。
func okxSign(secret, ts, method, path, body string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + method + path + body))
	return base64.StdEncoding.EncodeToString(mac.Sum(nil))
}

// okxResp 解析 OKX 业务码（HTTP 200 但业务失败也要识别）。
type okxResp struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
}

// parseOKXResp 校验 OKX 响应业务码（非 0 → 错误）。
func parseOKXResp(body []byte) error {
	var r okxResp
	if err := json.Unmarshal(body, &r); err != nil {
		return nil // 非 JSON（公共 time 端点返回 JSON 文本 "xxx"？）→ 视为连通
	}
	if r.Code != "" && r.Code != "0" {
		return fmt.Errorf("okx code=%s msg=%s", r.Code, r.Msg)
	}
	return nil
}
