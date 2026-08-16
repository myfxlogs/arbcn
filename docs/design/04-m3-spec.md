# M3 规格：模拟执行验证（施工权威文档 · 讨论稿）

> 决策依据：D-034（方向批准 + testnet key 豁免条款 + 加密模拟盘 Binance+OKX 都接 + 先细化设计后动工）。
> **本规格已获决策层确认**（D-034 ⑦ + 业主对 §4 风险门禁默认值的确认，2026-08-15）。施工 agent 照此实现；遇设计疑问回找决策者 Claude，不自行变更设计（AGENTS.md §0）。
> 前置依赖：**M2-b 已完成**（RMB 折算为 M3 对账基准）。M3 在 M2-b 后开工。

## 0. 定位与铁律（先读）

- **M3 目的**：把监控信号从"理论"变成"实证"——验证「信号 → 建议订单 → 人工确认 → 模拟成交 → 息差收敛」整条链路，全程不触真金。
- **真金执行不在 M3 范围**（D-034 ⑥）。M3 结束产出的是"模拟盘实证结论 + 可审计的模拟对账"，真金执行另立决策。
- **两条铁律全程生效**：
  - **无密钥铁律（D-010）**：真金路径零密钥不变。testnet key 按 D-034 豁免条款隔离使用（§2）。
  - **不赌原则（D-019）**：执行器只建议套利/息差类动作；无对冲的方向性建议**拒单**（白名单生息资产除外，见 §4 白名单行）。

## 0.1 M3 阶段划分

| 阶段 | 内容 | 验证目标 | 依赖 |
|------|------|---------|------|
| M3-a | 订单生成器（信号→建议订单）+ 本地模拟盘回填 | 信号→订单转换逻辑、盈亏计算、风险门禁 | M2-b |
| M3-b | testnet 只读接入 + 模拟持仓跑息差收敛 | 执行连通性 + **机制收敛**（结算管线 + 双边价差行为观察）；统计性收敛结论由历史数据出（§5.3，D-036） | M3-a |
| M3-c | 一键确认 UI + 风险门禁闭环 | 人工审价差后确认的完整流程 | M3-b |

> 顺序依赖明确：a → b → c。M3-a 纯本地（零外部接入），M3-b 引入 testnet 只读，M3-c 收口为 UI 流程。

---

## 1. 数据模型与模拟盘（M3 全局）

### 1.1 建议订单（SimOrder）
新表 `sim_orders`（独立前缀，与主业务表物理可辨）：

| 字段 | 类型 | 说明 |
|------|------|------|
| id | bigint identity | |
| ts | timestamptz | 生成时刻 |
| src_rule | text | 触发规则名（funding_warn / defi_* 等） |
| kind | text | 套利类型：`funding_hedge`（现货+永续对冲）/ `carry_asset`（白名单生息）/ `repo` 等 |
| venue | text | binance_testnet / okx_demo / sim_local |
| symbol | text | 如 BTCUSDT |
| side | text | long / short / hedge |
| qty | numeric | 名义数量 |
| ref_price | numeric | 生成时参考价（testnet 行情 or 本地） |
| expected_spread | numeric | 预期年化价差 % |
| risk_flags | text[] | 门禁未过标记（`UNHEDGED` / `SPREAD_LOW` / `SIZE_OVER`…） |
| status | text | `suggested` → `confirmed` → `filled` / `rejected` / `expired` |
| note | text | 拒单原因 / 结算备注 |

### 1.2 模拟持仓（SimPosition）
新表 `sim_positions`：记录模拟成交的腿（hedge = 两行，carry = 一行），按 funding 周期结算息差累计到每腿 `sim_positions.pnl` 列（0005 迁移实现；早期文档稿的 `sim_pnl` 表已废弃，实现以迁移为准——P3 单一真相源）。结算逻辑 = 实际资金费率 × 名义 × 持有天数（RMB 折算后入账，依赖 M2-b）。

### 1.3 命名空间
- 模拟盘表统一 `sim_` 前缀；模拟配置独立文件（§2）；不混入主 facts/rules/alerts 表。
- 模拟盘数据**不进** freshness/告警链路（那是真实监控）；只进模拟对账视图。

---

## 2. testnet key 隔离（D-034 ② 落地）

- 独立配置文件 `/etc/arbcn/arbcn-sim.env`（root:root 0600），键名 `SIM_*`；加载于独立 sim 包。
- 每 key 显式 `SIMULATED=true` 标记；配置校验：缺 `SIMULATED=true` 的 key **拒绝加载**（防误用真金 key）。
- 只读原则（§3.2）：M3 只用 testnet **公共行情 + 账户只读查询**（余额/费率），**不发送任何委托单**。因此 key 实际只需只读权限；实现上按"能读就行"要求最小权限。
- 模拟盘逻辑包 `internal/sim`，**不 import** 任何真实账户/交易代码路径；真实 collector 只读行情可复用（行情本身无 key）。
- 代码审查门禁：`internal/sim` 包内禁止出现真实交易端点域名（`fapi.binance.com` 等主网域），只允许 testnet/demo 域。

---

## 3. M3-a：订单生成器 + 本地模拟盘

### 3.1 输入
- 监控信号：规则命中（funding_* / defi_* / repo 等）+ 机会面板当前快照。
- 转换器 `signalToOrder(rule, snapshot)`：信号 → 建议订单结构。**转换逻辑是 M3-a 核心，须纯函数、可单测。**（实现签名：`SignalToOrder(sig, cfg)`，`Signal` 由驱动层组装，见 §3.1.2。）

### 3.1.1 规则 → Signal 映射（D-036 G1 补齐 · M3-b 集成前必读）
规则命中**不自动**产生模拟单；只有可映射为套利/息差动作的规则才组装 Signal（宁缺毋滥：无法映射的规则不建单）。映射为 sim 包内不可变常量表 + 对抗测试（未知规则名 → 断言不建单）：

| 触发规则（defaults.go 首版规则名） | Signal.Kind | Signal 填充 |
|------|------|------|
| `funding_warn` / `funding_critical` / `trx_funding_positive` | `funding_hedge` | Symbol=规则 scope 命中实体；RefPrice=现货最新价；FundingAnn=命中时年化 funding（规则 cond 的 avg_30d 口径）；SpotPrice / PerpPrice=现货 / 永续最新价；Notional=默认（capital×20%） |
| `funding_drill`（D-041 演练档，band `avg_30d > 5 && < 15`） | `funding_hedge` | 同上。Info 级、只喂模拟盘：band 下限=跨过 SPREAD_LOW 5%、上限=低于 funding_warn 门槛 15%（不重复真实窗口档）；供演练确认→成交→8h 结算全链路。 |
| `reverse_repo_timing` | `repo` | RefPrice=面值；ExpectedSpread=当日回购年化（manual 补录事实）；天然无方向敞口（§4 已对冲） |
| carry 信号（白名单生息资产 sUSDe/USDe，D-021 档位） | `carry_asset` | RefPrice=资产价值；ExpectedSpread=生息年化；CarryWhite=白名单命中（驱动层信任边界，M2 接受项；白名单显式配置在 M3-b 接 testnet 前落，`ARBCN_SIM_CARRY_WHITELIST`） |
| `defi_large_tier_change` / `ladder_trap` / `iv_opportunity` / `usdcnh_buy_line` / `collector_heartbeat` | — | **不产生模拟单**（信息类 / IV 非 M3 范围 / 遥测） |

组装发生在驱动层（§3.1.2），从 facts 快照（LatestFacts）取数；`SignalToOrder` 保持纯函数、零 I/O。

### 3.1.2 运行驱动（M3-b 集成时接线 · M3-a 交付为纯库有测试）
- **触发器**：挂钩 `rule.Engine.OnActive`（armed→active 转变 = "机会出现"时刻；避免持续满足时每评估周期重复建单的噪声）。M3-b 集成时把 `OnActive` 签名扩展为携带命中实体（symbol）列表——规则引擎本就逐实体独立聚合，传命中实体是自然小改（engine.go）。
- **流程**：`OnActive(rule, entities)` → 对每个命中实体查 LatestFacts（spot / perp / funding）→ 按 §3.1.1 组装 Signal → `SignalToOrder` → `InsertSimOrder`（生成时落库 `suggested` + risk_flags）。
- **结算调度**：按 8h 周期对 open 腿 `SettleFunding`（复用 collect.Scheduler 骨架）。
- **驱动层独立单测**：OnActive 回调 → 断言 InsertSimOrder 落库；映射表未知规则 → 断言不建单。
- **降级**：sim 配置缺失 → sim 驱动禁用（§7 warn 不退出），监控主循环不受影响。

### 3.2 本地模拟盘回填
- 无 testnet 也运行：成交 = 按 `ref_price` 全额即时成交（简化：忽略滑点/深度，标注为模拟假设）。
- `filled` + 建 `sim_positions` 原子完成（store 单事务 `FillSimOrder`，复审 M1：不留"filled 但缺腿"的半对冲状态，D-019）；按 funding 周期（8h，三班）结算：`pnl = funding_rate × notional`，其中 `funding_rate` 为**分数费率** = 年化点数 ÷100 ÷1095（复审 H1：facts 的 pct_annualized 是百分点点数，先 ÷100 转分数再乘名义；原实现缺 ÷100 使模拟 PnL 虚高 100 倍）；日终 RMB 折算复用 M2-b 折算函数，RMBValue（点数）同样 ÷100 转分数。
- **资金基数口径（D-036 G3）**：模拟资金基数 `Capital`（默认 100_000，模拟 USD，`ARBCN_SIM_CAPITAL`）是**独立模拟量纲**，不映射真实组合规模（真实 20 万 RMB ≈ 2.8 万 USD，且为 50/50 结构性配置）。模拟对账报告以**比例口径（PnL / Capital %）**为主列、绝对模拟数值为次列，明示"不直接映射真实组合"。
- **确认价时刻（D-036 G5）**：M3-c 确认时成交价取**确认时刻最新 ref_price**（重新查行情），不沿用生成时 ref_price；生成 → 确认窗口内 ref_price 漂移 > 2%（或预期年化变化 > 20%）→ 确认时二次门禁拒单（新标记 `SPREAD_DRIFT`，M3-c 实现）。

### 3.3 对抗测试锚点
- 转换器：删"无对冲拒单"分支 → 测试必红；删"预期价差 < 阈值拒单" → 必红。
- 结算：删 `× funding_rate` → 累计 PnL 断言必红。

---

## 4. 风险门禁（M3 核心 · §4 全阶段生效）

> **已按默认值定稿**（业主确认，2026-08-15）；改动走 D#。

| 门禁 | 规则 | 拒绝行为 |
|------|------|---------|
| 对冲 | `kind=funding_hedge` 必须双腿齐备（现货+永续，方向对冲） | 缺腿 → `UNHEDGED` 拒单 |
| 方向性 | 非白名单、无对冲 → 拒单 | `UNHEDGED` |
| 价差 | 预期年化 < **5%**（摩擦覆盖）→ 拒单 | `SPREAD_LOW` |
| 单笔 | 名义 > 模拟资金 **20%** → 拒单 | `SIZE_OVER` |
| 日累计 | 当日新增名义 > 模拟资金 **50%** → 拒单 | `DAILY_OVER` |
| 白名单 | `carry_asset` 标的须在显式白名单（sUSDe/USDe 等生息资产） | 未白名单 → 拒单（M3-a 裁定：白名单解析在驱动层 `Signal.CarryWhite`，属信任边界；M3-b 接 testnet 行情数据前须落显式白名单配置，由 `SignalToOrder` 校验 symbol） |

- 门禁在**生成时**执行一次（risk_flags 落库），**确认时**二次校验（M3-c，UI 层再算一遍——防生成到确认之间的状态漂移）。
- 拒单不是失败：拒单记录 = 有价值的负样本（哪些信号不该执行），纳入对账分析。

---

## 5. M3-b：testnet 只读接入 + 息差收敛

### 5.1 接入
- Binance Testnet + OKX Demo 双接入（D-034 ④）：公共行情源（复用既有 collector 骨架，换 testnet 域 + testnet 鉴权）+ 账户只读查询（余额核对用，验证 key 连通）。**用途 = §2 key 隔离机制验证（D-034 豁免条款落地），不是结算数据源**。
- **结算数据源裁决（D-037）**：模拟持仓的 funding 结算取**真实市场公开 funding**（既有 binance_funding/okx_funding collector 已采，无 key）——testnet 费率有偏差（本 spec 自标），喂结算会污染机制验证数字；testnet 只做连通性/key 隔离验证（M3-b §9.4）。已知限制保留：真实采集瞬时偏差不可避免，但无 testnet 额外污染。

### 5.2 对账
- 模拟持仓的 funding 由**真实市场公开 funding 事实**喂入（§5.1 裁决）→ 按 §3.2 结算 → 息差曲线（周视图）。
- 对账输出 `sim_report`（周频）：模拟净值 vs **理论净值（无摩擦理想曲线）**，差异 = 摩擦 + testnet 偏差 → 归因。
- **理论曲线定义（D-036 G4）**：cash-and-carry 定价下永续价收敛于现货（价差 → 0）；理想 funding 累计 = 建仓时预期年化 × 名义 × 持有天数 ÷ 365（每 8h 结算）；摩擦模型 = 手续费（双边）+ 滑点（按 testnet 深度差）+ 现货腿资金占用成本。差异 = 实测 − 理论 = 摩擦 + testnet 偏差。
- **验证目标分层（D-036 收敛口径修正）**：
  - M3-b 前向模拟**只验证机制收敛**：结算管线正确（每 8h 收付、双边价差在 testnet 数据下的行为观察）——这是前向模拟能证明的。
  - **统计性结论**（是否真收敛、收敛速度、残差分布）由**历史数据**出（§5.3）：testnet 周级前向小样本量 + testnet 费率偏差，回答不了统计问题且污染结论。

### 5.3 历史收敛分析（收敛统计的唯一证据 · D-036）
- **数据**：Binance funding 历史（`data-api.binance.vision/fapi/v1/fundingRate`，D-031 公开数据域）+ OKX funding 历史（`/api/v5/public/funding-history`）+ 现货/永续价差历史。公开只读 API，无密钥（D-010）。
- **分析**（周频报告）：实际累计 funding vs 理论累计、价差残差分布、收敛半衰期（价差回落到 X% 的时间）、摩擦后净收益 vs 5% 门槛。
- **定位**：收敛统计的唯一证据来源；前向模拟（§5.2）只给机制证据。两者结论可对照，不互替。
- **实现**：M3-b 前置小任务（历史回填 + 统计报告），不另立阶段（D-034 ⑤ 的 a→b→c 顺序不变）。

---

## 6. M3-c：一键确认 UI + 闭环

- 新页面/面板「模拟执行」（与机会面板平级 tab）：建议订单列表（标的/价格/数量/预期价差/风险标记/状态）+ 模拟持仓（实时 PnL，RMB）+ 对账报告入口。
- 确认流：人工审价差 → 二次门禁校验（§4）→ 确认 → 模拟成交 → 入持仓。**确认后仍是模拟**，明示 SIMULATED 徽标。
- 不做：任何通往真实资金的按钮/路径。

---

## 7. 部署与测试硬要求

- 同 M1/M2：对抗测试、`go vet` + `go test -race`、行数门禁（豁免 `gen/` + `*_test.go`）。
- `internal/sim` 包强制：真实主网域名**不得出现**（§2 审查门禁，编码为测试：grep 主网域名的单测）。
- sim 配置缺失 → sim 模块**降级禁用**（warn 日志，不退出进程），与 D-032 同口径；监控主循环不受影响。

## 8. 明确不做（M3）

- ❌ 真金执行 / 真实交易密钥（无密钥铁律不变，D-010）
- ❌ testnet 真实下单（M3 只读 + 本地模拟；真实成交路径验证属未来决策）
- ❌ 自动下单免人工确认（形态 = 决策监控 + 人工一键确认，D-034 ①）
- ❌ 交易策略回测引擎 / 顺单历史回放（只评估策略表现、不顺单、不模拟下单执行）
- ✔ 历史 funding / 价差数据回填 + 统计收敛分析（§5.3）是 M3 收敛验证的**数据基础，须做**——不在"不做"之列（D-036 收敛口径修正）
- ❌ 策略自动路由 / 跨所自动撮合

---

## 9. M3-b 施工细化设计（D-034/D-036 落地 · 施工权威 · D-037 定稿）

### 9.0 范围裁决（先读）
- 五件套 S1–S5（§9.1）。核心裁决（D-037）：**模拟结算数据源 = 真实市场公开 funding**（§5.1 已改），不是 testnet——testnet 费率偏差会污染机制验证数字；testnet 只做 §2 key 隔离机制验证（S3）。历史回填直接落 facts 表（kind=funding + 真实 ts），顺带让 `funding_warn`/`funding_critical` 的 `avg_30d` 立即有真实回溯（此前仅 1–3 天实时数据）——双赢。

### 9.1 子任务拆解

| # | 子任务 | 内容 | 验收锚点（对抗测试） |
|---|--------|------|----------------------|
| S1 | 规则→Signal 驱动接线（G1 落地） | rule.OnActive 携带命中实体 → sim.Driver 按 §3.1.1 映射组装 Signal → Generate 落库 | OnActive(funding_warn, BTC@binance 命中) → sim_orders 落 funding_hedge；未知规则 → 不建单（删映射 → 必红） |
| S2 | 8h 结算调度 | 对 open funding 腿按 (symbol,venue) 取真实 funding 结算 | 结算 pnl 正确；BTC@binance 与 BTC@okx 隔离（错 rate / 串 venue → 必红） |
| S3 | testnet 只读 + key 隔离验证（key 门控） | 新包 `internal/simtestnet`（key 承载层）：SIMULATED 标记加载校验 + 只读查询 + 零下单路径 | 缺 SIMULATED 标记 → 拒绝加载；sim 包无任何网络域名 / simtestnet 仅 testnet-demo 域无下单域（domains_test） |
| S4 | 历史收敛分析（§5.3 落地） | exchange 包历史 collector（无 key）+ boot 一次性回填（facts 表）+ 周频统计报告 | 回填幂等（跑两遍不重复）；年化折算正确（删 annualize → 必红） |
| S5 | 白名单 + 降级 | ARBCN_SIM_CARRY_WHITELIST 显式配置；sim 配置缺失 → 禁用 warn | 白名单解析；carry 未白名单 → WHITELIST 拒单 |

### 9.2 S1 规则→Signal 驱动

**rule 包改动（单点小改）**：
- `Config.OnActive` 签名 `func(ctx, store.Rule)` → `func(ctx context.Context, r store.Rule, entities []store.EntityHit)`；`store.EntityHit{Venue, Symbol string, Value float64}`（store 包新增）。
- 改点仅 `state.go:37`（`matches []match` 已在作用域）→ 映射 `[]match` → `[]store.EntityHit`。
- 既有调用方仅两处：main.go startPipeline 的 OnActive、exporter.OnRuleActive（签名同步改，忽略 entities，仍只用 r 刷新快照）；exporter_test 两处同步。

**sim.Driver（新 internal/sim/driver.go）**：
```
type Driver struct {
    st  store.Store
    cfg Config            // 含 CarryWhitelist（§9.6）
    now func() time.Time  // 注入时钟
}
func (d *Driver) OnRuleActive(ctx context.Context, r store.Rule, entities []store.EntityHit) error
```
- 映射表：sim 包内不可变常量表（§3.1.1 表编码：规则名 → kind + 组装函数）+ 对抗测试（未知规则名 → 不建单，宁缺毋滥）。
- 组装（每个命中实体 venue/symbol/value）：
  - `funding_*` → Kind=funding_hedge，Venue=hit.venue，Symbol=hit.symbol，RefPrice=LatestFacts(ticker, venue, symbol)，SpotPrice/PerpPrice=同取该 ticker 最新价（**诚实标注：系统无现货 collector，ticker 即永续价；现货/永续腿存在性由门禁把关（>0），basis/现货腿差留真实执行层，M3 只验证 funding 机制**），FundingAnn=hit.value（cond 口径 avg_30d），Notional=0（Generate 默认 capital×20%）。
  - `reverse_repo_timing` → Kind=repo（全局模式命中，单信号，无实体 venue/symbol）。
  - carry 白名单 → Kind=carry_asset，CarryWhite=symbol ∈ whitelist（§9.6）。
  - 其余规则 → 不建单。
- 调 `Generate(ctx, sig)`（SignalToOrder + InsertSimOrder，DayNotional 自动回填）。
- 单次激活只建一单（OnActive 仅 armed→active 转变触发，持续满足不重复建单——§3.1.2 语义）；生成时门禁已落库（§4）。
- 降级：cfg 加载失败 → Driver nil，main.go 不接 OnActive（§7 warn 不退出）。

### 9.3 S2 8h 结算调度

**store 扩展（小改）**：
- `ListOpenSimPositions(ctx, symbol, venue string)`（venue 空 = 不限）——现签名仅 symbol。
- `sim.SettleFunding` 扩展 `(ctx, symbol, venue string, annualized float64)`——按 (symbol,venue) 分组结算，避免 BTC@binance 与 BTC@okx 互相污染（诚实数字）。

**settleLoop（Driver 内 goroutine）**：每 8h tick：`ListOpenSimPositions(ctx, "", "")` → 按 (symbol,venue) 分组 open funding 腿 → 每组 `LatestFacts(kind=funding, venue, symbol)` → 无事实则 skip（warn 一次）→ `SettleFunding(ctx, symbol, venue, value)`。
- tick 实现：`time.Ticker` + ctx 取消（复用 rule.sleepCtx 模式）；注入时钟可测。
- **数据源裁决落地**：结算 funding = LatestFacts **真实市场值**（binance_funding/okx_funding 采集）；testnet 费率不参与结算（§9.0）。

### 9.4 S3 testnet 只读 + key 隔离（key 门控）

- **包拆分（D-037 补充·派工前设计修正）**：key 承载层与 sim 核心**物理隔离**——
  - `internal/sim`：**保持零网络零密钥**（M3-a 复审验证的 D-010 属性不变，纵深防御：即使 sim 核心配置错误也碰不到网络/key）——纯计算：Driver 组装、结算数学、报告数学、Capital/白名单配置。
  - `internal/simtestnet`（新包）：**key 承载层**（S3 探针）——加载 `/etc/arbcn/arbcn-sim.env` `SIM_*`（独立文件 root:root 0600，D-034 ② 物理隔离），每 key 显式 `SIMULATED=true`，**缺标记拒绝加载**（对抗测试：删校验 → 必红）。
- 只读探针（随 settle tick + 启动一次（D-040），key 可用时）：对 binance_testnet / okx_demo：
  - 公共行情 + 账户只读查询（余额/费率）验证 key 连通；
  - 成功后经 `alert.Heartbeat.Record("sim_testnet_binance"/"sim_testnet_okx", now)` 登记 → 出现在 ListSourceHealth（复用 M2-a freshness 面，D-032 降级同口径：失败 warn 不退出）；
  - **D-040 扩展**：余额解析为账户快照 → `sim_testnet_accounts`（migration 0006，source 主键 upsert，details JSONB）→ `GetTestnetAccounts` RPC → SimExec 测试网账户区。**equity_usd 口径因 source 而异（诚实标注）**：okx = `totalEq`（精确）；binance = 稳定币合计**近似**（无行情折算非稳定币，非全量净值）。启动一次探针让账户区立即有数据（不等 8h tick）。
  - **零下单路径**：simtestnet 不含任何下单端点代码；domains_test：sim 包**无任何网络域名**，simtestnet **仅 testnet/demo 域、无主网交易域、无 order/place 下单域**。
- 依赖：testnet key 由业主提供（缺失 → S3 降级禁用 + degraded 提示，**不阻塞 S1/S2/S4/S5**）。

### 9.5 S4 历史收敛分析（§5.3 落地）

**历史回填（一次性，幂等）**：
- **包归属**：历史收集器放 `internal/collect/exchange`（公开数据层，无 key，复用既有 getJSON/annualize 骨架）——`NewBinanceFundingHistory` / `NewOKXFundingHistory` 实现 `collect.Collector`（Kind=funding），Poll 内翻页拉满窗口；main.go boot **一次性调用**（不进 Scheduler 常驻循环）。
- Binance：`data-api.binance.vision/fapi/v1/fundingRate?symbol=BTCUSDT&startTime&endTime&limit=1000`（D-031 公开数据域；翻页拉满窗口）→ 每行 {fundingTime, fundingRate} → annualize（×1095 / ×2190，按 fundingInfo interval）→ fact{Kind=funding, Venue=binance, Symbol=BTC, Ts=fundingTime, Unit=pct_annualized}。
- OKX：`/api/v5/public/funding-history?instId=BTC-USDT-SWAP&limit=100&after=` 分页 → 同口径。
- 窗口默认 365d（`ARBCN_SIM_HISTORY_DAYS`，0 = 禁用）。
- **幂等**：回填前 `QueryFacts(funding, venue, symbol, from=window)` 取已有 ts 集合 → 跳过已覆盖时段（跑两遍不重复，对抗测试必红锚点）。落库走 `InsertFacts`（既有管线，无 dedup 依赖）。
- 顺带收益：`funding_warn`/`funding_critical` 的 `avg_30d` 立即有真实回溯（此前仅 1–3 天实时）。

**周频统计报告 sim_report**（settle loop 每 7×8h 渲染，导出 markdown，类似 facts.md 独占段）：
- 每 (venue,symbol)：实际累计 funding 收益（Σ rate_frac × notional）、理论累计（窗口均值年化 × 名义 × 天数/365）、残差 = 实际 − 理论、残差分布（均值/σ）、**收敛半衰期**（|残差| 减半所需天数，滚动窗口）、**摩擦后净收益**（实际 − 双边手续费 − 滑点估计）vs 5% 门槛。
- 纯函数计算（internal/sim/report.go）+ 对抗测试（删 Σ 累计 → 必红）。
- 数据面定位：收敛统计只来自历史（§5.3）；前向模拟只证机制（§5.2）。

### 9.6 S5 白名单 + 降级

- Config 增 `CarryWhitelist []string`（`ARBCN_SIM_CARRY_WHITELIST` 逗号分隔；**默认空**）。默认空 = carry 信号被 WHITELIST 拒单直到显式配置（安全默认，宁缺毋滥，M3-a 复审 M2 接受项落地）。
- Driver 组装 carry 信号时 `CarryWhite = contains(whitelist, symbol)`；SignalToOrder 白名单门禁已存在（§4 白名单行）。
- sim 配置缺失/非法 → Driver nil、settle loop/backfill 跳过（warn 不退出，§7 与 D-032 同口径）。

### 9.7 main.go 接线

- startPipeline（store 可用时）：
  1. 迁移后回填：sim cfg 可用 → `backfill.Run(ctx)`（一次性、幂等、阻塞至完成）。
  2. `simDriver := sim.NewDriver(st, simCfg)`（cfg 失败 → nil = 降级）。
  3. `rule.Config.OnActive = compose`：`func(ctx, r, entities){ factsExporter.OnRuleActive(ctx, r); if simDriver != nil { simDriver.OnRuleActive(ctx, r, entities) } }`。
  4. `go settleLoop(ctx)`（simDriver != nil 时）。
  5. testnet 探针（S3，key 可用时）随 settle tick。
- 不新增端口/RPC（模拟对账视图属 M3-c UI；M3-b 只产出落库数据 + sim_report 文件）。

### 9.8 验收标准（§7 硬要求不变）

- `go vet` + `go test -race` 全仓绿；行数门禁（gen/ + *_test 豁免）。
- 对抗测试锚点：删 rule→Signal 映射 → 必红；删 funding 历史 annualize → 必红；删幂等跳过 → 重复回填断言必红；删 SIMULATED 校验 → 缺标记 key 加载断言必红；结算按 venue 分组 → 跨 venue 污染断言必红。
- domains_test 增：sim 包无主网交易域、无下单端点域。
- 线上：SIGKILL 部署 healthz ok；回填后 `funding_warn` 的 `avg_30d` 有 30d 数据（可查 facts）。

### 9.9 依赖与阻塞

- testnet key（业主提供）→ S3 门控，缺失降级，不阻塞 S1/S2/S4/S5。
- 白名单默认空 → carry 先被 WHITELIST 拒单，显式配置后生效。
- M3-c（确认 UI + SPREAD_DRIFT 漂移门禁）在 M3-b 后开工（D-034 ⑤ 顺序不变）。

---

## 10. M3-c 施工细化设计（D-034⑤ + D-036 G5 落地 · 施工权威 · D-038 定稿）

### 10.0 范围裁决（先读）

- **形态**：决策监控 + 人工一键确认（D-034 ①）。**确认后仍是模拟**（SIMULATED 徽标明示），无任何通往真实资金的按钮/路径（§6/§8）。
- **确认流**：人工审价差 → **二次门禁（SPREAD_DRIFT，G5）** → 确认 → 本地模拟成交 → 入持仓。二次门禁是生成时六门禁的**确认时复核**（§4"确认时二次校验"落地），不是第七道生成时门禁。
- **结算数据源不变**（D-037 裁决）：持仓 funding 结算仍取真实市场公开 funding（sim.settleLoop 每 8h），M3-c 只加人工确认触发成交 + 视图；不新增结算/喂价路径。
- **确认成交 = store 层单事务原子**（practices #8：置状态+建从属行必须原子，防"已确认未成交"悬挂 + 并发双插重复建腿）。
- **RMB 口径**：持仓 PnL（模拟 USD 绝对金额）→ 即期汇率折算（USDCNH 事实）；**非** RMBDayEnd 年化口径（那是费率折算，H1/R6#1 刻度线）。汇率缺失 → 显示 USD 原值 + 标注（诚实）。
- **实时 vs 已结算 PnL**：显示**已结算累计**（settleOnce 每 8h 结算）+ 最新 funding 年化标注；**不做**未结算实时估算（范围蔓延，M3 只验证机制）。
- **proto 域**：新建独立 `SimService`（新 proto 域 arbcn.sim.v1），**不动 dashboardv1 生成物**（其 .proto 源缺失且无 sim 域；独立域零回归）。
- **expired 状态**：默认不触发（suggested 永续待确认，直到人工确认/拒单）；自动过期 = 可选项，本次不做（避免无人消费的时间窗复杂度）。

### 10.1 子任务拆解

| # | 子任务 | 内容 | 验收锚点（对抗测试） |
|---|--------|------|----------------------|
| C1 | SimService proto + 生成物 | 新 `proto/arbcn/sim/v1/sim.proto` + buf generate（后端 protoc-gen-go + connect-go，前端 protoc-gen-es）；4 个 RPC | 生成物编译通过；mux 挂 `/arbcn.sim.v1.SimService/`；ListSimOrders 空库返回空不报错 |
| C2 | SPREAD_DRIFT 二次门禁 | `RiskSpreadDrift` 标记 + 纯函数 `ConfirmDriftCheck`（G5 口径：ref 漂移 >2% 或 年化变化 >20% → 拒；有限性 fail-closed）；数据面 = **按 kind 分派权威源**（D-039：funding_hedge→ticker/funding；repo→reverse_repo 利率；carry→defi_rate 年化；权威源查不到 → fail-closed 拒） | 删漂移比较 → 必红；NaN/零 ref → 拒；漂移 2.01% → 拒、1.99% → 过；删 kind 分派 → repo/carry 确认流测试必红 |
| C3 | 确认成交流（ConfirmSimOrder） | RPC 唯一写路径：GetSimOrder → 非 suggested 拒（防重复确认）→ 二次门禁（拒 → RejectSimOrder 追加 SPREAD_DRIFT）→ 通过 → store 原子 `AcceptSimOrder`（suggested→confirmed→filled + INSERT 全腿，WHERE status='suggested' 守卫） | 已确认/已填/已拒订单确认 → 报错；门禁拒 → rejected+SPREAD_DRIFT+note；并发双确认 → 状态守卫拦第二次（无重复建腿） |
| C4 | 模拟执行 UI tab | App.tsx 第 4 个 tab「模拟执行」：建议订单列表（待确认/拒单负样本分组）/ 模拟持仓（PnL USD + 即期 RMB）/ 对账报告入口（GetSimReport 渲染 markdown）；SIMULATED 徽标贯穿 | 组件构建通过；SIMULATED 徽标固定渲染（可检查）；确认按钮仅 suggested 可点 → ConfirmSimOrder → 刷新 |
| C5 | 可检查性 + main.go 接线 + 验收 | domains_test 增 simapi 无真实账户/下单端点（grep 断言）；ConfirmSimOrder 是唯一写路径（无自动确认定时器）；mux 接线；全量测试 + 部署 | simapi 包 grep 无主网交易域/真实 API 端点；go vet + test -race 全绿；部署后 tab 可用 |

### 10.2 C1：SimService proto + 生成物

**新 proto 文件** `proto/arbcn/sim/v1/sim.proto`（包 `arbcn.sim.v1`，独立于 dashboardv1）：

```proto
syntax = "proto3";
package arbcn.sim.v1;

message SimOrder {
  int64 id = 1;
  int64 ts_ms = 2;            // 毫秒时间戳（前端 bigint 承载）
  string src_rule = 3;
  string kind = 4;
  string venue = 5;
  string symbol = 6;
  string side = 7;
  double qty = 8;
  double ref_price = 9;
  double expected_spread = 10;
  repeated string risk_flags = 11;
  string status = 12;
  string note = 13;
}

message SimPosition {
  int64 id = 1;
  int64 order_id = 2;
  int64 ts_ms = 3;
  string kind = 4;
  string venue = 5;
  string symbol = 6;
  string side = 7;
  double qty = 8;
  double ref_price = 9;
  bool funding = 10;
  double pnl = 11;            // 已结算累计（模拟 USD）
  double pnl_rmb = 12;        // 即期折算（汇率缺失 = 0，前端标注）
  string status = 13;
}

message ListSimOrdersRequest  { string status = 1; }        // 空 = 全部
message ListSimOrdersResponse { repeated SimOrder orders = 1; }
message ConfirmSimOrderRequest  { int64 id = 1; }
message ConfirmSimOrderResponse {
  SimOrder order = 1;         // 确认后（filled 或 rejected+SPREAD_DRIFT）
  bool accepted = 2;          // true = 成交；false = 二次门禁拒单
}
message ListSimPositionsRequest  {}
message ListSimPositionsResponse { repeated SimPosition positions = 1; }
message GetSimReportRequest {}
message GetSimReportResponse {
  string markdown = 1;        // 报告内容（文件不存在 = 空 + note 说明）
  bool exists = 2;
  string note = 3;            // 未生成时说明"周频报告每 7×8h tick 渲染"
}

service SimService {
  rpc ListSimOrders(ListSimOrdersRequest) returns (ListSimOrdersResponse);
  rpc ConfirmSimOrder(ConfirmSimOrderRequest) returns (ConfirmSimOrderResponse);
  rpc ListSimPositions(ListSimPositionsRequest) returns (ListSimPositionsResponse);
  rpc GetSimReport(GetSimReportRequest) returns (GetSimReportResponse);
}
```

**生成**（对齐现有生成物版本：后端 connect v1.20 / protobuf v1.36.11，前端 protoc-gen-es v2）：
- 建 `buf.yaml`（现有缺失，补）；后端输出 `internal/simapi/gen/...`（protoc-gen-go + protoc-gen-connect-go），前端输出 `web/src/gen/arbcn/sim/v1/`（protoc-gen-es）。
- main.go mux 挂新 handler：`simapi.NewService(st, cfg).Handler()` → `mux.Handle("/arbcn.sim.v1.SimService/", h)`（ConnectRPC 同源托管，vite dev 代理加同名路径）。
- **不新增端口**（§6 同源单端口）；**不动 dashboardv1**（其源缺失，硬改有反推风险——C1 关键边界）。

**验收锚点**：空库 ListSimOrders → `[]` 不报错；proto 字段与 store.SimOrder 逐一对齐（id/ts/src_rule/kind/venue/symbol/side/qty/ref_price/expected_spread/risk_flags/status/note）。

### 10.3 C2：SPREAD_DRIFT 二次门禁

**新常量**（`internal/sim/order.go` Risk 值域追加）：
```go
const RiskSpreadDrift = "SPREAD_DRIFT" // 确认时刻 ref_price 漂移 >2% 或 年化变化 >20%（D-036 G5）
```

**新纯函数**（`internal/sim/confirm.go`，零 I/O、可注入输入）：
```go
// ConfirmDriftCheck 确认时二次门禁（D-036 G5）：生成 vs 确认时刻的 ref_price 漂移
// >2%（或 预期年化变化 >20%）→ 拒单。genRef==0/非有限 → fail-closed 拒（practices #7：
// 确认重查的价可能是 NaN/缺，不得静默放行）。
func ConfirmDriftCheck(genRef, genSpread, curRef, curSpread float64) (reject bool, reason string)
```

G5 口径逐字落地：
- `|curRef-genRef|/genRef > 0.02` → 拒：`SPREAD_DRIFT: ref_price 漂移 X%`
- `|curSpread-genSpread|/genSpread > 0.20` → 拒：`SPREAD_DRIFT: 预期年化变化 X%`
- 两条件**各自独立触发**（任一过线即拒，note 记具体原因；G5"或"关系）
- 有限性守卫：`genRef==0 || IsNaN(genRef|curRef|genSpread|curSpread) || IsInf` → 拒（fail-closed）

**数据面**（RPC handler 层，C3 内；**D-039 kind 分派**）：确认时刻按 kind 选**权威数据源**——上述 ticker/funding 双查是 funding_hedge 语义；repo/carry 数据源不同（硬编码双查会恒拒，M3-c 验收发现）：
- **funding_hedge**：`LatestFacts(kind=ticker, venue, symbol)` 最新价 → curRef；`LatestFacts(kind=funding, venue, symbol)` 最新年化 → curSpread。双查，缺一 fail-closed 拒。
- **repo**：ref = 面值锚（GC001 面值 100，无价格行情；curRef=genRef 漂移恒 0）；spread = `LatestFacts(kind=reverse_repo)` 当日逆回购利率（与生成侧 repoSignal 同权威源）。查不到 → fail-closed 拒。
- **carry_asset**：spread = `LatestFacts(kind=defi_rate, venue, symbol)` 生息年化（权威源）；查不到 → fail-closed 拒。ref = ticker 有则查（稳定币现价漂移），无 → curRef=genRef（面值锚 1.0 漂移恒 0，跳过 ref 检查——稳定币无方向风险，核心漂移是生息年化，D-019 白名单已对冲）。
- 未知 kind → fail-closed 拒（与 SignalToOrder L1 同口径）。

fail-closed 哲学保持：**每类订单的权威源查不到 → 拒**（无数据不确认，§4"任一操作数无数据 → 不告警"同哲学，确认流从严）。`ConfirmDriftCheck` 纯函数签名不变。

**[对抗测试锚点]**：删 `> 0.02` 漂移比较 → TestConfirmDriftRejectsDrift 必红；`ConfirmDriftCheck(100, 5, 100, 5)` → 过；`(100, 5, 102.01, 5)` → 拒；`(0, 5, 100, 5)` → 拒（零 ref fail-closed）；`(100, 5, NaN, 5)` → 拒。

### 10.4 C3：确认成交流（ConfirmSimOrder RPC）

**流程**（handler 层，唯一写路径）：
1. `GetSimOrder(id)` → ErrNotFound 报错；`status != suggested` → 报错（已确认/已填/已拒/过期 → 防重复确认）。
2. 二次门禁（§10.3 + D-039 kind 分派数据面）：确认时刻重查该 kind 权威源 → `ConfirmDriftCheck`。
   - **拒** → 调新 store 方法 `RejectSimOrder(id, reason, "SPREAD_DRIFT")`（原子：status=rejected + risk_flags 追加标记 + note 覆盖）→ 返回 `{accepted:false, order}`。拒单 = 负样本保留（§4）。
   - **过** → 组 legs（funding_hedge = 两腿 现货 long + 永续 short；carry/repo = 单生息腿）→ 调新 store 原子方法。
3. **新 store 原子方法 `AcceptSimOrder(id, note, legs)`**（替代"先置 confirmed 再 ConfirmAndFill"两步——practices #8 原子性）：
   ```
   单事务：
     UPDATE sim_orders SET status='confirmed' WHERE id=$1 AND status='suggested'  -- 守卫 1
     UPDATE sim_orders SET status='filled', note=$2 WHERE id=$1 AND status='confirmed' -- 守卫 2
     INSERT 全部 sim_positions 腿
     COMMIT
   ```
   `RowsAffected` 任一步为 0 → 整体回滚 + 报错。事务内 confirmed 是中间态，外部永远看到 suggested 或 filled（无"已确认未成交"悬挂态）；并发双确认 → 守卫 1 拦第二次（无重复建腿）。
   - 语义与 M3-b 的 `FillSimOrder`（confirmed→filled）互补：FillSimOrder 供内部/测试从 confirmed 成交；AcceptSimOrder 是 M3-c 人工流从 suggested 一次性确认成交。**共享建腿逻辑**（组 legs 复用 sim.ConfirmAndFill 的 legs 组装，提取公共函数）。

**[对抗测试锚点]**：`ConfirmSimOrder` 对 confirmed/filled/rejected 订单 → 报错（删非 suggested 检查 → 必红）；门禁拒 → 订单 rejected + risk_flags 含 SPREAD_DRIFT + note 记漂移原因；并发双确认 → 第二次失败（删 WHERE status='suggested' 守卫 → 重复建腿断言必红）；拒单后不建腿（ListSimPositions 空）。

### 10.5 C4：模拟执行 UI tab

**App.tsx**：`Tab` 加 `"sim"`，TABS 加 `{ key: "sim", label: "模拟执行" }`（第 4 个 tab，与机会面板平级，§6）。新组件 `web/src/components/SimExec.tsx`。

**SimExec 三区**（仿现有组件模式：`hooks.ts` 加 `useSim()` 快照，调 SimService）：
1. **建议订单列表**：`ListSimOrders(status="")` → 分组——待确认（suggested）/ 拒单负样本（rejected，含 SPREAD_DRIFT 原因徽标）/ 已成交（filled）。行：标的/venue/方向/数量/预期年化/risk_flags 徽标/状态。**确认按钮仅 suggested 行渲染**（防误点 + 二次点击确认交互）→ `ConfirmSimOrder(id)` → 刷新。
2. **模拟持仓**：`ListSimPositions()` → 行：标的/venue/腿方向/数量/pnl（USD）/pnl_rmb（即期折算，汇率缺失标注 "USD 原值"）/funding 年化标注（最新）。PnL = 已结算累计（诚实标注"每 8h 结算"）。
3. **对账报告入口**：`GetSimReport()` → 渲染 markdown（或 "周频报告每 7×8h tick 渲染" note）。
4. **SIMULATED 徽标**：tab 顶部固定渲染 + 每个订单/持仓行明示（"模拟"）。**永不出现真金按钮/路径**（§6/§8）。

**RMB 折算实现**（C4 关键口径）：RPC handler 层用 `LatestFacts(kind=fx, venue=sina, symbol=usdcnh)` 即期价 × pnl（绝对金额用即期，**非** RMBDayEnd 年化口径——spec 明示避免 H1 刻度错位重演）；汇率缺失 → pnl_rmb=0 + 前端标注。

**[验收锚点]**：前端 `tsc --noEmit && vite build` 过；SimExec 组件含 `SIMULATED`/「模拟」常量（可 grep）；确认按钮渲染条件 `status==='suggested'`；后端 RPC 锚点测试为主（C2/C3），前端构建通过 + 手动冒烟。

### 10.6 C5：可检查性 + main.go 接线 + 验收

**可检查性（对抗测试）**：
- **domains_test 增**：`internal/simapi` 包 grep 断言——无真实账户端点（无 `account`、无 `withdraw`、无 `transfer`、无 `order` 下单路径）、无主网交易域名（fapi/www.okx 的真实交易端点不在 handler 路径）。**ConfirmSimOrder 是唯一写路径**：包内无定时器/自动确认（grep 无 `time.Ticker`）。
- **SIMULATED 徽标可检查**：SimExec.tsx 含固定 SIMULATED 渲染（grep）。
- **go vet + go test -race 全绿**；行数门禁（gen/ + *_test 豁免）。

**main.go 接线**（C1 内）：`simapi.NewService(st, simCfg).Handler()` → mux。**降级**：sim 配置缺失（simOK=false）→ SimService 仍挂载但返回说明（或 store 无 sim 表不可用时 ListSimOrders 报 degraded），不退出（D-032 同口径）。

**部署验收**：SIGKILL 重启 healthz ok；前端「模拟执行」tab 打开；建议订单列表/持仓/报告入口可用；手动触发一次 funding 命中 → 订单出现在待确认 → 确认 → 持仓出现 → 8h 结算后 PnL 更新；SPREAD_DRIFT 拒单路径手动验证（人工改确认窗口价差大）。

### 10.7 依赖与阻塞

- 无新外部依赖：复用 store（sim 接口）/ sim（SignalToOrder/ConfirmAndFill legs 组装）/ rmb / dashboard 模式；新增 store 原子方法 AcceptSimOrder + RejectSimOrder（pgstore 单事务，照 FillSimOrder 模式）。
- proto 工具链：protoc / protoc-gen-go / buf 已装，protoc-gen-es 在 web devDeps（connect-go v1.20 / protobuf v1.36.11 对齐现有生成物）。
- **不阻塞**：testnet key 缺失不影响 M3-c（确认流走本地模拟成交，与 S3 探针正交）。
- M3-c 验收依赖 M3-b 已部署（sim 表/驱动/settle 已在线上，D-037 commit 后）。
