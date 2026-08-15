-- M2-b 台账起步（docs/design/02-monitor-architecture.md §6 v2 预留）：
-- ledger 表存出入金流水（人工录入，资金动作永远人工 §1）。
-- 字段：date 出入金日期 / channel 通道 / currency 币种 / amount 金额（正入负出）/
--       fee_rate 费率 % / tier 档位（entry 自带，不推断，D-026 三档 + 持有层单列可选）/
--       note 备注。tier 用自由 TEXT（演进预留，不设 CHECK），已知值域在 store 常量。
-- 与 0001/0002/0003 同策略：幂等（IF NOT EXISTS），兼容既有库。

CREATE TABLE IF NOT EXISTS ledger (
    id       BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    date     TIMESTAMPTZ NOT NULL,
    channel  TEXT NOT NULL,
    currency TEXT NOT NULL,
    amount   DOUBLE PRECISION NOT NULL,
    fee_rate DOUBLE PRECISION NOT NULL DEFAULT 0,
    tier     TEXT NOT NULL DEFAULT '',
    note     TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS ledger_date_idx ON ledger (date);
CREATE INDEX IF NOT EXISTS ledger_tier_idx ON ledger (tier);
