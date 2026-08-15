// M3-b §9.5 历史回填幂等辅助（纯函数，零 I/O）。
// 回填前 QueryFacts 取已有结算时刻集合 → 跳过已覆盖时段（跑两遍不重复）。
package sim

import "arbcn/internal/fact"

// UncoveredFacts 过滤掉批次中已被既有事实覆盖的 (venue,symbol,结算时刻) 的事实
// （幂等回填跳过依据，§9.5）。键 = (Venue, Symbol, Ts UnixMilli)：同一交易所不同
// symbol 的结算时刻相同（如 Binance 全 USDT-M 合约 00:00/08:00/16:00 同时结算），
// 仅按 ts 判重会把同刻的其他 symbol 误判为已覆盖。落库走 InsertFacts（既有管线）。
//
// [对抗测试锚点] §9.5/§9.8：删除覆盖跳过 → sim/history_test.go
// TestUncoveredFactsSkipsCovered（跑两遍不重复断言）必红。
func UncoveredFacts(existing, batch []fact.Fact) []fact.Fact {
	type key struct{ venue, symbol string; ts int64 }
	covered := make(map[key]bool, len(existing))
	for _, e := range existing {
		covered[key{e.Venue, e.Symbol, e.Ts.UnixMilli()}] = true
	}
	out := make([]fact.Fact, 0, len(batch))
	for _, f := range batch {
		if !covered[key{f.Venue, f.Symbol, f.Ts.UnixMilli()}] {
			out = append(out, f)
		}
	}
	return out
}
