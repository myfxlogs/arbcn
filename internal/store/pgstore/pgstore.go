// Package pgstore：store.Store 的 PostgreSQL 实现（pgx v5）。
// Schema 由 Migrate 保证（migrations/ 版本化迁移，启动时执行）。
package pgstore

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"arbcn/internal/fact"
	"arbcn/internal/store"
)

// factCols 与 migrations/0001_init.sql 的 facts 表列一致。
var factCols = []string{"kind", "venue", "symbol", "value", "unit", "ts", "src"}

// Store 包装 pgx 连接池；池生命周期由调用方管理。
type Store struct {
	pool *pgxpool.Pool
}

// New 构造 PG 实现。
func New(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

// InsertFacts 批量写入（COPY 协议）；先整批校验，任一非法则整批拒绝。
func (s *Store) InsertFacts(ctx context.Context, facts []fact.Fact) error {
	if len(facts) == 0 {
		return nil
	}
	rows := make([][]any, len(facts))
	for i, f := range facts {
		if err := f.Validate(); err != nil {
			return fmt.Errorf("pgstore: insert facts[%d]: %w", i, err)
		}
		rows[i] = []any{f.Kind, f.Venue, f.Symbol, f.Value, f.Unit, f.Ts, f.Src}
	}
	_, err := s.pool.CopyFrom(ctx, pgx.Identifier{"facts"}, factCols, pgx.CopyFromRows(rows))
	return err
}

// QueryFacts 按 FactQuery 过滤，ts 升序返回（窗口 [From, To)）。
func (s *Store) QueryFacts(ctx context.Context, q store.FactQuery) ([]fact.Fact, error) {
	where := []string{}
	args := []any{}
	addEq := func(col, val string) {
		if val == "" {
			return
		}
		args = append(args, val)
		where = append(where, fmt.Sprintf("%s = $%d", col, len(args)))
	}
	addEq("kind", q.Kind)
	addEq("venue", q.Venue)
	addEq("symbol", q.Symbol)

	to := q.To
	if to.IsZero() {
		to = time.Now()
	}
	args = append(args, q.From, to)
	where = append(where, fmt.Sprintf("ts >= $%d AND ts < $%d", len(args)-1, len(args)))

	limit := q.Limit
	if limit <= 0 {
		limit = 1000
	}
	args = append(args, limit)

	query := "SELECT " + strings.Join(factCols, ", ") + " FROM facts"
	if len(where) > 0 {
		query += " WHERE " + strings.Join(where, " AND ")
	}
	query += fmt.Sprintf(" ORDER BY ts ASC LIMIT $%d", len(args))

	rows, err := s.pool.Query(ctx, query, args...)
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

// UpsertRule 按 Name 幂等写入，返回 id（存在则覆盖字段）。
func (s *Store) UpsertRule(ctx context.Context, r store.Rule) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `
		INSERT INTO rules (name, kind, cond, level, enabled, venue, symbol, interval_sec)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (name) DO UPDATE
			SET kind = EXCLUDED.kind, cond = EXCLUDED.cond,
			    level = EXCLUDED.level, enabled = EXCLUDED.enabled,
			    venue = EXCLUDED.venue, symbol = EXCLUDED.symbol,
			    interval_sec = EXCLUDED.interval_sec
		RETURNING id`,
		r.Name, r.Kind, r.Cond, r.Level, r.Enabled, r.Venue, r.Symbol, r.IntervalSec,
	).Scan(&id)
	return id, err
}

// ListRules 按 name 升序返回全部规则。
func (s *Store) ListRules(ctx context.Context) ([]store.Rule, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, name, kind, cond, level, enabled, venue, symbol, interval_sec FROM rules ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []store.Rule{}
	for rows.Next() {
		var r store.Rule
		if err := rows.Scan(&r.ID, &r.Name, &r.Kind, &r.Cond, &r.Level, &r.Enabled,
			&r.Venue, &r.Symbol, &r.IntervalSec); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// GetTriggerState 读取规则状态机；无记录返回 store.ErrNotFound。
func (s *Store) GetTriggerState(ctx context.Context, ruleID int64) (store.TriggerState, error) {
	var st store.TriggerState
	err := s.pool.QueryRow(ctx,
		`SELECT rule_id, state, since, last_value FROM trigger_states WHERE rule_id = $1`,
		ruleID,
	).Scan(&st.RuleID, &st.State, &st.Since, &st.LastValue)
	if errors.Is(err, pgx.ErrNoRows) {
		return store.TriggerState{}, store.ErrNotFound
	}
	return st, err
}

// PutTriggerState upsert 状态机记录；Since 零值 = now。
func (s *Store) PutTriggerState(ctx context.Context, st store.TriggerState) error {
	since := st.Since
	if since.IsZero() {
		since = time.Now()
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO trigger_states (rule_id, state, since, last_value)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (rule_id) DO UPDATE
			SET state = EXCLUDED.state, since = EXCLUDED.since, last_value = EXCLUDED.last_value`,
		st.RuleID, st.State, since, st.LastValue,
	)
	return err
}

// InsertAlert 追加告警行；Ts 零值 = now。
func (s *Store) InsertAlert(ctx context.Context, a store.Alert) error {
	ts := a.Ts
	if ts.IsZero() {
		ts = time.Now()
	}
	_, err := s.pool.Exec(ctx,
		`INSERT INTO alerts (rule_id, ts, level, message) VALUES ($1, $2, $3, $4)`,
		a.RuleID, ts, a.Level, a.Message,
	)
	return err
}
