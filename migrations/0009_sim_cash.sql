-- D-056 模拟账户「按真实账户对待」：完整现金账本数据面。
-- sim_account：单模拟账户（id 恒 1），capital = 初始本金（首启 InitSimAccount 写入，
-- 重启不重置），cash = 现金余额（随每笔 sim_cash_flow 原子增减，跨重启持久）。
-- sim_cash_flow：逐笔现金流流水（审计账本）。kind = capital_in / open / funding / close。
--   amount 正 = 入金（+ 现金），负 = 出金（− 现金）。
--   open：long 腿 −qty×ref（买入付钱）/ short 腿 +qty×ref（卖出收钱）；
--   funding：每 8h 资金费 +add（add = SettleFundingPnl(Per8hRate, qty)）；
--   close：long 腿 +qty×cur / short 腿 −qty×cur（cur = 平仓当前价，服务端算好传入）。
-- 不变量（D-056）：equity = cash + Σ_open(dir×qty×cur) = capital + realized + unrealized。
-- 与其他迁移一致用 IF NOT EXISTS（TestMigrateRollsBackFailedFile 会污染 schema_migrations，
-- 后续 Migrate 需能在表已存在时安全跳过，不报错）。
CREATE TABLE IF NOT EXISTS sim_account (
  id int PRIMARY KEY DEFAULT 1 CHECK (id = 1),
  capital double precision NOT NULL DEFAULT 0,
  cash    double precision NOT NULL DEFAULT 0,
  updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS sim_cash_flow (
  id bigserial PRIMARY KEY,
  ts timestamptz NOT NULL DEFAULT now(),
  order_id bigint,
  leg_id bigint,
  kind text NOT NULL,
  amount double precision NOT NULL,
  note text NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS sim_cash_flow_ts_idx ON sim_cash_flow (ts DESC);
