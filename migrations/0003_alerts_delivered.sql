-- M1-f Alerter（docs/design/02-monitor-architecture.md §4 告警流）：
-- alerts 增加投递状态列，Alerter 消费 delivered=false 的行、投递成功后置 true。
-- 与 0001/0002 同策略：幂等（IF NOT EXISTS），兼容既有库；存量行默认 false
-- （SMTP 配置后自动补投）。

ALTER TABLE alerts ADD COLUMN IF NOT EXISTS delivered BOOLEAN NOT NULL DEFAULT FALSE;
