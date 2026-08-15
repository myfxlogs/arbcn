package rule

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"arbcn/internal/fact"
	"arbcn/internal/store"
)

// TestStateMachineLifecycle：§11③ armed→active（1 告警）→ 持续满足不重复 →
// resolved 补发解除 → 再触发（第 3 告警）全路径。
// [对抗测试锚点] 关键行 = engine.go transition() 的
// "if st.State == store.StateActive" 早退分支（状态转变才告警）。删除该分支
// → 第 3 步"持续满足"与第 5 步"resolved 后再触发"的告警计数断言必红（§11②）。
func TestStateMachineLifecycle(t *testing.T) {
	st := newFakeStore([]store.Rule{
		{Name: "r1", Kind: fact.KindFunding, Cond: "avg_30d > 15", Level: store.LevelWarn, Enabled: true},
	}, nil)
	clock := t0
	e, err := New(context.Background(), st, Config{Now: func() time.Time { return clock }})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	step := func(v float64) {
		t.Helper()
		clock = clock.Add(time.Minute)
		if err := st.InsertFacts(context.Background(), []fact.Fact{
			fct(fact.KindFunding, "binance", "BTC", v, -time.Hour),
		}); err != nil {
			t.Fatal(err)
		}
		if err := e.EvaluateAll(context.Background()); err != nil {
			t.Fatalf("EvaluateAll: %v", err)
		}
	}
	state := func() store.TriggerState {
		t.Helper()
		s, err := st.GetTriggerState(context.Background(), 1)
		if err != nil {
			t.Fatalf("GetTriggerState: %v", err)
		}
		return s
	}

	step(16) // armed → active
	if got := st.alertsCopy(); len(got) != 1 || got[0].Message != "r1 触发: BTC@binance=16" {
		t.Fatalf("alerts after armed→active = %+v, want 1 active", got)
	}
	if s := state(); s.State != store.StateActive || s.LastValue != 16 {
		t.Fatalf("state after active = %+v, want active/16", s)
	}

	step(17) // active 持续满足 → 不重复告警，只更新 last_value、since 保留
	if got := st.alertsCopy(); len(got) != 1 {
		t.Fatalf("alerts after 持续满足 = %+v, want 仍 1 条（状态转变才告警）", got)
	}
	if s := state(); s.State != store.StateActive || s.LastValue != 16.5 || !s.Since.Equal(t0.Add(time.Minute)) {
		t.Fatalf("state after 持续满足 = %+v, want active/16.5/since 不变", s)
	}

	step(5) // 均值 (16+17+5)/3 ≈ 12.7 < 15 → resolved 补发解除
	got := st.alertsCopy()
	if len(got) != 2 || got[1].Message != "r1 已解除" {
		t.Fatalf("alerts after resolved = %+v, want 2 条且第 2 条为 resolved", got)
	}
	if s := state(); s.State != store.StateResolved {
		t.Fatalf("state after resolved = %+v, want resolved", s)
	}

	step(5) // resolved 且仍未命中 → 无转变，不落库
	if got := st.alertsCopy(); len(got) != 2 {
		t.Fatalf("alerts after resolved+false = %+v, want 仍 2 条", got)
	}

	step(30) // 均值 (16+17+5+5+30)/5 = 14.6 < 15 → 仍 resolved，无转变
	step(50) // 均值 (16+17+5+5+30+50)/6 ≈ 20.5 > 15 → resolved → active（第 3 告警）
	got = st.alertsCopy()
	if len(got) != 3 || got[2].Message != "r1 触发: BTC@binance=20.5" {
		t.Fatalf("alerts after re-active = %+v, want 3 条且第 3 条 active", got)
	}
}

// TestNoDataStaysArmed：armed 且条件假 → 无任何落库（不写空状态行）。
func TestNoDataStaysArmed(t *testing.T) {
	st := newFakeStore([]store.Rule{
		{Name: "r1", Kind: fact.KindFunding, Cond: "avg_30d > 15", Level: store.LevelWarn, Enabled: true},
	}, nil)
	e, err := New(context.Background(), st, Config{Now: func() time.Time { return t0 }})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := e.EvaluateAll(context.Background()); err != nil {
		t.Fatalf("EvaluateAll: %v", err)
	}
	if got := st.alertsCopy(); len(got) != 0 {
		t.Fatalf("alerts = %+v, want 0", got)
	}
	if _, err := st.GetTriggerState(context.Background(), 1); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("GetTriggerState err = %v, want ErrNotFound（armed 不落行）", err)
	}
}

// TestTRXCrossingSemantics：此前 48h 均值为正（未穿越）→ 不告警；负历史出现
// （穿越发生）→ 告警。验证 avg_48h@24h 的"此前"语义。
func TestTRXCrossingSemantics(t *testing.T) {
	r := defaultByName(t, "trx_funding_positive")
	st := newFakeStore([]store.Rule{r}, []fact.Fact{
		fct(fact.KindFunding, "binance", "TRX", 1, -time.Hour),
		fct(fact.KindFunding, "binance", "TRX", 2, -30*time.Hour),
	})
	e, err := New(context.Background(), st, Config{Now: func() time.Time { return t0 }})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := e.EvaluateAll(context.Background()); err != nil {
		t.Fatalf("EvaluateAll: %v", err)
	}
	if got := st.alertsCopy(); len(got) != 0 {
		t.Fatalf("无穿越历史时 alerts = %+v, want 0", got)
	}
	// 此前 48h 窗口加入负值 → 均值 (2−4)/2 ≤ 0 → 穿越成立。
	if err := st.InsertFacts(context.Background(), []fact.Fact{
		fct(fact.KindFunding, "binance", "TRX", -4, -40*time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	if err := e.EvaluateAll(context.Background()); err != nil {
		t.Fatalf("EvaluateAll(2nd): %v", err)
	}
	got := st.alertsCopy()
	if len(got) != 1 || got[0].Level != store.LevelWarn {
		t.Fatalf("穿越后 alerts = %+v, want 1 warn", got)
	}
}

// TestScopeSymbolFilter：规则 scope（BTC,ETH）过滤掉 TRX；无 scope 命中不告警。
func TestScopeSymbolFilter(t *testing.T) {
	r := defaultByName(t, "funding_warn")
	st := newFakeStore([]store.Rule{r}, []fact.Fact{
		fct(fact.KindFunding, "binance", "TRX", 30, -time.Hour), // scope 外
	})
	e, err := New(context.Background(), st, Config{Now: func() time.Time { return t0 }})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := e.EvaluateAll(context.Background()); err != nil {
		t.Fatalf("EvaluateAll: %v", err)
	}
	if got := st.alertsCopy(); len(got) != 0 {
		t.Fatalf("TRX 命中 scope 外 alerts = %+v, want 0", got)
	}
	if err := st.InsertFacts(context.Background(), []fact.Fact{
		fct(fact.KindFunding, "binance", "BTC", 16, -time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	if err := e.EvaluateAll(context.Background()); err != nil {
		t.Fatalf("EvaluateAll(2nd): %v", err)
	}
	if got := st.alertsCopy(); len(got) != 1 {
		t.Fatalf("BTC 命中后 alerts = %+v, want 1", got)
	}
}

// TestBadCondRejectedAtNew：§11④ 坏表达式在引擎加载（启动）时即报错；
// 禁用规则同样校验（配置错误不因 disabled 而藏匿）。
func TestBadCondRejectedAtNew(t *testing.T) {
	for _, r := range []store.Rule{
		{Name: "bad1", Kind: fact.KindFunding, Cond: "avg_30d", Level: store.LevelWarn, Enabled: true},
		{Name: "bad2", Kind: fact.KindFunding, Cond: "foo_30d > 15", Level: store.LevelWarn, Enabled: false},
	} {
		st := newFakeStore([]store.Rule{r}, nil)
		if _, err := New(context.Background(), st, Config{}); err == nil {
			t.Errorf("New(%q) = nil, want error", r.Cond)
		}
	}
	if _, err := New(context.Background(), newFakeStore(nil, nil), Config{}); err == nil {
		t.Error("New(空规则集) = nil, want error")
	}
}

// TestRunPerRuleInterval：§7 调度——每规则独立间隔循环评估；状态机跨轮去重
// （快速间隔下仍只 1 条告警）；ctx 取消后 Run 及时返回。
func TestRunPerRuleInterval(t *testing.T) {
	st := newFakeStore([]store.Rule{
		{Name: "r1", Kind: fact.KindFunding, Cond: "avg_30d > 15", Level: store.LevelWarn, Enabled: true, IntervalSec: 3600},
	}, []fact.Fact{fct(fact.KindFunding, "binance", "BTC", 16, -time.Hour)})
	e, err := New(context.Background(), st, Config{
		Now:      func() time.Time { return t0 },
		Interval: func(store.Rule) time.Duration { return 5 * time.Millisecond },
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { _ = e.Run(ctx); close(done) }()

	waitFor(t, time.Second, func() bool { return st.queryCount() >= 5 })
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not return after cancel")
	}
	if got := st.alertsCopy(); len(got) != 1 {
		t.Fatalf("跨轮评估 alerts = %d, want 1（状态机去重）", len(got))
	}
}

// TestActiveMsgTruncates：命中实体 >3 时消息截断。
func TestActiveMsgTruncates(t *testing.T) {
	matches := make([]match, 4)
	for i := range matches {
		matches[i] = match{venue: "v", symbol: string(rune('A' + i)), value: float64(i)}
	}
	got := activeMsg("x", matches)
	if !strings.HasSuffix(got, ", …") || !strings.HasPrefix(got, "x 触发: ") {
		t.Errorf("activeMsg = %q, want 前缀 + 截断后缀", got)
	}
}

// TestOnActiveFiresOnTransitionOnly：关键规则触发事件（M2-b §5）只在
// armed/resolved→active 转变时回调；持续满足（状态不变）与 resolved 不回调。
// [对抗测试锚点] 删除 state.go transition() 里 e.onActive 调用 → 本测试必红。
func TestOnActiveFiresOnTransitionOnly(t *testing.T) {
	st := newFakeStore([]store.Rule{
		{Name: "r1", Kind: fact.KindFunding, Cond: "avg_30d > 15", Level: store.LevelWarn, Enabled: true},
	}, nil)
	clock := t0
	var fired []string
	onActive := func(_ context.Context, r store.Rule, _ []store.EntityHit) { fired = append(fired, r.Name) }
	e, err := New(context.Background(), st, Config{
		Now:      func() time.Time { return clock },
		OnActive: onActive,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	step := func(v float64) {
		t.Helper()
		clock = clock.Add(time.Minute)
		if err := st.InsertFacts(context.Background(), []fact.Fact{
			fct(fact.KindFunding, "binance", "BTC", v, -time.Hour),
		}); err != nil {
			t.Fatal(err)
		}
		if err := e.EvaluateAll(context.Background()); err != nil {
			t.Fatalf("EvaluateAll: %v", err)
		}
	}

	step(16) // armed → active：回调 1 次
	step(17) // active 持续满足：不回调
	step(5)  // 均值回落 → resolved：不回调（解除不是触发事件）
	step(50) // resolved → active 再触发：回调第 2 次
	if len(fired) != 2 || fired[0] != "r1" || fired[1] != "r1" {
		t.Errorf("onActive fired = %v, want [r1 r1]（仅 2 次 armed/resolved→active 转变）", fired)
	}
}
