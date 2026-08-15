# M2 规格：通知闭环 + 数据质量 + RMB 视角（施工权威文档）

> 决策依据：D-033（通知通道变更 + 范围定稿）。基于 02-monitor-architecture.md（M1 已交付），本规格只写增量。
> 施工 agent 照此实现；遇设计疑问回找决策者 Claude，不自行变更设计（AGENTS.md §0）。
> 数据来源与阈值以 `docs/handoff/facts.md` 为准；本规格改动须走 decisions.md。

## 0. M2 阶段划分

| 阶段 | 内容 | 依赖 |
|------|------|------|
| M2-a | 通知中心（铃铛）+ 源 freshness 徽标 + 事实去重 | M1（已交付） |
| M2-b | RMB 折算 + facts.md 自动导出 + 台账起步 | M2-a |

> 本轮施工先做 M2-a（业主明确要的铃铛 + 今日两个生产实证缺口）。M2-b 规格同文档备施工。

---

## 1. M2-a：通知中心（铃铛）

### 1.1 数据语义
- `alerts` 表已存在（`acked` 字段，M1）。**未读 = acked=false**；未读数 = `SELECT count(*) FROM alerts WHERE acked=false`。
- 通知通道变更（D-033）：铃铛为主通道；SMTP 为可选（未配置时告警照常进 alerts 表，M1 已实现，铃铛照常显示）。

### 1.2 proto 扩展（arbcn.dashboard.v1.DashboardService）
新增 2 个方法，复用既有 `ListAlerts`/`AckAlert`：
- `ListUnacked() → UnackedAlerts{items:[{id, rule, level, message, ts}], total}`：未读告警列表（含计数；一次拉全，未读数小）。
- `AckAll() → AckAllResponse{acked_count}`：全部已读（单事务 `UPDATE alerts SET acked=true WHERE acked=false`）。
- buf lint 全过；与既有 5 方法同服务。

### 1.3 前端（web/）
- 右上角**铃铛图标 + 未读红色徽标**（数字 = total）。
- 点击 → 下拉抽屉：未读告警列表（level 色徽标 + 规则名 + 消息 + 相对时间 + 逐条 ✓ack）+ 底部"全部标记已读"按钮。
- 空态："暂无新通知"。
- 轮询与现有 60s 合并（可复用告警流视图数据）；移动端抽屉全屏化（PWA 后置，D-030）。
- 深浅主题跟随现有设计。

### 1.4 对抗测试（AGENTS.md §7.3 D）
- 合成插入未读告警 → 铃铛计数显示 → 逐条 ack → 计数递减 → AckAll → 归零；删除 ListUnacked/AckAll 关键行 → 测试必红。

---

## 2. M2-a：源 freshness 徽标（今日生产实证缺口）

### 2.1 状态语义（区分"闭市"与"源死"）
| 状态 | 判定 | 含义 |
|------|------|------|
| `live` | now - last_poll ≤ 2×interval | 采集器正常 |
| `stale` | last_poll 新，但 now - last_fact > 2×interval | 采集正常但无新事实 = 市场闭市/报价冻结（R4#2 裁定：sched nextWait 抖动 ±10%，stale 阈值取 2×interval 留余量，防单源抖动瞬时误报 stale） |
| `down` | now - last_poll > 2×interval | 采集器失联（元监控同口径，collector_heartbeat >2） |

### 2.2 proto
- `ListSourceHealth() → SourceHealths{items:[{name, interval_sec, last_poll_at, last_fact_at, status}]}`
- 实现数据来源：heartbeat facts（每源 lastOK 时刻，`alert.Heartbeat`）+ 各 kind 最新 fact ts + 源间隔（sched 配置）。

### 2.3 前端
- 机会面板各 tile 加状态点（🟢 live / 🟡 stale / 🔴 down）+ 悬停 tooltip："最近更新 X 前 · 源间隔 Y · 状态 Z"。
- stale 文案区分业务含义：轮询正常但无新事实 → "市场闭市/冻结"；down → "采集器失联"。

### 2.4 对抗测试
- 注入 last_poll 老于 2×interval → down；last_poll 新但 last_fact 老 → stale；合成 heartbeat 验证边界切换。

---

## 3. M2-a：连续重复事实去重（今日生产实证缺口）

### 3.1 实现
- **位置**：`collect.Scheduler` 的 Sink 包装（dedup wrapper），对全部源统一生效，Collector 无感（P3 单点）。
- **规则**：按 `(kind, venue, symbol)` 记忆最后 `(value, ts)`；与上条相同 → 跳过本次 InsertFacts（不调 Sink）。
- **并发**：Scheduler 每源独立 goroutine，dedup map 须加互斥。
- **边界**：不同源同名 symbol 不冲突（venue 参与 key）；heartbeat 的 value 持续变化不受影响；value 变或 ts 变 → 照常落库。

### 3.2 对抗测试
- 喂相同 (value,ts) 两批 → 只落一批；值变化 → 落库；删除去重关键行 → 测试必红。

---

## 4. M2-b：RMB 折算（D-023 必测项）

- 展示层：USD 计价事实 × 当日 USDCNH → RMB 净收益视角；**原始事实不污染**（02-monitor-architecture.md §8）。
- 汇率源：`fx` kind 最新值（sina）；汇率缺失时展示 USD 原值 + "汇率不可用"标记。
- 覆盖：funding / defi_rate / deposit_rate（非 RMB 计价者）。
- **口径裁定（2026-08-15，决策层确认）**：对**年化收益率类**事实，"× 当日 USDCNH"不成立（6.84×7.25=49.6 荒谬）。
  按 D-023 算例（稳定币 4.5–6% 折算人民币、升值 3% 情景净 1.5–3%）裁定公式：
  **RMB 净收益 ≈ USD 收益率 − 年化人民币升值率**（30d 尾窗，`(last/first−1)/天数×365` 取负）。
  窗口 30d 与规则引擎同源；改动走 D#。实现 = `internal/rmb`（纯函数包）。

## 5. M2-b：facts.md 自动导出（D-028 闭环）

- `FactsExporter` 组件：定时（日）+ 关键规则触发事件，把监控最新值渲染进 `docs/handoff/facts.md`，旧值标"已过期"不删除（D-028 规则）。
- 与 pre-commit 门禁交互：facts.md 变更随 STATE 一起提交；写文件动作由 exporter 完成，git commit 需人工 review（或默认自动 + 可禁）。
- 机器可读投影：dashboard 事实快照视图（02 §9 挂起项）。

## 6. M2-b：台账起步

- `ledger` 表（02 §6 v2 预留）：日期/通道/币种/金额/费率/备注。
- 手工录入 RPC + 前端台账视图（出入金流水）。
- 归因：按档位（保本凸性/稳定币基档/现金管理）汇总。

---

## 7. 部署与测试硬要求

- 同 M1（02 §10/§11）：对抗测试、`go vet` + `go test -race`、行数门禁（豁免 `gen/` + `*_test.go`）。
- 通知中心不依赖 SMTP：SMTP 未配置 → 告警照常进 alerts 表（M1 已实现），铃铛照常显示（D-033）。

## 8. 明确不做（M2）

- ❌ Web Push / Service Worker 系统通知（D-030 PWA 化时评估）
- ❌ SMTP 真实投递验证（D-033：业主不申请授权码；实现保留可选通道）
- ❌ 自动执行、任何密钥
