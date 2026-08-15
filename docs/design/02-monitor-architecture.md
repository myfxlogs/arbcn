# 监控系统 v1 架构规格（施工权威文档）

> 决策依据：D-011（技术栈）/ D-020（监控先行）/ D-022（v1 范围）/ D-028（数据闭环）。
> 施工 agent 照此实现；遇设计疑问回找决策者 Claude，不自行变更设计（AGENTS.md §0）。
> 数据来源与阈值以 `docs/handoff/facts.md` 为准；本规格改动须走 decisions.md。

## 1. 定位与铁律

- 定位：**采集 → 归一 → 规则 → 告警 → 留痕**的数据管线 + 决策仪表盘。非交易系统。
- **铁律：无密钥**——只用公开只读 API；系统内永不存交易密钥；资金动作永远人工（D-010）。

## 2. 技术栈（D-011）

| 层 | 选择 |
|----|------|
| 核心 | Go（单二进制；每 collector 一个轮询协程） |
| 数据库 | PostgreSQL（docker-compose 独立容器 `arbcn-postgres`，宿主端口 5434；存储层走接口封装） |
| Web | React 19 + TS + Vite，go:embed 嵌入单二进制 |
| 通信 | ConnectRPC（net/http stdlib），单端口 :50052 |
| 日志 | log/slog 结构化 |
| 调度 | 自实现 cron（每源独立间隔 + 抖动防限流），不引第三方调度框架 |

## 3. 架构

```
Collectors(插件) → Normalizer → Fact{kind,venue,symbol,value,ts,src}
  → PG(facts/rules/trigger_states/alerts) → RuleEngine(声明式+状态机)
  → Alerter(推送) + FactsExporter(facts.md 更新)
  → HTTP :50052 (ConnectRPC + 静态仪表盘)
```

## 4. 组件接口（核心抽象，仅签名）

```go
// 每个数据源 = 一个 collector，插件化；失败重试独立，不拖垮全局
type Collector interface {
    Kind() string                  // 如 "funding" / "defi_rate" / "domestic" / "fx" / "calendar"
    Poll(ctx context.Context) ([]Fact, error)  // 只读公开 API，无密钥
}

// 统一事实模型：规则引擎只认 Fact，不认来源
type Fact struct {
    Kind   string    // funding / defi_rate / reverse_repo / fx / iv / calendar
    Venue  string    // binance / okx / aave / eastmoney ...
    Symbol string    // BTC / TRX / USDT / GC001 / USDCNH ...
    Value  float64   // 年化 % 或价格；口径由 Kind 约定
    Unit   string    // "pct_annualized" | "price" | ...
    Ts     time.Time
    Src    string    // 端点或采集器版本
}

// 规则引擎：声明式规则（配置行）+ 状态机（armed→active→resolved，状态转变才告警）
type Rule struct {
    Name    string
    Kind    string        // 命中哪个 Fact 流
    Cond    string        // 条件表达式（如 "avg_30d > 15"），由表达式求值器解析
    Level   string        // info / warn / critical
    Enabled bool
}
```

- 规则存 PG `rules` 表 = 配置，非代码。改阈值 = 改一行，不发布版本。
- 告警状态机：同一规则持续满足期间只发一次 active + 状态转变通知；resolved 时补发解除。

## 5. 数据源清单（v1 · 全公开无密钥）

| Collector | 源 | 内容 | 频率 |
|-----------|-----|------|------|
| Exchange | Binance fapi / OKX public / Bybit v5 / HTX 公开端点 | funding（含 TRX）、ticker | 1–5 分钟 |
| DeFiRates | DefiLlama Yields API | Aave / Morpho / sUSDe / 代币化美债 | 30–60 分钟 |
| Domestic | 东财/新浪公开行情 | 逆回购（GC001/R-001 等）、银行利率（爬取挂牌，失败降级人工录入） | 5–15 分钟 |
| FX | 新浪/东财公开行情 | USDCNH | 5 分钟 |
| Calendar | 规则计算（季末/月末/周四）+ 人工维护表 | 逆回购时点、国债发行、Launchpool | 日 |
| OptionsIV | Deribit DVOL / OKX 公开（受阻降级人工录入） | BTC/ETH IV（期权预算动态化） | 30 分钟 |

## 6. 数据库 Schema（v1）

| 表 | 字段要点 | 用途 |
|----|---------|------|
| facts | kind, venue, symbol, value, unit, ts, src；索引 (kind, symbol, ts) | 时序事实 |
| rules | name, kind, cond, level, enabled | 声明式规则 |
| trigger_states | rule_id, state, since, last_value | 告警状态机 |
| alerts | rule_id, ts, level, message, acked | 告警流（去重后） |
| ledger（v2 预留） | 台账/归因（M2） | — |

## 7. 规则集（v1 首版，阈值锚 facts.md）

| 规则 | 条件 | 级别 |
|------|------|------|
| funding 预警 | BTC/ETH 30 日滚动 funding >15% | warn |
| funding 激活 | 同上 >20% | critical（D-016 门禁线） |
| TRX 费率转正 | TRX funding 由负转正持续 24h | warn |
| 金额档利率变动 | 任一稳定币大额档利率环比 ±0.5% | info |
| 阶梯陷阱识别 | 头条档利率 > 大额档 3× | warn |
| 逆回购时点 | 日历事件前 1 天 + 当日 10:30 未配置提醒 | warn |
| 汇率线 | USDCNH < 6.6 | warn（D-015 加仓线） |
| IV 机会 | BTC IV 低于 30 日 P25 | info（期权加购窗口） |
| 计价币种陷阱 | 非稳定币计价收益产品利率变动 | warn（D-024 规则） |
| 元监控 | collector 心跳超时 / 数据库不可达 | critical |

## 8. RMB 折算层（M2）

所有事实的展示与告警均可切换计价：USD 计价事实 × 当日 USDCNH → RMB 净收益视角（D-023 必测项）。折算在查询/展示层完成，不污染原始事实。

## 9. 仪表盘（最小集）

| 视图 | 内容 |
|------|------|
| 机会面板 | funding 矩阵（币 × 所）、稳定币金额档利率表、IV、逆回购时点倒计时 |
| 触发器 | 各规则状态（armed/active/resolved）+ 历史转变 |
| 告警流 | alerts 时间线 + ack |
| 事实快照 | facts.md 的机器可读投影 |

**客户端形态（D-030）**：web 仪表盘先行（M1 浏览器直接可用）→ PWA 化（M2：manifest + service worker，手机主屏可装 + 推送）→ 原生 App 挂起（触发条件另立 D#）。

## 10. 部署与可靠性

- 部署：本机 systemd 常驻；`docker compose up -d arbcn-postgres`；单二进制 + 配置文件（数据源开关/间隔/告警通道）。
- `/healthz` + 元监控：collector 心跳超时 → critical 告警（错过窗口的代价最大）。
- 每 collector 独立 goroutine + 独立重试/退避；单源故障不影响其余。
- 告警通道：邮件 SMTP（QQ/163）默认；微信 Server酱 可选。

## 11. 测试硬要求（AGENTS.md §7.3 D 落地）

- 对抗测试：喂合成 Fact 序列 → 断言对应告警必发；删规则引擎状态机关键行 → 测试必红。
- 每个 collector 带样例响应 fixture（离线可测）。
- `go vet` + `go test -race` + 文件规模检查（≤300 行软 / 450 行硬）。

## 12. 里程碑验收

| 里程碑 | 内容 | 验收标准 |
|--------|------|---------|
| M1 | 5 类 collector + Fact + 规则引擎 + 告警 + 最小仪表盘 | 合成数据触发 funding>20% → 告警必达（邮件） |
| M2 | RMB 折算 + facts.md 自动导出 + 台账起步 | facts.md 自动更新、旧事实标过期（D-028 闭环） |
| M3 | 跨所费率差 + 出入金台账 + IV 期权预算建议 | 季末逆回购提醒实测触发 |

## 13. 明确不做（v1）

- ❌ 交易执行、任何密钥、下单 API
- ❌ 回测框架（机会策略验证先人工，M3 后另议）
- ❌ 多用户/权限体系（单人工具）
- ❌ 移动端 App（浏览器 + 邮件告警够用）
