package dashboard

import (
	"context"
	"math"
	"testing"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	dashboardv1 "arbcn/internal/dashboard/gen/arbcn/dashboard/v1"
	"arbcn/internal/fact"
	"arbcn/internal/rmb"
	"arbcn/internal/store"
)

// TestListFactsRMBConversion：事实快照 + RMB 折算（M2-b §4/§5）。
// 覆盖 kind（funding）× 当日 USDCNH → RMBValue = Value − 年化升值；非覆盖 kind 原样；
// 汇率可用时 FxRate/FxAvailable 回填、原始 Value 不被改写（不污染）。
func TestListFactsRMBConversion(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	// D-023 现实场景：30 天窗口 USDCNH 7.25→7.232 ≈ 年化升值 ~3 个百分点，
	// 稳定币 6% 收益率 → RMB 净 ~3%（6 − 3）。刻度：appreciation 与 Value 同点数（R6#1）。
	st := &fakeStore{facts: []fact.Fact{
		{Kind: fact.KindFX, Venue: "sina", Symbol: "USDCNH", Value: 7.25, Ts: now.Add(-30 * 24 * time.Hour)},
		{Kind: fact.KindFX, Venue: "sina", Symbol: "USDCNH", Value: 7.232, Ts: now.Add(-time.Minute)},
		{Kind: fact.KindFunding, Venue: "binance", Symbol: "BTC", Value: 6.0, Unit: fact.UnitPctAnnualized, Ts: now.Add(-time.Minute)},
		{Kind: fact.KindReverseRepo, Venue: "sse", Symbol: "GC001", Value: 2.0, Ts: now.Add(-time.Minute)}, // RMB 计价，不折算
	}}
	svc := New(st, nil, nil, nil)
	svc.Now = func() time.Time { return now }
	client := newTestServer(t, svc)
	ctx := context.Background()

	resp, err := client.ListFacts(ctx, connect.NewRequest(&dashboardv1.ListFactsRequest{}))
	if err != nil {
		t.Fatalf("ListFacts: %v", err)
	}
	msg := resp.Msg
	if !msg.FxAvailable || math.Abs(msg.FxRate-7.232) > 1e-9 || msg.FxTs.AsTime() != now.Add(-time.Minute) {
		t.Errorf("fx 快照 = %+v, want available/7.232/ts", msg)
	}
	byKind := map[string]*dashboardv1.FactRmb{}
	for _, f := range msg.Facts {
		byKind[f.Kind] = f
	}

	funding := byKind[fact.KindFunding]
	if funding == nil {
		t.Fatal("缺少 funding 事实")
	}
	if funding.Value != 6.0 {
		t.Errorf("funding.Value = %v, want 6.0（原始值不污染）", funding.Value)
	}
	want := 6.0 - rmb.AnnualizedRMBAppreciation([]fact.Fact{
		{Ts: now.Add(-30 * 24 * time.Hour), Value: 7.25},
		{Ts: now.Add(-time.Minute), Value: 7.232},
	})
	if math.Abs(funding.RmbValue-want) > 1e-6 {
		t.Errorf("funding.RmbValue = %v, want %v（USD 收益率 − 年化升值）", funding.RmbValue, want)
	}
	if !funding.FxAvailable || funding.FxRate != 7.232 {
		t.Errorf("funding fx = %v/%v, want available/7.232", funding.FxAvailable, funding.FxRate)
	}

	repo := byKind[fact.KindReverseRepo]
	if repo == nil {
		t.Fatal("缺少 reverse_repo 事实")
	}
	if repo.RmbValue != 2.0 || repo.FxAvailable {
		t.Errorf("reverse_repo = %v/%v, want 原样 2.0/false（RMB 计价不折算）", repo.RmbValue, repo.FxAvailable)
	}
}

// TestListFactsExcludesHeartbeat：[对抗测试锚点] 快照投影排除 heartbeat 内部遥测
//（与 exporter skipKinds 同语义）——删 ListFacts 里 excludeHeartbeat 一行 → 本测试必红。
func TestListFactsExcludesHeartbeat(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	st := &fakeStore{facts: []fact.Fact{
		{Kind: fact.KindHeartbeat, Venue: "collector", Symbol: "binance_funding", Value: 0.5, Ts: now},
		{Kind: fact.KindFunding, Venue: "binance", Symbol: "BTC", Value: 6.0, Ts: now},
	}}
	client := newTestServer(t, New(st, nil, nil, nil))
	resp, err := client.ListFacts(context.Background(), connect.NewRequest(&dashboardv1.ListFactsRequest{}))
	if err != nil {
		t.Fatalf("ListFacts: %v", err)
	}
	for _, f := range resp.Msg.Facts {
		if f.Kind == fact.KindHeartbeat {
			t.Errorf("快照含 heartbeat 内部遥测：%+v", f)
		}
	}
	if len(resp.Msg.Facts) != 1 {
		t.Errorf("facts = %d, want 1（heartbeat 已排除）", len(resp.Msg.Facts))
	}
}

// TestListFactsFXMissingDegrades：汇率缺失（无 fx 事实）→ 覆盖 kind 回退 USD 原值 +
// FxAvailable=false（前端「汇率不可用」），RPC 不崩（03-m2-spec §4 硬要求）。
func TestListFactsFXMissingDegrades(t *testing.T) {
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)
	st := &fakeStore{facts: []fact.Fact{
		{Kind: fact.KindFunding, Venue: "binance", Symbol: "BTC", Value: 6.0, Ts: now},
		{Kind: fact.KindDefiRate, Venue: "aave", Symbol: "USDC", Value: 4.5, Ts: now},
	}}
	svc := New(st, nil, nil, nil)
	svc.Now = func() time.Time { return now }
	client := newTestServer(t, svc)

	resp, err := client.ListFacts(context.Background(), connect.NewRequest(&dashboardv1.ListFactsRequest{}))
	if err != nil {
		t.Fatalf("ListFacts(fx missing): %v", err)
	}
	if resp.Msg.FxAvailable {
		t.Error("fx 快照 = available, want false（汇率缺失）")
	}
	for _, f := range resp.Msg.Facts {
		if f.RmbValue != f.Value || f.FxAvailable {
			t.Errorf("%s/%s = %v/%v, want USD 原值 %v/false", f.Kind, f.Symbol, f.RmbValue, f.FxAvailable, f.Value)
		}
	}
}

// TestAddLedgerEntry：手工录入台账（M2-b §6）——合法录入返回 id 并可读回；
// 缺 date/channel/currency / 非法金额 → InvalidArgument。
func TestAddLedgerEntry(t *testing.T) {
	st := &fakeStore{}
	client := newTestServer(t, New(st, nil, nil, nil))
	ctx := context.Background()

	date := t0.Add(-time.Hour)
	resp, err := client.AddLedgerEntry(ctx, connect.NewRequest(&dashboardv1.AddLedgerEntryRequest{
		Date:     tsPtr(date),
		Channel:  "binance",
		Currency: "USDT",
		Amount:   1000,
		FeeRate:  0.1,
		Tier:     store.TierStableBase,
		Note:     "M2-b 测试入金",
	}))
	if err != nil {
		t.Fatalf("AddLedgerEntry: %v", err)
	}
	if resp.Msg.Id <= 0 {
		t.Errorf("id = %d, want > 0", resp.Msg.Id)
	}
	entries, err := client.ListLedgerEntries(ctx, connect.NewRequest(&dashboardv1.ListLedgerEntriesRequest{}))
	if err != nil {
		t.Fatalf("ListLedgerEntries: %v", err)
	}
	if len(entries.Msg.Entries) != 1 {
		t.Fatalf("entries = %d, want 1", len(entries.Msg.Entries))
	}
	got := entries.Msg.Entries[0]
	if got.Channel != "binance" || got.Currency != "USDT" || got.Amount != 1000 ||
		got.Tier != store.TierStableBase || got.Note != "M2-b 测试入金" {
		t.Errorf("entry = %+v", got)
	}

	for _, tc := range []struct {
		name string
		req  *dashboardv1.AddLedgerEntryRequest
	}{
		{"缺 date", &dashboardv1.AddLedgerEntryRequest{Channel: "a", Currency: "USDT", Amount: 1}},
		{"缺 channel", &dashboardv1.AddLedgerEntryRequest{Date: tsPtr(t0), Currency: "USDT", Amount: 1}},
		{"缺 currency", &dashboardv1.AddLedgerEntryRequest{Date: tsPtr(t0), Channel: "a", Amount: 1}},
		{"NaN 金额", &dashboardv1.AddLedgerEntryRequest{Date: tsPtr(t0), Channel: "a", Currency: "USDT", Amount: math.NaN()}},
		{"零金额", &dashboardv1.AddLedgerEntryRequest{Date: tsPtr(t0), Channel: "a", Currency: "USDT", Amount: 0}},
		{"NaN 费率", &dashboardv1.AddLedgerEntryRequest{Date: tsPtr(t0), Channel: "a", Currency: "USDT", Amount: 1, FeeRate: math.NaN()}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := client.AddLedgerEntry(ctx, connect.NewRequest(tc.req))
			if connect.CodeOf(err) != connect.CodeInvalidArgument {
				t.Errorf("code = %v, want InvalidArgument", connect.CodeOf(err))
			}
		})
	}
}

// TestListLedgerEntriesOrder：台账流水 date DESC, id DESC 稳定排序 + 分页。
func TestListLedgerEntriesOrder(t *testing.T) {
	st := &fakeStore{ledger: []store.LedgerEntry{
		{ID: 1, Date: t0.Add(3 * time.Hour), Channel: "c", Currency: "USDT", Amount: 1},
		{ID: 2, Date: t0.Add(time.Hour), Channel: "c", Currency: "USDT", Amount: 2},
		{ID: 3, Date: t0.Add(3 * time.Hour), Channel: "c", Currency: "USDT", Amount: 3},
	}}
	client := newTestServer(t, New(st, nil, nil, nil))
	ctx := context.Background()

	resp, err := client.ListLedgerEntries(ctx, connect.NewRequest(&dashboardv1.ListLedgerEntriesRequest{}))
	if err != nil {
		t.Fatalf("ListLedgerEntries: %v", err)
	}
	got := resp.Msg.Entries
	if len(got) != 3 || got[0].Id != 3 || got[1].Id != 1 || got[2].Id != 2 {
		t.Errorf("排序 = %+v, want 3,1,2（date DESC 同刻 id DESC）", got)
	}
	page, err := client.ListLedgerEntries(ctx, connect.NewRequest(&dashboardv1.ListLedgerEntriesRequest{Limit: 1, Offset: 1}))
	if err != nil {
		t.Fatalf("ListLedgerEntries(page): %v", err)
	}
	if len(page.Msg.Entries) != 1 || page.Msg.Entries[0].Id != 1 {
		t.Errorf("page = %+v, want 1 条 id=1", page.Msg.Entries)
	}
}

// TestLedgerSummary：按档位归因汇总（GROUP BY tier 简单分组）。
func TestLedgerSummary(t *testing.T) {
	st := &fakeStore{ledger: []store.LedgerEntry{
		{ID: 1, Date: t0, Channel: "a", Currency: "USDT", Amount: 1000, Tier: store.TierStableBase},
		{ID: 2, Date: t0, Channel: "b", Currency: "USDT", Amount: -200, Tier: store.TierStableBase},
		{ID: 3, Date: t0, Channel: "c", Currency: "RMB", Amount: 50000, Tier: store.TierProtectedConvexity},
		{ID: 4, Date: t0, Channel: "d", Currency: "RMB", Amount: 1000, Tier: ""}, // 未分类
	}}
	client := newTestServer(t, New(st, nil, nil, nil))
	ctx := context.Background()

	resp, err := client.LedgerSummary(ctx, connect.NewRequest(&dashboardv1.LedgerSummaryRequest{}))
	if err != nil {
		t.Fatalf("LedgerSummary: %v", err)
	}
	byTier := map[string]*dashboardv1.TierSummary{}
	for _, it := range resp.Msg.Items {
		byTier[it.Tier] = it
	}
	stable := byTier[store.TierStableBase]
	if stable == nil || stable.Inflow != 1000 || stable.Outflow != 200 || stable.Net != 800 || stable.EntryCount != 2 {
		t.Errorf("stable_base = %+v, want inflow 1000/outflow 200/net 800/2 笔", stable)
	}
	if p := byTier[store.TierProtectedConvexity]; p == nil || p.Inflow != 50000 || p.Net != 50000 || p.EntryCount != 1 {
		t.Errorf("protected_convexity = %+v", p)
	}
	if u := byTier[""]; u == nil || u.Net != 1000 {
		t.Errorf("未分类 = %+v, want net 1000", u)
	}
}

func tsPtr(t time.Time) *timestamppb.Timestamp {
	return timestamppb.New(t)
}
