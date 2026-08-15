// 台账数据面（M2-b §6）：出入金流水读写 + 按 tier 归因汇总。
// 资金动作永远人工（§1）——本层只读/写人工录入的流水，无任何自动执行能力。
package pgstore

import (
	"context"
	"errors"
	"fmt"

	"arbcn/internal/store"
)

// InsertLedgerEntry 追加台账行；返回新行 id。date 零值 / channel / currency 为空 → 拒绝。
func (s *Store) InsertLedgerEntry(ctx context.Context, e store.LedgerEntry) (int64, error) {
	if e.Date.IsZero() {
		return 0, errors.New("pgstore: ledger: date required")
	}
	if e.Channel == "" || e.Currency == "" {
		return 0, errors.New("pgstore: ledger: channel and currency required")
	}
	var id int64
	err := s.pool.QueryRow(ctx, `
		INSERT INTO ledger (date, channel, currency, amount, fee_rate, tier, note)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id`,
		e.Date, e.Channel, e.Currency, e.Amount, e.FeeRate, e.Tier, e.Note,
	).Scan(&id)
	return id, err
}

// ListLedgerEntries 按 date DESC, id DESC 分页返回台账流水（稳定排序）。
func (s *Store) ListLedgerEntries(ctx context.Context, limit, offset int) ([]store.LedgerEntry, error) {
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, date, channel, currency, amount, fee_rate, tier, note
		FROM ledger
		ORDER BY date DESC, id DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []store.LedgerEntry{}
	for rows.Next() {
		var e store.LedgerEntry
		if err := rows.Scan(&e.ID, &e.Date, &e.Channel, &e.Currency, &e.Amount,
			&e.FeeRate, &e.Tier, &e.Note); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// LedgerSummary 按 tier 分组归因汇总（GROUP BY tier 简单分组，M2-b §6）：
// 入金和 / 出金绝对值之和 / 净额 / 笔数。
func (s *Store) LedgerSummary(ctx context.Context) ([]store.TierSummary, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT tier,
		       COALESCE(SUM(CASE WHEN amount > 0 THEN amount ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN amount < 0 THEN -amount ELSE 0 END), 0),
		       COALESCE(SUM(amount), 0),
		       COUNT(*)
		FROM ledger
		GROUP BY tier
		ORDER BY tier`)
	if err != nil {
		return nil, fmt.Errorf("pgstore: ledger summary: %w", err)
	}
	defer rows.Close()

	out := []store.TierSummary{}
	for rows.Next() {
		var ts store.TierSummary
		if err := rows.Scan(&ts.Tier, &ts.Inflow, &ts.Outflow, &ts.Net, &ts.EntryCount); err != nil {
			return nil, err
		}
		out = append(out, ts)
	}
	return out, rows.Err()
}
