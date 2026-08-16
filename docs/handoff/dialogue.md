# 对话纪要

> 有痕可阅。每次实质对话追加一段。超 450 行 → 最旧段移 LOG.md（留索引行）。
> 全量原文在各工具自身 session 文件中（人类可读），此处只落实质纪要。


> #1~#53 已移 LOG.md（T2 归档，git 追溯）

## #54 · 2026-08-16 · 告警流高度封顶修复（页面被撑高 → 触发器脱节）· 业主报障 → 决策层
- **参与方**：业主（报障）、Claude（决策层）
- **议题**：仪表盘「监控总览」告警流把页面撑得太高，触发器表被顶出视口（脱节）。
- **根因**：后端 `maxAlertLimit=500`（internal/dashboard/service.go）一次最多回 500 条，前端 `.timeline`（告警流 `<ol>`）无高度上限 → 双栏 `.row` 网格（`align-items:start`）被无限撑高，下方 Triggers 卡片被顶飞。
- **修复**：`web/src/style.css` `.timeline` 加 `max-height: min(60vh, 480px)` + `overflow-y: auto`（卡片内滚动；铃铛通知中心 70vh 浮动层已覆盖全量浏览，语义不重复）。纯 CSS，无 TS/后端改动。
- **部署**：`npm run build`（新哈希 index-BfWXmvtq.css）+ `go build -o bin/arbcn ./cmd/arbcn` + systemd restart（sudo 正常路径，新 PID 2695905）；healthz 200，托管页面引用新 CSS，托管 CSS 实测含 `max-height:min(60vh,480px)`；ListAlerts 零回归（当前 18 条）。
- **决策号**：无新 D#（纯 UI 布局修复，未触及架构/合规/资金面）。

## #55 · 2026-08-16 · 模拟执行无开仓机会核查 + 演练档选型 · 业主提问 → 决策层
- **参与方**：业主（提问 + 选型）、Claude（决策层）
- **议题**：模拟执行 tab 一直没有开仓机会，是否异常？
- **核查**：设计内行为，非 bug——`sim_orders=0`、`funding_warn/funding_critical` 从未激活（trigger_states 仅 heartbeat/defi 两条且已 resolved）；实时 funding avg_30d：okx BTC 6.66% / binance BTC 4.47% / ETH 3.3–3.9%，全远低于 15%/20% 门槛；SPREAD_LOW 5% + carry 白名单空 = 双重宁缺毋滥。模拟执行是「资金费率窗口猎人」，只在 ≥15% 高水位出单（牛市短空费高企时），连真实可交易的 BTC cash-and-carry ~7% 都在门槛下。
- **决策**：业主三选一选型「**降门槛演练档（funding ≥5%）**」→ D-041：新规则 `funding_drill`（band `avg_30d > 5 && < 15`，Info 级）→ 复用 fundingHedgeSignal 映射；让真实市场 funding 6.6%（BTC@okx）进模拟盘，补真实端到端冒烟（确认→成交→8h 结算）。
- **结论**：D-041 落定（defaults/driver/对抗测试/spec 表），全量测试 + vet 绿；部署后预期 okx BTC 触发演练单，业主可确认成交走通全链路。

## #56 · 2026-08-16 · 演练单连续拒单根因排查 + 修复 · 决策层（对话 #54/#55 后续）
- **参与方**：Claude（决策层 + 施工）、业主（选型授权"重触发 + 代码加固"）
- **议题**：D-041 funding_drill 部署后演练单**拒单**（sim_orders id=1/2 rejected，UNHEDGED，ref_price=-0.16/-0.23 负值），本应「重启即触发建议单」。
- **排查链**：① xmin 事务序证实 tickers 在订单前已落库（排除数据竞态）；② 反汇编 + RPC 组合查询暴露过滤器错乱；③ inode 对比一度误判"运行旧二进制"（`stat` 不带 `-L` 返回 procfs 伪 inode，practices #16）；④ 逐字段 RPC 测试（kind 单独有效 / venue、symbol 单独失效）→ 锁定 **`pgstore.LatestFacts` SQL 运算符优先级 bug**（where 子句缺括号，DB 直查复现：`ticker/okx/BTC` 返回 5 行首行 funding@binance 负值）。
- **决策**：D-042——① 每 where 子句加括号 `($1='' OR kind=$1)` + 对抗测试 TestLatestFactsFilters（删括号必红）；② 引擎 boot 竞态加固 `rule.Config.BootDelay`（Scheduler 与 Engine 并行启动，collector 首轮 poll 可能晚于引擎首评 → 首评空跑；main.go 接 15s）+ 对抗测试 TestRunBootDelay（删 sleep 必红）；③ 修正 migrate_test 陈旧断言（want 5→6，D-040 加 migration 0006）。
- **结论**：全量测试 + vet 绿；部署实测 **sim_orders id=3 = suggested（okx BTC ref 63063.30 spread 6.64% risk_flags={}）**——演练单可确认→成交→8h 结算全链路闭环；拒单 id=1/2 保留为负样本（拒单不是失败）。教训入 practices #15（SQL where 子句括号）+ #16（stat -L）。

## #57 · 2026-08-16 · 演练单确认按钮"再次点击确认"无响应 · 业主反馈 → 决策层
- **参与方**：业主（反馈"再次点击确认？没有响应"）、Claude（决策层 + 施工）
- **议题**：D-042 后演练单 id=3 suggested 可确认，业主在 SimExec 点"确认"——首次点击按钮变"再次点击确认？"（进入待确认态），再点**无响应**。
- **排查**：① 后端 ConfirmSimOrder RPC 直测（`{"id":"3"}`）→ accepted=true 订单变 filled（**后端正常**）；② 读前端 OrderRow 按钮 → 发现 `disabled={pending === o.id}`：首次点击 setPending(id) 后按钮文字变"再次点击确认？"，但**同时被 disabled** → 第二次点击 onClick 触发不了，确认动作悬死。防误点设计自相矛盾（等待用户再点的状态 = 禁用条件）。
- **决策**：修三处——OrderRow/OrderZone/调用点把 `disabled={pending === o.id}` 改为 `disabled={busy}`（确认请求进行中才禁用，pending 态保持可点）；onConfirm 逻辑不变（首次 setPending，二次真确认）。教训入 practices #17。
- **结论**：前端构建 + 嵌入二进制重建部署；后端 RPC 已实测订单 id=3 确认成交（filled，二次门禁通过）。re-arm funding_drill 触发新演练单供业主实测二次确认路径。

## #58 · 2026-08-16 · SimExec 三处体验修复（提示条关闭 / 风险标记中文 / 持仓实时数值）· 业主反馈 → 决策层
- **参与方**：业主（反馈 + 口径澄清）、Claude（决策层 + 施工）
- **议题**：① 拒单提示条（如「订单 4 拒单（二次门禁未过，已存负样本）」）无关闭入口；② 风险标记 UNHEDGED 等英文硬编码；③ 模拟持仓缺实时价格/预期收益/实时收益。
- **口径澄清**（AskUserQuestion 三项）：实时收益 = **已结算资金费 + 未结算价格浮动**；预期收益 = **预期年化%**（当前 funding 年化）；提示条 = **手动 × 关闭**。
- **决策**：① `result` banner 加 `.banner-close` × 按钮（onClick setResult("")，style.css 加 flex 布局 + close 样式）；② `riskLabel()` 补全 internal/sim/order.go 全部 7 个 Risk* 常量中文徽标（UNHEDGED→未对冲 / SPREAD_LOW→价差过低 / SIZE_OVER→单笔超限 / DAILY_OVER→日额超限 / INVALID_INPUT→输入无效，SPREAD_DRIFT→漂移 / WHITELIST→未白名单保留，未知名回退原名）；③ 持仓实时数值：proto SimPosition 追加 +14 cur_price / +15 expected_ann / +16 unrealized_pnl（buf generate 同步 Go+TS）→ ListSimPositions 逐腿 `latestValue`（curPrice=ticker 最新、expectedAnn=仅生息腿查 funding 年化、unrealized=(cur-ref)×qty×方向 short=-1/long=+1，**ticker 缺失→0 不编造浮动**）→ toSimPosition 扩签名 → PositionZone 加 4 列（开仓价/当前价/预期年化/实时收益=已结算+未实现）。
- **对抗测试**：TestListSimPositionsRealtime——short 腿 (105-100)×10000×-1=-50000 / long 腿 +50000 / expected_ann 6.6 / 现货腿 0；**删 unrealized 计算必红（已实测短路验证）**。
- **结论**：全量测试 + vet 绿，npm run build 通过；部署实测——API：永续空腿 cur_price=63049 expected_ann=5.53%（当前 okx funding 年化，较生成时 avg_30d 6.63% 回落）unrealized=+286k，现货多腿 unrealized=−286k，**两腿对冲相消=0（delta 中性 ✓，D-019 不赌原则）**；前端：新 JS/CSS hash 托管匹配磁盘 dist，7 个中文徽标 + banner-close 均入 bundle。sim_orders id=4 拒单 SPREAD_DRIFT 20.08% 为设计内 fail-closed（确认时刻单点 funding 回落，二次门禁生效）。
- **决策号**：无新 D#（纯 UI + 数据面展示增强，未触及架构/合规/资金面；口径由对话澄清落档）。

## #59 · 2026-08-16 · 确认下单面板移入监控总览页（告警流下方）· 业主反馈 → 决策层
- **参与方**：业主（反馈 + 两项决策）、Claude（决策层 + 施工）
- **议题**：业主希望「确认下单面板简化一些，放到'告警流'卡片下方，监控总览页就整齐了」——确认下单藏第 4 tab（模拟执行）不易直达，移到总览页告警流下方（告警触发 → 下方确认，信息流更顺）。
- **业主决策**（AskUserQuestion 两项）：① sim tab 保留，只留持仓/账户/报告（确认下单抽到总览页）；② 确认下单面板只显示「待确认」订单 + 确认按钮（"简化一些"）。
- **决策**：新组件 `ConfirmPanel.tsx`（整宽卡片：告警流 row 之下、触发器之上；只列 suggested 订单 6 列=类型/标的/方向/数量/预期年化/确认按钮；二次确认 practices #17；结果提示条 banner-close；空态「暂无待确认订单」）；共享展示辅助抽 `sim.tsx`（SIMULATED 徽标 + kind/side/leg 中文映射，ConfirmPanel 与 SimExec 共用，无重复）；**useSim 提升 App 层共享**（ConfirmPanel + SimExec 同源，ConfirmSimOrder 成功后 useSim tick 刷新两处持仓/订单——避免两处各自轮询、确认后另一处不刷新的缺陷）；SimExec 瘦身为 props（positions/accounts/report/error/reload），OrderZone/OrderRow/RiskFlags/riskLabel/statusText 删除；锚点 `TestSimExecBadgeRenderable` 扩展（sim.tsx 定义 SIMULATED/「模拟」+ SimExec/ConfirmPanel 引用 SimulatedBadge，删徽标必红）。
- **结论**：npm run build 通过（新 hash index-DxLmi9bv.js，确认下单/暂无待确认订单/banner-close 入 bundle）+ go test/vet 全绿（含扩展锚点）+ 部署实测 healthz ok + 托管 hash 匹配磁盘 dist。**funding_drill 已 re-arm**（rule_id=223，avg_30d 6.61% 在 band）→ 约 5 分钟内出新的 suggested 订单，业主可在总览页告警流下方端到端确认（确认→成交→持仓同刷新）。
- **决策号**：无新 D#（纯前端信息架构调整，后端/协议零改动；useSim 提升是前端状态共享，无新数据面）。

## #60 · 2026-08-16 · 确认下单卡片布局再调整（与机会面板同行）+ 交付闭环补课 · 业主反馈 → 决策层
- **参与方**：业主（反馈 + 决策）、Claude（决策层 + 施工）
- **议题**：#59 把 ConfirmPanel 做成告警流下方**整宽卡片**，业主再反馈「确认下单卡片，与机会面板同行，在告警流下方」——整宽空卡太突兀，应收进右列。
- **决策**：App.tsx 双栏改造——`.row`（2fr/1fr 网格）内机会面板左列 `grid-row: span 2` 跨两行，右列包 `.row-col`（flex column）= 告警流（上）+ ConfirmPanel（下），确认卡片与机会面板下部**同行**、位于告警流正下方；style.css 加 `.row-col`（flex + gap 16px + min-width:0 防溢出）。移动端单列时自然堆叠不受影响。
- **端到端验证**（#59 承诺的业主可确认订单，由 Claude 代跑闭环）：re-arm funding_drill → sim_orders **id=5 suggested**（ref 63031.8 spread 6.606%）→ ListSimOrders 正常返回（ConfirmPanel 数据源）→ ConfirmSimOrder **accepted=true**（二次门禁通过，年化变化 16.3%<20%）→ 持仓新增两腿（short expectedAnn=5.98% / long 现货 0）+ 旧腿实时值正确（+63万/−63万 **对冲相消=0，delta 中性 ✓**）。0 值字段 proto JSON 省略是正常行为，前端 connect-es 解析填充默认 0。
- **交付闭环教训**（业主：「提交，推送，应用」）：Claude 完成 #57/#58/#59 自审落档 + 本地 commit 后，把「部署应用 + 推送远端」当可选项晾着等业主催——§7.3 自审不是交付终点。**practices #19 写入**：运行态改动 check-out 收尾必须「构建 → 部署应用 → 推送」三步闭环，本地 commit ≠ 上线。
- **结论**：布局改造部署应用完成（新 hash index-DdoEXWry.js，served 匹配磁盘 dist，row-col/确认下单 均入 bundle）+ go test/vet 全绿 + healthz ok；本次会话全部 commit 推送 GitHub。
- **决策号**：无新 D#（纯前端布局调整 + 流程教训，后端/协议零改动）。

## #61 · 2026-08-16 · 业主三问（defi_rate 五项是什么 / 矩阵横向滑块根因 / 实盘操作 / 是否接 LLM）· 问答 → 决策层
- **参与方**：业主（提问）、Claude（决策层 + 施工）
- **议题 1（defi_rate 身份）**：业主问 `aave-v3/blackrock-buidl/ethena-usde/morpho-blue/ondo-yield-assets` 是什么——**不是交易所，是 DeFi 协议资金池**（DefiLlama yields 数据源，格式 `资产@协议`），喂 carry 稳定币生息档（D-021 第二档）。五项当前年化：USDC@aave-v3 **12.57%（Aave 借贷利率尖峰，异常值，单点不代表长期）** / SUSDE@ethena-usde 4.35% / STEAKUSDC@morpho-blue 4.16% / BUIDL@blackrock-buidl 3.57% / USDY@ondo-yield-assets 3.55%。**注意：这些池只在事实库，carry 白名单默认空（M3-b §9.6 安全默认），未走 D# 显式配置前 carry 订单会被 WHITELIST 拒单。**
- **议题 2（横向滑块根因 + 修复）**：业主问「名称太长导致左右滑块？」——**确认**：总览页机会面板「稳定币金额档利率」矩阵把 5 个协议名当列头（14~16 字符 × 5 列，`white-space: nowrap` + `.table-scroll overflow-x:auto`）→ 矩阵撑宽出滑块。**修复**：`MatrixTable` 加可选 `colLabel` 显示名（key 不变）+ `Opportunity.tsx` 加 `venueLabel` 映射（aave-v3→Aave / blackrock-buidl→BlackRock / ethena-usde→Ethena / morpho-blue→Morpho / ondo-yield-assets→Ondo，未知名回退原名，practices #18 全集对照）；列头变短矩阵收进卡片。部署实测新 hash index-CJDJqr2V.js served 匹配，5 label 全入 bundle。
- **议题 3（实盘操作）**：业主问实盘如何操作——见下方「实盘操作」答复（系统不自动执行，人工下单 + 台账记账）。
- **议题 4（是否接 LLM）**：业主问本项目有无必要接入 LLM——**Claude 裁决：不建议接入**（见答复：与「先核实再采纳/可机械检查」原则冲突 + 结构化数据无自然语言需求 + 模板叙述可替代）。若业主接受，落为 D# 决策。
- **结论**：Q1 修复已构建部署应用 + 测试绿；本 commit 推送 GitHub。

## #62 · 2026-08-16 · 进化建议引擎 L0（半自动增强）确认 + 施工 · 业主需求 → 决策层
- **参与方**：业主（需求）、Claude（决策层 + 施工）
- **议题**：业主「不接 LLM，希望更智能化，目标象 Claude 一样聪明，随数据采集知识库能支撑这一点」→ 确认**半自动增强**。
- **决策层把关**：目标修正为**「资金运营的领域专家」而非「通用智能」**——证据驱动的收敛（数据提示方向），不引入 LLM 式发散（D-043 已锚定）。进化回路五环（数据面/执行面/学习面/记忆面/方向面）缺「半自动增强」环，L0 补上。
- **决策（D-044）**：四信号 L0（reject_dist 拒单分布 / defi_anomaly 利率尖峰 / no_order 连续无单 / source_down·stale 停更），RPC 归 dashboard 域 `ListInsights`，按需计算不加表，截面中位数×2.0 因子检测（regime shift 稳健），**只读证据表面——永不自动改规则，action 一律指向 D# 人工决策**。业主此前要求「自我审计未过不可施工」——本方案先过 §7.3 A–F 自审（A 否决抽新包改为包内纯函数；C 删 stddev 重复；D 补 NaN 守卫）再 ExitPlanMode。
- **施工**：proto Insight/ListInsights + buf generate + `internal/dashboard/insights.go`（纯函数 + RPC 编排）+ `insights_test.go` 对抗（删判定/删计数必红已实证）+ fakeStore ListSimOrders 真语义 + 前端「进化建议」卡（触发器下方整宽卡，severity 色 + 类别中文 + actions 列表）。全量测试/vet/npm build 绿。
- **部署实测**：`reject_dist`（UNHEDGED ×2/SPREAD_DRIFT ×1）、`source_stale` ×3（repo/fx/deribit_iv 周末低活跃源）正确出现；`no_order` 被近 7 天 filled 单正确抑制；`defi_anomaly` 因最新截面平稳**正确不报**（Aave 12.57% 尖峰两轮采集间已回落至 3.29%——L0 报「当前截面离群」非「历史曾冲高」，窗口内 max 语义留 L1）。
- **结论**：构建 → 部署应用 → 推送闭环完成（practices #19）；docs 落档（D-044 / practices #20 / dialogue #62 / STATE）。L1 候选：窗口 max 冲高 / 阈值自适应统计自校准、拒单-阈值联动归因。

## #63 · 2026-08-16 · 全面模拟（carry+repo 接入）+ 施工前自审纪律确认 · 业主需求 → 决策层
- **参与方**：业主（需求 + 纪律追问）、Claude（决策层 + 施工）
- **议题 1（全面模拟）**：业主「当前模拟只接进现货与永续，能否把其他机会都接入模拟」→ 业主定案 **carry + repo 都接**，carry 白名单 **SUSDE / USDE / BUIDL / STEAKUSDC / USDY（能做的都做上）**。调查定位三个真实缺口（非门禁太严）：① 结算数据面只查 funding 事实 → carry/repo 腿建仓后永不生息；② repoSignal 硬编码 domestic/GC001 vs 事实真实 sina/GC001 → 结算永 miss；③ MinSpread=5%（funding 摩擦假设）误用于 carry → 当前 defi 利率全被拒。
- **决策（D-045）**：结算按腿 kind 分派数据面（settleFactKind：funding_hedge→funding / carry_asset→defi_rate / repo→reverse_repo）+ SettleFunding 带 kind 维；repoSignal 落单 venue/symbol 取事实真实值；carry 独立低门槛 `CarryMinSpread`（默认 1%，纠正口径错配非放宽门禁造数据，repo 5%/funding 15% 不变）；生产 env 显式白名单。
- **议题 2（纪律追问）**：业主问「施工前的自我审计，是否写入到了纪律？」——如实回答：此前只在 D-045 计划文件里做了一次 A–F 自审，**没进共享纪律层**。随即落地：AGENTS.md §7.3 加「两时点强制」条款（施工前对设计文档 A–F + 交付前对成品 A–F），practices #21 记录该打回模式（自审是施工前置门禁，不是汇报材料，不待业主追问）。
- **施工**：driver.go（settleFactKind + settleOnce 按 (kind,symbol,venue) 分派 + repoSignal venue 对齐）+ backfill.go（SettleFunding 带 kind）+ config.go（CarryMinSpread，env + NaN 拒载）+ order.go（carry 门槛分档）+ TestDriverRepoBuildsOrder 事实带真实 venue。对抗测试 3 个新锚点（删分派 / 改回硬编码 domestic / 删分档）已逐一实证必红后还原；全量测试/vet/-race 绿。
- **部署**：新建 /etc/arbcn/arbcn-monitor.env（此前 systemd `EnvironmentFile=-` 指向该路径但文件缺失 = 全默认运行；本次创建仅含 `ARBCN_SIM_CARRY_WHITELIST` 一行，无影子冲突）→ npm build + go build → 重启服务 → active + 新二进制 inode 匹配（stat -L）+ 启动日志干净 + served bundle == 新 dist。**诚实标注**：carry 单是否真触发取决于 defi 池出现 ≥0.5%/h 变动（不造数据）；repo 当前 0.865% < 5% 会被 SPREAD_LOW 拒（负样本，符合时点逆回购意图）。
- **结论**：构建 → 部署 → 推送闭环完成（practices #19）；docs 落档（D-045 / practices #21+#22 / dialogue #63 / STATE）。

## #64 · 2026-08-16 · 投运后系统自算账（机会实算卡）+ 经验资产自我进化 · 业主核心追问 → 决策层
- **参与方**：业主（核心追问 + 方向定案 + 摩擦核实）、Claude（决策层 + 施工）
- **议题 1（投运后自算账）**：业主「投运后我不可能永远带着你，让你帮我算账——这种情况下系统如何解决这个问题？这也是之前我提到要不要加入 LLM 的疑问。」——**裁决**：我在场做的「算账」（瞬时 9.14% vs 30 日均值 4.16% / 0.3% 摩擦需 12 天保本 = 尖峰陷阱）**不是聪明，是公式**——确定性计算可机械执行，无需 LLM（D-043 已锚定不接 LLM）。
- **议题 2（自我进化方向）**：业主「要，这个很有用，而且'新情况'，我们要吸收成为知识库，成为我们的经验资产，这才是这套系统最有价值的地方，能随着行情数据，市场变化自我进化成长。」——方向定案：**经验库吸收 = 人工 D# 落盘 + 确定性签名匹配 + 只读呈现**（系统主动匹配与呈现 = 增强；自动吸收改判 = 越权，practices #20 边界）。
- **议题 3（摩擦核实）**：业主问「关于摩擦，这是确定的，是准确的吗？需不需要验证？」→ 业主确认「**两个交易所都不是 VIP，都是普通用户主户费率**」——**0.3%（现货 taker 0.1%×2 + 永续 taker 0.05%×2）为已核实值**，facts.md 落档「已核实 · 普通主户」；env 常量保留（升 VIP/启用抵扣改配置不改代码）。
- **决策（D-046）**：① 机会实算卡——对实时机会（funding/carry/repo）确定性算账（瞬时/30日均值/保本天数/扣摩擦净年化/三档判定/中文模板叙述），纯函数 + `ListOppCards` RPC，判定基准 = 稳定币基档 4.5%（D-021）；**只读证据不碰任何执行门禁**（D-016 15%/20%、MinSpread/CarryMinSpread、carry 白名单不动）。② 市场结构经验库——`knowledge_entries` 表 + `internal/knowledge` 包（签名探测器纯函数 + 3 条 seed）+ `knowledge_match` 只读 insight + `ListKnowledgeEntries` RPC；吸收=人工 D#（git 跟踪），系统只匹配与呈现。
- **施工**：oppcalc.go（纯函数公式 + 对抗测试）/ oppcalc_rpc.go / knowledge.go（签名探测器 + Defaults）/ migrations 0007 / store 两方法 + 4 处 test fake 补接口 / insights.go 信号 5 接线 / 前端「机会实算卡」区块（评级徽标 + 数值瓦片 + 摩擦明示 + 中文叙述）+/「市场结构经验库」卡（KnowledgeBoard）+ Insights「knowledge→经验」/ hooks（snapshot 第 8 路 RPC + useKnowledge 低频）。对抗测试 3 组新锚点（删公式/删匹配/删中位数因子必红已实证）；全量测试/vet/npm build 绿。
- **结论**：构建 → 部署 → 推送闭环（practices #19）；docs 落档（D-046 / practices #23 / dialogue #64 / STATE）；部署实测 ListOppCards（ETH@okx 尖峰陷阱卡）/ ListKnowledgeEntries（3 条 seed）/ ListInsights（knowledge_match）。

## #65 · 2026-08-16 · 前端第一性原则审计（P0 根因 + P1 双源 + P2 小项）· 业主发起 → 决策层审计 + 修复
- **参与方**：业主（发起审计需求 + 拍板全修）、Claude（决策层 + 审计 + 施工）
- **议题**：业主「完美，现在我们需要对前端每一个页面做第 1 性原则审计」——对监控总览 / 事实快照 / 出入金台账 / 模拟执行四页 + 共享层逐页审计。
- **审计结论**：页面组件本身全是干净薄渲染，问题集中在三层——**P0 根因**：数据 hook 绑 App 根生命周期而非视图生命周期（三症状：① 跨 tab 60s 轮询空转；② 顶部「刷新」只刷 useSnapshot，同页 ConfirmPanel/KnowledgeBoard 不动；③ useSim 无轮询，8h 结算新单确认面板脱节）；**P1 双源映射**（F1 SimKind 中文映射两份文案不同 = Opportunity.kindLabel vs sim.kindText；F2 COVERED_KINDS 复制后端 rmb.go CoveredKinds，P3 违背）；**P2 小项**（F3 freshness 死分支 / F4 pnl_rmb=0 当「汇率缺失」信号 / F5 ledgerDate 归置错位 / F6 fresh-dot 重复 / F7 Alerts-Bell 行重复观察项）。
- **决策**：业主 AskUserQuestion 三选一拍板 **「P0 + P1 + P2 全修」** → D-047：① P0 数据层随视图生命周期——新增 OverviewPage/FactsPage/SimPage 三页面组件承载各自数据 hook（hook 生命周期 = 视图生命周期，切 tab 即卸载停轮询），App 根只留 useSnapshot（header 全局 chrome）；useSim 加 60s 轮询；顶部刷新 = useSnapshot.reload + refreshKey 信号联动总览页 sim/knowledge。② P1 收敛——kindText 单源（Opportunity 删本地 kindLabel）+ grep 锚点测试 TestSimKindLabelCoverage；covered 单源在后端（proto FactRmb.covered + rmb.Converted.Covered），前端删 COVERED_KINDS。③ P2——F4 ListSimPositionsResponse.fx_available presence flag（镜像 ListFactsResponse，前端删 0 占位启发式）；F3 删死分支；F5 ledgerDate 移 format.ts；F6 抽 FreshDot；F7 保持（D# 记 rationale）。
- **设计回归**：对话 #59 曾把 useSim 提升 App 层共享（「确认后两处同刷新」）。本次下沉后，ConfirmPanel 确认 → 切 sim tab → SimPage 挂载即重拉最新，天然覆盖原共享动机——回归「数据随视图」，留痕 D-047。
- **施工**：三页面组件 + hooks.ts（useSim/useKnowledge 加 refreshKey + useSim 轮询 + alive 守卫 + 返回 fxAvailable）+ proto 两字段 + buf generate + App.tsx 重写（header 保留 useSnapshot + refreshKey + 三页面分派）+ 前端映射收敛 + FreshDot + FactsSnapshot covered + SimExec fxAvailable + Opportunity kindText。
- **验证**：对抗测试 TestSimKindLabelCoverage 红→绿已实证（写测试时 Opportunity 仍持 funding_hedge 字面量必红，删后转绿）；全量 go test（含真库 DSN）/vet/npm build 绿；go vet 通过（无 TS6133 未用变量：OverviewPage ackAll prop 移除后 build 干净）。
- **部署**：构建 → 部署 → 推送闭环（practices #19）。构建后 sudo 交互受阻 → 用 D-035 既有「SIGKILL 重启」模式（systemd Restart=on-failure 5s 拉起，状态全在 PG 持久化）；生产实测 healthz ok + 新二进制 inode 匹配（stat -L）+ served bundle == 新 dist（index-c9bH9JKr.js）+ ListFacts covered 18/32 + ListSimPositions fx_available=true + 4 持仓。
- **决策号**：D-047（决策 + 回归留痕）; 教训入 practices #24（数据 hook 随视图生命周期 + 0 占位二犯）。

## #66 · 2026-08-16 · 总览页布局按「第一眼原则」重构（D-048）· 业主指示方向 → 决策层审计 + 施工
- **参与方**：业主（方向指示 + 拍板全量重构）、Claude（决策层 + 审计 + 施工）
- **议题 1（UI 的第一性原则）**：业主「UI 部份，我是觉得用户第 1 眼最需要看到什么，从而来决写 UI 页面布局。」——UI 第一性原则 = **第一眼问题层级决定布局顺序**。
- **审计结论**：推导业主（唯一运营者）打开面板的第一眼问题（重要性降序）= ① 该我行动（有没有待确认订单）② 机会裁决（钱在招手还是坑）③ 系统健康（header 徽标，已就位）④ 信号流（告警/触发器/进化建议）⑤ 数据下钻（矩阵）。定位三处违背：**U1** ConfirmPanel 在 `.row-col` 右列底部、告警流之下，待确认单易被顶出首屏；**U2** 机会实算卡压在本应属于它的机会面板最底部、4 个数据矩阵之下——裁决在数据之后；**U3** 低频经验库（D-046，参考面）占整卡高度。
- **决策**：业主 AskUserQuestion 拍板 **「U1+U2+U3+U4（全量重构）」** → D-048：U1 确认下单置顶整宽（该我行动永远第一眼可见，空态保底不消失）；U2 实算卡移面板顶部（裁决先于数据），4 数据块包 Collapse（funding 默认展开 / defi/IV/repo 折叠下钻）；U3 经验库折叠（`hasKnowledgeMatch` 命中才默认展开，knowledge_match → 进化建议「经验」类目 → 展开判定记录供裁决对照）；U4 触发器仅 active 显示 + 全量规则表折叠；新组件 Collapse（原生 `<details>` 受控，零依赖可 grep），三处复用；节序定稿 error → ConfirmPanel → `.row`(机会+告警双栏) → 进化建议 → 触发器 → 经验库，删 `.row-col`/span 2 死样式。
- **施工**：Collapse.tsx + OverviewPage.tsx（ConfirmPanel 置顶 + `.row` 简化 + Insights 提序 + hasKnowledgeMatch）+ Opportunity.tsx（实算卡置前 + 折叠）+ Triggers.tsx（active 顶部 + Collapse 全量）+ KnowledgeBoard.tsx（defaultOpen prop）+ style.css（.collapse 系列 + 删死样式）。
- **验证**：grep 锚点恒绿（TestSimKindLabelCoverage 要求 Opportunity 无 funding_hedge 字面量、TestSimExecBadgeRenderable 要求 ConfirmPanel 引 SimulatedBadge——均未触及）；全量 go test（含真库 DSN）/vet/npm build 绿（tsc strict 无未用变量/import）。
- **部署**：构建 → 部署 → 推送闭环（practices #19）。构建后 sudo 交互受阻 → D-035 既有「SIGKILL 重启」模式（systemd Restart=on-failure 拉起，状态全在 PG 持久化）；生产实测 healthz ok + 新二进制 inode 匹配（stat -L，21244274→21244337）+ served bundle == 新 dist（index-DTovgGTO.js 匹配磁盘）+ 前端第一眼顺序核对（待确认下单置顶 → 实算卡裁决在前 funding 展开 → 告警流右列 → 进化建议 → 触发器仅 active → 经验库折叠）。
- **决策号**：D-048（决策 + 留痕）; 教训入 practices #25（布局 = 第一眼问题层级，裁决在数据前）。

## #67 · 2026-08-16 · 资金费率矩阵数据源范围（要不要加更多交易所）· 业主提问 → 决策层裁决
- **参与方**：业主（提问）、Claude（决策层）
- **议题**：业主「资金费率矩阵里，只有 binance/okx 2 家，有必要引入更多吗？」
- **裁决（D-049）**：**暂不加所**——① 矩阵瓶颈在标的维度不在所维度：流动性所 funding 高度趋同，加第三家不改「流动性币 funding 极少过 D-016 15% 门槛」基本面（对话 #52 实证：极端 funding 全在微盘陷阱币）；② 业主可交易面就是这两家（普通主户费率已核实），展示不能下单的所 = 噪音；③ 加所成本（collector + 部署机端点实测 practices #12 + 故障面）在收益未确证前不加（D-028）。**值得扩的是标的维度**（SOL/XRP 等，受宁缺毋滥约束）；**例外触发** = 跨所费率分歧在流动性标的上反复命中且业主确证可套 → 才考虑加第三所（Bybit，3 所 = 每币 3 分歧对）。
- **结论**：数据面维持 binance+okx，零代码改动；facts.md 落档数据源边界（现行）+ decisions.md D-049 记录触发条件，未来提「引入 Bybit」须先过该条件。
- **决策号**：D-049（数据源范围裁决，方向记录在案）。
