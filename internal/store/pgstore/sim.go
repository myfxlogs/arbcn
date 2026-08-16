// 模拟盘数据面（M3-a，docs/design/04-m3-spec.md §1/§3）：sim_orders（建议订单）+
// sim_positions（模拟成交腿）读写 + 日累计名义（DAILY_OVER 门禁数据面）。
// 照 M2-b ledger 模式：校验 + 稳定排序 + 幂等迁移（0005_sim.sql）。
// 纯本地模拟，无任何真实账户/交易路径（D-010 无密钥铁律）。
package pgstore

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"

	"arbcn/internal/store"
)

// insertSimOrder 字段序与 0005_sim.sql 列一致（含 id 读取回填）。
var simOrderCols = []string{"ts", "src_rule", "kind", "venue", "symbol", "side",
	"qty", "ref_price", "expected_spread", "risk_flags", "status", "note"}

// InsertSimOrder 追加建议订单；ts 零值 = now；kind/symbol/side 空或 qty ≤ 0 → 拒绝。
func (s *Store) InsertSimOrder(ctx context.Context, o store.SimOrder) (int64, error) {
	if o.Kind == "" || o.Symbol == "" || o.Side == "" {
		return 0, errors.New("pgstore: sim order: kind/symbol/side required")
	}
	if o.Qty <= 0 {
		return 0, errors.New("pgstore: sim order: qty must be > 0")
	}
	if o.RiskFlags == nil {
		o.RiskFlags = []string{} // NOT NULL DEFAULT '{}'：显式空切片避免 NULL
	}
	ts := o.Ts
	if ts.IsZero() {
		ts = time.Now()
	}
	var id int64
	err := s.pool.QueryRow(ctx, `
		INSERT INTO sim_orders (ts, src_rule, kind, venue, symbol, side, qty, ref_price,
		                        expected_spread, risk_flags, status, note)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id`,
		ts, o.SrcRule, o.Kind, o.Venue, o.Symbol, o.Side, o.Qty, o.RefPrice,
		o.ExpectedSpread, o.RiskFlags, o.Status, o.Note,
	).Scan(&id)
	return id, err
}

// ListSimOrders 按 ts DESC, id DESC 分页返回（稳定排序）。
func (s *Store) ListSimOrders(ctx context.Context, limit, offset int) ([]store.SimOrder, error) {
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, ts, src_rule, kind, venue, symbol, side, qty, ref_price,
		       expected_spread, risk_flags, status, note
		FROM sim_orders
		ORDER BY ts DESC, id DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []store.SimOrder{}
	for rows.Next() {
		var o store.SimOrder
		if err := rows.Scan(&o.ID, &o.Ts, &o.SrcRule, &o.Kind, &o.Venue, &o.Symbol, &o.Side,
			&o.Qty, &o.RefPrice, &o.ExpectedSpread, &o.RiskFlags, &o.Status, &o.Note); err != nil {
			return nil, err
		}
		out = append(out, o)
	}
	return out, rows.Err()
}

// GetSimOrder 按 id 取单条订单；未知 id → store.ErrNotFound。
func (s *Store) GetSimOrder(ctx context.Context, id int64) (store.SimOrder, error) {
	var o store.SimOrder
	err := s.pool.QueryRow(ctx, `
		SELECT id, ts, src_rule, kind, venue, symbol, side, qty, ref_price,
		       expected_spread, risk_flags, status, note
		FROM sim_orders WHERE id = $1`, id).
		Scan(&o.ID, &o.Ts, &o.SrcRule, &o.Kind, &o.Venue, &o.Symbol, &o.Side,
			&o.Qty, &o.RefPrice, &o.ExpectedSpread, &o.RiskFlags, &o.Status, &o.Note)
	if errors.Is(err, pgx.ErrNoRows) {
		return store.SimOrder{}, store.ErrNotFound
	}
	return o, err
}

// UpdateSimOrderStatus 更新状态 + 备注（note 非空时覆盖）；未知 id 无操作。
func (s *Store) UpdateSimOrderStatus(ctx context.Context, id int64, status, note string) error {
	if status == "" {
		return errors.New("pgstore: sim order: status required")
	}
	_, err := s.pool.Exec(ctx,
		`UPDATE sim_orders SET status = $1, note = CASE WHEN $2 <> '' THEN $2 ELSE note END WHERE id = $3`,
		status, note, id)
	return err
}

// TodaySimNotional 当日 [00:00, now) 活跃订单（suggested/confirmed/filled）名义和。
// 日界取 now 的服务器本地日（与"当日"人类语义一致）。
func (s *Store) TodaySimNotional(ctx context.Context, now time.Time) (float64, error) {
	start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	var sum float64
	err := s.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(qty), 0)
		FROM sim_orders
		WHERE status IN ($1, $2, $3) AND ts >= $4 AND ts < $5`,
		store.SimStatusSuggested, store.SimStatusConfirmed, store.SimStatusFilled,
		start, now,
	).Scan(&sum)
	return sum, err
}

// queryRower 抽象 *pgxpool.Pool 与 pgx.Tx 的 QueryRow（InsertSimPosition / FillSimOrder 共用）。
type queryRower interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// insertSimPosition 向 q（pool 或 tx）插入一条持仓腿；order_id 必填，qty ≤ 0 拒绝。
func insertSimPosition(ctx context.Context, q queryRower, p store.SimPosition) (int64, error) {
	if p.OrderID <= 0 {
		return 0, errors.New("pgstore: sim position: order_id required")
	}
	if p.Qty <= 0 {
		return 0, errors.New("pgstore: sim position: qty must be > 0")
	}
	ts := p.Ts
	if ts.IsZero() {
		ts = time.Now()
	}
	funding := p.Funding
	status := p.Status
	if status == "" {
		status = store.SimPosStatusOpen
	}
	var id int64
	err := q.QueryRow(ctx, `
		INSERT INTO sim_positions (order_id, ts, kind, venue, symbol, side, qty, ref_price,
		                           funding, pnl, status, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id`,
		p.OrderID, ts, p.Kind, p.Venue, p.Symbol, p.Side, p.Qty, p.RefPrice,
		funding, p.PnL, status, ts,
	).Scan(&id)
	return id, err
}

// InsertSimPosition 追加模拟成交腿（复用 insertSimPosition，pool 直连单条）。
func (s *Store) InsertSimPosition(ctx context.Context, p store.SimPosition) (int64, error) {
	return insertSimPosition(ctx, s.pool, p)
}

// FillSimOrder 原子成交（M3-a 复审 M1）：单事务「订单 confirmed→filled + 建全部腿」。
// 事务内先 UPDATE 且要求 status='confirmed'（RowsAffected 守卫：订单不存在/非 confirmed/
// 并发已成交 → 0 行 → 回滚拒绝），再 INSERT 全部腿；任一失败整体回滚。
// 不变量：filled 订单必有完整腿（不留半对冲裸敞口，D-019）；并发双插被状态守卫拦截。
func (s *Store) FillSimOrder(ctx context.Context, id int64, note string, legs []store.SimPosition) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("pgstore: fill sim order: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() // 成功后 Rollback 是 no-op

	tag, err := tx.Exec(ctx,
		`UPDATE sim_orders
		 SET status = $1, note = CASE WHEN $2 <> '' THEN $2 ELSE note END
		 WHERE id = $3 AND status = $4`,
		store.SimStatusFilled, note, id, store.SimStatusConfirmed)
	if err != nil {
		return fmt.Errorf("pgstore: fill sim order: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("pgstore: fill sim order %d: 非 confirmed/不存在（拒绝成交，防状态漂移）", id)
	}
	// D-056：建腿 + 开仓现金流同事务（applyOpenCashFlows 内含 insertSimPosition）。
	if err := applyOpenCashFlows(ctx, tx, id, legs); err != nil {
		return fmt.Errorf("pgstore: fill sim order: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("pgstore: fill sim order: commit: %w", err)
	}
	return nil
}

// AcceptSimOrder 人工确认原子成交（M3-c C3，替代"先置 confirmed 再 ConfirmAndFill"
// 两步——practices #8 原子性）：单事务 suggested→confirmed→filled + INSERT 全腿。
// 守卫：第一次 UPDATE 要求 status='suggested'，第二次要求 status='confirmed'
// （RowsAffected 任一为 0 → 整体回滚）。并发双确认 → 守卫 1 拦第二次（无重复建腿）；
// 事务内 confirmed 是中间态，外部永远看到 suggested 或 filled（无"已确认未成交"悬挂）。
func (s *Store) AcceptSimOrder(ctx context.Context, id int64, note string, legs []store.SimPosition) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("pgstore: accept sim order: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() // 成功后 Rollback 是 no-op

	tag, err := tx.Exec(ctx,
		`UPDATE sim_orders SET status = $1 WHERE id = $2 AND status = $3`,
		store.SimStatusConfirmed, id, store.SimStatusSuggested)
	if err != nil {
		return fmt.Errorf("pgstore: accept sim order: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("pgstore: accept sim order %d: 非 suggested/不存在（拒绝，防并发双确认/状态漂移）", id)
	}
	tag, err = tx.Exec(ctx,
		`UPDATE sim_orders
		 SET status = $1, note = CASE WHEN $2 <> '' THEN $2 ELSE note END
		 WHERE id = $3 AND status = $4`,
		store.SimStatusFilled, note, id, store.SimStatusConfirmed)
	if err != nil {
		return fmt.Errorf("pgstore: accept sim order: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("pgstore: accept sim order %d: confirmed 守卫未命中（回滚）", id)
	}
	// D-056：建腿 + 开仓现金流同事务（applyOpenCashFlows 内含 insertSimPosition）。
	if err := applyOpenCashFlows(ctx, tx, id, legs); err != nil {
		return fmt.Errorf("pgstore: accept sim order: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("pgstore: accept sim order: commit: %w", err)
	}
	return nil
}

// RejectSimOrder 确认时拒单（M3-c C3）：原子置 rejected + note 覆盖 + risk_flags
// 追加 flags（去重，array_agg(DISTINCT) 保序）。仅 status='suggested' 时生效
// （RowsAffected 守卫）；未知 id / 非 suggested → 报错。拒单 = 负样本保留（§4）。
// flags 为空 → 拒绝调用（调用方必须给至少一个标记，如 SPREAD_DRIFT）。
func (s *Store) RejectSimOrder(ctx context.Context, id int64, reason string, flags ...string) error {
	if len(flags) == 0 {
		return errors.New("pgstore: reject sim order: flags required")
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE sim_orders
		SET status = $1, note = $2,
		    risk_flags = (SELECT array_agg(DISTINCT f) FROM unnest(risk_flags || $3::text[]) f)
		WHERE id = $4 AND status = $5`,
		store.SimStatusRejected, reason, flags, id, store.SimStatusSuggested)
	if err != nil {
		return fmt.Errorf("pgstore: reject sim order: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("pgstore: reject sim order %d: 非 suggested/不存在（拒绝）", id)
	}
	return nil
}

// scanSimPositions 是 ListSimPositions / ListOpenSimPositions 共用的行扫描器。
func scanSimPositions(rows pgx.Rows) ([]store.SimPosition, error) {
	out := []store.SimPosition{}
	for rows.Next() {
		var p store.SimPosition
		if err := rows.Scan(&p.ID, &p.OrderID, &p.Ts, &p.Kind, &p.Venue, &p.Symbol, &p.Side,
			&p.Qty, &p.RefPrice, &p.Funding, &p.PnL, &p.Status, &p.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ListSimPositions 按 ts DESC, id DESC 分页返回持仓腿（稳定排序）。
func (s *Store) ListSimPositions(ctx context.Context, limit, offset int) ([]store.SimPosition, error) {
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, order_id, ts, kind, venue, symbol, side, qty, ref_price,
		       funding, pnl, status, updated_at
		FROM sim_positions
		ORDER BY ts DESC, id DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSimPositions(rows)
}

// ListOpenSimPositions 返回 open 持仓腿（symbol 空 = 不限；venue 空 = 不限；ts ASC
// 建仓序）。M3-b §9.3：venue 过滤支撑按 (symbol,venue) 分组结算，避免跨 venue 污染。
func (s *Store) ListOpenSimPositions(ctx context.Context, symbol, venue string) ([]store.SimPosition, error) {
	query := `
		SELECT id, order_id, ts, kind, venue, symbol, side, qty, ref_price,
		       funding, pnl, status, updated_at
		FROM sim_positions
		WHERE status = $1`
	args := []any{store.SimPosStatusOpen}
	if symbol != "" {
		args = append(args, symbol)
		query += fmt.Sprintf(" AND symbol = $%d", len(args))
	}
	if venue != "" {
		args = append(args, venue)
		query += fmt.Sprintf(" AND venue = $%d", len(args))
	}
	query += ` ORDER BY ts ASC, id ASC`
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("pgstore: list open sim positions: %w", err)
	}
	defer rows.Close()
	return scanSimPositions(rows)
}

// CloseSimOrder 人工平仓（D-055）：单事务原子——订单必须 status='filled'
// （RowsAffected 守卫：未知/非 filled/已平 → 0 行 → 回滚 ErrNotFound），全部
// closes 腿逐条「pnl += AddPnl + status='settled' + updated_at=now()」且要求
// 腿属于该订单且 status='open'（任一腿守卫 miss → 整体回滚，不留半仓裸敞口
// D-019），订单置 closed + note 追加。返回实际平掉的腿数。
// [对抗测试锚点] 删 filled 守卫 / 删腿 open 守卫 / 删 note 追加 → TestCloseSimOrder 必红。
func (s *Store) CloseSimOrder(ctx context.Context, orderID int64, note string, closes []store.SimLegClose) (int, error) {
	if len(closes) == 0 {
		return 0, errors.New("pgstore: close sim order: closes required")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("pgstore: close sim order: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }() // 成功后 Rollback 是 no-op

	tag, err := tx.Exec(ctx,
		`UPDATE sim_orders
		 SET status = $1, note = CASE WHEN $2 <> '' THEN $2 ELSE note END
		 WHERE id = $3 AND status = $4`,
		store.SimStatusClosed, note, orderID, store.SimStatusFilled)
	if err != nil {
		return 0, fmt.Errorf("pgstore: close sim order: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return 0, store.ErrNotFound
	}
	n := 0
	for _, c := range closes {
		tag, err := tx.Exec(ctx,
			`UPDATE sim_positions
			 SET pnl = pnl + $1, status = $2, updated_at = now()
			 WHERE id = $3 AND order_id = $4 AND status = $5`,
			c.AddPnl, store.SimPosStatusSettled, c.ID, orderID, store.SimPosStatusOpen)
		if err != nil {
			return 0, fmt.Errorf("pgstore: close sim order: leg %d: %w", c.ID, err)
		}
		if tag.RowsAffected() == 0 {
			return 0, fmt.Errorf("pgstore: close sim order %d: 腿 %d 非 open/不属于本单（回滚，防半仓）", orderID, c.ID)
		}
		// D-056：平仓现金流同事务（amount = c.CashDelta，long +qty×cur / short −qty×cur）。
		if err := applyCashFlow(ctx, tx, orderID, c.ID, store.CashKindClose, c.CashDelta); err != nil {
			return 0, fmt.Errorf("pgstore: close sim order %d: 腿 %d 现金流: %w", orderID, c.ID, err)
		}
		n++
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("pgstore: close sim order: commit: %w", err)
	}
	return n, nil
}
