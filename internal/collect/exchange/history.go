// M3-b §9.5 S4：历史 funding 回填数据源（一次性、幂等、无 key）。
// Binance data-api 公开数据域（D-031）+ OKX 公开 funding-history，翻页拉满窗口。
// 产出 fact{Kind=funding, Venue, Symbol, Value=年化, Ts=结算时刻, Unit=pct_annualized}。
// 本文件无密钥、无下单端点；实现 sim.HistoryCollector（Venue + Poll），由 main.go 注入
// sim.BackfillHistory 编排。days<=0 = 禁用（Poll 返回空）。
package exchange

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"arbcn/internal/fact"
)

// binanceDataAPIRoot Binance 公开数据域（D-031）：历史 fundingRate 镜像，与实时 fapi 域隔离。
const binanceDataAPIRoot = "https://data-api.binance.vision"

const (
	binanceFundingRatePath = "/fapi/v1/fundingRate"
	okxFundingHistoryPath  = "/api/v5/public/funding-history"
)

// binanceFundingRateHistory 单行历史结算记录（data-api fundingRate）。
type binanceFundingRateHistory struct {
	Symbol      string `json:"symbol"`
	FundingTime int64  `json:"fundingTime"`
	FundingRate string `json:"fundingRate"`
}

// okxFundingHistory 单行历史结算记录（funding-history）。
type okxFundingHistory struct {
	InstID      string `json:"instId"`
	FundingRate string `json:"fundingRate"`
	FundingTime string `json:"fundingTime"`
}

// binanceHistoryBase 历史数据域（测试注入优先）。
func (c Config) binanceHistoryBase() string {
	if c.BinanceHistoryBaseURL != "" {
		return c.BinanceHistoryBaseURL
	}
	return binanceDataAPIRoot
}

// BinanceFundingHistory 是 Binance 历史 funding 回填数据源。
type BinanceFundingHistory struct {
	cfg  Config
	days int
}

// NewBinanceFundingHistory 构造 Binance 历史回填 collector（S4）。days = 窗口天数。
func NewBinanceFundingHistory(cfg Config, days int) *BinanceFundingHistory {
	return &BinanceFundingHistory{cfg: cfg, days: days}
}

// Kind 实现 collect.Collector。
func (*BinanceFundingHistory) Kind() string { return fact.KindFunding }

// Venue 实现 sim.HistoryCollector（幂等查询按 venue 过滤）。
func (*BinanceFundingHistory) Venue() string { return VenueBinance }

// Poll 翻页拉满窗口内全部 symbol 的历史 funding。days<=0 → 返回空（禁用）。
func (c *BinanceFundingHistory) Poll(ctx context.Context) ([]fact.Fact, error) {
	if c.days <= 0 {
		return nil, nil
	}
	client := c.cfg.client()
	base := c.cfg.binanceHistoryBase()
	now := time.Now()
	from := now.Add(-time.Duration(c.days) * 24 * time.Hour)

	var out []fact.Fact
	for _, sym := range c.cfg.Symbols {
		inst := binanceInst(sym)
		rows, err := c.paginate(ctx, client, base, inst, from, now)
		if err != nil {
			return nil, fmt.Errorf("binance funding history %s: %w", inst, err)
		}
		interval := binanceIntervalHours(rows)
		for _, r := range rows {
			rate, err := strconv.ParseFloat(r.FundingRate, 64)
			if err != nil {
				return nil, fmt.Errorf("binance funding history %s: bad rate %q", inst, r.FundingRate)
			}
			v, err := annualize(rate, interval)
			if err != nil {
				return nil, fmt.Errorf("binance funding history %s: %w", inst, err)
			}
			out = append(out, fact.Fact{
				Kind: fact.KindFunding, Venue: VenueBinance, Symbol: sym,
				Value: v, Unit: fact.UnitPctAnnualized,
				Ts:  time.UnixMilli(r.FundingTime),
				Src: fmt.Sprintf("data-api/fapi/v1/fundingRate rate=%s", r.FundingRate),
			})
		}
	}
	return out, nil
}

// paginate 从 from 向 now 翻页拉取（limit=1000/页；满页则推进 startTime）。
func (c *BinanceFundingHistory) paginate(ctx context.Context, client *http.Client, base, inst string, from, to time.Time) ([]binanceFundingRateHistory, error) {
	var all []binanceFundingRateHistory
	start := from.UnixMilli()
	end := to.UnixMilli()
	for {
		url := fmt.Sprintf("%s%s?symbol=%s&startTime=%d&endTime=%d&limit=1000",
			base, binanceFundingRatePath, inst, start, end)
		var page []binanceFundingRateHistory
		if err := getJSON(ctx, client, url, &page); err != nil {
			return nil, err
		}
		if len(page) == 0 {
			break
		}
		all = append(all, page...)
		if len(page) < 1000 {
			break
		}
		start = page[len(page)-1].FundingTime + 1 // 下一翻页从最后一条之后
		if start >= end {
			break
		}
	}
	return all, nil
}

// binanceIntervalHours 从相邻结算时刻差推断结算频率（8h/4h 等）；<2 点或异常 → 8h。
func binanceIntervalHours(rows []binanceFundingRateHistory) int {
	ts := make([]int64, 0, len(rows))
	for _, r := range rows {
		ts = append(ts, r.FundingTime)
	}
	return inferIntervalHours(ts)
}

// OKXFundingHistory 是 OKX 历史 funding 回填数据源。
type OKXFundingHistory struct {
	cfg  Config
	days int
}

// NewOKXFundingHistory 构造 OKX 历史回填 collector（S4）。
func NewOKXFundingHistory(cfg Config, days int) *OKXFundingHistory {
	return &OKXFundingHistory{cfg: cfg, days: days}
}

// Kind 实现 collect.Collector。
func (*OKXFundingHistory) Kind() string { return fact.KindFunding }

// Venue 实现 sim.HistoryCollector。
func (*OKXFundingHistory) Venue() string { return VenueOKX }

// Poll 翻页拉满窗口内全部 symbol 的历史 funding。days<=0 → 返回空（禁用）。
func (c *OKXFundingHistory) Poll(ctx context.Context) ([]fact.Fact, error) {
	if c.days <= 0 {
		return nil, nil
	}
	client := c.cfg.client()
	base := c.cfg.OKXBaseURL
	now := time.Now()
	from := now.Add(-time.Duration(c.days) * 24 * time.Hour)

	var out []fact.Fact
	for _, sym := range c.cfg.Symbols {
		inst := okxInst(sym)
		rows, err := c.paginate(ctx, client, base, inst, from)
		if err != nil {
			return nil, fmt.Errorf("okx funding history %s: %w", inst, err)
		}
		interval := okxHistoryIntervalHours(rows)
		for _, r := range rows {
			ts := parseOKXTS(r.FundingTime)
			if ts.Before(from) {
				continue // 末页越界到窗口前的记录丢弃（keep 窗口内）
			}
			rate, err := strconv.ParseFloat(r.FundingRate, 64)
			if err != nil {
				return nil, fmt.Errorf("okx funding history %s: bad rate %q", inst, r.FundingRate)
			}
			v, err := annualize(rate, interval)
			if err != nil {
				return nil, fmt.Errorf("okx funding history %s: %w", inst, err)
			}
			out = append(out, fact.Fact{
				Kind: fact.KindFunding, Venue: VenueOKX, Symbol: sym,
				Value: v, Unit: fact.UnitPctAnnualized,
				Ts:  ts,
				Src: fmt.Sprintf("api/v5/public/funding-history rate=%s", r.FundingRate),
			})
		}
	}
	return out, nil
}

// paginate 从最新向 from 翻页拉取（after 分页，返回新→旧；到窗口边界停）。
func (c *OKXFundingHistory) paginate(ctx context.Context, client *http.Client, base, inst string, from time.Time) ([]okxFundingHistory, error) {
	var all []okxFundingHistory
	after := ""
	fromMs := from.UnixMilli()
	for {
		url := base + okxFundingHistoryPath + "?instId=" + inst + "&limit=100"
		if after != "" {
			url += "&after=" + after
		}
		var resp okxResp
		if err := getJSON(ctx, client, url, &resp); err != nil {
			return nil, err
		}
		if err := resp.check(); err != nil {
			return nil, err
		}
		var page []okxFundingHistory
		if err := json.Unmarshal(resp.Data, &page); err != nil {
			return nil, err
		}
		if len(page) == 0 {
			break
		}
		oldest := parseOKXTS(page[len(page)-1].FundingTime)
		all = append(all, page...)
		if len(page) < 100 || oldest.UnixMilli() < fromMs {
			break
		}
		after = page[len(page)-1].FundingTime
	}
	return all, nil
}

// okxHistoryIntervalHours 从结算时刻序列推断结算频率。
func okxHistoryIntervalHours(rows []okxFundingHistory) int {
	ts := make([]int64, 0, len(rows))
	for _, r := range rows {
		ts = append(ts, parseOKXTS(r.FundingTime).UnixMilli())
	}
	return inferIntervalHours(ts)
}

// inferIntervalHours 从结算时刻序列推断结算频率（相邻差众数，取整小时）；<2 点或异常 → 8h。
//
// [对抗测试锚点] §9.5 S4：删除 annualize 调用 → history_test.go
// TestHistoryAnnualizes（年化折算断言）必红。
func inferIntervalHours(tsMs []int64) int {
	if len(tsMs) < 2 {
		return 8
	}
	counts := map[int64]int{}
	for i := 1; i < len(tsMs); i++ {
		if d := tsMs[i] - tsMs[i-1]; d > 0 {
			counts[d]++
		}
	}
	var best int64
	bestCnt := 0
	for d, c := range counts {
		if c > bestCnt {
			bestCnt, best = c, d
		}
	}
	if best <= 0 {
		return 8
	}
	h := int(math.Round(float64(best) / float64(time.Hour/time.Millisecond)))
	if h <= 0 {
		return 8
	}
	return h
}
