package exchange

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"arbcn/internal/fact"
)

// Binance USDT-M 合约（fapi）公开端点：
//   - /fapi/v1/fundingInfo  → 每 symbol 的结算频率 fundingIntervalHours（4h/8h）
//   - /fapi/v1/premiumIndex → 最近一次结算费率 lastFundingRate + 服务器时间
//   - /fapi/v1/ticker/price → 全 symbol 最新价（本地过滤目标币种）
const (
	binanceFundingInfoPath  = "/fapi/v1/fundingInfo"
	binancePremiumIndexPath = "/fapi/v1/premiumIndex"
	binanceTickerPricePath  = "/fapi/v1/ticker/price"
)

// binanceInst 基础币 → USDT-M 合约 symbol（BTC → BTCUSDT）。
func binanceInst(base string) string { return base + "USDT" }

type binanceFundingInfo struct {
	Symbol               string `json:"symbol"`
	FundingIntervalHours int    `json:"fundingIntervalHours"`
}

type binancePremiumIndex struct {
	Symbol          string `json:"symbol"`
	LastFundingRate string `json:"lastFundingRate"`
	Time            int64  `json:"time"`
}

type binanceTickerPrice struct {
	Symbol string `json:"symbol"`
	Price  string `json:"price"`
	Time   int64  `json:"time"`
}

// BinanceFunding 采集 BTC/ETH/TRX 等 USDT-M 永续资金费率（Kind=funding）。
type BinanceFunding struct{ cfg Config }

// NewBinanceFunding 构造 Binance funding collector。
func NewBinanceFunding(cfg Config) *BinanceFunding { return &BinanceFunding{cfg: cfg} }

// Kind 实现 collect.Collector。
func (*BinanceFunding) Kind() string { return fact.KindFunding }

// Poll 拉取 fundingInfo + premiumIndex，按结算频率折算年化；
// 请求币种缺失 → 整个 Poll 失败（缺币种 = 漏监控窗口，不静默降级）。
func (c *BinanceFunding) Poll(ctx context.Context) ([]fact.Fact, error) {
	client := c.cfg.client()
	base := c.cfg.BinanceBaseURL

	var infos []binanceFundingInfo
	if err := getJSON(ctx, client, base+binanceFundingInfoPath, &infos); err != nil {
		return nil, err
	}
	intervalByInst := make(map[string]int, len(infos))
	for _, i := range infos {
		intervalByInst[i.Symbol] = i.FundingIntervalHours
	}

	var rows []binancePremiumIndex
	if err := getJSON(ctx, client, base+binancePremiumIndexPath, &rows); err != nil {
		return nil, err
	}
	rateByInst := make(map[string]binancePremiumIndex, len(rows))
	for _, r := range rows {
		rateByInst[r.Symbol] = r
	}

	var out []fact.Fact
	for _, sym := range c.cfg.Symbols {
		inst := binanceInst(sym)
		e, ok := rateByInst[inst]
		if !ok {
			return nil, fmt.Errorf("binance funding: %s: missing in premiumIndex", inst)
		}
		hours, ok := intervalByInst[inst]
		if !ok {
			return nil, fmt.Errorf("binance funding: %s: missing in fundingInfo", inst)
		}
		rate, err := strconv.ParseFloat(e.LastFundingRate, 64)
		if err != nil {
			return nil, fmt.Errorf("binance funding: %s: bad rate %q", inst, e.LastFundingRate)
		}
		v, err := annualize(rate, hours)
		if err != nil {
			return nil, fmt.Errorf("binance funding: %s: %w", inst, err)
		}
		out = append(out, fact.Fact{
			Kind:   fact.KindFunding,
			Venue:  VenueBinance,
			Symbol: sym,
			Value:  v,
			Unit:   fact.UnitPctAnnualized,
			Ts:     time.UnixMilli(e.Time),
			Src:    fmt.Sprintf("fapi/v1/premiumIndex rate=%s per%dh", e.LastFundingRate, hours),
		})
	}
	return out, nil
}

// BinanceTicker 采集 USDT-M 永续最新价（Kind=ticker）。
type BinanceTicker struct{ cfg Config }

// NewBinanceTicker 构造 Binance ticker collector。
func NewBinanceTicker(cfg Config) *BinanceTicker { return &BinanceTicker{cfg: cfg} }

// Kind 实现 collect.Collector。
func (*BinanceTicker) Kind() string { return fact.KindTicker }

// Poll 拉取全 symbol 最新价并过滤目标币种；缺失币种 → 整个 Poll 失败。
func (c *BinanceTicker) Poll(ctx context.Context) ([]fact.Fact, error) {
	var rows []binanceTickerPrice
	if err := getJSON(ctx, c.cfg.client(), c.cfg.BinanceBaseURL+binanceTickerPricePath, &rows); err != nil {
		return nil, err
	}
	byInst := make(map[string]binanceTickerPrice, len(rows))
	for _, r := range rows {
		byInst[r.Symbol] = r
	}
	var out []fact.Fact
	for _, sym := range c.cfg.Symbols {
		inst := binanceInst(sym)
		r, ok := byInst[inst]
		if !ok {
			return nil, fmt.Errorf("binance ticker: %s: missing in ticker/price", inst)
		}
		price, err := strconv.ParseFloat(r.Price, 64)
		if err != nil {
			return nil, fmt.Errorf("binance ticker: %s: bad price %q", inst, r.Price)
		}
		out = append(out, fact.Fact{
			Kind:   fact.KindTicker,
			Venue:  VenueBinance,
			Symbol: sym,
			Value:  price,
			Unit:   fact.UnitPrice,
			Ts:     time.UnixMilli(r.Time),
			Src:    "fapi/v1/ticker/price",
		})
	}
	return out, nil
}
