// 版本化迁移（决策层复审 M1-a 时裁决的 M1-b 任务项，dialogue #22）：
// 由 docker-entrypoint 迁出，改为程序启动时按文件名序执行 dir 下
// 尚未记账的 *.sql，schema_migrations 表记账。
package pgstore

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Migrate 执行 dir 下未记账的迁移文件，返回本次应用的数量。
// 每个文件在单事务内执行：失败整体回滚、不记账、立即中止。
// 已执行版本（schema_migrations 有记录）跳过；重复调用幂等。
// 迁移文件可用 IF NOT EXISTS 编写，以兼容 M1-a 时期
// 经 docker-entrypoint 初始化的既有库（0001 即如此）。
// 并发安全：advisory 锁串行化——多进程/并行测试包共享一库时，版本表查不到
// 与插入之间的窗口会让两个应用者同时应用同一文件（M1-e 引入第二测试包后实测
// 撞车），锁按连接持有，故迁移全程走专用连接。
func Migrate(ctx context.Context, pool *pgxpool.Pool, dir string) (int, error) {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return 0, fmt.Errorf("pgstore: acquire conn: %w", err)
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, `SELECT pg_advisory_lock(hashtext('arbcn_schema_migrations'))`); err != nil {
		return 0, fmt.Errorf("pgstore: lock migrations: %w", err)
	}
	defer func() {
		_, _ = conn.Exec(context.WithoutCancel(ctx), `SELECT pg_advisory_unlock(hashtext('arbcn_schema_migrations'))`)
	}()

	if _, err := conn.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version    TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`); err != nil {
		return 0, fmt.Errorf("pgstore: ensure schema_migrations: %w", err)
	}

	pending, err := listPending(ctx, conn, dir)
	if err != nil {
		return 0, err
	}

	n := 0
	for _, name := range pending {
		body, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return n, fmt.Errorf("pgstore: read migration %s: %w", name, err)
		}
		tx, err := conn.Begin(ctx)
		if err != nil {
			return n, fmt.Errorf("pgstore: begin migration %s: %w", name, err)
		}
		if _, err := tx.Exec(ctx, string(body)); err != nil {
			_ = tx.Rollback(ctx)
			return n, fmt.Errorf("pgstore: migration %s failed: %w", name, err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations (version) VALUES ($1)`, name); err != nil {
			_ = tx.Rollback(ctx)
			return n, fmt.Errorf("pgstore: record migration %s: %w", name, err)
		}
		if err := tx.Commit(ctx); err != nil {
			return n, fmt.Errorf("pgstore: commit migration %s: %w", name, err)
		}
		n++
	}
	return n, nil
}

// querier 是 Migrate / PendingMigrations 共用的最小查询面。
type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// listPending 返回 dir 下尚未在 schema_migrations 记账的 *.sql（升序，ReadDir 序）。
// schema_migrations 表必须已存在（调用方保证）。
func listPending(ctx context.Context, q querier, dir string) ([]string, error) {
	applied := map[string]bool{}
	rows, err := q.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("pgstore: read schema_migrations: %w", err)
	}
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			rows.Close()
			return nil, fmt.Errorf("pgstore: read schema_migrations: %w", err)
		}
		applied[v] = true
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("pgstore: read schema_migrations: %w", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("pgstore: read migrations dir %s: %w", dir, err)
	}

	pending := []string{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".sql") || applied[name] {
			continue
		}
		pending = append(pending, name)
	}
	return pending, nil
}

// PendingMigrations 报告 dir 下尚未应用的迁移文件（dialogue #23：未全部应用
// = degraded，/healthz 据此 503）。只读查询，不加 advisory 锁。
func PendingMigrations(ctx context.Context, pool *pgxpool.Pool, dir string) ([]string, error) {
	return listPending(ctx, pool, dir)
}
