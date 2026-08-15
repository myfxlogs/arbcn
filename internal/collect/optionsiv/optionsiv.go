// Package optionsiv：BTC/ETH 隐含波动率采集（docs/design/02-monitor-architecture.md §5 OptionsIV 行）。
// Deribit DVOL 公开指数（get_volatility_index_data，日线最近一根 close = 当前 DVOL，单位 %）。
// 受阻（如被墙）→ 本源失败走 Scheduler 退避 + 人工录入通道补位（§5 授权降级）。
// OKX 公开 opt-summary（ATM markVol）为备选源（§5 列名），当前未启用。
// 铁律：公开只读、无密钥（§1）。
package optionsiv

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"arbcn/internal/collect"
	"arbcn/internal/fact"
)

// VenueDeribit 是 Deribit DVOL 的 Fact.Venue 值。
const VenueDeribit = "deribit"

const DefaultBaseURL = "https://www.deribit.com"

// DefaultCurrencies 是 DVOL 覆盖币种（Deribit 公开指数支持 BTC/ETH）。
var DefaultCurrencies = []string{"BTC", "ETH"}

// Config 是 optionsiv 采集的最小配置面。
type Config struct {
	Currencies []string     // 币种（默认 BTC,ETH）
	BaseURL    string       // 默认 DefaultBaseURL
	Client     *http.Client // nil = 10s 超时默认客户端
}

// FromEnv 从环境变量构建 Config（ARBCN_IV_CURRENCIES 逗号分隔，大小写归一；空 = 默认 BTC,ETH）。
func FromEnv(getenv func(string) string) Config {
	cfg := Config{Currencies: DefaultCurrencies, BaseURL: DefaultBaseURL}
	if v := strings.TrimSpace(getenv("ARBCN_IV_CURRENCIES")); v != "" {
		var curs []string
		for _, s := range strings.Split(v, ",") {
			if s = strings.ToUpper(strings.TrimSpace(s)); s != "" {
				curs = append(curs, s)
			}
		}
		if len(curs) > 0 {
			cfg.Currencies = curs
		}
	}
	return cfg
}

// All 返回命名 collector：deribit_iv，默认间隔 30m（§5）。
func All(cfg Config) []collect.Named {
	return []collect.Named{
		{Name: "deribit_iv", Interval: 30 * time.Minute, Collector: NewDeribitIV(cfg)},
	}
}

func (c Config) client() *http.Client {
	if c.Client != nil {
		return c.Client
	}
	return &http.Client{Timeout: 10 * time.Second}
}

// dvResp 是 get_volatility_index_data 的响应（Error 非 nil = JSON-RPC 业务错误）。
type dvResp struct {
	Error *struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Result *struct {
		Data [][]float64 `json:"data"` // 每行 [ts, open, high, low, close]（毫秒）
	} `json:"result"`
}

// DeribitIV 采集 DVOL 指数（Kind=iv）。
type DeribitIV struct{ cfg Config }

// NewDeribitIV 构造 DVOL collector。
func NewDeribitIV(cfg Config) *DeribitIV { return &DeribitIV{cfg: cfg} }

// Kind 实现 collect.Collector。
func (*DeribitIV) Kind() string { return fact.KindIV }

// Poll 逐币种拉最近 2 个自然日的日线，取最近一根 close = 当前 DVOL；
// 任一币种无数据/异常 → 整个 Poll 失败（缺 IV = 漏期权预算窗口）。
func (c *DeribitIV) Poll(ctx context.Context) ([]fact.Fact, error) {
	if len(c.cfg.Currencies) == 0 {
		return nil, nil // 无币种 = 源关闭，不发请求
	}
	client := c.cfg.client()
	now := time.Now()
	q := url.Values{
		"start_timestamp": {fmt.Sprintf("%d", now.Add(-48*time.Hour).UnixMilli())},
		"end_timestamp":   {fmt.Sprintf("%d", now.UnixMilli())},
		"resolution":      {"86400"},
	}
	var out []fact.Fact
	for _, cur := range c.cfg.Currencies {
		q.Set("currency", cur)
		var resp dvResp
		if err := collect.GetJSON(ctx, client, c.cfg.BaseURL+"/api/v2/public/get_volatility_index_data?"+q.Encode(), nil, 1<<20, &resp); err != nil {
			return nil, err
		}
		if resp.Error != nil {
			return nil, fmt.Errorf("deribit iv %s: code=%d msg=%s", cur, resp.Error.Code, resp.Error.Message)
		}
		if resp.Result == nil || len(resp.Result.Data) == 0 {
			return nil, fmt.Errorf("deribit iv %s: empty data", cur)
		}
		last := resp.Result.Data[len(resp.Result.Data)-1]
		if len(last) < 5 {
			return nil, fmt.Errorf("deribit iv %s: bad row length %d", cur, len(last))
		}
		out = append(out, fact.Fact{
			Kind:   fact.KindIV,
			Venue:  VenueDeribit,
			Symbol: cur,
			Value:  last[4],
			Unit:   fact.UnitPct,
			Ts:     time.UnixMilli(int64(last[0])),
			Src:    "api/v2/public/get_volatility_index_data DVOL",
		})
	}
	return out, nil
}
