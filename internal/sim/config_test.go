package sim

import (
	"slices"
	"testing"
)

// TestFromEnvDefaults：未设置任何 ARBCN_SIM_* → 定稿默认值（§4 表 + M3-b §9.5/§9.6）。
func TestFromEnvDefaults(t *testing.T) {
	cfg, err := FromEnv(func(string) string { return "" })
	if err != nil {
		t.Fatalf("FromEnv(empty): %v", err)
	}
	d := DefaultConfig()
	if cfg.Capital != d.Capital || cfg.MinSpread != d.MinSpread ||
		cfg.MaxSizePct != d.MaxSizePct || cfg.MaxDailyPct != d.MaxDailyPct ||
		cfg.HistoryDays != d.HistoryDays || cfg.ReportPath != d.ReportPath ||
		!slices.Equal(cfg.CarryWhitelist, d.CarryWhitelist) {
		t.Fatalf("cfg = %+v, want default %+v", cfg, d)
	}
	if cfg.Capital != 100_000 || cfg.MinSpread != 5 || cfg.MaxSizePct != 0.20 || cfg.MaxDailyPct != 0.50 {
		t.Fatalf("default values drift: %+v", cfg)
	}
	// 安全默认：白名单默认空（carry 先被 WHITELIST 拒单，§9.6）；历史窗口默认 365d（§9.5）。
	if len(cfg.CarryWhitelist) != 0 || cfg.HistoryDays != 365 {
		t.Fatalf("安全默认漂移：whitelist=%v history_days=%d", cfg.CarryWhitelist, cfg.HistoryDays)
	}
}

// TestFromEnvOverrides：ARBCN_SIM_* 逐项覆盖生效。
func TestFromEnvOverrides(t *testing.T) {
	env := map[string]string{
		"ARBCN_SIM_CAPITAL":      "200000",
		"ARBCN_SIM_MIN_SPREAD":   "6",
		"ARBCN_SIM_MAX_SIZE_PCT": "0.10",
		"ARBCN_SIM_MAX_DAILY_PCT": "0.40",
	}
	cfg, err := FromEnv(func(k string) string { return env[k] })
	if err != nil {
		t.Fatalf("FromEnv: %v", err)
	}
	if cfg.Capital != 200_000 || cfg.MinSpread != 6 || cfg.MaxSizePct != 0.10 || cfg.MaxDailyPct != 0.40 {
		t.Fatalf("cfg = %+v, want overridden values", cfg)
	}
}

// TestFromEnvPartial：只覆盖一项，其余保持默认。
func TestFromEnvPartial(t *testing.T) {
	cfg, err := FromEnv(func(k string) string {
		if k == "ARBCN_SIM_CAPITAL" {
			return "50000"
		}
		return ""
	})
	if err != nil {
		t.Fatalf("FromEnv(partial): %v", err)
	}
	if cfg.Capital != 50_000 || cfg.MinSpread != 5 {
		t.Fatalf("cfg = %+v, want capital=50000 min_spread=5（默认）", cfg)
	}
}

// TestFromEnvInvalid：非法数值 → 错误（配置错误 fail fast；调用方可按 §7 降级禁用 sim）。
func TestFromEnvInvalid(t *testing.T) {
	if _, err := FromEnv(func(k string) string {
		if k == "ARBCN_SIM_CAPITAL" {
			return "abc"
		}
		return ""
	}); err == nil {
		t.Fatal("FromEnv(capital=abc) = nil, want error")
	}
	if _, err := FromEnv(func(k string) string {
		if k == "ARBCN_SIM_MAX_SIZE_PCT" {
			return "0"
		}
		return ""
	}); err == nil {
		t.Fatal("FromEnv(max_size=0) = nil, want error（非正数拒绝）")
	}
	if _, err := FromEnv(func(k string) string {
		if k == "ARBCN_SIM_HISTORY_DAYS" {
			return "abc"
		}
		return ""
	}); err == nil {
		t.Fatal("FromEnv(history_days=abc) = nil, want error")
	}
}

// TestFromEnvCarryWhitelist：[对抗测试锚点 §9.6] 白名单逗号分隔解析 + 大小写/空白归一；
// 默认空（安全默认，carry 先被 WHITELIST 拒单）。删 FromEnv 白名单解析 → 本测试必红。
func TestFromEnvCarryWhitelist(t *testing.T) {
	env := map[string]string{"ARBCN_SIM_CARRY_WHITELIST": " sUSDe , USDe, sUSDe "}
	cfg, err := FromEnv(func(k string) string { return env[k] })
	if err != nil {
		t.Fatalf("FromEnv: %v", err)
	}
	if !slices.Equal(cfg.CarryWhitelist, []string{"sUSDe", "USDe"}) {
		t.Fatalf("CarryWhitelist = %v, want [sUSDe USDe]（去重 + 归一）", cfg.CarryWhitelist)
	}

	// 默认空。
	cfg0, err := FromEnv(func(string) string { return "" })
	if err != nil {
		t.Fatalf("FromEnv(empty): %v", err)
	}
	if len(cfg0.CarryWhitelist) != 0 {
		t.Fatalf("CarryWhitelist 默认 = %v, want 空（安全默认）", cfg0.CarryWhitelist)
	}
}

// TestFromEnvHistoryDays：ARBCN_SIM_HISTORY_DAYS 覆盖（0 = 禁用；负值视为 0）。
func TestFromEnvHistoryDays(t *testing.T) {
	cfg, err := FromEnv(func(k string) string {
		if k == "ARBCN_SIM_HISTORY_DAYS" {
			return "180"
		}
		return ""
	})
	if err != nil {
		t.Fatalf("FromEnv: %v", err)
	}
	if cfg.HistoryDays != 180 {
		t.Fatalf("HistoryDays = %d, want 180", cfg.HistoryDays)
	}
	cfg0, err := FromEnv(func(k string) string {
		if k == "ARBCN_SIM_HISTORY_DAYS" {
			return "0"
		}
		return ""
	})
	if err != nil || cfg0.HistoryDays != 0 {
		t.Fatalf("HistoryDays(0) = %d, %v, want 0（禁用）", cfg0.HistoryDays, err)
	}
}
