-- D-046 市场结构经验库（knowledge_entries）：把「新情况 → 人工判定 → 结构化条目」落盘。
-- 吸收 = 人工 + D#（internal/knowledge.Defaults 落盘，git 跟踪；boot 幂等 upsert），
-- 匹配 = 确定性签名纯函数（internal/knowledge 探测器），呈现 = 只读 knowledge_match insight
-- + ListKnowledgeEntries RPC。
-- 系统永不自动吸收/自动改 verdict（practices #20：只读证据表面，优化方向由数据提示、
-- 动作由决策层 D# 拍板）。同签名同实体演进走「人工复核 → 新 D# → 更新条目」。
CREATE TABLE IF NOT EXISTS knowledge_entries (
    id              BIGSERIAL PRIMARY KEY,
    ts              TIMESTAMPTZ NOT NULL DEFAULT now(),   -- 吸收时刻
    signature       TEXT NOT NULL UNIQUE,                 -- 受控签名键（knowledge.Signature*；ON CONFLICT 键，镜像 rules.name UNIQUE）
    venue           TEXT NOT NULL DEFAULT '',             -- seed 实例 venue（溯源用）
    symbol          TEXT NOT NULL DEFAULT '',             -- seed 实例 symbol
    verdict         TEXT NOT NULL,                        -- 人工判定（D# 落）
    rationale       TEXT NOT NULL DEFAULT '',             -- 判定依据（中文）
    source          TEXT NOT NULL DEFAULT '',             -- 出处（对话 #N / D#）
    status          TEXT NOT NULL DEFAULT 'active',       -- active / superseded / retracted
    validated_at    TIMESTAMPTZ,                          -- 复核时刻；NULL = 待复核
    validation_note TEXT NOT NULL DEFAULT ''              -- 复核结论
);
