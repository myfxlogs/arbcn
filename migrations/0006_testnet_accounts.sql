-- D-040 testnet 账户快照（SimExec 测试网账户区数据面，docs/design/04-m3-spec §9.4 S3 扩展）。
-- 探针每次余额查询成功后 upsert 一行（source 主键）；details 存每资产余额明细 JSONB。
-- equity_usd 口径因 source 而异（诚实标注，前端明示）：
--   sim_testnet_binance → 稳定币合计近似（USDT/USDC/BUSD/FDUSD 1:1，无行情折算非稳定币）
--   sim_testnet_okx     → totalEq（交易所精确折算）
-- 纯模拟盘数据，不进 freshness/告警链路（与 0005 同策略）。
CREATE TABLE IF NOT EXISTS sim_testnet_accounts (
    source        TEXT NOT NULL PRIMARY KEY,            -- sim_testnet_binance / sim_testnet_okx
    account_alias TEXT NOT NULL DEFAULT '',             -- binance accountAlias / okx 无
    equity_usd    DOUBLE PRECISION NOT NULL DEFAULT 0,
    details       JSONB NOT NULL DEFAULT '[]',          -- [{asset,balance,equity_usd}]
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
