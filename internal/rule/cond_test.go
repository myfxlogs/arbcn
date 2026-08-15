package rule

import (
	"strings"
	"testing"
	"time"

	"arbcn/internal/fact"
)

// TestParseCondOK：文法全量正例（阈值/穿越/环比/分位/缩放聚合/显式 scope）。
func TestParseCondOK(t *testing.T) {
	cases := []string{
		"avg_30d > 15",
		"last_24h < 6.6",
		"avg_24h > 0 && avg_48h@24h <= 0",
		"chg_1h >= 0.5 || chg_1h <= -0.5",
		"last_24h < p25_30d",
		"avg_7d(binance_ear, USDT_H) > 3 * avg_7d(binance_ear, USDT_L)",
		"last_15m <= 1",
	}
	for _, s := range cases {
		if _, err := parseCond(s); err != nil {
			t.Errorf("parseCond(%q) = %v, want nil", s, err)
		}
	}
}

// TestParseCondRejected：坏表达式启动即报错（§4；§11④）。
func TestParseCondRejected(t *testing.T) {
	cases := []string{
		"",                // 空
		"avg_30d",         // 缺比较
		"avg_30d >",       // 缺右操作数
		"> 15",            // 缺左操作数
		"avg_30d > 15 &&", // 悬空 &&
		"avg_30d > 15 ||", // 悬空 ||
		"avg_30x > 15",    // 非法时长单位
		"avg_-5d > 15",    // 负时长
		"avg_0d > 15",     // 零时长
		"foo_30d > 15",    // 未知聚合
		"avg_30d == 15",   // 未知比较符
		"avg_30d > 1 + 2", // 不支持算术
		"avg_30d(binance_ear, USDT_H) > 3 * avg_7d",              // 显式/隐式 scope 混用
		"avg_30d(binance_ear) > 3 * avg_7d(binance_ear, USDT_L)", // scope 缺 symbol
	}
	for _, s := range cases {
		if _, err := parseCond(s); err == nil {
			t.Errorf("parseCond(%q) = nil, want error", s)
		}
	}
}

// TestParseDur：时长解析边界。
func TestParseDur(t *testing.T) {
	ok := map[string]time.Duration{
		"15m": 15 * time.Minute, "24h": 24 * time.Hour, "30d": 30 * 24 * time.Hour,
	}
	for s, want := range ok {
		got, err := parseDur(s)
		if err != nil || got != want {
			t.Errorf("parseDur(%q) = %v, %v, want %v", s, got, err, want)
		}
	}
	for _, s := range []string{"", "d", "0d", "-3d", "30", "30s", "1.5h"} {
		if _, err := parseDur(s); err == nil {
			t.Errorf("parseDur(%q) = nil, want error", s)
		}
	}
}

// TestAggregates：avg/last/p25/chg/窗口前移/显式 scope 的求值语义。
func TestAggregates(t *testing.T) {
	now := time.Now()
	// 输入必须 ts 升序（QueryFacts 保证；此处按组分开验证）。
	binance := []fact.Fact{
		{Kind: fact.KindFunding, Venue: "binance", Symbol: "BTC", Value: 1, Ts: now.Add(-3 * time.Hour)},
		{Kind: fact.KindFunding, Venue: "binance", Symbol: "BTC", Value: 2, Ts: now.Add(-2 * time.Hour)},
		{Kind: fact.KindFunding, Venue: "binance", Symbol: "BTC", Value: 3, Ts: now.Add(-1 * time.Hour)},
	}
	okx := []fact.Fact{
		{Kind: fact.KindFunding, Venue: "okx", Symbol: "BTC", Value: 150, Ts: now.Add(-40 * time.Minute)},
		{Kind: fact.KindFunding, Venue: "okx", Symbol: "BTC", Value: 100, Ts: now.Add(-30 * time.Minute)},
	}

	if v, ok := aggValue(aggregate{kind: aggAvg, window: time.Hour}, binance, now); !ok || v != 3 {
		t.Errorf("avg_1h(binance) = %v, %v, want 3", v, ok)
	}
	if v, ok := aggValue(aggregate{kind: aggLast, window: time.Hour}, binance, now); !ok || v != 3 {
		t.Errorf("last_1h(binance) = %v, %v, want 3", v, ok)
	}
	// 显式 scope：只看 okx 组。
	if v, ok := aggValue(aggregate{kind: aggAvg, window: time.Hour, venue: "okx", symbol: "BTC"}, okx, now); !ok || v != 125 {
		t.Errorf("avg_1h(okx,BTC) = %v, %v, want 125", v, ok)
	}
	// 窗口前移 [−3h,−1h)：只含 −3h、−2h 两点，均值 1.5。
	if v, ok := aggValue(aggregate{kind: aggAvg, window: 2 * time.Hour, offset: time.Hour}, binance, now); !ok || v != 1.5 {
		t.Errorf("avg_2h@1h = %v, %v, want 1.5", v, ok)
	}
	// p25 最近秩：1,2,3 → ceil(0.75)−1 = 0 → 值 1。
	if v, ok := aggValue(aggregate{kind: aggP25, window: 4 * time.Hour}, binance, now); !ok || v != 1 {
		t.Errorf("p25_4h = %v, %v, want 1", v, ok)
	}
	// chg：okx 最新 100 − 更早 150 = −50。
	if v, ok := aggValue(aggregate{kind: aggChg, window: time.Hour, venue: "okx", symbol: "BTC"}, okx, now); !ok || v != -50 {
		t.Errorf("chg_1h(okx,BTC) = %v, %v, want -50", v, ok)
	}
	// 无数据 → 假。
	if _, ok := aggValue(aggregate{kind: aggAvg, window: time.Hour}, nil, now); ok {
		t.Error("aggValue(empty) ok = true, want false")
	}
}

// TestEvalUndefinedIsFalse：任一操作数无数据 → 比较为假（无数据不告警）。
func TestEvalUndefinedIsFalse(t *testing.T) {
	now := time.Now()
	c, err := parseCond("avg_30d > 15")
	if err != nil {
		t.Fatal(err)
	}
	if ok, _ := c.eval(nil, now); ok {
		t.Error("eval(no facts) = true, want false")
	}
	// 窗口外的事实 → avg_30d 无数据 → 假。
	stale := []fact.Fact{{Kind: fact.KindFunding, Value: 16, Ts: now.Add(-31 * 24 * time.Hour)}}
	if ok, _ := c.eval(stale, now); ok {
		t.Error("eval(fact outside window) = true, want false")
	}
	// 缩放聚合与常数比较：3 * avg_7d(USDT_L) = 6 ≤ 6 → 真；代表值 = lhs（含系数）= 6。
	c2, err := parseCond("3 * avg_7d(binance_ear, USDT_L) <= 6")
	if err != nil {
		t.Fatal(err)
	}
	fs := []fact.Fact{{Kind: fact.KindDefiRate, Venue: "binance_ear", Symbol: "USDT_L", Value: 2, Ts: now.Add(-time.Hour)}}
	if ok, v := c2.eval(fs, now); !ok || v != 6 {
		t.Errorf("eval(3*avg<=6) = %v, %v, want true (代表值 6)", ok, v)
	}
}

// TestLookback：查询回溯覆盖窗口 + offset + chg 宽限。
func TestLookback(t *testing.T) {
	c, err := parseCond("avg_24h > 0 && avg_48h@24h <= 0")
	if err != nil {
		t.Fatal(err)
	}
	if got := c.lookback(); got != 72*time.Hour {
		t.Errorf("lookback(crossing) = %v, want 72h", got)
	}
	c, err = parseCond("chg_1h >= 0.5")
	if err != nil {
		t.Fatal(err)
	}
	if got := c.lookback(); got != 25*time.Hour {
		t.Errorf("lookback(chg_1h) = %v, want 25h", got)
	}
}

// TestScopedModeFlag：显式 scope 聚合置全局模式标记。
func TestScopedModeFlag(t *testing.T) {
	c, err := parseCond("avg_7d(binance_ear, USDT_H) > 3 * avg_7d(binance_ear, USDT_L)")
	if err != nil {
		t.Fatal(err)
	}
	if !c.scoped {
		t.Error("scoped cond not marked scoped")
	}
	c, err = parseCond("avg_30d > 15")
	if err != nil {
		t.Fatal(err)
	}
	if c.scoped {
		t.Error("unscoped cond marked scoped")
	}
}

// TestTokenize：括号/逗号切词，@与_保持粘连。
func TestTokenize(t *testing.T) {
	got := tokenize("avg_48h@24h(x, y) > 3 * last_30d")
	want := strings.Join([]string{"avg_48h@24h", "(", "x", ",", "y", ")", ">", "3", "*", "last_30d"}, " ")
	if strings.Join(got, " ") != want {
		t.Errorf("tokenize = %v, want %v", got, want)
	}
}
