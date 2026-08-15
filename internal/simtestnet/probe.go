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

	"arbcn/internal/store"
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
	HB             RecordRecorder   // 非 nil：成功时 Record
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

// Run 执行 binance + okx 两路只读探针；每路成功 → hb.Record + 返回该路余额快照
// （D-040 测试网账户区数据面）。一路失败不影响另一路（各自独立判断）；返回成功路
// 快照 + 失败路聚合错误（调用方按快照持久化、按错误 warn，D-032 同口径）。零下单端点。
func (p *Probe) Run(ctx context.Context) ([]store.TestnetAccount, error) {
	var out []store.TestnetAccount
	var errs []error
	if p.cfg.BinanceAPIKey != "" {
		if a, err := p.probeBinance(ctx); err != nil {
			errs = append(errs, fmt.Errorf("simtestnet binance: %w", err))
		} else {
			out = append(out, a)
		}
	}
	if p.cfg.OKXAPIKey != "" {
		if a, err := p.probeOKX(ctx); err != nil {
			errs = append(errs, fmt.Errorf("simtestnet okx: %w", err))
		} else {
			out = append(out, a)
		}
	}
	return out, errors.Join(errs...)
}

// probeBinance：公共行情 /fapi/v1/time + 账户只读 /fapi/v2/balance（HMAC-SHA256 签名）；
// 成功 → 解析余额快照 + hb.Record。
func (p *Probe) probeBinance(ctx context.Context) (store.TestnetAccount, error) {
	client := p.client()
	base := p.binanceBase()

	// 公共行情（无签名）。
	if _, err := p.get(ctx, client, base+"/fapi/v1/time", nil); err != nil {
		return store.TestnetAccount{}, fmt.Errorf("public time: %w", err)
	}
	// 账户只读（签名；仅查余额，不建单）。TS + recvWindow 防重放。
	ts := p.now().UnixMilli()
	q := url.Values{}
	q.Set("timestamp", strconv.FormatInt(ts, 10))
	q.Set("recvWindow", "5000")
	sig := binanceSign(p.cfg.BinanceSecret, q.Encode())
	q.Set("signature", sig)
	hdr := http.Header{"X-MBX-APIKEY": []string{p.cfg.BinanceAPIKey}}
	body, err := p.get(ctx, client, base+"/fapi/v2/balance?"+q.Encode(), hdr)
	if err != nil {
		return store.TestnetAccount{}, fmt.Errorf("account balance: %w", err)
	}
	a, err := parseBinanceBalance(body)
	if err != nil {
		return store.TestnetAccount{}, err
	}
	if p.HB != nil {
		p.HB.Record(SourceBinanceTestnet, p.now())
	}
	return a, nil
}

// probeOKX：公共行情 /api/v5/public/time + 账户只读 /api/v5/account/balance
// （x-simulated-trading:1 + OK 签名头）；成功 → 解析余额快照 + hb.Record。
func (p *Probe) probeOKX(ctx context.Context) (store.TestnetAccount, error) {
	client := p.client()
	base := p.okxBase()

	// 公共行情（无签名）。
	if _, err := p.get(ctx, client, base+"/api/v5/public/time", nil); err != nil {
		return store.TestnetAccount{}, fmt.Errorf("public time: %w", err)
	}
	// 账户只读（签名；仅查余额，零下单）。模拟盘由 x-simulated-trading:1 头声明。
	// OKX 要求 OK-ACCESS-TIMESTAMP 为 ISO 8601 UTC（"2026-08-15T15:54:21.991Z"）——
	// Unix 毫秒会被拒（50102 Timestamp request expired，2026-08-15 部署机实测，probe_test 锚点）。
	ts := p.now().UTC().Format("2006-01-02T15:04:05.000Z")
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
		return store.TestnetAccount{}, fmt.Errorf("account balance: %w", err)
	}
	a, err := parseOKXBalance(body)
	if err != nil {
		return store.TestnetAccount{}, fmt.Errorf("account balance: %w", err)
	}
	if p.HB != nil {
		p.HB.Record(SourceOKXDemo, p.now())
	}
	return a, nil
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

// —— D-040 余额快照解析（binance /fapi/v2/balance、OKX /api/v5/account/balance）——
// 只解析展示所需最小面；余额/金额保留 API 原字符串（strconv 仅用于折算计算）。

// binanceBalanceRow 是 /fapi/v2/balance 单元素最小面（accountAlias 每行同值）。
type binanceBalanceRow struct {
	AccountAlias string `json:"accountAlias"`
	Asset        string `json:"asset"`
	Balance      string `json:"balance"`
}

// binanceStable 是 binance 余额中可按 1:1 折 USD 的稳定币（无行情折算，只做稳定币近似；
// 非稳定币 EquityUSD=0，前端标 —，诚实标注"非全量净值"）。
var binanceStable = map[string]bool{"USDT": true, "USDC": true, "BUSD": true, "FDUSD": true}

// parseBinanceBalance 解析 binance 余额为账户快照。equity_usd = 稳定币余额合计（近似）。
func parseBinanceBalance(body []byte) (store.TestnetAccount, error) {
	var rows []binanceBalanceRow
	if err := json.Unmarshal(body, &rows); err != nil {
		return store.TestnetAccount{}, fmt.Errorf("parse binance balance: %w", err)
	}
	a := store.TestnetAccount{Source: SourceBinanceTestnet}
	if len(rows) > 0 {
		a.AccountAlias = rows[0].AccountAlias
	}
	for _, r := range rows {
		if r.Asset == "" {
			continue
		}
		d := store.TestnetAccountDetail{Asset: r.Asset, Balance: r.Balance}
		if binanceStable[r.Asset] {
			if v, err := strconv.ParseFloat(r.Balance, 64); err == nil {
				d.EquityUSD = v
				a.EquityUSD += v
			}
		}
		a.Details = append(a.Details, d)
	}
	return a, nil
}

// okxBalance 是 /api/v5/account/balance 响应最小面。totalEq = 总权益 USD（交易所精确折算）；
// details = 每币种 eq（原值）与 eqUsd。HTTP 200 但业务失败也要识别（code != 0）。
type okxBalance struct {
	Code string `json:"code"`
	Msg  string `json:"msg"`
	Data []struct {
		TotalEq string `json:"totalEq"`
		Details []struct {
			Ccy   string `json:"ccy"`
			Eq    string `json:"eq"`
			EqUsd string `json:"eqUsd"`
		} `json:"details"`
	} `json:"data"`
}

// parseOKXBalance 解析 OKX 余额为账户快照。equity_usd = totalEq（精确）；空 data = 空快照
// （连通有效，账户无持仓不报错）。
func parseOKXBalance(body []byte) (store.TestnetAccount, error) {
	var r okxBalance
	if err := json.Unmarshal(body, &r); err != nil {
		return store.TestnetAccount{}, fmt.Errorf("parse okx balance: %w", err)
	}
	if r.Code != "" && r.Code != "0" {
		return store.TestnetAccount{}, fmt.Errorf("okx code=%s msg=%s", r.Code, r.Msg)
	}
	a := store.TestnetAccount{Source: SourceOKXDemo}
	if len(r.Data) == 0 {
		return a, nil
	}
	if v, err := strconv.ParseFloat(r.Data[0].TotalEq, 64); err == nil {
		a.EquityUSD = v
	}
	for _, d := range r.Data[0].Details {
		if d.Ccy == "" {
			continue
		}
		eqUSD, _ := strconv.ParseFloat(d.EqUsd, 64)
		a.Details = append(a.Details, store.TestnetAccountDetail{
			Asset: d.Ccy, Balance: d.Eq, EquityUSD: eqUSD,
		})
	}
	return a, nil
}
