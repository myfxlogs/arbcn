// 市场结构经验库读取/seed 路径（D-046）：knowledge_entries 表。
// 吸收 = 人工 + D#（git 跟踪 seed），呈现 = 只读；系统永不自动吸收/自动改 verdict。
package pgstore

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"arbcn/internal/store"
)

// ListKnowledgeEntries 按 signature 升序返回全部经验库条目（D-046 浏览）。
func (s *Store) ListKnowledgeEntries(ctx context.Context) ([]store.KnowledgeEntry, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, ts, signature, venue, symbol, verdict, rationale, source, status,
		       validated_at, validation_note, review_direction
		FROM knowledge_entries ORDER BY signature`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []store.KnowledgeEntry{}
	for rows.Next() {
		var (
			e  store.KnowledgeEntry
			va pgtype.Timestamptz
			rd *string // review_direction 可 NULL（复核早于 D-060 / 复核时数据面不可判定）
		)
		if err := rows.Scan(&e.ID, &e.Ts, &e.Signature, &e.Venue, &e.Symbol, &e.Verdict,
			&e.Rationale, &e.Source, &e.Status, &va, &e.ValidationNote, &rd); err != nil {
			return nil, err
		}
		if va.Valid {
			t := va.Time
			e.ValidatedAt = &t
		}
		if rd != nil {
			e.ReviewDirection = *rd
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// UpsertKnowledgeEntry 按 signature 确保条目存在并返回 id（镜像 UpsertRule 语义：
// 已存在 **不覆盖**——保留 DB 后续人工修订，seed 只引导新装）。冲突时取现有行 id。
func (s *Store) UpsertKnowledgeEntry(ctx context.Context, e store.KnowledgeEntry) (int64, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `
		INSERT INTO knowledge_entries (signature, venue, symbol, verdict, rationale, source, status, validation_note)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (signature) DO NOTHING
		RETURNING id`,
		e.Signature, e.Venue, e.Symbol, e.Verdict, e.Rationale, e.Source, e.Status, e.ValidationNote,
	).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		// 已存在（可能被人工修订过）→ 取现有 id，不覆盖。
		err = s.pool.QueryRow(ctx, `SELECT id FROM knowledge_entries WHERE signature = $1`, e.Signature).Scan(&id)
	}
	return id, err
}

// ReviewKnowledgeEntry 人工复核（D-054/D-060）：置 validated_at=now + 生命周期 status +
// 可选判定文本 verdict（NULLIF 空串 = 保留原判定）+ validation_note + 复核方向快照
// direction（NULLIF 空串 = 数据面不可判定，保留旧快照，不覆盖历史）。
// 只改判定记录（呈现面），不碰规则/门禁；未知 signature 返回 store.ErrNotFound。
func (s *Store) ReviewKnowledgeEntry(ctx context.Context, signature, status, verdict, note, direction string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE knowledge_entries
		SET validated_at = now(),
		    status = $2,
		    verdict = COALESCE(NULLIF($3, ''), verdict),
		    validation_note = $4,
		    review_direction = COALESCE(NULLIF($5, ''), review_direction)
		WHERE signature = $1`,
		signature, status, verdict, note, direction)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return store.ErrNotFound
	}
	return nil
}
