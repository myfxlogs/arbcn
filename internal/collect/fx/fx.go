// Package fx：USDCNH 汇率采集（docs/design/02-monitor-architecture.md §5 FX 行）。
// 新浪公开行情 hq.sinajs.cn（无密钥，须带 Referer 头，否则 403/空响应）。
// 响应格式 fx_s 系列：逗号分隔，idx3 = 最新价，idx17 = 日期，idx0 = 时间（CST）。
// Fact.Ts：优先报价日期+时间（CST），缺失回退本地采集时间。
package fx

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"arbcn/internal/collect"
	"arbcn/internal/fact"
)

// VenueSina 是新浪公开行情的 Fact.Venue 值。
const VenueSina = "sina"

const (
	DefaultBaseURL = "https://hq.sinajs.cn"
	DefaultReferer = "https://finance.sina.com.cn"
	codeUSDCNH     = "fx_susdcnh"
)

// Config 是 fx 采集的最小配置面。
type Config struct {
	BaseURL string       // 默认 DefaultBaseURL
	Referer string       // 默认 DefaultReferer（hq API 必需）
	Client  *http.Client // nil = 10s 超时默认客户端
}

// FromEnv 从环境变量构建 Config（ARBCN_FX_BASE_URL 可覆盖基址，便于代理调试；其余取默认）。
func FromEnv(getenv func(string) string) Config {
	cfg := Config{BaseURL: DefaultBaseURL, Referer: DefaultReferer}
	if v := strings.TrimSpace(getenv("ARBCN_FX_BASE_URL")); v != "" {
		cfg.BaseURL = v
	}
	return cfg
}

// All 返回命名 collector：fx，默认间隔 5m（§5）。
func All(cfg Config) []collect.Named {
	return []collect.Named{
		{Name: "fx", Interval: 5 * time.Minute, Collector: NewFX(cfg)},
	}
}

func (c Config) client() *http.Client {
	if c.Client != nil {
		return c.Client
	}
	return &http.Client{Timeout: 10 * time.Second}
}

// sinaHeader 新浪 hq API 必需的 Referer 头；配置留空时取默认（测试/直构 Config 兜底）。
func (c Config) sinaHeader() http.Header {
	referer := c.Referer
	if referer == "" {
		referer = DefaultReferer
	}
	return http.Header{"Referer": {referer}}
}

// FX 采集 USDCNH 最新价（Kind=fx）。
type FX struct{ cfg Config }

// NewFX 构造 FX collector。
func NewFX(cfg Config) *FX { return &FX{cfg: cfg} }

// Kind 实现 collect.Collector。
func (*FX) Kind() string { return fact.KindFX }

// Poll 拉取 fx_susdcnh 报价；价字段缺失/非法 → 失败（缺报价 = 漏监控窗口）。
func (c *FX) Poll(ctx context.Context) ([]fact.Fact, error) {
	body, err := collect.GetText(ctx, c.cfg.client(), c.cfg.BaseURL+"/list="+codeUSDCNH,
		c.cfg.sinaHeader(), 1<<20)
	if err != nil {
		return nil, err
	}
	fields, err := parseQuote(body)
	if err != nil {
		return nil, fmt.Errorf("fx: %w", err)
	}
	price, err := strconv.ParseFloat(fields[3], 64)
	if err != nil || price <= 0 {
		return nil, fmt.Errorf("fx: bad price %q", fields[3])
	}
	return []fact.Fact{{
		Kind:   fact.KindFX,
		Venue:  VenueSina,
		Symbol: "USDCNH",
		Value:  price,
		Unit:   fact.UnitPrice,
		Ts:     quoteTs(fields),
		Src:    "hq.sinajs.cn/list=" + codeUSDCNH,
	}}, nil
}

// parseQuote 从 `var hq_str_<code>="...";` 脚本中取出引号内逗号分隔字段。
func parseQuote(body string) ([]string, error) {
	_, after, ok := strings.Cut(body, `"`)
	if !ok {
		return nil, fmt.Errorf("parse: no quoted payload")
	}
	payload, _, _ := strings.Cut(after, `"`)
	fields := strings.Split(payload, ",")
	if len(fields) < 17 {
		return nil, fmt.Errorf("parse: %d fields, want >= 17", len(fields))
	}
	return fields, nil
}

// quoteTs 组合 idx17 日期 + idx0 时间（CST）；解析失败回退本地时间。
func quoteTs(fields []string) time.Time {
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", fields[17]+" "+fields[0], collect.CST); err == nil {
		return t
	}
	return time.Now()
}
