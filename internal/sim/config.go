// Package sim：M3-a 模拟执行（docs/design/04-m3-spec.md §3/§4）。
// 信号 → 建议订单纯函数转换（SignalToOrder）+ 六道风险门禁 + 本地模拟盘回填 +
// funding 结算（复用 internal/rmb 做 RMB 日终折算）。纯本地、零外部连接、无密钥（D-010）。
// 门禁数值已定稿（业主确认 2026-08-15），改动走 D#。
package sim

import (
	"slices"
	"fmt"
	"strconv"
	"strings"
)

// Config 是 M3-a 本地模拟盘配置（04-m3-spec §3/§4）。缺省值 = 决策层定稿。
type Config struct {
	Capital        float64 // 模拟资金基数（模拟 USD；ARBCN_SIM_CAPITAL；默认 100_000）
	MinSpread      float64 // 预期年化价差门槛 %（ARBCN_SIM_MIN_SPREAD；默认 5）
	MaxSizePct     float64 // 单笔名义上限 = Capital×MaxSizePct（ARBCN_SIM_MAX_SIZE_PCT；默认 0.20）
	MaxDailyPct    float64 // 日累计名义上限 = Capital×MaxDailyPct（ARBCN_SIM_MAX_DAILY_PCT；默认 0.50）
	// CarryWhitelist 白名单生息资产（sUSDe/USDe 等，M3-b §9.6）。
	// ARBCN_SIM_CARRY_WHITELIST 逗号分隔；**默认空** = carry 被 WHITELIST 拒单直到显式
	// 配置（安全默认，宁缺毋滥，M3-a 复审 M2 信任边界落地）。
	CarryWhitelist []string
	// HistoryDays 历史 funding 回填/周报窗口天数（M3-b §9.5，ARBCN_SIM_HISTORY_DAYS）。
	// 0 = 禁用历史回填与周频报告。
	HistoryDays int
	// ReportPath sim_report 周频报告输出路径（M3-b §9.5；ARBCN_SIM_REPORT_PATH）。
	ReportPath string
}

// DefaultConfig 返回定稿默认值（04-m3-spec §4 表 + M3-b §9.5/§9.6 默认）。
func DefaultConfig() Config {
	return Config{
		Capital: 100_000, MinSpread: 5, MaxSizePct: 0.20, MaxDailyPct: 0.50,
		HistoryDays: 365, ReportPath: "docs/handoff/sim_report.md",
	}
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
	// 白名单（逗号分隔；默认空；大小写/空白归一 + 去重）。
	// [对抗测试锚点] §9.6：删去重 → config_test.go TestFromEnvCarryWhitelist 必红。
	if v := strings.TrimSpace(getenv("ARBCN_SIM_CARRY_WHITELIST")); v != "" {
		for _, p := range strings.Split(v, ",") {
			if p = strings.TrimSpace(p); p != "" && !slices.Contains(cfg.CarryWhitelist, p) {
				cfg.CarryWhitelist = append(cfg.CarryWhitelist, p)
			}
		}
	}
	// 历史窗口天数（0 = 禁用；负数视为 0）。
	if v := getenv("ARBCN_SIM_HISTORY_DAYS"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return Config{}, fmt.Errorf("sim: ARBCN_SIM_HISTORY_DAYS: %q: %w", v, err)
		}
		if n < 0 {
			n = 0
		}
		cfg.HistoryDays = n
	}
	// 周报输出路径。
	if v := strings.TrimSpace(getenv("ARBCN_SIM_REPORT_PATH")); v != "" {
		cfg.ReportPath = v
	}
	if cfg.Capital <= 0 || cfg.MinSpread <= 0 || cfg.MaxSizePct <= 0 || cfg.MaxDailyPct <= 0 {
		return Config{}, fmt.Errorf("sim: ARBCN_SIM_* 必须为正数（capital=%v min_spread=%v max_size=%v max_daily=%v）",
			cfg.Capital, cfg.MinSpread, cfg.MaxSizePct, cfg.MaxDailyPct)
	}
	return cfg, nil
}
