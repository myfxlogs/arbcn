package pgstore

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"arbcn/internal/fact"
	"arbcn/internal/store"
)

// migrationsDir：go test 的 cwd = 包目录（internal/store/pgstore），仓库根在 ../../../。
var migrationsDir = filepath.Join("..", "..", "..", "migrations")

// testPool 连接 ARBCN_TEST_PG_DSN 指定的专用测试库（可销毁数据）。
// 未设置时跳过：go test -race ./... 无 PG 环境仍全过。
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("ARBCN_TEST_PG_DSN")
	if dsn == "" {
		t.Skip("ARBCN_TEST_PG_DSN 未设置，跳过需真库的测试")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func resetTables(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tables ...string) {
	t.Helper()
	for _, tbl := range tables {
		if _, err := pool.Exec(ctx, "TRUNCATE "+tbl+" CASCADE"); err != nil {
			t.Fatalf("TRUNCATE %s: %v", tbl, err)
		}
	}
}

// ensureSchema 保证测试库 schema 就绪（包内测试相互独立，测试库可被前序用例清空）。
func ensureSchema(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := Migrate(ctx, pool, migrationsDir); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
}

// TestFactsRoundtrip：合成 Fact 写入 → 按 Kind/窗口/Symbol 查询，字段与排序逐项比对。
func TestFactsRoundtrip(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	ensureSchema(t, ctx, pool)
	resetTables(t, ctx, pool, "facts")

	now := time.Now().UTC().Truncate(time.Microsecond)
	facts := []fact.Fact{
		{Kind: fact.KindFunding, Venue: "binance", Symbol: "BTC", Value: 0.5, Unit: fact.UnitPctAnnualized, Ts: now.Add(-2 * time.Minute), Src: "test:binance"},
		{Kind: fact.KindFunding, Venue: "okx", Symbol: "BTC", Value: 0.25, Unit: fact.UnitPctAnnualized, Ts: now.Add(-time.Minute), Src: "test:okx"},
		{Kind: fact.KindFX, Venue: "sina", Symbol: "USDCNH", Value: 7.25, Unit: fact.UnitPrice, Ts: now.Add(-30 * time.Second), Src: "test:sina"},
	}
	s := New(pool)
	if err := s.InsertFacts(ctx, nil); err != nil {
		t.Fatalf("InsertFacts(empty): %v", err)
	}
	if err := s.InsertFacts(ctx, facts); err != nil {
		t.Fatalf("InsertFacts: %v", err)
	}

	got, err := s.QueryFacts(ctx, store.FactQuery{Kind: fact.KindFunding, From: now.Add(-time.Hour)})
	if err != nil {
		t.Fatalf("QueryFacts(funding): %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("QueryFacts(funding) len = %d, want 2", len(got))
	}
	assertFact(t, got[0], facts[0])
	assertFact(t, got[1], facts[1])

	// 窗口 [From, To) 右开：只含第一条。
	got, err = s.QueryFacts(ctx, store.FactQuery{From: facts[0].Ts, To: facts[1].Ts})
	if err != nil {
		t.Fatalf("QueryFacts(window): %v", err)
	}
	if len(got) != 1 || got[0].Venue != "binance" {
		t.Fatalf("QueryFacts(window) = %+v, want 1 row (binance)", got)
	}

	// Venue + Symbol 过滤。
	got, err = s.QueryFacts(ctx, store.FactQuery{Kind: fact.KindFX, Symbol: "USDCNH"})
	if err != nil {
		t.Fatalf("QueryFacts(fx): %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("QueryFacts(fx) len = %d, want 1", len(got))
	}
	assertFact(t, got[0], facts[2])

	// 窗口外应查无。
	got, err = s.QueryFacts(ctx, store.FactQuery{From: now.Add(-time.Hour), To: now.Add(-5 * time.Minute)})
	if err != nil {
		t.Fatalf("QueryFacts(empty): %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("QueryFacts(empty) len = %d, want 0", len(got))
	}
}

func assertFact(t *testing.T, got, want fact.Fact) {
	t.Helper()
	if got.Kind != want.Kind || got.Venue != want.Venue || got.Symbol != want.Symbol ||
		got.Value != want.Value || got.Unit != want.Unit || !got.Ts.Equal(want.Ts) || got.Src != want.Src {
		t.Errorf("fact = %+v, want %+v", got, want)
	}
}

// TestAlertRoundtrip：告警行写入 → 原样读回（acked 默认 false、ts 落库）。
func TestAlertRoundtrip(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	ensureSchema(t, ctx, pool)
	resetTables(t, ctx, pool, "rules", "alerts")

	s := New(pool)
	id, err := s.UpsertRule(ctx, store.Rule{
		Name: "r1", Kind: fact.KindFunding, Cond: "avg_30d > 15", Level: store.LevelWarn, Enabled: true,
	})
	if err != nil {
		t.Fatalf("UpsertRule: %v", err)
	}
	ts := time.Now().UTC().Truncate(time.Microsecond)
	if err := s.InsertAlert(ctx, store.Alert{RuleID: id, Ts: ts, Level: store.LevelWarn, Message: "r1 active"}); err != nil {
		t.Fatalf("InsertAlert: %v", err)
	}
	var (
		gotTS time.Time
		level string
		msg   string
		acked bool
	)
	if err := pool.QueryRow(ctx, `SELECT ts, level, message, acked FROM alerts WHERE rule_id = $1`, id).
		Scan(&gotTS, &level, &msg, &acked); err != nil {
		t.Fatalf("query alert: %v", err)
	}
	if !gotTS.Equal(ts) || level != store.LevelWarn || msg != "r1 active" || acked {
		t.Fatalf("alert = %v/%q/%q/%v, want %v/warn/r1 active/false", gotTS, level, msg, acked, ts)
	}
}

// TestInsertFactsRejectsUnknownKind：存储层兜底校验——删掉 InsertFacts 里的
// Validate 调用本测试必红（§11 对抗测试精神）。
func TestInsertFactsRejectsUnknownKind(t *testing.T) {
	pool := testPool(t)
	ensureSchema(t, context.Background(), pool)
	s := New(pool)
	err := s.InsertFacts(context.Background(), []fact.Fact{
		{Kind: fact.KindFunding, Venue: "binance", Symbol: "BTC"},
		{Kind: "stake", Venue: "x", Symbol: "y"},
	})
	if err == nil {
		t.Fatal("InsertFacts with unknown kind = nil, want error")
	}
}

// TestRulesAndTriggerStateRoundtrip：规则幂等 upsert + 状态机读写全链路。
func TestRulesAndTriggerStateRoundtrip(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	ensureSchema(t, ctx, pool)
	resetTables(t, ctx, pool, "rules", "trigger_states")

	s := New(pool)
	id1, err := s.UpsertRule(ctx, store.Rule{
		Name: "funding_warn", Kind: fact.KindFunding, Cond: "avg_30d > 15",
		Level: store.LevelWarn, Enabled: true, Symbol: "BTC,ETH", IntervalSec: 300,
	})
	if err != nil {
		t.Fatalf("UpsertRule: %v", err)
	}
	id2, err := s.UpsertRule(ctx, store.Rule{
		Name: "funding_warn", Kind: fact.KindFunding, Cond: "avg_30d > 20",
		Level: store.LevelCritical, Enabled: true, Symbol: "BTC,ETH", IntervalSec: 300,
	})
	if err != nil {
		t.Fatalf("UpsertRule(again): %v", err)
	}
	if id1 != id2 {
		t.Fatalf("UpsertRule ids = %d, %d, want equal", id1, id2)
	}

	rules, err := s.ListRules(ctx)
	if err != nil {
		t.Fatalf("ListRules: %v", err)
	}
	if len(rules) != 1 || rules[0].Cond != "avg_30d > 20" || rules[0].Level != store.LevelCritical {
		t.Fatalf("ListRules = %+v, want 1 updated rule", rules)
	}
	// scope/间隔字段随 upsert 往返（M1-e 迁移 0002）。
	if rules[0].Symbol != "BTC,ETH" || rules[0].IntervalSec != 300 {
		t.Fatalf("ListRules scope/interval = %q/%d, want BTC,ETH/300", rules[0].Symbol, rules[0].IntervalSec)
	}

	if _, err := s.GetTriggerState(ctx, id1); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("GetTriggerState(before) err = %v, want ErrNotFound", err)
	}

	st := store.TriggerState{RuleID: id1, State: store.StateActive, LastValue: 0.25}
	if err := s.PutTriggerState(ctx, st); err != nil {
		t.Fatalf("PutTriggerState: %v", err)
	}
	got, err := s.GetTriggerState(ctx, id1)
	if err != nil {
		t.Fatalf("GetTriggerState: %v", err)
	}
	if got.State != store.StateActive || got.LastValue != 0.25 || got.Since.IsZero() {
		t.Fatalf("GetTriggerState = %+v, want active/0.25/非零 since", got)
	}

	st.State = store.StateResolved
	if err := s.PutTriggerState(ctx, st); err != nil {
		t.Fatalf("PutTriggerState(resolved): %v", err)
	}
	got, err = s.GetTriggerState(ctx, id1)
	if err != nil {
		t.Fatalf("GetTriggerState(resolved): %v", err)
	}
	if got.State != store.StateResolved {
		t.Fatalf("GetTriggerState state = %q, want resolved", got.State)
	}
}
