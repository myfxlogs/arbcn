// M3-b §9.5/§9.7：历史 funding 回填编排（幂等，一次性，阻塞至完成）。
// 本文件零网络零密钥（§9.4）：只依赖 store 与本地抽象 HistoryCollector，具体数据源
// （internal/collect/exchange 的 data-api 实现）由 main.go 注入——sim 包不 import 网络包。
package sim

import (
	"context"
	"fmt"
	"time"

	"arbcn/internal/fact"
	"arbcn/internal/store"
)

// HistoryCollector 是历史回填数据源的最小抽象（避免 sim 依赖网络实现）。
// main.go 注入 exchange.NewBinanceFundingHistory / NewOKXFundingHistory。
type HistoryCollector interface {
	// Venue 返回产出事实的 venue（binance / okx），用于幂等查询。
	Venue() string
	// Poll 翻页拉满 HistoryDays 窗口内的历史 funding，产 fact{Kind=funding, Venue, Symbol, Ts=结算时刻}。
	Poll(ctx context.Context) ([]fact.Fact, error)
}

// BackfillHistory 一次性幂等回填：对每个 collector 拉取窗口内历史 → QueryFacts 取已有
// (venue,symbol,ts) 集合 → UncoveredFacts 跳过已覆盖 → InsertFacts 落库。跑两遍不重复。
// days <= 0 = 禁用（返回 nil）。单 collector 失败 → 整体失败（回填是 boot 一次性任务，
// 数据源挂了应显式报错；但由调用方 warn 处理，不退出进程——§7/D-032 同口径）。
//
// [对抗测试锚点] §9.5/§9.8：删除 UncoveredFacts 覆盖跳过 → history_test.go
// TestUncoveredFactsSkipsCovered / TestBackfillHistoryIdempotent 必红。
func BackfillHistory(ctx context.Context, st store.Store, collectors []HistoryCollector, days int) error {
	if days <= 0 {
		return nil
	}
	from := time.Now().Add(-time.Duration(days) * 24 * time.Hour)
	for _, c := range collectors {
		batch, err := c.Poll(ctx)
		if err != nil {
			return fmt.Errorf("sim backfill %s: poll: %w", c.Venue(), err)
		}
		if len(batch) == 0 {
			continue
		}
		existing, err := st.QueryFacts(ctx, store.FactQuery{
			Kind: fact.KindFunding, Venue: c.Venue(), From: from, Limit: 500_000,
		})
		if err != nil {
			return fmt.Errorf("sim backfill %s: query existing: %w", c.Venue(), err)
		}
		toInsert := UncoveredFacts(existing, batch)
		if len(toInsert) == 0 {
			continue
		}
		if err := st.InsertFacts(ctx, toInsert); err != nil {
			return fmt.Errorf("sim backfill %s: insert %d facts: %w", c.Venue(), len(toInsert), err)
		}
	}
	return nil
}
