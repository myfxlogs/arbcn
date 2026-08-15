-- M3-a 模拟执行（docs/design/04-m3-spec.md §1）：sim_orders（建议订单）+ sim_positions
-- （模拟成交腿）。独立 sim_ 前缀命名空间（§1.3），与主业务表物理可辨；模拟盘数据
-- 不进 freshness/告警链路，只进模拟对账视图。
-- 与 0001-0004 同策略：幂等（IF NOT EXISTS），兼容既有库。
--
-- sim_orders：建议订单（§1.1）。risk_flags 为门禁未过标记（text[]，UNHEDGED /
-- SPREAD_LOW / SIZE_OVER / DAILY_OVER / WHITELIST）；status 状态机
-- suggested → confirmed → filled / rejected / expired（§1.1 表 + §4 拒单负样本）。
--
-- sim_positions：模拟成交腿（§1.2）。hedge = 两行（现货 long + 永续 short），
-- carry/repo = 一行。funding 列标定资金费率结算腿（funding_hedge 的永续腿 /
-- carry_asset 生息腿），避免现货腿重复计息；pnl 按 funding 周期累计
-- （pnl = funding_rate × notional，§3.2）。

CREATE TABLE IF NOT EXISTS sim_orders (
    id              BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    ts              TIMESTAMPTZ NOT NULL,
    src_rule        TEXT NOT NULL,
    kind            TEXT NOT NULL,
    venue           TEXT NOT NULL,
    symbol          TEXT NOT NULL,
    side            TEXT NOT NULL,
    qty             DOUBLE PRECISION NOT NULL,
    ref_price       DOUBLE PRECISION NOT NULL,
    expected_spread DOUBLE PRECISION NOT NULL DEFAULT 0,
    risk_flags      TEXT[] NOT NULL DEFAULT '{}',
    status          TEXT NOT NULL CHECK (status IN ('suggested', 'confirmed', 'filled', 'rejected', 'expired')),
    note            TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS sim_orders_ts_idx ON sim_orders (ts);
CREATE INDEX IF NOT EXISTS sim_orders_status_idx ON sim_orders (status);
CREATE INDEX IF NOT EXISTS sim_orders_symbol_ts_idx ON sim_orders (symbol, ts);

CREATE TABLE IF NOT EXISTS sim_positions (
    id         BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    order_id   BIGINT NOT NULL REFERENCES sim_orders (id) ON DELETE CASCADE,
    ts         TIMESTAMPTZ NOT NULL,
    kind       TEXT NOT NULL,
    venue      TEXT NOT NULL,
    symbol     TEXT NOT NULL,
    side       TEXT NOT NULL,
    qty        DOUBLE PRECISION NOT NULL,
    ref_price  DOUBLE PRECISION NOT NULL,
    funding    BOOLEAN NOT NULL DEFAULT TRUE,
    pnl        DOUBLE PRECISION NOT NULL DEFAULT 0,
    status     TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'settled')),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS sim_positions_symbol_status_idx ON sim_positions (symbol, status);
CREATE INDEX IF NOT EXISTS sim_positions_order_id_idx ON sim_positions (order_id);
