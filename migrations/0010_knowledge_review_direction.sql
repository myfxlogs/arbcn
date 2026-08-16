-- D-060 复核方向快照：knowledge_entries 加 review_direction 列。
-- 复核时把「当时证据方向」（hit=命中 / miss=未命中）快照下来，供系统后续做方向翻转
-- 检测（当前方向 ≠ 快照方向 → 建议复核；一致 → 仍适用无需动作）。NULL = 未快照
-- （该条复核早于本特性 / 复核时数据面不可判定），此时不做翻转检测（无从比较，宁缺毋滥，
-- practices #20：只呈现证据，不可比数据不吓人）。
-- 与 0001-0009 同策略：幂等（IF NOT EXISTS），兼容既有库（TestMigrateRollsBackFailedFile
-- 会重放全部真实迁移，非幂等 ALTER 必红）。
ALTER TABLE knowledge_entries ADD COLUMN IF NOT EXISTS review_direction text;
