package sim

import (
	"testing"
)

// TestFromEnvDefaults：未设置任何 ARBCN_SIM_* → 定稿默认值（§4 表）。
func TestFromEnvDefaults(t *testing.T) {
	cfg, err := FromEnv(func(string) string { return "" })
	if err != nil {
		t.Fatalf("FromEnv(empty): %v", err)
	}
	d := DefaultConfig()
	if cfg != d {
		t.Fatalf("cfg = %+v, want default %+v", cfg, d)
	}
	if cfg.Capital != 100_000 || cfg.MinSpread != 5 || cfg.MaxSizePct != 0.20 || cfg.MaxDailyPct != 0.50 {
		t.Fatalf("default values drift: %+v", cfg)
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
}
