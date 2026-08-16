package rule

import (
	"context"
	"testing"
	"time"

	"arbcn/internal/fact"
	"arbcn/internal/store"
)

// defaultByName 从 Defaults 取指定规则（§7 首版 11 条逐条测试）。
func defaultByName(t *testing.T, name string) store.Rule {
	t.Helper()
	for _, r := range Defaults() {
		if r.Name == name {
			return r
		}
	}
	t.Fatalf("default rule %q not found", name)
	return store.Rule{}
}

// ivHistory 30 日每日 IV 历史（40..69），p25_30d = 47（31 样本含最新值 45 时）。
func ivHistory() []fact.Fact {
	out := make([]fact.Fact, 0, 30)
	for i := 0; i < 30; i++ {
		out = append(out, fct(fact.KindIV, "deribit", "BTC", float64(40+i), -time.Duration(30-i)*24*time.Hour))
	}
	return out
}

// TestEachDefaultFiresOnSyntheticFacts：§11① 合成 fact 序列 → 11 规则各至少一条
// 正例（用 Defaults 原样规则，阈值按 §7 表 + D-041 演练档）。
func TestEachDefaultFiresOnSyntheticFacts(t *testing.T) {
	cases := []struct {
		rule    string
		facts   []fact.Fact
		level   string
		message string
	}{
		{
			rule: "funding_warn",
			facts: []fact.Fact{
				fct(fact.KindFunding, "binance", "BTC", 16.25, -time.Hour),
				fct(fact.KindFunding, "okx", "BTC", 16.75, -2*time.Hour),
			},
			level:   store.LevelWarn,
			message: "资金费率预警 触发: BTC@binance=16.25, BTC@okx=16.75",
		},
		{
			rule: "funding_critical",
			facts: []fact.Fact{
				fct(fact.KindFunding, "binance", "BTC", 21, -time.Hour),
				fct(fact.KindFunding, "binance", "ETH", 21.5, -2*time.Hour),
			},
			level:   store.LevelCritical,
			message: "资金费率激活 触发: BTC@binance=21, ETH@binance=21.5",
		},
		{
			rule: "funding_drill",
			facts: []fact.Fact{
				fct(fact.KindFunding, "binance", "BTC", 7.5, -time.Hour),
				fct(fact.KindFunding, "okx", "BTC", 6.8, -2*time.Hour),
			},
			level:   store.LevelInfo,
			message: "资金费率演练档 触发: BTC@binance=7.5, BTC@okx=6.8",
		},
		{
			rule: "trx_funding_positive",
			facts: []fact.Fact{
				fct(fact.KindFunding, "binance", "TRX", 1, -time.Hour),
				fct(fact.KindFunding, "binance", "TRX", 0.5, -2*time.Hour),
				fct(fact.KindFunding, "binance", "TRX", -2, -30*time.Hour),
				fct(fact.KindFunding, "binance", "TRX", -1, -40*time.Hour),
			},
			level:   store.LevelWarn,
			message: "TRX 费率转正 触发: TRX@binance=0.75",
		},
		{
			rule: "defi_large_tier_change",
			facts: []fact.Fact{
				fct(fact.KindDefiRate, "aave", "USDC", 4.67, -10*time.Minute),
				fct(fact.KindDefiRate, "aave", "USDC", 4.0, -40*time.Minute),
			},
			level:   store.LevelInfo,
			message: "金额档利率变动 触发: USDC@aave=0.67",
		},
		{
			rule: "ladder_trap",
			facts: []fact.Fact{
				fct(fact.KindDefiRate, "binance_ear", "USDT_H", 9.8, -24*time.Hour),
				fct(fact.KindDefiRate, "binance_ear", "USDT_L", 2.0, -24*time.Hour),
			},
			level:   store.LevelWarn,
			message: "阶梯陷阱识别 触发: 9.8",
		},
		{
			rule: "reverse_repo_timing",
			facts: []fact.Fact{
				fct(fact.KindCalendar, "rule", "thursday", 1, -30*time.Minute),
				fct(fact.KindCalendar, "rule", "month_end", 12, -30*time.Minute),
			},
			level:   store.LevelWarn,
			message: "逆回购时点 触发: thursday@rule=1",
		},
		{
			rule:    "usdcnh_buy_line",
			facts:   []fact.Fact{fct(fact.KindFX, "sina", "USDCNH", 6.55, -10*time.Minute)},
			level:   store.LevelWarn,
			message: "汇率加仓线 触发: USDCNH@sina=6.55",
		},
		{
			rule:    "iv_opportunity",
			facts:   append(ivHistory(), fct(fact.KindIV, "deribit", "BTC", 45, -30*time.Minute)),
			level:   store.LevelInfo,
			message: "IV 机会 触发: BTC@deribit=45",
		},
		{
			rule: "nonstable_quote_change",
			facts: []fact.Fact{
				fct(fact.KindDefiRate, "morpho", "WETH", 2.1, -10*time.Minute),
				fct(fact.KindDefiRate, "morpho", "WETH", 1.5, -40*time.Minute),
			},
			level:   store.LevelWarn,
			message: "计价币种陷阱 触发: WETH@morpho=0.6",
		},
		{
			rule: "collector_heartbeat",
			facts: []fact.Fact{
				fct(fact.KindHeartbeat, "collector", "binance_funding", 3, -time.Minute),
				fct(fact.KindHeartbeat, "collector", "deribit_iv", 0.5, -time.Minute),
			},
			level:   store.LevelCritical,
			message: "采集器心跳 触发: binance_funding@collector=3",
		},
	}

	for _, c := range cases {
		t.Run(c.rule, func(t *testing.T) {
			st := newFakeStore([]store.Rule{defaultByName(t, c.rule)}, c.facts)
			e, err := New(context.Background(), st, Config{Now: func() time.Time { return t0 }})
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			if err := e.EvaluateAll(context.Background()); err != nil {
				t.Fatalf("EvaluateAll: %v", err)
			}
			alerts := st.alertsCopy()
			if len(alerts) != 1 {
				t.Fatalf("alerts = %+v, want 1", alerts)
			}
			if alerts[0].Level != c.level || alerts[0].Message != c.message {
				t.Errorf("alert = %q/%q, want %q/%q", alerts[0].Level, alerts[0].Message, c.level, c.message)
			}
			if !alerts[0].Ts.Equal(t0) {
				t.Errorf("alert Ts = %v, want %v", alerts[0].Ts, t0)
			}
			s, err := st.GetTriggerState(context.Background(), 1)
			if err != nil || s.State != store.StateActive {
				t.Errorf("state = %+v, %v, want active", s, err)
			}
		})
	}
}

// TestDefaultCondsParse：Defaults 全部条件串可解析（New 级校验的兜底快速通道）。
func TestDefaultCondsParse(t *testing.T) {
	for _, r := range Defaults() {
		if _, err := parseCond(r.Cond); err != nil {
			t.Errorf("default %q cond %q: %v", r.Name, r.Cond, err)
		}
	}
}
