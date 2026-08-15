-- M1-e 规则引擎（docs/design/02-monitor-architecture.md §4/§7）：
-- rules 增加实体 scope（venue/symbol 过滤，逗号分隔 = IN 列表；空 = 不限）
-- 与每规则独立评估间隔（秒，§7 调度："每规则独立间隔，配置化"）。
-- 与 0001 同策略：幂等（IF NOT EXISTS），兼容既有库。

ALTER TABLE rules ADD COLUMN IF NOT EXISTS venue TEXT NOT NULL DEFAULT '';
ALTER TABLE rules ADD COLUMN IF NOT EXISTS symbol TEXT NOT NULL DEFAULT '';
ALTER TABLE rules ADD COLUMN IF NOT EXISTS interval_sec INTEGER NOT NULL DEFAULT 300;
