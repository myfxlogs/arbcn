package exchange

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"time"

	"arbcn/internal/fact"
)

// OKX v5 公开端点：
//   - /api/v5/public/funding-rate?instId=BTC-USDT-SWAP → 当前费率 + 结算时间戳对
//     （结算间隔 = nextFundingTime − fundingTime，8h 为主、部分 4h）
//   - /api/v5/market/tickers?instType=SWAP → 全部永续最新价（本地过滤目标币种）
const (
	okxFundingRatePath = "/api/v5/public/funding-rate"
	okxTickersPath     = "/api/v5/market/tickers"
)

// okxInst 基础币 → 永续合约 instrument（BTC → BTC-USDT-SWAP）。
func okxInst(base string) string { return base + "-USDT-SWAP" }

type okxResp struct {
	Code string          `json:"code"`
	Msg  string          `json:"msg"`
	Data json.RawMessage `json:"data"`
}

// check 校验 OKX 业务码；非 "0" 返回错误（HTTP 200 但业务失败）。
func (r okxResp) check() error {
	if r.Code != "0" {
		return fmt.Errorf("okx: code=%s msg=%s", r.Code, r.Msg)
	}
	return nil
}

type okxFundingRate struct {
	InstID          string `json:"instId"`
	FundingRate     string `json:"fundingRate"`
	FundingTime     string `json:"fundingTime"`
	NextFundingTime string `json:"nextFundingTime"`
}

type okxTicker struct {
	InstID string `json:"instId"`
	Last   string `json:"last"`
	TS     string `json:"ts"`
}

// OKXFunding 采集 BTC/ETH/TRX 等永续资金费率（Kind=funding）。
type OKXFunding struct{ cfg Config }

// NewOKXFunding 构造 OKX funding collector。
func NewOKXFunding(cfg Config) *OKXFunding { return &OKXFunding{cfg: cfg} }

// Kind 实现 collect.Collector。
func (*OKXFunding) Kind() string { return fact.KindFunding }

// Poll 逐币种拉取 funding-rate（端点为单 instId 查询），按结算时间戳对折算年化；
// 任一币种缺失/异常 → 整个 Poll 失败（不静默降级）。
func (c *OKXFunding) Poll(ctx context.Context) ([]fact.Fact, error) {
	client := c.cfg.client()
	base := c.cfg.OKXBaseURL
	var out []fact.Fact
	for _, sym := range c.cfg.Symbols {
		inst := okxInst(sym)
		var resp okxResp
		if err := getJSON(ctx, client, base+okxFundingRatePath+"?instId="+inst, &resp); err != nil {
			return nil, err
		}
		if err := resp.check(); err != nil {
			return nil, fmt.Errorf("okx funding %s: %w", inst, err)
		}
		var rows []okxFundingRate
		if err := json.Unmarshal(resp.Data, &rows); err != nil {
			return nil, fmt.Errorf("okx funding %s: decode data: %w", inst, err)
		}
		if len(rows) == 0 {
			return nil, fmt.Errorf("okx funding %s: empty data", inst)
		}
		r := rows[0]
		rate, err := strconv.ParseFloat(r.FundingRate, 64)
		if err != nil {
			return nil, fmt.Errorf("okx funding %s: bad rate %q", inst, r.FundingRate)
		}
		hours, err := okxIntervalHours(r)
		if err != nil {
			return nil, fmt.Errorf("okx funding %s: %w", inst, err)
		}
		v, err := annualize(rate, hours)
		if err != nil {
			return nil, fmt.Errorf("okx funding %s: %w", inst, err)
		}
		out = append(out, fact.Fact{
			Kind:   fact.KindFunding,
			Venue:  VenueOKX,
			Symbol: sym,
			Value:  v,
			Unit:   fact.UnitPctAnnualized,
			Ts:     time.Now(), // funding-rate 无响应时间戳，用本地采集时间
			Src:    fmt.Sprintf("api/v5/public/funding-rate rate=%s per%dh", r.FundingRate, hours),
		})
	}
	return out, nil
}

// okxIntervalHours 从结算时间戳对推断结算频率（8h/4h）；异常（非正/不可解析）报错。
func okxIntervalHours(r okxFundingRate) (int, error) {
	cur, err1 := strconv.ParseInt(r.FundingTime, 10, 64)
	next, err2 := strconv.ParseInt(r.NextFundingTime, 10, 64)
	if err1 != nil || err2 != nil {
		return 0, fmt.Errorf("bad funding time %q/%q", r.FundingTime, r.NextFundingTime)
	}
	h := int(math.Round(float64(next-cur) / float64(time.Hour/time.Millisecond)))
	if h <= 0 {
		return 0, fmt.Errorf("non-positive funding interval %dms", next-cur)
	}
	return h, nil
}

// OKXTicker 采集永续最新价（Kind=ticker）。
type OKXTicker struct{ cfg Config }

// NewOKXTicker 构造 OKX ticker collector。
func NewOKXTicker(cfg Config) *OKXTicker { return &OKXTicker{cfg: cfg} }

// Kind 实现 collect.Collector。
func (*OKXTicker) Kind() string { return fact.KindTicker }

// Poll 拉取全部 SWAP 最新价并过滤目标币种；缺失币种 → 整个 Poll 失败。
func (c *OKXTicker) Poll(ctx context.Context) ([]fact.Fact, error) {
	var resp okxResp
	if err := getJSON(ctx, c.cfg.client(), c.cfg.OKXBaseURL+okxTickersPath+"?instType=SWAP", &resp); err != nil {
		return nil, err
	}
	if err := resp.check(); err != nil {
		return nil, err
	}
	var rows []okxTicker
	if err := json.Unmarshal(resp.Data, &rows); err != nil {
		return nil, fmt.Errorf("okx tickers: decode data: %w", err)
	}
	byInst := make(map[string]okxTicker, len(rows))
	for _, r := range rows {
		byInst[r.InstID] = r
	}
	var out []fact.Fact
	for _, sym := range c.cfg.Symbols {
		inst := okxInst(sym)
		r, ok := byInst[inst]
		if !ok {
			return nil, fmt.Errorf("okx tickers: %s missing", inst)
		}
		price, err := strconv.ParseFloat(r.Last, 64)
		if err != nil {
			return nil, fmt.Errorf("okx tickers: %s: bad last %q", inst, r.Last)
		}
		out = append(out, fact.Fact{
			Kind:   fact.KindTicker,
			Venue:  VenueOKX,
			Symbol: sym,
			Value:  price,
			Unit:   fact.UnitPrice,
			Ts:     parseOKXTS(r.TS),
			Src:    "api/v5/market/tickers",
		})
	}
	return out, nil
}

// parseOKXTS 解析 OKX 毫秒时间戳；失败回退本地时间（时间戳异常不阻断行情）。
func parseOKXTS(s string) time.Time {
	if ms, err := strconv.ParseInt(s, 10, 64); err == nil {
		return time.UnixMilli(ms)
	}
	return time.Now()
}
