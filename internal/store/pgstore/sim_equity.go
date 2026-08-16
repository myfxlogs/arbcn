// equity 时点快照（D-062，migration 0011）：sim_equity_snapshots 表。
// 判定门① 测量引擎数据面——settleOnce 每 8h tick 落一份，供 TWR/MWR 跨窗口收益
// 测量 + 判定门① 判定（跨 30 天窗口 paper 净年化 ≥ 诚实基线 3.2-3.7% + 摩擦余量 0.3%）。
// 只读测量，不参与任何执行/门禁/规则/阈值（零执行改动，D-062）。切出本文件防超行。
package pgstore

import (
	"context"
	"fmt"
	"time"

	"arbcn/internal/store"
)

// InsertEquitySnapshot 落一份快照；ts 主键 ON CONFLICT 幂等（同 tick 重复落 = 保留首份）。
func (s *Store) InsertEquitySnapshot(ctx context.Context, snap store.EquitySnapshot) error {
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO sim_equity_snapshots (ts, equity, cash, realized, unrealized, market_value)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (ts) DO NOTHING`,
		snap.Ts, snap.Equity, snap.Cash, snap.Realized, snap.Unrealized, snap.MarketValue); err != nil {
		return fmt.Errorf("pgstore: insert equity snapshot: %w", err)
	}
	return nil
}

// ListEquitySnapshots 按 ts ASC 返回 [since, now) 内快照（升序 = TWR 链乘顺序）。
// since 零值 = 不过滤；limit ≤ 0 = 默认 10000（30 天 × 8h ≈ 90 份，默认足够）。
func (s *Store) ListEquitySnapshots(ctx context.Context, since time.Time, limit int) ([]store.EquitySnapshot, error) {
	if limit <= 0 {
		limit = 10000
	}
	q := `
		SELECT ts, equity, cash, realized, unrealized, market_value
		FROM sim_equity_snapshots`
	args := []any{}
	if !since.IsZero() {
		q += ` WHERE ts >= $1`
		args = append(args, since)
	}
	q += fmt.Sprintf(` ORDER BY ts ASC LIMIT $%d`, len(args)+1)
	args = append(args, limit)

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("pgstore: list equity snapshots: %w", err)
	}
	defer rows.Close()
	out := []store.EquitySnapshot{}
	for rows.Next() {
		var e store.EquitySnapshot
		if err := rows.Scan(&e.Ts, &e.Equity, &e.Cash, &e.Realized, &e.Unrealized, &e.MarketValue); err != nil {
			return nil, fmt.Errorf("pgstore: scan equity snapshot: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
