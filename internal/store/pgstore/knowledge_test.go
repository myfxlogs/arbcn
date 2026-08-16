package pgstore

import (
	"context"
	"errors"
	"testing"

	"arbcn/internal/knowledge"
	"arbcn/internal/store"
)

// TestReviewKnowledgeEntryRoundtrip：人工复核写入 validated_at/note/verdict/方向快照 →
// 读回核对；未知 signature 返回 ErrNotFound。对抗：删 Update 的 validated_at 子句必红。
func TestReviewKnowledgeEntryRoundtrip(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	ensureSchema(t, ctx, pool)
	resetTables(t, ctx, pool, "knowledge_entries")

	s := New(pool)
	sig := knowledge.SignatureFundingSpikeTrap
	_, err := s.UpsertKnowledgeEntry(ctx, store.KnowledgeEntry{
		Signature: sig, Venue: "binance", Symbol: "ETH", Verdict: "陷阱", Rationale: "seed 判定", Source: "D#", Status: knowledge.StatusActive,
	})
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}

	if err := s.ReviewKnowledgeEntry(ctx, sig, knowledge.StatusRetracted, "判定文本已更新", "市场结构变化，撤回该判定", "miss"); err != nil {
		t.Fatalf("review: %v", err)
	}

	entries, err := s.ListKnowledgeEntries(ctx)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("want 1 entry, got %d", len(entries))
	}
	e := entries[0]
	if e.ValidatedAt == nil {
		t.Fatal("validated_at not set after review（删复核 SQL 的 validated_at 子句必红）")
	}
	if e.Status != knowledge.StatusRetracted {
		t.Fatalf("status = %q, want %q", e.Status, knowledge.StatusRetracted)
	}
	if e.Verdict != "判定文本已更新" {
		t.Fatalf("verdict = %q（复核判定文本未写入）", e.Verdict)
	}
	if e.ValidationNote != "市场结构变化，撤回该判定" {
		t.Fatalf("note = %q", e.ValidationNote)
	}
	if e.ReviewDirection != "miss" {
		t.Fatalf("review_direction = %q, want miss（复核方向快照未写入）", e.ReviewDirection)
	}

	// verdict 空 → 保留原判定（COALESCE(NULLIF(..)) 路径）；方向空 → 保留旧快照（同路径）。
	if err := s.ReviewKnowledgeEntry(ctx, sig, knowledge.StatusActive, "", "再次复核仅更新状态", ""); err != nil {
		t.Fatalf("review2: %v", err)
	}
	entries, _ = s.ListKnowledgeEntries(ctx)
	if entries[0].Verdict != "判定文本已更新" {
		t.Fatalf("空 verdict 未保留原判定：%q", entries[0].Verdict)
	}
	if entries[0].Status != knowledge.StatusActive {
		t.Fatalf("status = %q, want active", entries[0].Status)
	}
	if entries[0].ReviewDirection != "miss" {
		t.Fatalf("空 direction 未保留旧快照：%q", entries[0].ReviewDirection)
	}

	// 未知 signature → ErrNotFound。
	if err := s.ReviewKnowledgeEntry(ctx, "unknown:signature", knowledge.StatusActive, "", "x", ""); !errors.Is(err, store.ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}

// TestReviewDirectionNullScan：未复核条目（review_direction 为 NULL，复核早于 D-060）
// 浏览不报错——NULL 不能 scan 进 string，必须经 *string 兜底（本会话真实事故：
// 上线后 ListInsights 因 ListKnowledgeEntries 扫 NULL 直接 503）。[对抗测试锚点]
// 删掉 *string 处理改回直接 scan 进 string → 本测试必红。
func TestReviewDirectionNullScan(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	ensureSchema(t, ctx, pool)
	resetTables(t, ctx, pool, "knowledge_entries")

	s := New(pool)
	// 直接插一条未复核条目（validated_at = NULL、review_direction = NULL）。
	if _, err := s.UpsertKnowledgeEntry(ctx, store.KnowledgeEntry{
		Signature: knowledge.SignatureFundingSpikeTrap, Venue: "binance", Symbol: "ETH",
		Verdict: "陷阱", Rationale: "seed 判定", Status: knowledge.StatusActive,
	}); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	entries, err := s.ListKnowledgeEntries(ctx)
	if err != nil {
		t.Fatalf("list with NULL review_direction: %v（NULL 扫描兜底缺失）", err)
	}
	if len(entries) != 1 || entries[0].ReviewDirection != "" || entries[0].ValidatedAt != nil {
		t.Fatalf("entry = %+v, want 未复核（direction 空 / validated_at nil）", entries[0])
	}
}
