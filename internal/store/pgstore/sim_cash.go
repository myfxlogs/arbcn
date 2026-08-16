// 模拟账户现金账本（D-056，migration 0009）：sim_account（单账户 id=1，capital 初始本金 +
// cash 现金余额）+ sim_cash_flow（逐笔流水，kind ∈ capital_in/open/funding/close）。
// 不变量（一套账两种展开）：equity = cash + Σ_open(dir×qty×cur) = capital + realized + unrealized。
// 现金流全部在既有事务内原子入账（开仓/平仓/结算各在其事务边界），纯本地模拟，
// 无任何真实账户/交易路径（D-010 无密钥铁律）。切出本文件防 sim.go 超 450 行。
package pgstore

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"

	"arbcn/internal/store"
)

// cashExer 抽象 *pgxpool.Pool 与 pgx.Tx 的 Exec（现金流流水 + 现金余额更新共用）。
type cashExer interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// applyCashFlow 入一笔现金流流水 + 原子更新现金余额（D-056）：INSERT sim_cash_flow
// （order_id/leg_id = 0 → NULL）+ upsert sim_account（id 恒 1）cash += amount。
// q 为 pool 或事务内 tx——调用方决定原子边界（开仓/平仓/结算各在其既有事务内）。
// 入金前账户不存在时用 ON CONFLICT upsert 兜底（cash 从 0 起加；capital 不受影响）。
func applyCashFlow(ctx context.Context, q cashExer, orderID, legID int64, kind string, amount float64) error {
	var oid, lid any
	if orderID != 0 {
		oid = orderID
	}
	if legID != 0 {
		lid = legID
	}
	if _, err := q.Exec(ctx, `
		INSERT INTO sim_cash_flow (order_id, leg_id, kind, amount)
		VALUES ($1, $2, $3, $4)`, oid, lid, kind, amount); err != nil {
		return fmt.Errorf("pgstore: cash flow %s: %w", kind, err)
	}
	if _, err := q.Exec(ctx, `
		INSERT INTO sim_account (id, capital, cash)
		VALUES (1, 0, $1)
		ON CONFLICT (id) DO UPDATE SET cash = sim_account.cash + EXCLUDED.cash, updated_at = now()`, amount); err != nil {
		return fmt.Errorf("pgstore: cash balance %s: %w", kind, err)
	}
	return nil
}

// applyOpenCashFlows 建腿 + 开仓现金流入账（D-056；FillSimOrder / AcceptSimOrder 两条
// 成交路径共用，口径必须一致）：逐腿 insertSimPosition（返回 leg id）+ open 流水
// （long −qty×ref / short +qty×ref）+ 现金余额，全部在同一事务。任一步失败 → 调用方
// 整体回滚，不留「成交了但现金没动」的半账。
func applyOpenCashFlows(ctx context.Context, tx pgx.Tx, orderID int64, legs []store.SimPosition) error {
	for _, p := range legs {
		if p.OrderID <= 0 {
			p.OrderID = orderID // 缺省回填（insertSimPosition 必填）
		}
		legID, err := insertSimPosition(ctx, tx, p)
		if err != nil {
			return fmt.Errorf("pgstore: open leg: %w", err)
		}
		dir := 1.0 // long 买入付钱
		if p.Side == store.SimSideShort {
			dir = -1.0 // short 卖出收钱
		}
		if err := applyCashFlow(ctx, tx, p.OrderID, legID, store.CashKindOpen, dir*p.Qty*p.RefPrice); err != nil {
			return err
		}
	}
	return nil
}

// SettleSimPositionFunding 结算资金费入账（D-056）：单事务——腿 pnl += addPnl
// （status 保持 open）+ 插 funding 现金流（+addPnl）+ 现金余额 cash += addPnl。
// 未知 id 无操作（与原 SettleSimPosition 语义一致）。仅在资金费结算路径调用
// （internal/sim/backfill.go SettleFunding）；平仓走 CloseSimOrder（close 现金流在内）。
func (s *Store) SettleSimPositionFunding(ctx context.Context, id, orderID int64, addPnl float64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("pgstore: settle funding: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() // 成功后 Rollback 是 no-op

	tag, err := tx.Exec(ctx, `
		UPDATE sim_positions
		SET pnl = pnl + $1, updated_at = now()
		WHERE id = $2`, addPnl, id)
	if err != nil {
		return fmt.Errorf("pgstore: settle funding: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return nil // 未知 id：无操作（幂等）
	}
	if err := applyCashFlow(ctx, tx, orderID, id, store.CashKindFunding, addPnl); err != nil {
		return fmt.Errorf("pgstore: settle funding: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("pgstore: settle funding: commit: %w", err)
	}
	return nil
}

// InitSimAccount 首启入金（D-056）：无 sim_account 行则插入（id=1, capital=capital,
// cash=0）+ capital_in 现金流 +capital（→ cash=capital）；已存在则幂等不动
// （重启不重置 cash，跨重启资金持久）。capital ≤ 0 拒绝。
// [对抗测试锚点 D-056] 单事务——账户行与 capital_in 流水原子（任一步失败整体回滚，
// 不留「有账户没流水」的半账；重启后首启失败可安全重试）。
func (s *Store) InitSimAccount(ctx context.Context, capital float64) error {
	if capital <= 0 {
		return errors.New("pgstore: init sim account: capital must be > 0")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("pgstore: init sim account: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() // 成功后 Rollback 是 no-op

	var id int
	err = tx.QueryRow(ctx, `
		INSERT INTO sim_account (id, capital, cash)
		VALUES (1, $1, 0)
		ON CONFLICT (id) DO NOTHING
		RETURNING id`, capital).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // 已存在：幂等不动（重启不重置 cash）
	}
	if err != nil {
		return fmt.Errorf("pgstore: init sim account: %w", err)
	}
	if err := applyCashFlow(ctx, tx, 0, 0, store.CashKindCapitalIn, capital); err != nil {
		return fmt.Errorf("pgstore: init sim account: capital_in: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("pgstore: init sim account: commit: %w", err)
	}
	return nil
}

// GetSimAccount 取单模拟账户；无行 → 返回零值 SimAccount（服务端可兜底，不报错）。
func (s *Store) GetSimAccount(ctx context.Context) (store.SimAccount, error) {
	var a store.SimAccount
	err := s.pool.QueryRow(ctx, `
		SELECT capital, cash, updated_at FROM sim_account WHERE id = 1`).
		Scan(&a.Capital, &a.Cash, &a.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return store.SimAccount{}, nil
	}
	return a, err
}

// ListCashFlows 按 ts DESC, id DESC 分页返回现金流流水（稳定排序，审计账本）。
// limit ≤ 0 = 默认 100，offset < 0 = 0。order_id/leg_id 可空（入金流水 = 0）。
func (s *Store) ListCashFlows(ctx context.Context, limit, offset int) ([]store.CashFlow, error) {
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, ts, order_id, leg_id, kind, amount, note
		FROM sim_cash_flow
		ORDER BY ts DESC, id DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("pgstore: list cash flows: %w", err)
	}
	defer rows.Close()
	out := []store.CashFlow{}
	for rows.Next() {
		var f store.CashFlow
		var oid, lid *int64
		if err := rows.Scan(&f.ID, &f.Ts, &oid, &lid, &f.Kind, &f.Amount, &f.Note); err != nil {
			return nil, err
		}
		if oid != nil {
			f.OrderID = *oid
		}
		if lid != nil {
			f.LegID = *lid
		}
		out = append(out, f)
	}
	return out, rows.Err()
}
