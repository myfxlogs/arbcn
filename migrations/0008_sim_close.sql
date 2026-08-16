-- D-055 平仓（模拟持仓平仓闭环）：sim_orders.status 扩展 'closed'。
-- 平仓 = 人工在环（RPC CloseSimOrder）：订单全部持仓腿结算浮动 → status=settled，
-- 订单本身标记 closed（对冲结构整体退出，不赌原则 D-019——平仓必须整单，绝不单腿
-- 留裸敞口）。幂等：重建 CHECK 前先 drop 既有（0005 建的是无名内联约束，Postgres
-- 自动命名为 sim_orders_status_check）。

ALTER TABLE sim_orders DROP CONSTRAINT IF EXISTS sim_orders_status_check;
ALTER TABLE sim_orders ADD CONSTRAINT sim_orders_status_check
    CHECK (status IN ('suggested', 'confirmed', 'filled', 'rejected', 'expired', 'closed'));
