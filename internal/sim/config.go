// Package sim：M3-a 模拟执行（docs/design/04-m3-spec.md §3/§4）。
// 信号 → 建议订单纯函数转换（SignalToOrder）+ 六道风险门禁 + 本地模拟盘回填 +
// funding 结算（复用 internal/rmb 做 RMB 日终折算）。纯本地、零外部连接、无密钥（D-010）。
// 门禁数值已定稿（业主确认 2026-08-15），改动走 D#。
package sim

import (
	"fmt"
	"strconv"
)

// Config 是 M3-a 本地模拟盘配置（04-m3-spec §3/§4）。缺省值 = 决策层定稿。
type Config struct {
	Capital     float64 // 模拟资金基数（模拟 USD；ARBCN_SIM_CAPITAL；默认 100_000）
	MinSpread   float64 // 预期年化价差门槛 %（ARBCN_SIM_MIN_SPREAD；默认 5）
	MaxSizePct  float64 // 单笔名义上限 = Capital×MaxSizePct（ARBCN_SIM_MAX_SIZE_PCT；默认 0.20）
	MaxDailyPct float64 // 日累计名义上限 = Capital×MaxDailyPct（ARBCN_SIM_MAX_DAILY_PCT；默认 0.50）
}

// DefaultConfig 返回定稿默认值（04-m3-spec §4 表）。
func DefaultConfig() Config {
	return Config{Capital: 100_000, MinSpread: 5, MaxSizePct: 0.20, MaxDailyPct: 0.50}
}

// FromEnv 从 ARBCN_SIM_* 读取覆盖；未设置 → 默认。getenv 便于测试注入。
// 非法数值 → 错误（配置错误 fail fast；调用方可按 §7 降级禁用 sim 模块，不退出进程）。
func FromEnv(getenv func(string) string) (Config, error) {
	cfg := DefaultConfig()
	overrides := []struct {
		key string
		dst *float64
	}{
		{"ARBCN_SIM_CAPITAL", &cfg.Capital},
		{"ARBCN_SIM_MIN_SPREAD", &cfg.MinSpread},
		{"ARBCN_SIM_MAX_SIZE_PCT", &cfg.MaxSizePct},
		{"ARBCN_SIM_MAX_DAILY_PCT", &cfg.MaxDailyPct},
	}
	for _, ov := range overrides {
		v := getenv(ov.key)
		if v == "" {
			continue
		}
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return Config{}, fmt.Errorf("sim: %s: %q: %w", ov.key, v, err)
		}
		*ov.dst = f
	}
	if cfg.Capital <= 0 || cfg.MinSpread <= 0 || cfg.MaxSizePct <= 0 || cfg.MaxDailyPct <= 0 {
		return Config{}, fmt.Errorf("sim: ARBCN_SIM_* 必须为正数（capital=%v min_spread=%v max_size=%v max_daily=%v）",
			cfg.Capital, cfg.MinSpread, cfg.MaxSizePct, cfg.MaxDailyPct)
	}
	return cfg, nil
}
