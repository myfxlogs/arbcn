package sim

import (
	"testing"
	"time"

	"arbcn/internal/fact"
)

// TestUncoveredFactsSkipsCovered：[对抗测试锚点 §9.5 S4] 回填幂等——已覆盖结算时刻跳过。
// 模拟"跑两遍不重复"：第二遍 existing 含第一遍落库的全部结算时刻 → UncoveredFacts 返回空。
// 删 UncoveredFacts 的 covered 跳过 → 本测试必红。
func TestUncoveredFactsSkipsCovered(t *testing.T) {
	ts := func(h int) time.Time { return t0.Add(time.Duration(h) * 8 * time.Hour) }
	// 第一遍：existing 空 → 全部 uncovered（3 条）。
	first := []fact.Fact{
		{Kind: fact.KindFunding, Venue: "binance", Symbol: "BTC", Value: 10.95, Ts: ts(0), Src: "data-api"},
		{Kind: fact.KindFunding, Venue: "binance", Symbol: "BTC", Value: 10.5, Ts: ts(1), Src: "data-api"},
		{Kind: fact.KindFunding, Venue: "binance", Symbol: "BTC", Value: 11.4, Ts: ts(2), Src: "data-api"},
	}
	got1 := UncoveredFacts(nil, first)
	if len(got1) != 3 {
		t.Fatalf("第一遍 uncovered = %d, want 3（全量回填）", len(got1))
	}

	// 第二遍：existing = 第一遍已落库（含 ts(0)/ts(1)/ts(2)），批同样 3 条 → 0 条 uncovered。
	got2 := UncoveredFacts(first, first)
	if len(got2) != 0 {
		t.Fatalf("第二遍 uncovered = %d, want 0（已覆盖全部结算时刻，跑两遍不重复）", len(got2))
	}

	// 部分覆盖：批含 1 条新结算时刻 → 只返回它。
	batch := []fact.Fact{first[0], {Kind: fact.KindFunding, Venue: "binance", Symbol: "BTC", Value: 12.0, Ts: ts(3), Src: "data-api"}}
	got3 := UncoveredFacts(first, batch)
	if len(got3) != 1 || !got3[0].Ts.Equal(ts(3)) {
		t.Fatalf("部分覆盖 uncovered = %+v, want 仅 ts(3)", got3)
	}
}

// TestUncoveredFactsCrossSymbol：同刻不同 symbol 不互相误判（Binance 全合约同时结算，
// 仅按 ts 判重会误删同刻的 ETH）。键必须是 (venue,symbol,ts)。
func TestUncoveredFactsCrossSymbol(t *testing.T) {
	ts := t0.Add(8 * time.Hour)
	existing := []fact.Fact{{Kind: fact.KindFunding, Venue: "binance", Symbol: "BTC", Value: 10.95, Ts: ts, Src: "data-api"}}
	batch := []fact.Fact{
		{Kind: fact.KindFunding, Venue: "binance", Symbol: "BTC", Value: 10.95, Ts: ts, Src: "data-api"}, // 已覆盖
		{Kind: fact.KindFunding, Venue: "binance", Symbol: "ETH", Value: 11.5, Ts: ts, Src: "data-api"},  // 同刻未覆盖
	}
	got := UncoveredFacts(existing, batch)
	if len(got) != 1 || got[0].Symbol != "ETH" {
		t.Fatalf("uncovered = %+v, want 仅 ETH（同刻 BTC 已覆盖，ETH 未覆盖）", got)
	}
}
