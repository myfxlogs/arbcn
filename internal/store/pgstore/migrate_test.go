package pgstore

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
)

func dropTables(t *testing.T, ctx context.Context, pool *pgxpool.Pool, tables ...string) {
	t.Helper()
	for _, tbl := range tables {
		if _, err := pool.Exec(ctx, "DROP TABLE IF EXISTS "+tbl+" CASCADE"); err != nil {
			t.Fatalf("DROP %s: %v", tbl, err)
		}
	}
}

// TestMigrateIdempotent：对真库执行 migrations/ 目录（真实迁移文件）——
// 首跑建 5 表 + 记账；二跑 0 应用、无错误（幂等）。
func TestMigrateIdempotent(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	dropTables(t, ctx, pool, "facts", "rules", "trigger_states", "alerts", "ledger", "sim_orders", "sim_positions", "sim_testnet_accounts", "sim_account", "sim_cash_flow", "sim_equity_snapshots", "knowledge_entries", "schema_migrations")

	n, err := Migrate(ctx, pool, migrationsDir)
	if err != nil {
		t.Fatalf("Migrate(first): %v", err)
	}
	// 0001_init + 0002_rule_scope + 0003_alerts_delivered + 0004_ledger + 0005_sim +
	// 0006_testnet_accounts + 0007_knowledge + 0008_sim_close + 0009_sim_cash +
	// 0010_metric_tier_snapshot + 0011_equity_snapshot
	if n != 11 {
		t.Fatalf("Migrate(first) applied = %d, want 11", n)
	}

	for _, tbl := range []string{"facts", "rules", "trigger_states", "alerts", "ledger", "sim_orders", "sim_positions", "sim_testnet_accounts", "sim_account", "sim_cash_flow", "sim_equity_snapshots", "knowledge_entries"} {
		var exists bool
		if err := pool.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = $1)`,
			tbl).Scan(&exists); err != nil {
			t.Fatalf("check table %s: %v", tbl, err)
		}
		if !exists {
			t.Errorf("table %s missing after migrate", tbl)
		}
	}
	var versions int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM schema_migrations`).Scan(&versions); err != nil {
		t.Fatalf("count versions: %v", err)
	}
	if versions != 11 {
		t.Fatalf("schema_migrations count = %d, want 11", versions)
	}

	n, err = Migrate(ctx, pool, migrationsDir)
	if err != nil {
		t.Fatalf("Migrate(second): %v", err)
	}
	if n != 0 {
		t.Fatalf("Migrate(second) applied = %d, want 0", n)
	}
}

// TestMigrateRollsBackFailedFile：失败文件整体回滚——已成功的记账保留，
// 失败文件不记账、其 DDL 不生效。
func TestMigrateRollsBackFailedFile(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	dropTables(t, ctx, pool, "mtest_ok", "mtest_bad", "schema_migrations")

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "0001_ok.sql"),
		[]byte("CREATE TABLE mtest_ok (id INT);"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "0002_bad.sql"),
		[]byte("CREATE TABLE mtest_bad (id INT);\nTHIS IS NOT SQL;"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Migrate(ctx, pool, dir); err == nil {
		t.Fatal("Migrate with broken file = nil, want error")
	}

	var applied []string
	rows, err := pool.Query(ctx, `SELECT version FROM schema_migrations ORDER BY version`)
	if err != nil {
		t.Fatalf("query versions: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			t.Fatal(err)
		}
		applied = append(applied, v)
	}
	if len(applied) != 1 || applied[0] != "0001_ok.sql" {
		t.Fatalf("applied versions = %v, want [0001_ok.sql]", applied)
	}

	for tbl, want := range map[string]bool{"mtest_ok": true, "mtest_bad": false} {
		var exists bool
		if err := pool.QueryRow(ctx,
			`SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'public' AND table_name = $1)`,
			tbl).Scan(&exists); err != nil {
			t.Fatalf("check table %s: %v", tbl, err)
		}
		if exists != want {
			t.Errorf("table %s exists = %v, want %v", tbl, exists, want)
		}
	}

	// 恢复真实迁移记账（本用例 drop 了 schema_migrations 并指向 temp dir；
	// 不恢复则下一用例会把真实迁移全部重放一遍）。
	ensureSchema(t, ctx, pool)
}

// TestPendingMigrations：/healthz degraded 信号的数据面（dialogue #23）——
// 全部应用后为空；目录多出未记账文件 → 报告；记账表缺失 → 报错。
func TestPendingMigrations(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	ensureSchema(t, ctx, pool)

	pending, err := PendingMigrations(ctx, pool, migrationsDir)
	if err != nil {
		t.Fatalf("PendingMigrations(applied): %v", err)
	}
	if len(pending) != 0 {
		t.Fatalf("pending = %v, want empty", pending)
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "9999_future.sql"), []byte("SELECT 1;"), 0o644); err != nil {
		t.Fatal(err)
	}
	pending, err = PendingMigrations(ctx, pool, dir)
	if err != nil {
		t.Fatalf("PendingMigrations(extra file): %v", err)
	}
	if len(pending) != 1 || pending[0] != "9999_future.sql" {
		t.Fatalf("pending = %v, want [9999_future.sql]", pending)
	}

	dropTables(t, ctx, pool, "schema_migrations")
	if _, err := PendingMigrations(ctx, pool, migrationsDir); err == nil {
		t.Fatal("PendingMigrations without schema_migrations = nil, want error")
	}
	ensureSchema(t, ctx, pool) // 恢复测试库，供后续用例
}
