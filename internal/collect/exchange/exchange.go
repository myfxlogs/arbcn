// Package exchange：交易所公开行情采集（docs/design/02-monitor-architecture.md §5 Exchange 行）。
// Binance fapi / OKX v5 公开端点：funding（含 TRX，按 8h/4h 结算频率折算年化）+ ticker。
// 铁律：公开只读端点、无密钥（§1）；币种清单配置可改（ARBCN_COLLECT_SYMBOLS）。
// Fact.Ts 口径：优先交易所报告时间戳，缺失回退本地采集时间（各 Poll 内注明）。
package exchange

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"arbcn/internal/collect"
)

// Venue 常量（Fact.Venue 值域：binance / okx）。
const (
	VenueBinance = "binance"
	VenueOKX     = "okx"
)

// Config 是 exchange 采集的最小配置面。
type Config struct {
	Symbols        []string     // 基础币种，如 BTC/ETH/TRX
	BinanceBaseURL string       // 默认 https://fapi.binance.com（USDT-M 合约）
	OKXBaseURL     string       // 默认 https://www.okx.com
	// BinanceHistoryBaseURL 历史 funding 回填数据域（M3-b §9.5，D-031 公开数据域；
	// 默认 https://data-api.binance.vision）。独立于实时 fapi 域；测试可注入 httptest。
	BinanceHistoryBaseURL string
	Client                *http.Client // nil = 10s 超时默认客户端
}

// FromEnv 从环境变量构建 Config（ARBCN_COLLECT_SYMBOLS 逗号分隔，默认 BTC,ETH,TRX；大小写归一）。
func FromEnv(getenv func(string) string) Config {
	cfg := Config{
		Symbols:        []string{"BTC", "ETH", "TRX"},
		BinanceBaseURL: "https://fapi.binance.com",
		OKXBaseURL:     "https://www.okx.com",
	}
	if v := strings.TrimSpace(getenv("ARBCN_COLLECT_SYMBOLS")); v != "" {
		var syms []string
		for _, s := range strings.Split(v, ",") {
			if s = strings.ToUpper(strings.TrimSpace(s)); s != "" {
				syms = append(syms, s)
			}
		}
		if len(syms) > 0 {
			cfg.Symbols = syms
		}
	}
	return cfg
}

// All 返回四个命名 collector（binance/okx × funding/ticker）及默认间隔：
// funding 5m（费率 8h/4h 结算一次，低频足够）；ticker 1m（价格变化快）。
// 间隔可经 ARBCN_COLLECT_SOURCES（collect.LoadSources）覆盖。
func All(cfg Config) []collect.Named {
	return []collect.Named{
		{Name: "binance_funding", Interval: 5 * time.Minute, Collector: NewBinanceFunding(cfg)},
		{Name: "binance_ticker", Interval: time.Minute, Collector: NewBinanceTicker(cfg)},
		{Name: "okx_funding", Interval: 5 * time.Minute, Collector: NewOKXFunding(cfg)},
		{Name: "okx_ticker", Interval: time.Minute, Collector: NewOKXTicker(cfg)},
	}
}

func (c Config) client() *http.Client {
	if c.Client != nil {
		return c.Client
	}
	return &http.Client{Timeout: 10 * time.Second}
}

// getJSON GET 并解码（公共工具 collect.GetJSON，1MB 上限，无自定义头）。
func getJSON(ctx context.Context, client *http.Client, url string, out any) error {
	return collect.GetJSON(ctx, client, url, nil, 1<<20, out)
}

// annualize 把单次结算费率折算为年化百分数：× (8760 / intervalHours) × 100。
// 8h → ×1095；4h → ×2190。负费率（如 TRX）原样保留。
func annualize(rate float64, intervalHours int) (float64, error) {
	if intervalHours <= 0 {
		return 0, fmt.Errorf("annualize: bad interval %dh", intervalHours)
	}
	return rate * (8760 / float64(intervalHours)) * 100, nil
}
