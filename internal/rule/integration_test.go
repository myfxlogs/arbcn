package rule

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"arbcn/internal/fact"
	"arbcn/internal/store"
	"arbcn/internal/store/pgstore"
)

// migrationsDir：go test 的 cwd = 包目录（internal/rule），仓库根在 ../../。
var migrationsDir = filepath.Join("..", "..", "migrations")

// testPool 连接 ARBCN_TEST_PG_DSN 的派生库（库名 + "_rule"，不存在则自动创建）：
// 与 pgstore 包共用同一 DSN 但表名相同，go test 并行跑包时互相 TRUNCATE 会污染，
// 故本包独占一库（M1-e 实测撞车后加）。
// 未设置 DSN 时跳过（与 pgstore 同策略：go test -race ./... 无 PG 环境仍全过）。
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("ARBCN_TEST_PG_DSN")
	if dsn == "" {
		t.Skip("ARBCN_TEST_PG_DSN 未设置，跳过需真库的测试")
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("parse DSN: %v", err)
	}
	name := cfg.ConnConfig.Database + "_rule"
	createDatabase(t, cfg.ConnConfig, name)
	cfg.ConnConfig.Database = name
	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		t.Fatalf("pgxpool.New: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// createDatabase 经 bootstrap 库（postgres）创建派生测试库（存在则跳过）。
func createDatabase(t *testing.T, cc *pgx.ConnConfig, name string) {
	t.Helper()
	boot := *cc
	boot.Database = "postgres"
	conn, err := pgx.ConnectConfig(context.Background(), &boot)
	if err != nil {
		t.Fatalf("bootstrap conn: %v", err)
	}
	defer conn.Close(context.Background())
	var exists bool
	if err := conn.QueryRow(context.Background(),
		`SELECT EXISTS (SELECT 1 FROM pg_database WHERE datname = $1)`, name).Scan(&exists); err != nil {
		t.Fatalf("check db %s: %v", name, err)
	}
	if !exists {
		if _, err := conn.Exec(context.Background(), "CREATE DATABASE "+name); err != nil {
			t.Fatalf("create db %s: %v", name, err)
		}
	}
}

func ensureSchema(t *testing.T, ctx context.Context, pool *pgxpool.Pool) {
	t.Helper()
	if _, err := pgstore.Migrate(ctx, pool, migrationsDir); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
}

func resetTables(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tables ...string) {
	t.Helper()
	for _, tbl := range tables {
		if _, err := pool.Exec(ctx, "TRUNCATE "+tbl+" CASCADE"); err != nil {
			t.Fatalf("TRUNCATE %s: %v", tbl, err)
		}
	}
}

// alertRows 按规则取告警（level, message），ts 升序。
func alertRows(t *testing.T, ctx context.Context, pool *pgxpool.Pool, ruleID int64) []store.Alert {
	t.Helper()
	rows, err := pool.Query(ctx,
		`SELECT level, message FROM alerts WHERE rule_id = $1 ORDER BY ts ASC, id ASC`, ruleID)
	if err != nil {
		t.Fatalf("query alerts: %v", err)
	}
	defer rows.Close()
	var out []store.Alert
	for rows.Next() {
		var a store.Alert
		if err := rows.Scan(&a.Level, &a.Message); err != nil {
			t.Fatal(err)
		}
		out = append(out, a)
	}
	return out
}

// TestSeedAndEvaluatePG：全链路——迁移 0002 生效（scope/interval 列）→ Seed 幂等
// 落 10 条 → 合成事实触发 funding 两档 + 心跳 → 告警去重 → resolved 补发。
func TestSeedAndEvaluatePG(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	ensureSchema(t, ctx, pool)
	resetTables(t, ctx, pool, "facts", "rules", "trigger_states", "alerts")

	st := pgstore.New(pool)
	if n, err := Seed(ctx, st); err != nil || n != 10 {
		t.Fatalf("Seed = %d, %v, want 10", n, err)
	}
	if n, err := Seed(ctx, st); err != nil || n != 10 {
		t.Fatalf("Seed(2nd) = %d, %v, want 10（幂等）", n, err)
	}
	rules, err := st.ListRules(ctx)
	if err != nil {
		t.Fatalf("ListRules: %v", err)
	}
	if len(rules) != 10 {
		t.Fatalf("ListRules = %d, want 10", len(rules))
	}
	var critID int64
	for _, r := range rules {
		if r.Name == "funding_critical" {
			critID = r.ID
			if r.Symbol != "BTC,ETH" || r.IntervalSec != 300 {
				t.Fatalf("funding_critical scope/interval = %q/%d, want BTC,ETH/300（迁移 0002）", r.Symbol, r.IntervalSec)
			}
		}
	}
	if critID == 0 {
		t.Fatal("funding_critical not seeded")
	}

	e, err := New(ctx, st, Config{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	now := time.Now().UTC().Truncate(time.Microsecond)
	if err := st.InsertFacts(ctx, []fact.Fact{
		{Kind: fact.KindFunding, Venue: "binance", Symbol: "BTC", Value: 21, Unit: fact.UnitPctAnnualized, Ts: now.Add(-time.Hour)},
		{Kind: fact.KindFunding, Venue: "binance", Symbol: "BTC", Value: 23, Unit: fact.UnitPctAnnualized, Ts: now.Add(-2 * time.Hour)},
		{Kind: fact.KindHeartbeat, Venue: "collector", Symbol: "binance_funding", Value: 3, Unit: fact.UnitRatio, Ts: now.Add(-time.Minute)},
	}); err != nil {
		t.Fatalf("InsertFacts: %v", err)
	}

	if err := e.EvaluateAll(ctx); err != nil {
		t.Fatalf("EvaluateAll: %v", err)
	}
	got := alertRows(t, ctx, pool, critID)
	if len(got) != 1 || got[0].Level != store.LevelCritical || got[0].Message != "funding_critical active: BTC@binance=22" {
		t.Fatalf("funding_critical alerts = %+v, want 1 条 critical/avg22", got)
	}

	// 第二轮：持续满足 → 不重复（§4 状态转变才告警）。
	if err := e.EvaluateAll(ctx); err != nil {
		t.Fatalf("EvaluateAll(2nd): %v", err)
	}
	if got := alertRows(t, ctx, pool, critID); len(got) != 1 {
		t.Fatalf("funding_critical alerts after 2nd eval = %d, want 仍 1", len(got))
	}

	// 拉低均值 → resolved 补发。
	low := make([]fact.Fact, 100)
	for i := range low {
		low[i] = fact.Fact{Kind: fact.KindFunding, Venue: "binance", Symbol: "BTC", Value: 5,
			Unit: fact.UnitPctAnnualized, Ts: now.Add(-3 * time.Hour)}
	}
	if err := st.InsertFacts(ctx, low); err != nil {
		t.Fatalf("InsertFacts(low): %v", err)
	}
	if err := e.EvaluateAll(ctx); err != nil {
		t.Fatalf("EvaluateAll(3rd): %v", err)
	}
	got = alertRows(t, ctx, pool, critID)
	if len(got) != 2 || got[1].Message != "funding_critical resolved" {
		t.Fatalf("funding_critical alerts after resolve = %+v, want 2 条（第 2 条 resolved）", got)
	}
}
