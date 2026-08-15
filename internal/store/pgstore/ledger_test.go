package pgstore

import (
	"context"
	"testing"
	"time"

	"arbcn/internal/store"
)

// TestLedgerRoundtrip：台账写入 → date DESC 读回 → 字段逐项比对；非法行拒绝。
func TestLedgerRoundtrip(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	ensureSchema(t, ctx, pool)
	resetTables(t, ctx, pool, "ledger")

	s := New(pool)
	now := time.Now().UTC().Truncate(time.Microsecond)
	entries := []store.LedgerEntry{
		{Date: now.Add(2 * time.Hour), Channel: "binance", Currency: "USDT", Amount: 1000, FeeRate: 0.1, Tier: store.TierStableBase, Note: "入金"},
		{Date: now.Add(time.Hour), Channel: "okx", Currency: "USDC", Amount: -200, FeeRate: 0, Tier: store.TierStableBase, Note: "出金"},
		{Date: now, Channel: "民营定期", Currency: "RMB", Amount: 50000, FeeRate: 0, Tier: store.TierProtectedConvexity},
	}
	ids := make([]int64, 0, len(entries))
	for i, e := range entries {
		id, err := s.InsertLedgerEntry(ctx, e)
		if err != nil {
			t.Fatalf("InsertLedgerEntry(%d): %v", i, err)
		}
		if id <= 0 {
			t.Fatalf("InsertLedgerEntry(%d) id = %d, want > 0", i, id)
		}
		ids = append(ids, id)
	}

	got, err := s.ListLedgerEntries(ctx, 0, 0)
	if err != nil {
		t.Fatalf("ListLedgerEntries: %v", err)
	}
	if len(got) != 3 {
		t.Fatalf("entries len = %d, want 3", len(got))
	}
	// date DESC：entries[0] = now+2h。
	if got[0].ID != ids[0] || !got[0].Date.Equal(now.Add(2*time.Hour)) ||
		got[0].Channel != "binance" || got[0].Currency != "USDT" || got[0].Amount != 1000 ||
		got[0].FeeRate != 0.1 || got[0].Tier != store.TierStableBase || got[0].Note != "入金" {
		t.Errorf("got[0] = %+v", got[0])
	}
	if got[2].Tier != store.TierProtectedConvexity || got[2].Currency != "RMB" {
		t.Errorf("got[2] = %+v", got[2])
	}

	// 分页：LIMIT 1 OFFSET 1 → 第二条（now+1h）。
	page, err := s.ListLedgerEntries(ctx, 1, 1)
	if err != nil {
		t.Fatalf("ListLedgerEntries(page): %v", err)
	}
	if len(page) != 1 || page[0].ID != ids[1] {
		t.Errorf("page = %+v, want 1 条 id=%d", page, ids[1])
	}

	// 非法行拒绝：date 零值 / channel 空 / currency 空。
	for name, bad := range map[string]store.LedgerEntry{
		"date 零值":  {Channel: "a", Currency: "USDT", Amount: 1},
		"channel 空": {Date: now, Currency: "USDT", Amount: 1},
		"currency 空": {Date: now, Channel: "a", Amount: 1},
	} {
		if _, err := s.InsertLedgerEntry(ctx, bad); err == nil {
			t.Errorf("%s: InsertLedgerEntry = nil, want error", name)
		}
	}
}

// TestLedgerSummaryByTier：GROUP BY tier 归因汇总——入金和/出金和/净额/笔数。
func TestLedgerSummaryByTier(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	ensureSchema(t, ctx, pool)
	resetTables(t, ctx, pool, "ledger")

	s := New(pool)
	now := time.Now().UTC()
	for _, e := range []store.LedgerEntry{
		{Date: now, Channel: "a", Currency: "USDT", Amount: 1000, Tier: store.TierStableBase},
		{Date: now, Channel: "b", Currency: "USDT", Amount: -200, Tier: store.TierStableBase},
		{Date: now, Channel: "c", Currency: "RMB", Amount: 50000, Tier: store.TierProtectedConvexity},
		{Date: now, Channel: "d", Currency: "RMB", Amount: -500, Tier: store.TierProtectedConvexity},
		{Date: now, Channel: "e", Currency: "RMB", Amount: 1000, Tier: ""}, // 未分类
	} {
		if _, err := s.InsertLedgerEntry(ctx, e); err != nil {
			t.Fatalf("InsertLedgerEntry: %v", err)
		}
	}

	sums, err := s.LedgerSummary(ctx)
	if err != nil {
		t.Fatalf("LedgerSummary: %v", err)
	}
	if len(sums) != 3 {
		t.Fatalf("summary len = %d, want 3（stable_base/protected_convexity/空）", len(sums))
	}
	byTier := map[string]store.TierSummary{}
	for _, ts := range sums {
		byTier[ts.Tier] = ts
	}
	stable := byTier[store.TierStableBase]
	if stable.Inflow != 1000 || stable.Outflow != 200 || stable.Net != 800 || stable.EntryCount != 2 {
		t.Errorf("stable_base = %+v, want inflow 1000/outflow 200/net 800/2 笔", stable)
	}
	prot := byTier[store.TierProtectedConvexity]
	if prot.Inflow != 50000 || prot.Outflow != 500 || prot.Net != 49500 || prot.EntryCount != 2 {
		t.Errorf("protected_convexity = %+v, want inflow 50000/outflow 500/net 49500/2 笔", prot)
	}
	uncat := byTier[""]
	if uncat.Net != 1000 || uncat.EntryCount != 1 {
		t.Errorf("未分类 = %+v, want net 1000/1 笔", uncat)
	}

	// 存储层故障注入路径不存在（PG 直连），但空表应返回空切片而非 nil 错误。
	resetTables(t, ctx, pool, "ledger")
	sums, err = s.LedgerSummary(ctx)
	if err != nil || len(sums) != 0 {
		t.Fatalf("empty summary = %v, %v, want 空/nil", sums, err)
	}
}

// TestLedgerUnknownTierFreeText：tier 为自由 TEXT（演进预留，不设 CHECK）——
// 任意非空 tier 可写入，归因按其分组。
func TestLedgerUnknownTierFreeText(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	ensureSchema(t, ctx, pool)
	resetTables(t, ctx, pool, "ledger")

	s := New(pool)
	if _, err := s.InsertLedgerEntry(ctx, store.LedgerEntry{
		Date: time.Now(), Channel: "x", Currency: "RMB", Amount: 100, Tier: "未来新档位",
	}); err != nil {
		t.Fatalf("InsertLedgerEntry(free tier): %v", err)
	}
	sums, err := s.LedgerSummary(ctx)
	if err != nil || len(sums) != 1 || sums[0].Tier != "未来新档位" {
		t.Fatalf("summary = %+v, %v", sums, err)
	}
}
