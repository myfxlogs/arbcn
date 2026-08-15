// Package domestic：国内利率采集（docs/design/02-monitor-architecture.md §5 Domestic 行）。
// ① 逆回购：新浪公开行情 hq.sinajs.cn（GC001/R-001 等，最新价 = 年化利率 %，须带 Referer）。
// ② 银行挂牌利率：BOC 官网人民币存款利率表两跳爬取（见 bankrate.go）；§5 允许失败降级——
//
//	该源独立命名（bank_rate），失败走 Scheduler 退避重试 + 人工录入通道补位，不拖垮逆回购。
//
// 铁律：公开只读、无密钥（§1）。
package domestic

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
	DefaultBaseURL      = "https://hq.sinajs.cn"
	DefaultReferer      = "https://finance.sina.com.cn"
	DefaultBankRateURL  = "https://www.bankofchina.com/fimarkets/lilv/fd31/"
	bankRateDisableFlag = "off" // ARBCN_BANK_RATE_URL=off → 不启用 bank_rate 源
)

// Repo 是逆回购标的映射：新浪行情代码 → Fact.Symbol（显示名，避开 GBK 名称字段）。
type Repo struct {
	Code   string // 如 sh204001
	Symbol string // 如 GC001
}

// defaultRepos：逆回购默认标的（沪市 GC001 + 深市 R-001，2026-08 行情核实）。
var defaultRepos = []Repo{{Code: "sh204001", Symbol: "GC001"}, {Code: "sz131810", Symbol: "R-001"}}

// Config 是 domestic 采集的最小配置面。
type Config struct {
	BaseURL     string       // 新浪行情基址（默认 DefaultBaseURL）
	Referer     string       // 新浪 hq API 必需（默认 DefaultReferer）
	Repos       []Repo       // 逆回购标的清单（默认 defaultRepos）
	BankRateURL string       // BOC 利率索引页；空 = 不启用 bank_rate 源
	Client      *http.Client // nil = 10s 超时默认客户端
}

// FromEnv 从环境变量构建 Config：
// ARBCN_REPO_SYMBOLS（"sh204001:GC001,sz131810:R-001"）与 ARBCN_BANK_RATE_URL（空 = 默认；off = 禁用）。
func FromEnv(getenv func(string) string) Config {
	cfg := Config{
		BaseURL:     DefaultBaseURL,
		Referer:     DefaultReferer,
		Repos:       defaultRepos,
		BankRateURL: DefaultBankRateURL,
	}
	if v := strings.TrimSpace(getenv("ARBCN_REPO_SYMBOLS")); v != "" {
		var repos []Repo
		for _, part := range strings.Split(v, ",") {
			code, sym, ok := strings.Cut(strings.TrimSpace(part), ":")
			code, sym = strings.TrimSpace(code), strings.TrimSpace(sym)
			if ok && code != "" && sym != "" {
				repos = append(repos, Repo{Code: code, Symbol: sym})
			}
		}
		if len(repos) > 0 {
			cfg.Repos = repos
		}
	}
	if v := strings.TrimSpace(getenv("ARBCN_BANK_RATE_URL")); v != "" {
		if v == bankRateDisableFlag {
			cfg.BankRateURL = ""
		} else {
			cfg.BankRateURL = v
		}
	}
	return cfg
}

// All 返回命名 collector：repo（5m，§5 5–15 分钟档）+ bank_rate（1h；挂牌利率变动罕见，
// 且为礼貌低频爬取；间隔可经 ARBCN_COLLECT_SOURCES 覆盖）。BankRateURL 为空则不启用 bank_rate。
func All(cfg Config) []collect.Named {
	out := []collect.Named{
		{Name: "repo", Interval: 5 * time.Minute, Collector: NewReverseRepo(cfg)},
	}
	if cfg.BankRateURL != "" {
		out = append(out, collect.Named{Name: "bank_rate", Interval: time.Hour, Collector: NewBankRate(cfg)})
	}
	return out
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

// ReverseRepo 采集逆回购最新价（= 年化利率 %，Kind=reverse_repo）。
type ReverseRepo struct{ cfg Config }

// NewReverseRepo 构造逆回购 collector。
func NewReverseRepo(cfg Config) *ReverseRepo { return &ReverseRepo{cfg: cfg} }

// Kind 实现 collect.Collector。
func (*ReverseRepo) Kind() string { return fact.KindReverseRepo }

// Poll 一次拉全部标的（list 逗号拼接）；缺标的/价非法 → 整个 Poll 失败（口径同 exchange）。
func (c *ReverseRepo) Poll(ctx context.Context) ([]fact.Fact, error) {
	if len(c.cfg.Repos) == 0 {
		return nil, nil // 无标的 = 源关闭，不发请求
	}
	codes := make([]string, len(c.cfg.Repos))
	for i, r := range c.cfg.Repos {
		codes[i] = r.Code
	}
	body, err := collect.GetText(ctx, c.cfg.client(), c.cfg.BaseURL+"/list="+strings.Join(codes, ","),
		c.cfg.sinaHeader(), 1<<20)
	if err != nil {
		return nil, err
	}
	quotes, err := parseRepoLines(body)
	if err != nil {
		return nil, fmt.Errorf("repo: %w", err)
	}
	var out []fact.Fact
	for _, r := range c.cfg.Repos {
		f, ok := quotes[r.Code]
		if !ok {
			return nil, fmt.Errorf("repo: %s: missing in response", r.Code)
		}
		out = append(out, fact.Fact{
			Kind:   fact.KindReverseRepo,
			Venue:  VenueSina,
			Symbol: r.Symbol,
			Value:  f.price,
			Unit:   fact.UnitPctAnnualized,
			Ts:     f.ts,
			Src:    "hq.sinajs.cn/list=" + r.Code,
		})
	}
	return out, nil
}

// repoQuote 是一条新浪行情脚本的解析结果。
type repoQuote struct {
	price float64
	ts    time.Time
}

// parseRepoLines 解析多行 `var hq_str_<code>="...";` 脚本（GBK 名称字段不影响逗号切分）。
func parseRepoLines(body string) (map[string]repoQuote, error) {
	out := make(map[string]repoQuote)
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "var hq_str_") {
			continue
		}
		rest, _, ok := strings.Cut(strings.TrimPrefix(line, "var hq_str_"), "=")
		if !ok {
			return nil, fmt.Errorf("parse line: no '=': %q", line)
		}
		code := rest
		_, quoted, ok := strings.Cut(line, `"`)
		if !ok {
			return nil, fmt.Errorf("parse %s: no quoted payload", code)
		}
		payload, _, _ := strings.Cut(quoted, `"`)
		fields := strings.Split(payload, ",")
		if len(fields) < 32 {
			return nil, fmt.Errorf("parse %s: %d fields, want >= 32", code, len(fields))
		}
		price, err := strconv.ParseFloat(fields[3], 64)
		if err != nil || price <= 0 {
			return nil, fmt.Errorf("parse %s: bad price %q", code, fields[3])
		}
		out[code] = repoQuote{price: price, ts: quoteTsCST(fields[30], fields[31])}
	}
	return out, nil
}

// quoteTsCST 组合日期/时间（CST）；解析失败回退本地时间。
func quoteTsCST(date, clock string) time.Time {
	if t, err := time.ParseInLocation("2006-01-02 15:04:05", date+" "+clock, collect.CST); err == nil {
		return t
	}
	return time.Now()
}
