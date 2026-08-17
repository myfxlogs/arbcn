-- D-098 测试网执行层：sim_order_executions（镜像下单执行记录）。
-- ConfirmSimOrder 对 testnet/demo venue 订单逐腿镜像下单 → 回读成交 → 落一行；本地
-- sim_orders/sim_positions 仍是 PnL 大脑（D-037 真实费率），本表只记录「执行机制验证」。
-- best-effort：execution 成败不影响本地成交；status 值域 = placed/filled/partial/rejected/error。
-- 纯模拟盘数据，不进 freshness/告警链路（与 0005/0006 同策略）。
CREATE TABLE IF NOT EXISTS sim_order_executions (
    id                 BIGSERIAL PRIMARY KEY,
    order_id           BIGINT NOT NULL REFERENCES sim_orders(id) ON DELETE CASCADE,
    ts                 TIMESTAMPTZ NOT NULL DEFAULT now(),
    leg                TEXT NOT NULL DEFAULT '',            -- 腿标识（spot / perp）
    venue              TEXT NOT NULL,                       -- binance_testnet / okx_demo
    exchange_order_id  TEXT NOT NULL DEFAULT '',            -- 交易所订单号（回读；失败 = ''）
    symbol             TEXT NOT NULL,                       -- 交易所 instrument（如 BTCUSDT / BTC-USDT-SWAP）
    side               TEXT NOT NULL,                       -- long / short（本地腿方向）
    qty                DOUBLE PRECISION NOT NULL,           -- 请求 base 数量（换算后）
    fill_price         DOUBLE PRECISION NOT NULL DEFAULT 0, -- 回读成交均价（未成交 = 0）
    fill_qty           DOUBLE PRECISION NOT NULL DEFAULT 0, -- 回读已成交数量（未成交 = 0）
    status             TEXT NOT NULL,                       -- placed / filled / partial / rejected / error
    note               TEXT NOT NULL DEFAULT ''             -- 拒单/错误原因；成功可空
);
CREATE INDEX IF NOT EXISTS idx_sim_order_executions_order_id ON sim_order_executions(order_id);
