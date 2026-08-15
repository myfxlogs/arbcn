package sim

import (
	"context"
	"math"
	"testing"
	"time"

	"arbcn/internal/fact"
	"arbcn/internal/store"
)

// seedTicker 给 stub 注入 (venue,symbol) 的 ticker 最新价。
func seedTicker(st *storeStub, venue, symbol string, price float64) {
	st.facts = append(st.facts, fact.Fact{
		Kind: fact.KindTicker, Venue: venue, Symbol: symbol, Value: price, Ts: t0, Src: "test",
	})
}

// newDriver 构造 Driver + stub；固定时钟。
func newDriver(t *testing.T, cfg Config) (*Driver, *storeStub) {
	t.Helper()
	st := &storeStub{}
	if cfg.Capital == 0 {
		cfg = DefaultConfig()
	}
	d := NewDriver(st, cfg)
	d.now = func() time.Time { return t0 }
	d.sim.Now = d.now
	return d, st
}

// TestDriverFundingHitCreatesOrder：[对抗测试锚点 §9.2 S1] funding_warn 命中
// BTC@binance → sim_orders 落 funding_hedge（venue/symbol/资金年化/双腿价）。
// 删除 signalMappers 中 funding_warn 映射 → 本测试必红。
func TestDriverFundingHitCreatesOrder(t *testing.T) {
	d, st := newDriver(t, DefaultConfig())
	seedTicker(st, "binance", "BTC", 60000)

	err := d.OnRuleActive(context.Background(), store.Rule{Name: "funding_warn"},
		[]store.EntityHit{{Venue: "binance", Symbol: "BTC", Value: 18.5}})
	if err != nil {
		t.Fatalf("OnRuleActive: %v", err)
	}
	if len(st.orders) != 1 {
		t.Fatalf("orders = %d, want 1", len(st.orders))
	}
	o := st.orders[0]
	if o.Kind != store.SimKindFundingHedge || o.Symbol != "BTC" || o.Venue != "binance" ||
		o.SrcRule != "funding_warn" {
		t.Fatalf("order = %+v, want funding_hedge BTC@binance", o)
	}
	// 诚实标注（§9.2）：ticker 即永续价；现货/永续腿存在性由门禁把关（>0）。
	if o.RefPrice != 60000 || o.ExpectedSpread != 18.5 {
		t.Fatalf("ref_price/spread = %v/%v, want 60000/18.5", o.RefPrice, o.ExpectedSpread)
	}
	if o.Status != store.SimStatusSuggested || len(o.RiskFlags) != 0 {
		t.Fatalf("status/flags = %q/%v, want suggested/empty", o.Status, o.RiskFlags)
	}
}

// TestDriverFundingHitNoTickerRejected：funding 命中但无 ticker（现货/永续价缺）→
// UNHEDGED 拒单（门禁把关，负样本落库）。
func TestDriverFundingHitNoTickerRejected(t *testing.T) {
	d, st := newDriver(t, DefaultConfig())
	if err := d.OnRuleActive(context.Background(), store.Rule{Name: "funding_warn"},
		[]store.EntityHit{{Venue: "binance", Symbol: "BTC", Value: 18.5}}); err != nil {
		t.Fatalf("OnRuleActive: %v", err)
	}
	if len(st.orders) != 1 || st.orders[0].Status != store.SimStatusRejected ||
		!hasFlag(st.orders[0], RiskUNHEDGED) {
		t.Fatalf("orders = %+v, want 1 条 UNHEDGED 拒单", st.orders)
	}
}

// TestDriverUnknownRuleNoOrder：[对抗测试锚点 §9.2 S1] 未知规则名 → 不建单（宁缺毋滥）。
// 删除 buildSignal 中"未知规则返回不建单"分支 → 本测试必红。
func TestDriverUnknownRuleNoOrder(t *testing.T) {
	d, st := newDriver(t, DefaultConfig())
	for _, name := range []string{"defi_large_tier_change", "ladder_trap", "iv_opportunity", "usdcnh_buy_line", "collector_heartbeat"} {
		if err := d.OnRuleActive(context.Background(), store.Rule{Name: name},
			[]store.EntityHit{{Venue: "binance", Symbol: "BTC", Value: 1}}); err != nil {
			t.Fatalf("OnRuleActive(%s): %v", name, err)
		}
	}
	if len(st.orders) != 0 {
		t.Fatalf("orders = %d, want 0（信息类/遥测规则不建单）", len(st.orders))
	}
}

// TestDriverRepoBuildsOrder：reverse_repo_timing（全局命中，时点资金面收紧）→ repo 单。
// 命中值 6.5% 代表季末/年末逆回购利率上冲（>5% 门槛放行；平时 2-4% 会 SPREAD_LOW 拒单负样本，
// 这是正确行为——宁缺毋滥）。
func TestDriverRepoBuildsOrder(t *testing.T) {
	d, st := newDriver(t, DefaultConfig())
	if err := d.OnRuleActive(context.Background(), store.Rule{Name: "reverse_repo_timing"},
		[]store.EntityHit{{Value: 6.5}}); err != nil {
		t.Fatalf("OnRuleActive: %v", err)
	}
	if len(st.orders) != 1 {
		t.Fatalf("orders = %d, want 1", len(st.orders))
	}
	o := st.orders[0]
	if o.Kind != store.SimKindRepo || o.Symbol != "GC001" || o.RefPrice != 100 || o.ExpectedSpread != 6.5 {
		t.Fatalf("order = %+v, want repo GC001 @100 spread 6.5", o)
	}
	if o.Status != store.SimStatusSuggested || len(o.RiskFlags) != 0 {
		t.Fatalf("status/flags = %q/%v, want suggested/empty", o.Status, o.RiskFlags)
	}
}

// TestDriverCarryWhitelisted：[对抗测试锚点 §9.6 S5] 白名单内 symbol → carry_asset
// 且 CarryWhite 命中（SignalToOrder 白名单门禁通过）。删 buildSignal carry 分支 → 必红。
func TestDriverCarryWhitelisted(t *testing.T) {
	cfg := DefaultConfig()
	cfg.CarryWhitelist = []string{"sUSDe", "USDe"}
	d, st := newDriver(t, cfg)

	if err := d.OnRuleActive(context.Background(), store.Rule{Name: "carry_monitor"},
		[]store.EntityHit{{Venue: "binance_ear", Symbol: "sUSDe", Value: 8.0}}); err != nil {
		t.Fatalf("OnRuleActive: %v", err)
	}
	if len(st.orders) != 1 {
		t.Fatalf("orders = %d, want 1", len(st.orders))
	}
	o := st.orders[0]
	if o.Kind != store.SimKindCarryAsset || o.Symbol != "sUSDe" || o.Status != store.SimStatusSuggested {
		t.Fatalf("order = %+v, want carry_asset sUSDe suggested", o)
	}
}

// TestDriverCarryNotWhitelisted：symbol 未白名单 → carry 不建单（默认空 = 全部 carry 拒门外）。
func TestDriverCarryNotWhitelisted(t *testing.T) {
	d, st := newDriver(t, DefaultConfig()) // 默认白名单空（安全默认）
	if err := d.OnRuleActive(context.Background(), store.Rule{Name: "carry_monitor"},
		[]store.EntityHit{{Venue: "binance_ear", Symbol: "sUSDe", Value: 8.0}}); err != nil {
		t.Fatalf("OnRuleActive: %v", err)
	}
	if len(st.orders) != 0 {
		t.Fatalf("orders = %d, want 0（默认白名单空 = carry 拒门外，§9.6 安全默认）", len(st.orders))
	}
}

// TestDriverGeneratePersistsRejectedNegativeSample：组装后 Generate 即使门禁拒单也落库
// （拒单 = 负样本，§4）。此例 funding 命中 + 无 ticker → UNHEDGED 已覆盖；这里验证
// Generate 路径确实经过 SignalToOrder（价差低于门槛时 SPREAD_LOW 拒单仍落库）。
func TestDriverGeneratePersistsRejectedNegativeSample(t *testing.T) {
	d, st := newDriver(t, DefaultConfig())
	seedTicker(st, "binance", "BTC", 60000)
	// 年化 3% < 门槛 5% → SPREAD_LOW 拒单（负样本）。
	if err := d.OnRuleActive(context.Background(), store.Rule{Name: "funding_warn"},
		[]store.EntityHit{{Venue: "binance", Symbol: "BTC", Value: 3}}); err != nil {
		t.Fatalf("OnRuleActive: %v", err)
	}
	if len(st.orders) != 1 || st.orders[0].Status != store.SimStatusRejected ||
		!hasFlag(st.orders[0], RiskSpreadLow) {
		t.Fatalf("orders = %+v, want 1 条 SPREAD_LOW 拒单（负样本）", st.orders)
	}
}

// openLeg 构造一条 open funding 腿（driver 结算测试用）。
func openLeg(id int64, venue, symbol string, qty float64, funding bool) store.SimPosition {
	return store.SimPosition{
		OrderID: id, Ts: t0, Kind: store.SimKindFundingHedge, Venue: venue, Symbol: symbol,
		Side: store.SimSideShort, Qty: qty, RefPrice: 1, Funding: funding, Status: store.SimPosStatusOpen,
	}
}

// TestSettleByVenue：[对抗测试锚点 §9.3 S2] BTC@binance 与 BTC@okx 隔离结算——
// 各取各 venue 的 LatestFacts(funding) 最新值，互不污染。删 settleOnce 分组中的 venue
// 维度（按 symbol 合并）→ 本测试必红。
func TestSettleByVenue(t *testing.T) {
	d, st := newDriver(t, DefaultConfig())
	// 同 symbol 不同 venue 两条 funding 腿。
	if _, err := st.InsertSimPosition(context.Background(), openLeg(1, "binance", "BTC", 10_000, true)); err != nil {
		t.Fatal(err)
	}
	if _, err := st.InsertSimPosition(context.Background(), openLeg(1, "okx", "BTC", 10_000, true)); err != nil {
		t.Fatal(err)
	}
	// 各 venue 真实 funding 年化不同：binance 10.95%（→ 每 8h 0.0001×10000=1），okx 21.90%（→ 2）。
	st.facts = append(st.facts,
		fact.Fact{Kind: fact.KindFunding, Venue: "binance", Symbol: "BTC", Value: 10.95, Ts: t0, Src: "test"},
		fact.Fact{Kind: fact.KindFunding, Venue: "okx", Symbol: "BTC", Value: 21.90, Ts: t0, Src: "test"},
	)

	if err := d.settleOnce(context.Background()); err != nil {
		t.Fatalf("settleOnce: %v", err)
	}
	byVenue := map[string]float64{}
	for _, p := range st.positions {
		byVenue[p.Venue] = p.PnL
	}
	if math.Abs(byVenue["binance"]-1) > 1e-9 {
		t.Fatalf("binance pnl = %v, want 1（10.95%% 单期 0.0001×10000）", byVenue["binance"])
	}
	if math.Abs(byVenue["okx"]-2) > 1e-9 {
		t.Fatalf("okx pnl = %v, want 2（21.90%% 单期 0.0002×10000）", byVenue["okx"])
	}
}

// TestSettleByVenueSkipsNonFunding：结算只动 funding 腿；非 funding（现货）腿不结算。
func TestSettleByVenueSkipsNonFunding(t *testing.T) {
	d, st := newDriver(t, DefaultConfig())
	if _, err := st.InsertSimPosition(context.Background(), openLeg(1, "binance", "BTC", 10_000, true)); err != nil {
		t.Fatal(err)
	}
	if _, err := st.InsertSimPosition(context.Background(), openLeg(1, "binance", "BTC", 10_000, false)); err != nil {
		t.Fatal(err)
	}
	st.facts = append(st.facts,
		fact.Fact{Kind: fact.KindFunding, Venue: "binance", Symbol: "BTC", Value: 10.95, Ts: t0, Src: "test"})
	if err := d.settleOnce(context.Background()); err != nil {
		t.Fatalf("settleOnce: %v", err)
	}
	nFunding := 0
	for _, p := range st.positions {
		if p.Funding {
			nFunding++
			if math.Abs(p.PnL-1) > 1e-9 {
				t.Fatalf("funding leg pnl = %v, want 1", p.PnL)
			}
		} else if p.PnL != 0 {
			t.Fatalf("non-funding leg pnl = %v, want 0（现货腿不结算）", p.PnL)
		}
	}
	if nFunding != 1 {
		t.Fatalf("funding legs = %d, want 1", nFunding)
	}
}

// TestSettleNoFactSkips：open funding 腿但无 funding 事实 → skip（warn），不结算不报错。
func TestSettleNoFactSkips(t *testing.T) {
	d, st := newDriver(t, DefaultConfig())
	if _, err := st.InsertSimPosition(context.Background(), openLeg(1, "binance", "BTC", 10_000, true)); err != nil {
		t.Fatal(err)
	}
	if err := d.settleOnce(context.Background()); err != nil {
		t.Fatalf("settleOnce: %v", err)
	}
	if len(st.positions) != 1 || st.positions[0].PnL != 0 {
		t.Fatalf("positions = %+v, want 未结算（无事实 skip）", st.positions)
	}
}

// TestSettleLoopTicks：settleLoop 消费注入 tick → 每次 tick 结算一轮；
// ctx 取消 → 及时返回（S2 调度骨架测试）。
func TestSettleLoopTicks(t *testing.T) {
	d, st := newDriver(t, DefaultConfig())
	if _, err := st.InsertSimPosition(context.Background(), openLeg(1, "binance", "BTC", 10_000, true)); err != nil {
		t.Fatal(err)
	}
	st.facts = append(st.facts,
		fact.Fact{Kind: fact.KindFunding, Venue: "binance", Symbol: "BTC", Value: 10.95, Ts: t0, Src: "test"})

	ctx, cancel := context.WithCancel(context.Background())
	ticks := make(chan time.Time)
	done := make(chan error, 1)
	go func() { done <- d.settleLoop(ctx, ticks) }()

	ticks <- t0
	ticks <- t0.Add(8 * time.Hour)
	waitFor(t, time.Second, func() bool {
		for _, p := range st.snapshotPositions() {
			if p.Funding && p.PnL >= 1.999 {
				return true
			}
		}
		return false
	})
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("settleLoop did not return after cancel")
	}
}

// TestDriverOnRuleActiveNoEntities：无命中实体（空列表）→ 无任何动作。
func TestDriverOnRuleActiveNoEntities(t *testing.T) {
	d, st := newDriver(t, DefaultConfig())
	if err := d.OnRuleActive(context.Background(), store.Rule{Name: "funding_warn"}, nil); err != nil {
		t.Fatalf("OnRuleActive: %v", err)
	}
	if len(st.orders) != 0 {
		t.Fatalf("orders = %d, want 0", len(st.orders))
	}
}
