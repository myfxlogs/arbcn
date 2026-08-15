// Package defirate：DeFi 收益率采集（docs/design/02-monitor-architecture.md §5 DeFiRates 行）。
// DefiLlama Yields API（公开无密钥）：一次拉全池列表（/pools，无服务端过滤参数），
// 按配置池 UUID 本地过滤，产出 Aave / Morpho / sUSDe / 代币化美债（BUIDL/USDY 等）年化 APY。
// 口径：pool.apy 为总年化百分数（apyBase+apyReward），原样入 Fact.Value。
// 缺池 / apy 为空 → 整个 Poll 失败（缺标的 = 漏监控窗口，与 exchange 口径一致）。
// Fact.Ts：API 无逐池时间戳，用本地采集时间。
package defirate

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"arbcn/internal/collect"
	"arbcn/internal/fact"
)

// DefaultBaseURL 是 DefiLlama Yields API 基址（公开只读，无密钥）。
const DefaultBaseURL = "https://yields.llama.fi"

// defaultPools：标的池默认清单（2026-08-15 从 /pools 全表核实 UUID）。
var defaultPools = []string{
	"aa70268e-4b52-42bf-a116-608b370f9501", // aave-v3 USDC Ethereum
	"931ea9be-5f4d-428e-beaf-205fc5b4e2b5", // morpho-blue Steakhouse USDC Ethereum
	"66985a81-9c51-46ca-9977-42b4fe7bc6df", // ethena sUSDe Ethereum
	"b663ca59-c7e6-4435-ae4a-28d339ce6a15", // BlackRock BUIDL Ethereum
	"ac61ee82-2fe4-4f9b-a9cd-7fb33f598859", // Ondo USDY Ethereum
}

// Config 是 defirate 采集的最小配置面。
type Config struct {
	Pools   []string     // 池 UUID 清单（缺省 defaultPools）
	BaseURL string       // 默认 DefaultBaseURL
	Client  *http.Client // nil = 30s 超时默认客户端（/pools 全表 ~10MB，放宽于 10s）
}

// FromEnv 从环境变量构建 Config（ARBCN_DEFI_POOLS 逗号分隔 UUID；空 = 默认池清单）。
func FromEnv(getenv func(string) string) Config {
	cfg := Config{Pools: defaultPools, BaseURL: DefaultBaseURL}
	if v := strings.TrimSpace(getenv("ARBCN_DEFI_POOLS")); v != "" {
		seen := make(map[string]bool)
		var ids []string
		for _, s := range strings.Split(v, ",") {
			if s = strings.TrimSpace(s); s != "" && !seen[s] {
				seen[s] = true
				ids = append(ids, s)
			}
		}
		if len(ids) > 0 {
			cfg.Pools = ids
		}
	}
	return cfg
}

// All 返回命名 collector：defi_rates，默认间隔 30m（§5：30–60 分钟）。
// 间隔可经 ARBCN_COLLECT_SOURCES（collect.LoadSources）覆盖。
func All(cfg Config) []collect.Named {
	return []collect.Named{
		{Name: "defi_rates", Interval: 30 * time.Minute, Collector: NewDefiRates(cfg)},
	}
}

func (c Config) client() *http.Client {
	if c.Client != nil {
		return c.Client
	}
	return &http.Client{Timeout: 30 * time.Second}
}

// pool 是 /pools 响应中命中过滤所需的字段（其余字段不落地）。
type pool struct {
	Pool    string   `json:"pool"`
	Project string   `json:"project"`
	Symbol  string   `json:"symbol"`
	Apy     *float64 `json:"apy"` // 总年化百分数；null = 无数据
}

// DefiRates 采集配置池的年化 APY（Kind=defi_rate）。
type DefiRates struct{ cfg Config }

// NewDefiRates 构造 DefiRates collector。
func NewDefiRates(cfg Config) *DefiRates { return &DefiRates{cfg: cfg} }

// Kind 实现 collect.Collector。
func (*DefiRates) Kind() string { return fact.KindDefiRate }

// Poll 拉取 /pools 全表并按配置池 UUID 过滤；配置池缺失或 apy 为空 → 整个 Poll 失败。
func (c *DefiRates) Poll(ctx context.Context) ([]fact.Fact, error) {
	if len(c.cfg.Pools) == 0 {
		return nil, nil // 无配置池 = 源关闭，不发请求
	}
	var resp struct {
		Status string          `json:"status"`
		Data   json.RawMessage `json:"data"`
	}
	if err := collect.GetJSON(ctx, c.cfg.client(), c.cfg.BaseURL+"/pools", nil, 64<<20, &resp); err != nil {
		return nil, err
	}
	if resp.Status != "success" {
		return nil, fmt.Errorf("defirate: status %q", resp.Status)
	}
	byID := make(map[string]pool)
	dec := json.NewDecoder(bytes.NewReader(resp.Data))
	if tok, err := dec.Token(); err != nil || tok != json.Delim('[') {
		return nil, fmt.Errorf("defirate: bad data array")
	}
	for dec.More() {
		var p pool
		if err := dec.Decode(&p); err != nil {
			return nil, fmt.Errorf("defirate: decode pool: %w", err)
		}
		byID[p.Pool] = p
	}
	ts := time.Now()
	var out []fact.Fact
	for _, id := range c.cfg.Pools {
		p, ok := byID[id]
		if !ok {
			return nil, fmt.Errorf("defirate: pool %s: missing in /pools", id)
		}
		if p.Apy == nil {
			return nil, fmt.Errorf("defirate: pool %s (%s/%s): nil apy", id, p.Project, p.Symbol)
		}
		out = append(out, fact.Fact{
			Kind:   fact.KindDefiRate,
			Venue:  p.Project,
			Symbol: p.Symbol,
			Value:  *p.Apy,
			Unit:   fact.UnitPctAnnualized,
			Ts:     ts,
			Src:    "yields.llama.fi/pools pool=" + id,
		})
	}
	return out, nil
}
