// 仪表盘读取路径（M1-g）：机会面板快照 / 告警流分页 + ack / 触发器视图。
package pgstore

import (
	"context"
	"strings"

	"github.com/jackc/pgx/v5/pgtype"

	"arbcn/internal/fact"
	"arbcn/internal/store"
)

// LatestFacts 每 (kind, venue, symbol) 取 ts 最新一条（DISTINCT ON；
// 空参数 = 不过滤）。结果按 kind, venue, symbol 排序（仪表盘快照友好）。
func (s *Store) LatestFacts(ctx context.Context, kind, venue, symbol string) ([]fact.Fact, error) {
	where := []string{"$1 = '' OR kind = $1", "$2 = '' OR venue = $2", "$3 = '' OR symbol = $3"}
	rows, err := s.pool.Query(ctx, `
		SELECT DISTINCT ON (kind, venue, symbol) kind, venue, symbol, value, unit, ts, src
		FROM facts
		WHERE `+strings.Join(where, " AND ")+`
		ORDER BY kind, venue, symbol, ts DESC`, kind, venue, symbol)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []fact.Fact{}
	for rows.Next() {
		var f fact.Fact
		if err := rows.Scan(&f.Kind, &f.Venue, &f.Symbol, &f.Value, &f.Unit, &f.Ts, &f.Src); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// ListAlerts 时间降序分页（ts DESC, id DESC 稳定排序）。
func (s *Store) ListAlerts(ctx context.Context, limit, offset int) ([]store.Alert, error) {
	if limit <= 0 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := s.pool.Query(ctx, `
		SELECT a.id, a.rule_id, r.name, a.ts, a.level, a.message, a.delivered, a.acked
		FROM alerts a JOIN rules r ON r.id = a.rule_id
		ORDER BY a.ts DESC, a.id DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []store.Alert{}
	for rows.Next() {
		var a store.Alert
		if err := rows.Scan(&a.ID, &a.RuleID, &a.RuleName, &a.Ts, &a.Level, &a.Message,
			&a.Delivered, &a.Acked); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// AckAlert 单条确认（幂等；未知 id 无操作，与 MarkAlertDelivered 同语义）。
func (s *Store) AckAlert(ctx context.Context, id int64) error {
	_, err := s.pool.Exec(ctx, `UPDATE alerts SET acked = TRUE WHERE id = $1`, id)
	return err
}

// ListUnacked 返回全部未读告警（acked=false；ts DESC, id DESC 稳定排序，M2-a §1.1）。
// 未读数小，一次拉全（不设 LIMIT）；JOIN rules 取规则名与 ListAlerts 同型。
func (s *Store) ListUnacked(ctx context.Context) ([]store.Alert, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT a.id, a.rule_id, r.name, a.ts, a.level, a.message, a.delivered, a.acked
		FROM alerts a JOIN rules r ON r.id = a.rule_id
		WHERE a.acked = FALSE
		ORDER BY a.ts DESC, a.id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []store.Alert{}
	for rows.Next() {
		var a store.Alert
		if err := rows.Scan(&a.ID, &a.RuleID, &a.RuleName, &a.Ts, &a.Level, &a.Message,
			&a.Delivered, &a.Acked); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// AckAll 全部已读：单条 UPDATE = 单事务（原子），返回 RowsAffected = 确认数。
func (s *Store) AckAll(ctx context.Context) (int64, error) {
	tag, err := s.pool.Exec(ctx, `UPDATE alerts SET acked = TRUE WHERE acked = FALSE`)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// ListTriggerStates 触发器视图：全部规则 LEFT JOIN 状态行；
// 未评估规则投影为 armed、Since 零值、LastValue nil。
func (s *Store) ListTriggerStates(ctx context.Context) ([]store.RuleState, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT r.name, COALESCE(t.state, 'armed'), t.since, t.last_value
		FROM rules r LEFT JOIN trigger_states t ON t.rule_id = r.id
		ORDER BY r.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []store.RuleState{}
	for rows.Next() {
		var (
			rs    store.RuleState
			since pgtype.Timestamptz
			last  pgtype.Float8
		)
		if err := rows.Scan(&rs.RuleName, &rs.State, &since, &last); err != nil {
			return nil, err
		}
		if since.Valid {
			rs.Since = since.Time
		}
		if last.Valid {
			v := last.Float64
			rs.LastValue = &v
		}
		out = append(out, rs)
	}
	return out, rows.Err()
}
