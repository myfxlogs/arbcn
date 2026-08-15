-- arbcn 监控 v1 schema（docs/design/02-monitor-architecture.md §6）。
-- 经 docker-entrypoint-initdb.d 在首次建库时自动执行；后续变更走新增迁移文件，不改本文件。

CREATE TABLE facts (
    id     BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    kind   TEXT NOT NULL,
    venue  TEXT NOT NULL,
    symbol TEXT NOT NULL,
    value  DOUBLE PRECISION NOT NULL,
    unit   TEXT NOT NULL DEFAULT '',
    ts     TIMESTAMPTZ NOT NULL,
    src    TEXT NOT NULL DEFAULT ''
);

CREATE INDEX facts_kind_symbol_ts_idx ON facts (kind, symbol, ts);

CREATE TABLE rules (
    id      BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name    TEXT NOT NULL UNIQUE,
    kind    TEXT NOT NULL,
    cond    TEXT NOT NULL,
    level   TEXT NOT NULL CHECK (level IN ('info', 'warn', 'critical')),
    enabled BOOLEAN NOT NULL DEFAULT TRUE
);

CREATE TABLE trigger_states (
    rule_id    BIGINT PRIMARY KEY REFERENCES rules (id) ON DELETE CASCADE,
    state      TEXT NOT NULL DEFAULT 'armed' CHECK (state IN ('armed', 'active', 'resolved')),
    since      TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_value DOUBLE PRECISION
);

CREATE TABLE alerts (
    id      BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    rule_id BIGINT NOT NULL REFERENCES rules (id) ON DELETE CASCADE,
    ts      TIMESTAMPTZ NOT NULL DEFAULT now(),
    level   TEXT NOT NULL CHECK (level IN ('info', 'warn', 'critical')),
    message TEXT NOT NULL,
    acked   BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX alerts_ts_idx ON alerts (ts);
