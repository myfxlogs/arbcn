-- D-062 阶段 0 判定门① 测量引擎数据面：equity 时点快照。
-- settleOnce 每 8h tick 落一份（ts 主键 + ON CONFLICT 幂等），供 TWR/MWR 跨窗口
-- 收益测量 + 判定门①（跨 30 天窗口 paper 净年化 ≥ 诚实基线 3.2-3.7% + 摩擦余量 0.3%）。
-- 只读测量数据面，不参与任何执行/门禁/规则/阈值（零执行改动，D-062）。
CREATE TABLE IF NOT EXISTS sim_equity_snapshots (
  ts           timestamptz PRIMARY KEY,
  equity       double precision NOT NULL,
  cash         double precision NOT NULL,
  realized     double precision NOT NULL,
  unrealized   double precision NOT NULL,
  market_value double precision NOT NULL
);
