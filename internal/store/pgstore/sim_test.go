package pgstore

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"arbcn/internal/store"
)

// TestSimOrderRoundtrip：建议订单写入 → ts DESC 读回逐项比对；非法行拒绝；
// 状态机更新 + GetSimOrder 读回。
func TestSimOrderRoundtrip(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	ensureSchema(t, ctx, pool)
	resetTables(t, ctx, pool, "sim_orders", "sim_positions")

	s := New(pool)
	now := time.Now().UTC().Truncate(time.Microsecond)
	orders := []store.SimOrder{
		{Ts: now.Add(time.Hour), SrcRule: "funding_warn", Kind: store.SimKindFundingHedge,
			Venue: "sim_local", Symbol: "BTC", Side: store.SimSideHedge, Qty: 10000,
			RefPrice: 60000, ExpectedSpread: 10, RiskFlags: []string{}, Status: store.SimStatusSuggested},
		{Ts: now, SrcRule: "defi_large_tier_change", Kind: store.SimKindCarryAsset,
			Venue: "sim_local", Symbol: "USDT", Side: store.SimSideLong, Qty: 5000,
			RefPrice: 1, ExpectedSpread: 6, RiskFlags: []string{simRiskWhitelist}, Status: store.SimStatusRejected,
			Note: "carry_asset 未白名单"},
	}
	ids := make([]int64, 0, len(orders))
	for i, o := range orders {
		id, err := s.InsertSimOrder(ctx, o)
		if err != nil {
			t.Fatalf("InsertSimOrder(%d): %v", i, err)
		}
		if id <= 0 {
			t.Fatalf("InsertSimOrder(%d) id = %d, want > 0", i, id)
		}
		ids = append(ids, id)
	}

	got, err := s.ListSimOrders(ctx, 0, 0)
	if err != nil {
		t.Fatalf("ListSimOrders: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("orders len = %d, want 2", len(got))
	}
	// ts DESC：got[0] = now+1h。
	if got[0].ID != ids[0] || !got[0].Ts.Equal(now.Add(time.Hour)) ||
		got[0].SrcRule != "funding_warn" || got[0].Kind != store.SimKindFundingHedge ||
		got[0].Symbol != "BTC" || got[0].Side != store.SimSideHedge || got[0].Qty != 10000 ||
		got[0].RefPrice != 60000 || got[0].ExpectedSpread != 10 ||
		len(got[0].RiskFlags) != 0 || got[0].Status != store.SimStatusSuggested {
		t.Errorf("got[0] = %+v", got[0])
	}
	if !slices.Equal(got[1].RiskFlags, []string{simRiskWhitelist}) || got[1].Status != store.SimStatusRejected {
		t.Errorf("got[1] = %+v（risk_flags 应含 WHITELIST，status=rejected）", got[1])
	}

	// GetSimOrder 读回 + 状态机更新。
	one, err := s.GetSimOrder(ctx, ids[0])
	if err != nil || one.Status != store.SimStatusSuggested {
		t.Fatalf("GetSimOrder = %+v, %v", one, err)
	}
	if err := s.UpdateSimOrderStatus(ctx, ids[0], store.SimStatusConfirmed, ""); err != nil {
		t.Fatalf("UpdateSimOrderStatus(confirmed): %v", err)
	}
	if err := s.UpdateSimOrderStatus(ctx, ids[0], store.SimStatusFilled, "本地模拟即时成交"); err != nil {
		t.Fatalf("UpdateSimOrderStatus(filled): %v", err)
	}
	filled, _ := s.GetSimOrder(ctx, ids[0])
	if filled.Status != store.SimStatusFilled || filled.Note != "本地模拟即时成交" {
		t.Fatalf("filled = %+v, want filled + note 覆盖", filled)
	}

	// 未知 id → ErrNotFound。
	if _, err := s.GetSimOrder(ctx, 999999); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("GetSimOrder(unknown) err = %v, want ErrNotFound", err)
	}

	// 非法行拒绝：kind 空 / symbol 空 / side 空 / qty ≤ 0。
	for name, bad := range map[string]store.SimOrder{
		"kind 空":   {Ts: now, Symbol: "BTC", Side: "long", Qty: 1},
		"symbol 空": {Ts: now, Kind: "funding_hedge", Side: "long", Qty: 1},
		"side 空":   {Ts: now, Kind: "funding_hedge", Symbol: "BTC", Qty: 1},
		"qty 零":    {Ts: now, Kind: "funding_hedge", Symbol: "BTC", Side: "long", Qty: 0},
	} {
		if _, err := s.InsertSimOrder(ctx, bad); err == nil {
			t.Errorf("%s: InsertSimOrder = nil, want error", name)
		}
	}
}

// TestSimTodayNotional：当日累计名义（suggested/confirmed/filled；排除 rejected）；
// 昨日/明日订单不计入。
func TestSimTodayNotional(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	ensureSchema(t, ctx, pool)
	resetTables(t, ctx, pool, "sim_orders")

	s := New(pool)
	// now = 查询时刻；窗口 [当日 00:00, now) 右开——订单须严格早于 now 才计入。
	now := time.Date(2026, 8, 15, 12, 0, 0, 0, time.Local)
	mk := func(ts time.Time, status string, qty float64) {
		id, err := s.InsertSimOrder(ctx, store.SimOrder{
			Ts: ts, SrcRule: "r", Kind: store.SimKindFundingHedge, Venue: "sim_local",
			Symbol: "BTC", Side: store.SimSideHedge, Qty: qty, RefPrice: 1, Status: status,
		})
		if err != nil || id <= 0 {
			t.Fatalf("InsertSimOrder: %v", err)
		}
	}
	mk(now.Add(-3*time.Hour), store.SimStatusConfirmed, 10_000)  // 计入（09:00）
	mk(now.Add(-2*time.Hour), store.SimStatusFilled, 20_000)     // 计入（10:00）
	mk(now.Add(-time.Hour), store.SimStatusRejected, 99_000)     // 排除（拒单负样本）
	mk(now.Add(-24*time.Hour), store.SimStatusConfirmed, 5_000)  // 昨日不计入

	sum, err := s.TodaySimNotional(ctx, now)
	if err != nil {
		t.Fatalf("TodaySimNotional: %v", err)
	}
	if sum != 30_000 {
		t.Fatalf("TodaySimNotional = %v, want 30000（10k+20k，排除 rejected/昨日）", sum)
	}

	// 空表 → 0 非 nil。
	resetTables(t, ctx, pool, "sim_orders")
	sum, err = s.TodaySimNotional(ctx, now)
	if err != nil || sum != 0 {
		t.Fatalf("empty TodaySimNotional = %v, %v, want 0/nil", sum, err)
	}
}

// TestSimPositionRoundtrip：持仓腿写入 → ListSimPositions 读回；ListOpenSimPositions
// 只返回 open；SettleSimPosition 累计 pnl + 关闭。
func TestSimPositionRoundtrip(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	ensureSchema(t, ctx, pool)
	resetTables(t, ctx, pool, "sim_orders", "sim_positions")

	s := New(pool)
	oid, err := s.InsertSimOrder(ctx, store.SimOrder{
		Ts: time.Now(), SrcRule: "funding_warn", Kind: store.SimKindFundingHedge,
		Venue: "sim_local", Symbol: "BTC", Side: store.SimSideHedge, Qty: 10000,
		RefPrice: 60000, Status: store.SimStatusFilled,
	})
	if err != nil {
		t.Fatalf("InsertSimOrder: %v", err)
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	legs := []store.SimPosition{
		{OrderID: oid, Ts: now, Kind: store.SimKindFundingHedge, Venue: "sim_local", Symbol: "BTC",
			Side: store.SimSideLong, Qty: 10000, RefPrice: 60000, Funding: false, Status: store.SimPosStatusOpen},
		{OrderID: oid, Ts: now.Add(time.Second), Kind: store.SimKindFundingHedge, Venue: "sim_local", Symbol: "BTC",
			Side: store.SimSideShort, Qty: 10000, RefPrice: 60000, Funding: true, Status: store.SimPosStatusOpen},
	}
	pids := make([]int64, 0, 2)
	for i, l := range legs {
		id, err := s.InsertSimPosition(ctx, l)
		if err != nil || id <= 0 {
			t.Fatalf("InsertSimPosition(%d): %v", i, err)
		}
		pids = append(pids, id)
	}

	got, err := s.ListSimPositions(ctx, 0, 0)
	if err != nil {
		t.Fatalf("ListSimPositions: %v", err)
	}
	if len(got) != 2 || got[0].ID != pids[1] || got[1].ID != pids[0] {
		t.Fatalf("ListSimPositions = %+v, want 2 条 ts DESC", got)
	}
	if !got[0].Funding || got[1].Funding {
		t.Fatalf("funding 标定错：got[0]=%+v got[1]=%+v", got[0], got[1])
	}

	open, err := s.ListOpenSimPositions(ctx, "BTC", "")
	if err != nil {
		t.Fatalf("ListOpenSimPositions: %v", err)
	}
	if len(open) != 2 {
		t.Fatalf("open legs = %d, want 2", len(open))
	}

	// 结算永续腿：pnl += 100。
	if err := s.SettleSimPosition(ctx, pids[1], 100, store.SimPosStatusOpen); err != nil {
		t.Fatalf("SettleSimPosition: %v", err)
	}
	// 关闭现货腿。
	if err := s.SettleSimPosition(ctx, pids[0], 0, store.SimPosStatusSettled); err != nil {
		t.Fatalf("SettleSimPosition(settled): %v", err)
	}
	open, err = s.ListOpenSimPositions(ctx, "BTC", "")
	if err != nil {
		t.Fatalf("ListOpenSimPositions(after): %v", err)
	}
	if len(open) != 1 || open[0].ID != pids[1] || open[0].PnL != 100 {
		t.Fatalf("open after = %+v, want 1 条（永续）pnl=100", open)
	}

	// 空 symbol → 不过滤（全 open）。
	allOpen, err := s.ListOpenSimPositions(ctx, "", "")
	if err != nil || len(allOpen) != 1 {
		t.Fatalf("ListOpenSimPositions('') = %+v, %v, want 1 条", allOpen, err)
	}

	// 非法行拒绝：order_id 0 / qty ≤ 0。
	for name, bad := range map[string]store.SimPosition{
		"order 缺":  {Ts: now, Symbol: "BTC", Qty: 1},
		"qty 零":    {OrderID: oid, Ts: now, Symbol: "BTC", Qty: 0},
	} {
		if _, err := s.InsertSimPosition(ctx, bad); err == nil {
			t.Errorf("%s: InsertSimPosition = nil, want error", name)
		}
	}
}

// TestFillSimOrderAtomicity（M3-a 复审 M1）：原子成交——confirmed→filled + 建全部腿
// 在同一事务；非 confirmed/不存在/重复成交一律拒绝，不留"filled 但缺腿"半对冲状态。
// [对抗测试锚点] 把 FillSimOrder 的状态守卫（WHERE status='confirmed'）去掉 → 重复成交
// 断言必红；把 INSERT 腿移出事务 → 失败回滚断言必红。
func TestFillSimOrderAtomicity(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	ensureSchema(t, ctx, pool)
	resetTables(t, ctx, pool, "sim_orders", "sim_positions")

	s := New(pool)
	now := time.Now().UTC().Truncate(time.Microsecond)

	newOrder := func(status string) int64 {
		id, err := s.InsertSimOrder(ctx, store.SimOrder{Ts: now, Kind: store.SimKindFundingHedge,
			Venue: "sim_local", Symbol: "BTC", Side: store.SimSideHedge, Qty: 10000,
			RefPrice: 60000, ExpectedSpread: 10, Status: status})
		if err != nil {
			t.Fatalf("InsertSimOrder: %v", err)
		}
		return id
	}
	legs := []store.SimPosition{
		{Ts: now, Kind: store.SimKindFundingHedge, Venue: "sim_local", Symbol: "BTC",
			Side: store.SimSideLong, Qty: 10000, RefPrice: 60000, Funding: false, Status: store.SimPosStatusOpen},
		{Ts: now, Kind: store.SimKindFundingHedge, Venue: "sim_local", Symbol: "BTC",
			Side: store.SimSideShort, Qty: 10000, RefPrice: 60000, Funding: true, Status: store.SimPosStatusOpen},
	}

	// 1) confirmed → filled + 两腿。
	id := newOrder(store.SimStatusConfirmed)
	if err := s.FillSimOrder(ctx, id, "即时成交", legs); err != nil {
		t.Fatalf("FillSimOrder: %v", err)
	}
	o, err := s.GetSimOrder(ctx, id)
	if err != nil || o.Status != store.SimStatusFilled || o.Note != "即时成交" {
		t.Fatalf("order after fill = %+v, %v, want filled/即时成交", o, err)
	}
	open, err := s.ListOpenSimPositions(ctx, "", "")
	if err != nil || len(open) != 2 {
		t.Fatalf("legs = %d, %v, want 2", len(open), err)
	}

	// 2) 重复成交（已 filled）→ 拒绝（状态守卫，防并发双插）。
	if err := s.FillSimOrder(ctx, id, "again", legs); err == nil {
		t.Fatal("FillSimOrder(filled) = nil, want error（状态守卫）")
	}
	if open, _ = s.ListOpenSimPositions(ctx, "", ""); len(open) != 2 {
		t.Fatalf("legs after dup = %d, want 2（不得双插）", len(open))
	}

	// 3) suggested（未确认）→ 拒绝，且不建腿。
	sid := newOrder(store.SimStatusSuggested)
	if err := s.FillSimOrder(ctx, sid, "", legs); err == nil {
		t.Fatal("FillSimOrder(suggested) = nil, want error")
	}
	if open, _ = s.ListOpenSimPositions(ctx, "", ""); len(open) != 2 {
		t.Fatalf("legs after suggested = %d, want 2（未确认不得成交）", len(open))
	}

	// 4) 不存在 id → 拒绝。
	if err := s.FillSimOrder(ctx, 999_999, "", legs); err == nil {
		t.Fatal("FillSimOrder(missing) = nil, want error")
	}
}

// TestAcceptSimOrderAtomicity（M3-c C3，practices #8）：人工确认原子成交——
// suggested→confirmed→filled + 建全部腿在同一事务；非 suggested/不存在/重复确认拒绝，
// 不留"已确认未成交"悬挂、无重复建腿。
// [对抗测试锚点] 删 AcceptSimOrder 的 status='suggested' 守卫（第一 UPDATE WHERE 条件）
// → 重复确认/confirmed 拒绝断言必红；把 INSERT 腿移出事务 → 失败回滚断言必红。
func TestAcceptSimOrderAtomicity(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	ensureSchema(t, ctx, pool)
	resetTables(t, ctx, pool, "sim_orders", "sim_positions")

	s := New(pool)
	now := time.Now().UTC().Truncate(time.Microsecond)
	legs := []store.SimPosition{
		{Ts: now, Kind: store.SimKindFundingHedge, Venue: "sim_local", Symbol: "BTC",
			Side: store.SimSideLong, Qty: 10000, RefPrice: 60000, Funding: false, Status: store.SimPosStatusOpen},
		{Ts: now, Kind: store.SimKindFundingHedge, Venue: "sim_local", Symbol: "BTC",
			Side: store.SimSideShort, Qty: 10000, RefPrice: 60000, Funding: true, Status: store.SimPosStatusOpen},
	}
	newOrder := func(status string) int64 {
		id, err := s.InsertSimOrder(ctx, store.SimOrder{Ts: now, Kind: store.SimKindFundingHedge,
			Venue: "sim_local", Symbol: "BTC", Side: store.SimSideHedge, Qty: 10000,
			RefPrice: 60000, ExpectedSpread: 10, Status: status})
		if err != nil {
			t.Fatalf("InsertSimOrder: %v", err)
		}
		return id
	}

	// 1) suggested → filled + 两腿。
	id := newOrder(store.SimStatusSuggested)
	if err := s.AcceptSimOrder(ctx, id, "人工确认成交", legs); err != nil {
		t.Fatalf("AcceptSimOrder: %v", err)
	}
	o, err := s.GetSimOrder(ctx, id)
	if err != nil || o.Status != store.SimStatusFilled || o.Note != "人工确认成交" {
		t.Fatalf("order after accept = %+v, %v, want filled/人工确认成交", o, err)
	}
	open, err := s.ListOpenSimPositions(ctx, "", "")
	if err != nil || len(open) != 2 {
		t.Fatalf("legs = %d, %v, want 2", len(open), err)
	}

	// 2) 重复确认（已 filled）→ 拒绝（suggested 守卫，防并发双确认/双插）。
	if err := s.AcceptSimOrder(ctx, id, "again", legs); err == nil {
		t.Fatal("AcceptSimOrder(filled) = nil, want error（守卫）")
	}
	if open, _ = s.ListOpenSimPositions(ctx, "", ""); len(open) != 2 {
		t.Fatalf("legs after dup = %d, want 2（不得双插）", len(open))
	}

	// 3) confirmed（已确认未成交 = 事务中间态，外部不应存在）→ 拒绝（suggested 守卫）。
	cid := newOrder(store.SimStatusConfirmed)
	if err := s.AcceptSimOrder(ctx, cid, "", legs); err == nil {
		t.Fatal("AcceptSimOrder(confirmed) = nil, want error（suggested 守卫）")
	}

	// 4) rejected → 拒绝。
	rid := newOrder(store.SimStatusRejected)
	if err := s.AcceptSimOrder(ctx, rid, "", legs); err == nil {
		t.Fatal("AcceptSimOrder(rejected) = nil, want error")
	}

	// 5) 不存在 id → 拒绝。
	if err := s.AcceptSimOrder(ctx, 999_999, "", legs); err == nil {
		t.Fatal("AcceptSimOrder(missing) = nil, want error")
	}
}

// TestCloseSimOrder（D-055 平仓闭环）：filled 订单 + 两腿 → 平仓 → 订单 closed +
// 两腿 settled + pnl += AddPnl + note 追加 + 返回腿数 2；非 filled / 未知订单 →
// ErrNotFound；腿非 open/不属于本单 → 整体回滚（订单保持 filled、腿不变）。
// [对抗测试锚点] 删 CloseSimOrder 的 filled 守卫 / 删腿 open 守卫 → 拒绝断言必红；
// 把腿 UPDATE 移出事务 → 回滚断言必红。
func TestCloseSimOrder(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	ensureSchema(t, ctx, pool)
	resetTables(t, ctx, pool, "sim_orders", "sim_positions")

	s := New(pool)
	now := time.Now().UTC().Truncate(time.Microsecond)
	legs := []store.SimPosition{
		{Ts: now, Kind: store.SimKindFundingHedge, Venue: "sim_local", Symbol: "BTC",
			Side: store.SimSideLong, Qty: 10000, RefPrice: 60000, Funding: false, Status: store.SimPosStatusOpen},
		{Ts: now, Kind: store.SimKindFundingHedge, Venue: "sim_local", Symbol: "BTC",
			Side: store.SimSideShort, Qty: 10000, RefPrice: 60000, Funding: true, Status: store.SimPosStatusOpen},
	}
	// 建 filled 订单（含两腿）。
	id, err := s.InsertSimOrder(ctx, store.SimOrder{Ts: now, Kind: store.SimKindFundingHedge,
		Venue: "sim_local", Symbol: "BTC", Side: store.SimSideHedge, Qty: 10000,
		RefPrice: 60000, ExpectedSpread: 10, Status: store.SimStatusSuggested})
	if err != nil {
		t.Fatalf("InsertSimOrder: %v", err)
	}
	if err := s.AcceptSimOrder(ctx, id, "确认成交", legs); err != nil {
		t.Fatalf("AcceptSimOrder: %v", err)
	}
	open, _ := s.ListOpenSimPositions(ctx, "", "")
	if len(open) != 2 {
		t.Fatalf("open legs = %d, want 2", len(open))
	}
	// 短腿先积累一期已结算 funding（模拟 8h settle），pnl 保留在平仓 realized 里。
	if err := s.SettleSimPosition(ctx, open[1].ID, 123.45, store.SimPosStatusOpen); err != nil {
		t.Fatalf("SettleSimPosition: %v", err)
	}

	// 平仓：短腿浮动 −8000、长腿浮动 +8000（对冲净 0）→ 两腿皆 settled。
	n, err := s.CloseSimOrder(ctx, id, "人工平仓", []store.SimLegClose{
		{ID: open[0].ID, AddPnl: 8000},
		{ID: open[1].ID, AddPnl: -8000},
	})
	if err != nil {
		t.Fatalf("CloseSimOrder: %v", err)
	}
	if n != 2 {
		t.Fatalf("closed legs = %d, want 2", n)
	}
	o, err := s.GetSimOrder(ctx, id)
	if err != nil || o.Status != store.SimStatusClosed || o.Note != "人工平仓" {
		t.Fatalf("order after close = %+v, %v, want closed/人工平仓", o, err)
	}
	all, _ := s.ListSimPositions(ctx, 0, 0)
	if len(all) != 2 {
		t.Fatalf("positions = %d, want 2", len(all))
	}
	for _, p := range all {
		if p.Status != store.SimPosStatusSettled {
			t.Fatalf("leg %d status = %q, want settled", p.ID, p.Status)
		}
	}
	// 短腿 pnl = settled(123.45) + (−8000)；长腿 = 8000。按 side 核对（运行时双精度
	// 计算与 DB float8 同 ULP，避免常量折叠差 1 ULP 的 flake）。
	settled := float64(123.45)
	if all[0].Side == store.SimSideShort {
		all[0], all[1] = all[1], all[0]
	}
	if all[0].PnL != 8000 || all[1].PnL != settled-8000 {
		t.Fatalf("pnl = long %v short %v, want 8000 / %v", all[0].PnL, all[1].PnL, settled-8000)
	}
	// 再平（已 closed）→ ErrNotFound（filled 守卫）。
	if _, err := s.CloseSimOrder(ctx, id, "", []store.SimLegClose{{ID: open[0].ID}}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("close(closed) = %v, want ErrNotFound", err)
	}

	// 非 filled 订单 / 未知订单 → ErrNotFound。
	sid, _ := s.InsertSimOrder(ctx, store.SimOrder{Ts: now, Kind: store.SimKindFundingHedge,
		Venue: "sim_local", Symbol: "BTC", Side: store.SimSideHedge, Qty: 10000,
		RefPrice: 60000, Status: store.SimStatusSuggested})
	if _, err := s.CloseSimOrder(ctx, sid, "", []store.SimLegClose{{ID: open[0].ID}}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("close(suggested) = %v, want ErrNotFound", err)
	}
	if _, err := s.CloseSimOrder(ctx, 999_999, "", []store.SimLegClose{{ID: open[0].ID}}); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("close(missing) = %v, want ErrNotFound", err)
	}

	// 腿非本单（另一订单的腿 id 混入）→ 回滚：订单保持 filled、原两腿仍 open。
	// 重建一个 filled 订单验证回滚语义。
	id2, _ := s.InsertSimOrder(ctx, store.SimOrder{Ts: now, Kind: store.SimKindFundingHedge,
		Venue: "sim_local", Symbol: "BTC", Side: store.SimSideHedge, Qty: 10000,
		RefPrice: 60000, Status: store.SimStatusSuggested})
	if err := s.AcceptSimOrder(ctx, id2, "", legs); err != nil {
		t.Fatalf("AcceptSimOrder#2: %v", err)
	}
	open2, _ := s.ListOpenSimPositions(ctx, "", "")
	if len(open2) != 2 {
		t.Fatalf("open legs #2 = %d, want 2", len(open2))
	}
	// 把第一单已 settled 的腿混进第二单的平仓 → 腿 open 守卫 miss → 回滚。
	if _, err := s.CloseSimOrder(ctx, id2, "", []store.SimLegClose{{ID: all[0].ID, AddPnl: 1}}); err == nil {
		t.Fatal("close with foreign/settled leg = nil, want error（回滚）")
	}
	o2, _ := s.GetSimOrder(ctx, id2)
	if o2.Status != store.SimStatusFilled {
		t.Fatalf("order#2 after failed close = %q, want filled（回滚）", o2.Status)
	}
	if open, _ = s.ListOpenSimPositions(ctx, "", ""); len(open) != 2 {
		t.Fatalf("open legs after failed close = %d, want 2（回滚）", len(open))
	}
	// 空 closes → 拒绝。
	if _, err := s.CloseSimOrder(ctx, id2, "", nil); err == nil {
		t.Fatal("close(empty closes) = nil, want error")
	}
}

// TestRejectSimOrderAppendsFlag（M3-c C3）：suggested → rejected + risk_flags 追加
// SPREAD_DRIFT（保留既有标记、去重）+ note 覆盖；非 suggested/未知 id → 报错（状态守卫）。
// [对抗测试锚点] 删 RejectSimOrder 的 status='suggested' 守卫 → 非 suggested 拒绝断言必红。
func TestRejectSimOrderAppendsFlag(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	ensureSchema(t, ctx, pool)
	resetTables(t, ctx, pool, "sim_orders")

	s := New(pool)
	now := time.Now().UTC().Truncate(time.Microsecond)
	newOrder := func(status string) int64 {
		id, err := s.InsertSimOrder(ctx, store.SimOrder{Ts: now, Kind: store.SimKindFundingHedge,
			Venue: "sim_local", Symbol: "BTC", Side: store.SimSideHedge, Qty: 10000,
			RefPrice: 60000, ExpectedSpread: 10, RiskFlags: []string{"SPREAD_LOW"}, Status: status})
		if err != nil {
			t.Fatalf("InsertSimOrder: %v", err)
		}
		return id
	}

	// 1) suggested → rejected + SPREAD_DRIFT 追加（保留既有 SPREAD_LOW）+ note 覆盖。
	id := newOrder(store.SimStatusSuggested)
	if err := s.RejectSimOrder(ctx, id, "SPREAD_DRIFT: ref_price 漂移 5.00%", "SPREAD_DRIFT"); err != nil {
		t.Fatalf("RejectSimOrder: %v", err)
	}
	o, err := s.GetSimOrder(ctx, id)
	if err != nil || o.Status != store.SimStatusRejected {
		t.Fatalf("order = %+v, %v, want rejected", o, err)
	}
	if !slices.Contains(o.RiskFlags, "SPREAD_DRIFT") {
		t.Fatalf("risk_flags = %v, want 含 SPREAD_DRIFT", o.RiskFlags)
	}
	if !slices.Contains(o.RiskFlags, "SPREAD_LOW") {
		t.Fatalf("risk_flags = %v, want 保留既有 SPREAD_LOW", o.RiskFlags)
	}
	if o.Note != "SPREAD_DRIFT: ref_price 漂移 5.00%" {
		t.Fatalf("note = %q, want 拒单原因覆盖", o.Note)
	}

	// 2) filled（非 suggested）→ 拒绝（守卫）。
	fid := newOrder(store.SimStatusFilled)
	if err := s.RejectSimOrder(ctx, fid, "x", "SPREAD_DRIFT"); err == nil {
		t.Fatal("RejectSimOrder(filled) = nil, want error（守卫）")
	}

	// 3) 未知 id → 拒绝。
	if err := s.RejectSimOrder(ctx, 999_999, "x", "SPREAD_DRIFT"); err == nil {
		t.Fatal("RejectSimOrder(missing) = nil, want error")
	}

	// 4) flags 为空 → 拒绝调用。
	if err := s.RejectSimOrder(ctx, id, "x"); err == nil {
		t.Fatal("RejectSimOrder(no flags) = nil, want error")
	}
}

// simRiskWhitelist 与 sim 包常量对齐的测试局部别名（pgstore 不 import internal/sim，
// 避免循环依赖；值域以 04-m3-spec §1.1 为准）。
const simRiskWhitelist = "WHITELIST"
