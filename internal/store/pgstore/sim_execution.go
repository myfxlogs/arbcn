// D-098 测试网执行层数据面：sim_order_executions 读写（Insert + List）。
// ConfirmSimOrder 对 testnet/demo venue 订单逐腿镜像下单 → 回读成交 → 落一行。
// best-effort：execution 成败不影响本地成交；本地 PnL 大脑（sim_positions/sim_cash，
// D-037 真实费率）不读本表——本表只记录「执行机制验证」（对账可见，不喂判定门）。
// 纯模拟盘数据（SIMULATED），无任何真实账户/交易路径（D-010 无密钥铁律）。
package pgstore

import (
	"context"
	"errors"
	"time"

	"arbcn/internal/store"
)

// InsertSimExecution 追加镜像下单执行记录（D-098）。order_id/venue/symbol 必填，qty > 0；
// ts 零值 = DB now()（不信任客户端时钟）。
func (s *Store) InsertSimExecution(ctx context.Context, e store.SimExecution) (int64, error) {
	if e.OrderID <= 0 {
		return 0, errors.New("pgstore: sim execution: order_id required")
	}
	if e.Venue == "" || e.Symbol == "" {
		return 0, errors.New("pgstore: sim execution: venue/symbol required")
	}
	if e.Qty <= 0 {
		return 0, errors.New("pgstore: sim execution: qty must be > 0")
	}
	ts := e.Ts
	if ts.IsZero() {
		ts = time.Now()
	}
	var id int64
	err := s.pool.QueryRow(ctx, `
		INSERT INTO sim_order_executions
			(order_id, ts, leg, venue, exchange_order_id, symbol, side, qty, fill_price, fill_qty, status, note)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id`,
		e.OrderID, ts, e.Leg, e.Venue, e.ExchangeOrderID, e.Symbol, e.Side,
		e.Qty, e.FillPrice, e.FillQty, e.Status, e.Note,
	).Scan(&id)
	return id, err
}

// ListSimExecutions 按 order_id 返回执行记录（ts ASC, id ASC 建单序）；
// orderID ≤ 0 = 全部（ts ASC）。无数据 = 空切片。
func (s *Store) ListSimExecutions(ctx context.Context, orderID int64) ([]store.SimExecution, error) {
	q := `
		SELECT id, order_id, ts, leg, venue, exchange_order_id, symbol, side, qty, fill_price, fill_qty, status, note
		FROM sim_order_executions`
	args := []any{}
	if orderID > 0 {
		q += ` WHERE order_id = $1`
		args = append(args, orderID)
	}
	q += ` ORDER BY ts ASC, id ASC`

	rows, err := s.pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []store.SimExecution{}
	for rows.Next() {
		var e store.SimExecution
		if err := rows.Scan(&e.ID, &e.OrderID, &e.Ts, &e.Leg, &e.Venue, &e.ExchangeOrderID,
			&e.Symbol, &e.Side, &e.Qty, &e.FillPrice, &e.FillQty, &e.Status, &e.Note); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
