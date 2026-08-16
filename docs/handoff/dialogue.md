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

## #68 · 2026-08-16 · 总览页布局第 2 版（业主指定 3×2 网格 + 告警流第1性原则）· 业主不满意 → 决策层执行
- **参与方**：业主（不满意 D-048 + 指定布局 + 原则追问）、Claude（决策层 + 执行）
- **议题 1（新布局）**：业主「监控总览页，我还不满意」——指定精确 3×2 网格：左上1 资金费率/稳定币/IV/逆回购 矩阵、右上1 市场结构经验库（现在的告警流位置）、左2 机会面板（缩短）、右2 确认下单（**鼠标顺手高度**）、左3 进化建议、右3 触发器。并给原则：「**打开首页第 1 版就能完成基本的信息和操作**」。
- **议题 2（告警流第1性原则）**：业主问「告警流确实是一个要处理的，它的第1性原则是什么？」——推导：告警 = 系统「有情况报告」，运营者第一眼问题是**状态问题**（有没有 N 条未处置）而非内容问题（逐条发生了什么）。**推论：第一眼形态 = 未读数徽标，内容/处置 = 拉取式**。因此告警流移入 header 铃铛（状态一眼可见、ack 一键可达），网格内时间线卡删除（信息不丢）。
- **决策（D-050）**：① 3×2 网格（MarketMatrix 左上1 全展开 / 经验库右上1 / 机会面板左2 / 确认下单右2 / 进化建议左3 / 触发器右3，auto-flow）；② 机会面板拆分——矩阵独立成 MarketMatrix.tsx，Opportunity 只剩实算卡（缩短）；③ 告警移铃铛——删 Alerts.tsx + `.timeline` 死 CSS + OverviewPage 删 ackBusy/ack props；④ D-048 U2「裁决先于数据/确认置顶」**反转**（业主新优先级 = 数据监控置顶 + 操作热区中右），走 D# 留痕。
- **施工**：MarketMatrix.tsx（新）+ Opportunity.tsx（实算卡瘦身）+ OverviewPage.tsx（网格）+ App.tsx（props 同步）+ 删 Alerts.tsx + style.css（.grid 3fr/2fr，删 .timeline）。
- **验证/部署**：锚点恒绿 + 全量测试/vet/npm build 绿；构建 → SIGKILL 部署 → healthz ok + 新 inode 匹配（21244337→21244374）+ served bundle == 新 dist（index-Bkl8ybxZ.js）。（practices #19 闭环）
- **决策号**：D-050（布局第 2 版 + 反转留痕）；教训入 practices #26（信息架构 vs 操作热区 + 告警流状态优先）。

## #69 · 2026-08-16 · 移动/小屏自适应（D-051）· 业主指令「需要做」→ 决策层审计 + 施工 + 实测
- **参与方**：业主（指令）、Claude（决策层 + 审计 + 施工 + 真机视口实测）
- **议题**：业主「我们有做移动设置小屏幕自适应吗？这个需要做」。
- **审计结论**：已有**桌面优先兜底**（viewport meta / `.grid` 860px 折叠单栏 / 铃铛 600px 全屏抽屉 / `.table-scroll` 横向滚动 / 瓦片 auto-fill / header+tabs flex-wrap），但**真移动适配缺失**——触摸目标过小（icon 25px / ack 22px / bell 34×30 / tab 31px / 折叠块 30px，指南 ≥44px）、iOS 聚焦输入 <16px 自动缩放、14px 基准字号小屏偏小、表格无惯性滚动、双击缩放未禁用。
- **决策（D-051）**：纯 CSS 媒体查询增量（零 JSX/后端/门禁改动）——主断点 600px（与铃铛一致）+ 极窄 480px（台账表单）+ `hover:none` 按压反馈；600px 内触摸目标 ≥44px / body 15px / main 12px+safe-area / 卡片 12px / 输入 16px 防缩放 / 表格惯性滚动；480px 内台账表单单栏。
- **施工**：style.css 追加 3 媒体块 + 2 处基础规则扩展（button/collapse-summary 的 touch-action + tap-highlight）+ 修 860px `minmax(0,1fr)` + 480px ledger-note 特异性。
- **真机视口实测（无头 chromium CDP，非截图推断）**：375px 首次抓出**两个真 bug**——① `.grid` 860px 折叠 `1fr` 隐式最小=auto，宽矩阵 min-content 撑破视口（docScrollWidth 594→修后 375）；② 台账 480px 规则 `.ledger-note{1/-1}` 特异性 (0,1,0) 不敌基础 `.ledger-form .ledger-note{span 2}` (0,2,0)，span 2 在单栏网格撑出隐式第二列溢出（docScrollWidth 511→修后 375）。修复后三档视口全过：**375px** 单栏无溢出 + 44px 触摸目标 + 台账单栏 325px + 输入 16px；**768px** 单栏网格无溢出 + 按钮不被 600px 规则污染（31px）；**1280px** 双栏 3fr/2fr 无溢出 + 14px 字号（移动规则零污染）。
- **验证/部署**：全量 go test（含真库 DSN）/vet/npm build/tsc 绿；锚点恒绿（纯 CSS 未动 TSX）；构建 → SIGKILL 部署 → healthz ok + inode 匹配（21244393→21244395→21244397，三次构建两次 bug 修复各重部署一次）+ served bundle == 新 dist（index-CRNqUktn.js / index-Cjd-E_-V.css）。（practices #19 闭环）
- **决策号**：D-051（移动适配 + 两实测 bug 修复）；教训入 practices #27（grid `1fr` 隐式 auto 最小撑破视口 + 媒体查询特异性须 ≥ 基础规则 + 375px 实测替代推断）。

## #70 · 2026-08-16 · 长卡片全局高度协调 + 确认下单提上去（D-052）· 业主多问齐发 → 决策层核实 + 施工
- **参与方**：业主（五连问）、Claude（决策层 + 核实 + 施工 + 真机视口实测）
- **议题**：① 市场结构经验库「待复核」是否自动复核？② 经验库条目会分页还是无限延伸？③ 确认下单需提上去；④ 机会面板需分页，否则向下无限拉伸、影响后面卡片阅读高度；⑤ 整个页面多个长卡片，需全局阅读便利协调。
- **核实（Q1 待复核）**：**永不自动复核**——`pgstore.UpsertKnowledgeEntry` 的 INSERT 不写 `validated_at`（保持 NULL），`ON CONFLICT (signature) DO NOTHING` 连人工编辑都保留；全库无设置路径。复核 = 人工 D# 落盘（practices #20/#23 同源：「待复核」提示给决策层裁决，系统永不自动判定）。
- **决策（D-052）**：
  ① **统一高度封顶**：新 `.scroll-cap`（max-height:min(68vh,520px)+overflow-y:auto，600px 内降 min(52vh,360px)）接入机会面板/经验库/进化建议三处列表——**用「卡内滚动」代替「分页」**（实时监控面板卡数 <20，分页引入翻页态丢失跨页一览，滚动封顶以更少交互达成同一目标；翻页态待条目数预期百级才值得做）。
  ② **矩阵表高度封顶护栏**：`.card[aria-labelledby="matrix-title"] .table-scroll` cap 440px（当前各表 <440 不生效，是未来加币种防延伸的护栏）。
  ③ **确认下单提上去**：DOM 重排 矩阵→**确认下单**→机会→经验库→建议→触发器（原矩阵→经验库→机会→确认下单→…）。业主观察实为移动单列序（机会面板 3092px 把确认下单顶到底）；重排后桌面 = 确认下单右上1（首行）、移动 = 第 2 位。
  ④ **「分页还是无限延伸」答案**：均不——有界 + 卡内滚动，永不无限延伸。
- **施工**：style.css（.scroll-cap + 矩阵护栏 + 600px 降级）+ Opportunity.tsx（.opp-cards 接 scroll-cap + 卡数标题）+ KnowledgeBoard.tsx（.insights 接 scroll-cap）+ Insights.tsx（.insights 接 scroll-cap）+ OverviewPage.tsx（DOM 重排 + 布局注释更新）。
- **真机视口实测（CDP 1280/375）**：桌面 docH **4781→2305**、机会面板 **3092→618**、确认下单 y121 首行、6 卡有序（矩阵|确认 / 机会|经验库 / 建议|触发器）各排对齐；移动 375 docSW==innerW 无横向溢出、确认下单第 2 位（y1229）、机会面板 450。矩阵仍 875px = 4 个紧凑分节累加（funding 134 + defi 202 + IV 97 + repo 202），无单节超 cap，业主指定全展开的主数据面保留。
- **验证/部署**：全量 go test（含锚点 TestSimKindLabelCoverage / TestSimExecBadgeRenderable，均 PASS）/vet/npm build/tsc 绿；构建 → SIGKILL 部署（arbcn-monitor.service，restart counter 8）→ healthz ok + inode 匹配（21244415→21244416）+ served bundle == 新 dist（index-DQsLtvVN.js / index-C170oVwK.css）。（practices #19 闭环）
- **决策号**：D-052（高度协调 + 提上去）；教训入 practices #28（长列表用高度封顶卡内滚动防无限延伸 + DOM 序 = 布局序，重排即提权 + 移动单列序是业主观察视图）。

## #71 · 2026-08-16 · 右1 双卡堆叠等高矩阵 + 进化建议右2 等高机会面板 + 经验库「如何复核」（D-053 + D-054）· 业主指定 → 决策层施工 + 实测
- **参与方**：业主（指定布局 + 问复核）、Claude（决策层 + 施工 + 真机视口实测）
- **议题 1（布局）**：业主「左1 是市场数据矩阵，右1 排两个卡片，市场结构经验库在上，确认下单在下，两个+在一起高度跟市场数据矩阵一样高」「进化建议排右2，跟左2 的机会面板一样高」。
- **决策（D-053）**：`.right-stack` flex column 单网格 cell 包 KnowledgeBoard（上）+ ConfirmPanel（下）；删 `.grid` 的 `align-items: start`（恢复默认 stretch）→ 同排单元格沿行高拉伸，**right-stack 自动 = 矩阵高、进化建议自动 = 机会面板高，零逐卡高度规则**。DOM 序 = 矩阵 → right-stack(经验库+确认下单) → 机会 → 进化建议 → 触发器。
- **议题 2（复核）**：业主「如何复核（待复核）」。
- **决策（D-054）**：复核 = **人工在环**（owner 决策层行为，系统永不自动复核）。新 RPC `ReviewKnowledgeEntry` + Store 方法：signature 必填 + status 三态白名单（active/superseded/retracted）+ verdict 自由文本（**留空 = COALESCE 保留原判定**）+ validation_note 必填留痕。**语义关键：verdict 是判定文本（如「坑」），status 是生命周期枚举，绝不混写**（初稿曾把枚举写进 verdict 列，自审纠正）。只改判定记录，不触规则/门禁/D-016/MinSpread/CarryMinSpread/白名单。前端 KnowledgeBoard 每行「复核/再次复核」按钮 → 内联表单（状态 select + 判定文本留空=保留 + 必填结论 + 确认/取消）。
- **施工**：OverviewPage.tsx（right-stack + review prop）+ style.css（.right-stack / .review-form 系，删 align-items:start）+ proto + buf 再生成 + store.go + pgstore/knowledge.go + dashboard/knowledge.go + knowledge 状态常量 + 6 处 fake 接口对齐 + hooks.ts（review 增 status 参数）+ KnowledgeBoard.tsx（ReviewForm 内联表单）。
- **真机视口实测（CDP 1280/375）**：桌面 matrix y121 h875 == right-stack y121 h875（经验库 603 + 确认下单 119 @ y755）、opp y1012 h618 == insights y1012 h618、trig y1645 h188、docH 1881；**D-054 加复核按钮后复测仍 875==875、618==618**（scroll-cap 封顶防撑破）；移动 375 docSW==innerW 无溢出、单列序 = 矩阵→经验库→确认下单→机会面板。
- **验证/部署**：全量 go test（含锚点 TestSimKindLabelCoverage / TestSimExecBadgeRenderable 均 PASS + pgstore roundtrip + service 校验）/vet/npm build/tsc 绿；构建 → SIGKILL 部署 → healthz ok + inode 匹配（21244439→21244442）+ served bundle == 新 dist（index-BRFIYzsI.js / index-_h0_Ebl1.css）。**RPC 实测**：空 note / 非法 status → 400 InvalidArgument、未知签名 → 503 Unavailable 三条拒绝路径 live 命中；CDP 复核表单渲染（select 三态、verdict 预填「坑·核实」、note 必填占位、确认复核按钮）。**零执行门禁/规则/阈值/D-016 改动；不接 LLM（D-043）；不赌（D-019）。**（practices #19 闭环）
- **决策号**：D-053（布局第 3 版）+ D-054（人工复核闭环）。

## #72 · 2026-08-16 · 模拟盘平仓补齐 + 浮动收益列 + 预期收益答疑（D-055）· 业主反馈 → 决策层施工 + 部署
- **参与方**：业主（反馈两问）、Claude（答疑 + 决策层 + 施工 + 部署实测）
- **议题**：① 「我们只做了开仓，还有平仓功能没做，也就只做了 1/2，模拟持仓中要显示浮动收益」；② 「持仓中预期收益是不是降了？」
- **答疑（Q2 预期收益降了？）**：**是**，但非 bug——`expected_ann` 是**实时投影**（当前 okx BTC funding 年化，`latestValue(KindFunding)` 每 tick 查最新），不是开仓时锁定值。facts.md 证据：okx BTC funding 年化 8/15 20:42 8.018% → 22:12 8.284% → 23:55 7.225% → 8/16 09:57 4.059% → 当前 4.899%（与持仓区 expectedAnn=4.899 一致）。**市场条件变化（funding 回落），非系统错误**；持仓收益 = 已按 open 时刻起每 8h 结算的 funding 落袋 + 浮动，预期年化只是「现在若续持一年」的投影刻度。
- **决策（D-055）**：
  ① **平仓 = 订单级整单平**（D-019 绝不单腿）：新 RPC `CloseSimOrder` + store 单事务「filled 守卫 → 逐腿 pnl+=浮动 + settled（腿须属本单且 open，任一 miss 回滚）→ 订单 closed + note」；funding_hedge 两腿一起退。
  ② **AddPnl = 当前价浮动** `(cur−ref)×qty×dir`，ticker 缺失 add=0（不编造浮动）；realized_pnl = 各腿 (pnl+add) 合计；realized_rmb = 即期折算（缺失 = 0 标 USD 原值）。
  ③ **双层防重复平**：服务层 status != filled → FailedPrecondition + store 守卫（并发双平仅先到者成功）。
  ④ **迁移 0008**：sim_orders.status CHECK 加 closed（部署自动应用）。
  ⑤ **前端**：持仓表独立「浮动收益」列（当前价缺失标 —）+ 每订单首腿「平仓」按钮（二次点击确认 armed 警示高亮，rowSpan 整单表达）+ 成功横幅显示实现净 PnL（USD + 即期 RMB）；settled 腿移出持仓表。
- **施工**：migration 0008_sim_close.sql + store.go（SimLegClose/SimStatusClosed/CloseSimOrder 接口）+ pgstore/sim.go CloseSimOrder（事务 + 三守卫 + 对抗锚点注释）+ sim_test TestCloseSimOrder + migrate_test want 7→8 + sim.proto（CloseSimOrderRequest/Response + rpc，头注释 4→6）+ `buf generate --template buf.gen.sim.yaml`（**教训：sim 域必须用独立模板，裸 `buf generate` 不碰 sim**）+ service.go CloseSimOrder（前置校验 + latestValue 浮动 + fx 折算，包 doc 5→6 RPC/双写路径）+ service_test fakeStore（ListOpenSimPositions/CloseSimOrder 真语义）+ 5 处 fake panic stub（dashboard/collect-manual/alert/rule）+ 6 场景服务测试 + hooks.ts useSim.close（返回响应含 realized_pnl）+ SimExec.tsx（浮动列 + rowSpan 平仓按钮 + 二次确认 + 结果横幅 + settled 过滤）+ SimPage 传参 + style.css .btn-close（默认警示描边 / armed 实心）。
- **验证/部署**：全量 go test（pgstore TestMigrateIdempotent want 8 通过 = migration 0008 应用 + TestCloseSimOrder 全绿 + simapi 6 场景）+ vet + tsc/npm build 绿；锚点 TestSimKindLabelCoverage / TestSimExecBadgeRenderable PASS（SimExec/ConfirmPanel 仍引 SimulatedBadge，Opportunity 无 funding_hedge 字面量）；构建 → SIGKILL 部署（arbcn-monitor.service，MainPID 2919956→2935391）→ **healthz `pending_migrations` → `ok`（0008 自动应用）** + inode 匹配（21244442→21244499）+ served bundle == 新 dist（index-EdR3ou5I.js 含「确认平仓/浮动收益/closeSimOrder」）。**零执行门禁/规则/阈值/D-016/MinSpread/CarryMinSpread/白名单改动；不接 LLM（D-043）；不赌（D-019）——平仓仍人工触发、模拟盘无真实资金路径。**（practices #19 闭环）
- **决策号**：D-055（平仓 + 浮动收益列 + 预期收益答疑）；教训候选入 practices #30（对称补全 checklist + buf 独立模板）。

## #73 · 2026-08-16 · 模拟账户「按真实账户对待」完整现金账本 + 秒级实时报价（D-056）· 业主两条需求 → 决策层施工 + live 核账 + 部署机实测
- **参与方**：业主（两条需求）、Claude（选型答复 + 决策层 + 施工 + 实测）
- **议题**：① 「虽然是模拟账户，但我们应该按真实账户对待」——校验策略真赚还是理论；② 「之前也要求过显示实时报价，好像没有实现」。
- **选型答复（业主确认）**：① 账户口径 = **完整现金账本**（现金余额 + 逐笔流水 + 净值，与开平结算原子联动）；② 报价刷新 = **秒级实时**，上游（交易所→后端）WebSocket（交易所只提供 WS 推送）、下游（后端→前端）SSE（EventSource 自动重连零新依赖，过现有 HTTP mux）。报价流只做展示，不喂策略不落库。
- **决策（D-056）**：
  - **Part A 现金账本**：migration 0009（sim_account 单账户 id=1 + sim_cash_flow 逐笔流水）+ 四 kind 事件（capital_in/open/funding/close）全在既有事务内原子入账，Fill/Accept 两成交路径口径一致；**不变量 = equity=cash+Σ_open(dir×qty×cur)=capital+realized+unrealized 双恒等式交叉校验**；InitSimAccount 启动 seed 幂等（重启不重置 cash 跨重启持久）+ 单事务（原实现非事务半账已修）；GetSimAccount RPC（净值 + 最近 100 条流水）；CloseSimOrder 服务端补 CashDelta；前端 AccountZone 五数对账网格 + 净值 USD/RMB + 逐笔流水表（账户区置顶第一眼）。
  - **Part B 秒级实时报价**：上游 binance USDT-M 合流 miniTicker + okx v5 tickers 公共行情（无密钥 D-010）/ 下游 SSE 1s 差分推送；新包 internal/quote + gorilla/websocket 唯一新依赖；断线指数退避自愈 + 8s 拨号上限 + 45s 读空闲；前端 QuoteStrip 顶部报价条（EventSource 自动重连）。
- **根因修复**：binance 合流 feed **静默丢帧**——Go json 字段名大小写不敏感匹配致帧内 `"e"`（事件类型字符串）撞上 `EventT`(int64)，整帧 Unmarshal 失败全丢；补显式 `EventType string json:"e"` 字段 + 测试用真实帧形（envelope + e/E 并存）锚定（删字段必红）。
- **施工**：见 STATE 施工表 D-056 Part A/B 两行（迁移/store/sim/backfill/main/proto/simapi/quote 包/frontend 全链路）。
- **验证/部署实测**：全量 go test（pgstore want 9）+ vet + -race + npm build 绿；**live 核账**：seed 100000/100000 + 重启幂等 1 条 capital_in + GetSimAccount RPC 双恒等式对平 + 历史对冲单（id=3/5）pnl 净 0 无漂移（无需回填）；**部署机双流实测**：binance BTC 62956.9/ETH 1879.79 + okx BTC 62965 秒级推送（SSE）。**零执行门禁/规则/阈值/D-016/MinSpread/CarryMinSpread/白名单改动；只展示不自动执行（D-013/D-016 门禁不动）；不赌（D-019）；无密钥公共端点（D-010）。**（practices #19：本对话交付未完——部署重启 + 推送在对话 #74 的 commit 一并闭环）
- **决策号**：D-056（Part A 现金账本 + Part B 实时报价）；教训候选入 practices #31（Go json 大小写不敏感匹配）+ #32（启动 seed 接线 main.go + 单事务）+ #12 续（binance WS 端点 geo-block 同样须部署机实测）。

## #74 · 2026-08-16 · 确认下单面板删减（对冲对合成单行摘要）· 业主「太长」反馈 → 决策层施工
- **参与方**：业主（反馈 + 选档）、Claude（数据盘点 + 施工）
- **议题**：「监控总览中的确认下单，都显示那些数据？太长了，我们要做一些删减」。
- **数据盘点（现状 6 列）**：类型（现货+永续对冲）+ 行内「模拟」标签 / 标的 / 方向（funding_hedge 单 side 恒为「对冲」）/ 数量 / 预期年化 / 操作。**冗余识别**：类型列与方向列同义重复；行内 SimTag 与标题 SimulatedBadge 双重「模拟」标注。
- **决策（业主选档）**：**单行摘要，对冲对合成一行**——每单一行 `BTC @okx · 对冲 20,000U · 预期 6.61%` + 确认按钮；删 类型/方向/数量 独立列 + 行内 SimTag + h2 SimTag（锚点 TestSimExecBadgeRenderable 仅要求 SimulatedBadge 引用，已保留）。
- **施工**：ConfirmPanel.tsx 重写（ConfirmRow → `<li>` 单行摘要 flex，.pending-list；header 保留 SimulatedBadge + 刷新）+ style.css 新增 .pending-list/.pending-summary/.dir-tag/.spread（可换行不横向溢出）。纯前端零门禁改动。
- **验证**：tsc/npm build 绿（新 hash index-BFYvO4mt.js）+ 全量 go test 绿（含锚点 TestSimExecBadgeRenderable PASS）。部署随 D-056 对话 #73 一并闭环。
- **决策号**：—（业主 UI 删减，非方向级；对话 #73 D-056 commit 一并交付）。

## #75 · 2026-08-16 · D-056 测试事故：go test 误清真实库 sim 账本 → 根因双修 + 复位 · 事故报告 → 决策层修复 + 业主确认清理
- **参与方**：Claude（事故发现 + 根因 + 修复 + 请求确认）、业主（确认清理复位）
- **议题**：D-056 收工核验发现真实 arbcn 库 sim 账本被污染（capital=0 / cash=123.45，真实 drill 单 1-5 消失）。
- **事故根因**：上会话 `go test` 的 `ARBCN_TEST_PG_DSN` 误指向真实库 `arbcn`（应为专用测试库 `arbcn_test`）；pgstore 测试 `resetTables` 对 sim 四表 `TRUNCATE CASCADE` → 清掉真实 drill 单 1-5（不可恢复）+ 遗留测试行：订单 16/17（src_rule 空、sim_local，来自 TestRejectSimOrderAppendsFlag/TestAcceptSimOrderAtomicity）、流水 7-13（open ±6e8 = 10000×60000 + funding 123.45 + close 0，TestCloseSimOrder 数据）、sim_account `capital=0 / cash=123.45` 卡死 0（applyCashFlow 兜底 upsert 建行后 InitSimAccount `ON CONFLICT DO NOTHING` 永不补本金，真实设计缺口）。订单 18（funding_drill @ binance）为真实演练拒单，保留。
- **根因双修（代码）**：① `testPool` 安全闸（pgstore_test.go）：`ARBCN_TEST_PG_DSN` 的 dbname 不含 "test" → `t.Fatalf` 拒绝执行（migrate_test 的 DROP 同受保护；负测：DSN 指向 arbcn 立即拒绝且零触碰）；② `InitSimAccount` 补正（sim_cash.go）：`ON CONFLICT DO UPDATE SET capital = CASE WHEN capital=0 THEN 配置本金 ELSE 保留 END` + 仅当无 `capital_in` 流水才入金（幂等，重启不重复入金）；对抗测试 `TestInitSimAccountRepairsUnfunded`（删补正/删入金必红）。
- **业主确认**：AskUserQuestion「确认清理（推荐）」——删除测试残留（sim_cash_flow 7 条 / sim_orders 16、17 / sim_positions 0 / sim_account 1 行），保留订单 18，重启重新入金。
- **清理 + 复位 + 部署闭环**：事务内删除（before 7 flows/3 orders/1 account → after 0/1/0）→ 旧二进制重启 seed 100000/100000 + 1 capital_in → 换 Fix-2 二进制（inode 21244553，mv 原子替换避开 text-busy）重启幂等（capital_in 仍 1 条）+ GetSimAccount RPC 双恒等式对平（equity=100000）+ served bundle == dist 不变 + healthz ok。
- **决策号**：D-057（事故修复，decisions.md）。

## #76 · 2026-08-16 · 确认：只用 binance/okx 报价接口，无任何交易接口（含模拟）· 业主提问 → 决策层核实
- **参与方**：业主（提问）、Claude（代码核实）
- **议题**：「我们只是用了 binance 和 okx 的报价接口，并不使用他们的交易接口？模拟交易接口都没用，是吗？」
- **核实结论（逐层）**：① **真实交易所 = 纯报价**：internal/quote 仅 WS 公共行情（binance `wss://fstream.binancefuture.com/stream?...@miniTicker`、okx `wss://ws.okx.com:8443/ws/v5/public`）+ facts REST ticker/funding（公共端点）；全库 grep 无任何 `x-api-key/secret/signature` 调真实交易 API（D-010 零密钥）；② **模拟盘订单生命周期 = 纯本地 PostgreSQL**（sim_orders/sim_positions/sim_cash_flow），面板点「确认成交」走 AcceptSimOrder/CloseSimOrder 纯 DB 事务，**不调用任何交易所模拟下单接口**；③ **唯一例外 simtestnet 探针（D-040）**：连 `https://testnet.binancefuture.com` + okx demo，函数仅 `probeBinance/parseBinanceBalance`、`probeOKX/parseOKXBalance`（**只读余额 GET**，无 create_order/close_position），用 testnet/demo 密钥（arbcn-sim.env）非真实资金。
- **决策号**：—（事实核实，无方向改动；§1 定位「人工下单、零执行」再确认）。

## #77 · 2026-08-16 · 实盘距离核实 + 推进建议 · 业主两问 → 决策层核验 + 建议
- **参与方**：业主（提问）、Claude（核验 + 建议）
- **议题**：「这样的设计，跟实盘交易差得了多远？」+「项目要推进，最终是要走到实盘真实资金（交易所真实交易，非模拟）。就目前策略盈利能力，你的建议是什么？」
- **核验（逐层）**：① 决策/记账/报价层 ≈ 生产就绪 80-90%（机会雷达/门禁/账本 D-056 双恒等式/报价流都是真数据真口径，可直接服务实盘决策）；② 执行/资金/保证金层 = 0（模拟订单纯本地 DB、确认即成交，无真实下单/成交回读/摩擦/保证金）；③ 诚实收益：当前 BTC funding ~4.9% / ETH ~4-7%，funding 窗口档（15-30%）不存在，真实可交易面 = 稳定币基档 3-4% + BTC cash-and-carry ~7%（delta 中性）+ sUSDe 4.44%；④ 系统上线 ~1 天，零累积 PnL 证据——「策略盈利能力」尚无任何实证。
- **建议（三阶段）**：A 真金冒烟（~5% 资本，先小钱买「摩擦后真实盈利」唯一实证）→ B 证据积累（3-6 月 + 硬门）→ C 按证据放大（D-026 三档）。
- **决策号**：D-058 前置（建议后被纠偏重排，见 #79）。

## #78 · 2026-08-16 · 阶段 A 可否用交易所模拟交易 API（D-010 放开）+ 起草授权 · 业主提问 + 授权
- **参与方**：业主（提问 + 授权）、Claude（可行性核验）
- **议题**：「阶段 A，用现在交易所的模拟交易 api 是否可以做到？D-010 可以放开」→ 我确认可行后：「好的，现在起草」。
- **可行性核验**：① D-010 已有 testnet key 豁免条款（D-034 ②），放开的是把 testnet key 从只读探针扩到可下单；② 工程可行——internal/simtestnet 已有 HMAC 签名 + 端点 + key 承载（SIMULATED 门控），扩展下单是增量；③ 定位为阶段 A-0 执行层验证；④ 诚实边界——testnet 验证执行机制，**不验证盈利能力**（testnet funding 假、成交薄、摩擦不真实）。
- **决策号**：D-058 起草授权（后被纠偏，见 #79）。

## #79 · 2026-08-16 · 盈利测试意义澄清 + 纠偏 + 方向确认 · 业主两次纠偏 → 决策层采纳定稿
- **参与方**：业主（澄清追问 + 两次纠偏 + 确认）、Claude（澄清 + 采纳）
- **议题**：①「看了诚实的边界，这么说来，其实对盈利能力的测试，意义不大？」→ 澄清：**「用 testnet 测盈利」意义不大；「盈利测试」只能靠真金，testnet 的作用是保证真金测试不被执行层 bug 污染**（三层测试分工：策略层=本地 sim 账本 + 真实行情，行情真成交假；执行层=testnet，机制真价格假；盈利层=真金，全真）。②「如果只是验证执行机制，那大可不必；第 1 原则是盈利，策略不盈利执行机制有什么用？本地模拟盘能否校验策略盈利能力？能，就不必浪费时间到 API 接入；确认能盈利再接入 testnet 不晚」→ 采纳：本地模拟盘**能**校验策略层盈利（真实行情 + 真实费率 D-037，只假设按报价成交），是第一道闸且不需要 API；testnet 从必经降为**可选预演**（冒烟前想假钱演练执行全流程才用，非必经随时可补）；路线重排为盈利验证优先。③「模拟和实盘有差距这个正确，但模拟都是亏损的，说明这条路就不通，何必再走下去？方向一致！」→ 确认，D-058 定稿。
- **决策号**：D-058（盈利验证优先路线，testnet 降级可选预演，D-010/D-034 维持只读不扩下单）。

## #81 · 2026-08-16 · 确认下单布局 + 平仓卡 + 拒单记录 + 复核自动证据 · 业主三问 → AskUserQuestion 确认 → 施工部署
- **参与方**：业主（三问 + AskUserQuestion 确认）、Claude（核验 + 施工 + 部署）
- **议题**：①「监控总览里的确认下单内容太长需滑块才能看到（刚改了不知是否生效）；市场结构与确认下单之间有空白可否加平仓卡？第二门槛拦下的订单没有记录？」②「市场结构复核手动完全对应不上，更填不上『本次复核的依据/结论』，自动验证审核是否行得通？」
- **核验**：部署确为最新（served bundle 含复核表单/平仓/账户区）；ConfirmPanel 待确认单多时无高度封顶会向下延伸（业主"需要滑块"）；拒单负样本在库（id=18 rejected SPREAD_DRIFT 53.28%）但 UI 不可见（OrderZone 对话 #59 移除后无出口）；复核"对应不上"根因 = 条目 rationale 是历史案例，当前数据无对照。
- **AskUserQuestion 确认**：复核自动验证 = **「自动证据 + 人确认（推荐）」**（系统自动生成当前数据证据，人工做最终裁决，边界 = 系统永不自动改判定）；拒单记录展示 = **「模拟执行 tab 订单历史」**。
- **施工（D-059）**：后端 proto KnowledgeEntry.current_evidence + knowledgeEvidence/三 helper + ListKnowledgeEntries 编排（故障降级/缺数据不编造）；前端 ClosePanel 平仓卡（市场结构与确认下单之间）+ ConfirmPanel/ClosePanel scroll-cap + OrderHistoryZone（模拟执行 tab 六态订单历史 + rejected 拒单原因）+ KnowledgeBoard 证据行 + ReviewForm note 自动预填。零执行门禁改动。
- **部署实测**：全量测试/vet/npm build 绿；原子替换重启 healthz ok；served bundle == 新 dist（index-DRVZi1Rk.js 含「订单历史/当前核验/确认平仓/已拒单/自动核验」）；live ListKnowledgeEntries 三条 evidence 正确（spike_trap 命中 ETH 10.95% vs 均值 3.75%（×2.9）/ cross_venue 命中 BTC 7.5pp·ETH 8.3pp / defi 未命中 SUSDE 4.39%）；ListSimOrders 拒单 18 负样本在库（SPREAD_DRIFT 53.28%）。
- **决策号**：D-059。

## #82 · 2026-08-16 · 画布加宽 + 复核按钮变形 + 右列间距/顺序 + 复核按钮同行 · 业主五问 → 施工部署
- **参与方**：业主（反馈）、Claude（核验 + 施工 + 部署）
- **议题**：① 市场结构经验库再宽一点；② 点开复核时确认按钮变形；③ 整个底画布更宽一些；④ 右列三卡（经验库/确认下单/平仓）间距太大、把左1 矩阵跟着拉高了，且确认下单应在平仓上方（对调）；⑤ 复核表单的取消/确认复核按钮应与复核结论同一行。
- **核验**：`main` 整体就是那个"底画布"（max-width 1100px）——加宽它即同时满足 ①③；经验库在右列（3fr/2fr 的 2fr，桌面 ~421px），右列变宽依赖画布与列比例；**② 确认按钮变形根因** = `.review-actions` 桌面端无 `grid-column`，grid auto-placement 把它塞进复核表单第一列 110px（row 2 col 3）→ 两按钮被挤变形（手机有 span 2 桌面漏了）；**④ CDP 实测 1280**：右列三卡间间距 31/32px = **双倍间距 bug**（每卡自带基础 `.card{margin-bottom:16px}` + 右列 flex gap 16px 叠加；`.grid > .right-stack` 的 margin 清零只作用于 right-stack 自身、未覆盖其内部卡），右列总高 998px 撑过矩阵（矩阵内容实为 996）→ 整行被右列拉高；DOM 序为 经验库→平仓→确认下单，业主要求 确认下单在平仓上方；**⑤ CDP**：复核表单 复核结论（row2 span 2）+ 按钮（row3 全宽）不同行。
- **施工（纯前端 CSS + OverviewPage DOM 序，零后端/RPC/门禁改动）**：①③ `main` max-width 1100→1440px；`.grid` 3fr:2fr→5fr:4fr（右列 ~421→619px、左列 631→773px 同步加宽无回归，860px 折叠单栏不受影响）；② `.review-actions` 显式 `grid-column` + 右对齐；④ `.right-stack > .card{margin-bottom:0}` 消双倍间距 + `.right-stack` gap 16→12px（三卡更紧凑聚合）+ OverviewPage 交换 ConfirmPanel/ClosePanel 顺序 = [经验库, 确认下单, 平仓]；⑤ 复核表单 `grid-template-columns:110px 1fr auto`，复核结论 span 2 + 按钮 `grid-column:3` 同行右对齐（align-items:end 底部与输入对齐），手机 ≤640px 改回 `span 2` 整行（基础 col 3 在 2 列网格会隐式建第三轨道）。
- **部署实测**（CDP 1280/375）：右列间距 31/32→**11/12px**；DOM 序 = 经验库→确认下单→平仓；页高 2055→**1999**（矩阵 998→941 回落，右列不再撑高左1）；复核表单按钮与复核结论输入底对齐 0px（同行 ✓）；375px 无横向溢出 + 手机 actions 整行落复核结论下方（触屏友好）；served bundle == 新 dist（index-Cyt2Tw60.js / index-Dok6zbWG.css）+ healthz ok；全量测试/vet 未受影响（纯 CSS）。
- **决策号**：—（纯呈现层，无方向改动；D-050/D-052/D-053 布局演进延续）。

## #83 · 2026-08-16 · 复核触发机制 + 无 claude 时系统如何判断 + 是否接 LLM · 业主四连概念问 → 选路线 A（不接 LLM）
- **参与方**：业主（四连问 + 确认 + 点名落「不做 B 的理由」）、Claude（概念澄清 + 方案对比 + 施工部署）
- **议题**：①「再次复核在什么情况下触发？」②「我没有判断信号的能力，怎么知道要不要再次复核？」③「系统上线后就没有 claude 了，系统怎么做判断？」④「要不要接入 LLM？接入能做到 claude 这么聪明吗？」；收尾明示「不做 B 的理由也要落文档，后期有痕可查」。
- **澄清与决策**：
  ① **复核触发（现状）**：当前无自动触发——validated_at 仅 ReviewKnowledgeEntry 人工写入；知识命中（knowledge_match）只在「进化建议」提示、不自动改判定。
  ② **「没有判断信号的能力」→ 业主不需要判断**：系统按 D-059 自动证据 + 本次翻转检测给结论性提示（建议复核 / 仍适用），业主只需二选确认。
  ③ **「上线后没有 claude」→ 三角色运营模型**：系统 7×24 自治（采集/计算/检测/证据/候选过滤）+ 业主二选确认 + 决策层定期会话（D# 判定变更）；Claude 从随时在线降为定期维护。
  ④ **LLM**：路线 A（不接 LLM，维持 D-043；复核方向快照 + 自动翻转检测 + 待决策层清单）vs 路线 B（接 LLM 自动判断）；**业主「同意你的建议」选 A**。不做 B 的理由（判断力错配 / 成本复杂度 / 责任不可归属——详见 D-060）按业主要求落决策记录留痕。
- **施工（D-060）**：migration 0010 review_direction（幂等 IF NOT EXISTS）+ store ReviewKnowledgeEntry 加 direction + evidenceResult{text,hit,known} 重构（方向同源产出不反解析文本）+ recheckNeeded 翻转检测（proto #14）+ ReviewKnowledgeEntry 服务端方向快照（COALESCE 不覆盖）+ 前端 KnowledgeBoard 四态徽标/建议复核置顶/警示条/warn 按钮/上次核验方向。
- **部署实测**：全量测试/vet/npm build 绿；**上线即抓真实事故**——ListInsights 503 = pgstore 扫 NULL review_direction 进 string（fake 用零值测不出、真库存量行有 NULL）→ *string 兜底 + NULL scan 回归测试；served bundle index-_k06EENF.js 匹配 dist + healthz ok；CDP 实测正常态（3 条「生效中+已复核」无警示条、间距 12px、右列序 经验库/确认下单/平仓、无横向溢出）+ 翻转态（Fetch 拦截注入 recheck → 警示条「建议复核 1 条」+ 资金费率尖峰陷阱置顶 warn 徽标 + 「建议复核」警示按钮 + 「上次核验 未命中」）。
- **决策号**：D-060。

---

## #84 · 2026-08-16 · 拒单理由自明性改进（SPREAD_DRIFT note 前后数值 + 方向 + 阈值）

- **议题**：业主查库问「2026-08-16 19:56:14 已拒单理由是什么？备注项是拒单理由吗？」→ 查证 = **sim_orders id=18**（funding_hedge 演练单 binance BTC，rejected，risk_flags={SPREAD_DRIFT}，note=`SPREAD_DRIFT: 预期年化变化 53.28%`，ref_price=62963，expected_spread=5.796%）。
- **答疑（两次纠偏业主误读）**：① note 就是拒单理由（人类可读），risk_flags 是机器可读标记，二者对应；② **53.28% 是变化比例不是目标值**——生成时按 30d 均值预期年化 5.80%，确认时刻 binance BTC 当前 funding 已回落至 ~2.7%（`|2.71−5.80|/5.80≈53.3%`，DB 实据 21:00 起 binance BTC 2.6–4.1%），触发 `ConfirmDriftCheck` 年化变化 >20% 阈值（另有 ref 漂移 >2% 独立触发，D-038 C2/D-036 G5）→ fail-closed 拒单，设计内宁缺毋滥，从未成交无持仓。业主最初把「变化 53.28%」误读成「涨到 53.28%」。
- **需求**：业主「明白了，那就是**理由说得不够详细清楚，这里需要改善**」。
- **结论**：`internal/sim/confirm.go` 两条漂移拒单理由改为**自明格式**——年化：`SPREAD_DRIFT: 预期年化 5.80% → 2.71%（回落 53.30%，超阈值 ±20%）`（前后数值 + 方向词回落/上行 + 变化比例 + 阈值）；参考价：`SPREAD_DRIFT: 参考价 62000 → 65000（漂移 +4.84%，超阈值 ±2%）`。id=18 同款场景实测产出「预期年化 5.80% → 2.71%（回落 53.30%，超阈值 ±20%）」。测试断言更新（含回落/上行方向词对抗）+ 新增回落场景；其他拒单理由（UNHEDGED/SPREAD_LOW/SIZE_OVER/DAILY_OVER/WHITELIST/INVALID）已带数值/说明，无需动。
- **部署实测**：全量测试/vet 绿 + 原子替换重启 + systemctl active + healthz ok + 运行二进制 = 新 /opt/arbcn/bin/arbcn。**历史记录 id=18 note 保持原样不改写**（审计留痕 P3，新格式只对后续新拒单生效）；如业主希望把 id=18 这条展示也改成清晰版可单独说。
- **决策号**：无（消息文案改进，非设计决策）。
- **后续（业主拍板）**：业主问「需要新的拒单才能看到新的理由吗？」→ 答：是（note 拒单时刻定格，新格式只对后续新拒单生效），可一次性回填 id=18 立即看到 → **业主「改成新的格式」** → 生产库 `UPDATE sim_orders SET note='SPREAD_DRIFT: 预期年化 5.80% → 2.71%（回落 53.28%，超阈值 ±20%）' WHERE id=18`（方向回落依据 = 当时 binance BTC funding 已回落到 ~2.7% 段，21:00 起 DB 实据 2.6–4.1%；53.28% 沿用原记录值、curSpread≈2.71 由 drift 反推，均为当时真值）→ ListSimOrders RPC 实测返回新文案（订单历史已显示）。**审计留痕例外**：历史 note 原定不改写（P3），本次经业主显式授权回填 = 纯展示修正（不改逻辑/门禁），已记 STATE。
- **后续②（业主 UI 反馈）**：业主「因为页面的问题，拒单理由不能显示完整，需要优化，让理由能完整显示」+「订单历史，表格的左边还有很多空白」→ CDP 实测定位根因：① `.note-cell` `max-width:220px + nowrap + ellipsis` 截断（新理由内容 519px 只显 300px）；② `table.rows td` 全右对齐 + 自动布局把列撑宽 → 文字内容贴列右、列内左侧空出大片空白（实测时间列 217px 内容仅 ~150px、状态列 120px 装 ~50px 徽标）。修法（作用域限 `[aria-labelledby="sim-orders-title"]`，不误伤 SimExec 流水表 note-cell）：`table-layout:fixed` + 显式列宽（时间148/状态76/类型90/标的88/方向44/数量72/预期64）+ 文字列（1-5）左对齐 / 数字列（6-7）右对齐 / **备注列吸收剩余宽度（桌面 617px）+ 去截断可换行** + 表格 `min-width:880px`（小屏备注不被压成 0 宽竖条，走 `.table-scroll` 横向滚动）。CDP 实测：桌面备注 617px 完整显示 truncated=false、首列内容从表格左缘起 colLeftBlank=0（左空白消除）、数字列右对齐保留 20-23px 自然对齐间隙；375 备注 298px 完整显示 + docSW==innerW 无溢出 + SimExec note-cell 保持原样。
- **后续③（业主反馈：桌面重叠）**：业主「桌面端表格收太紧，导致内容重叠了」→ CDP 实测溢出列：类型（现货+永续对冲 自然宽 116 vs 我设的 90）、标的（BTC@binance 117 vs 88）、时间（159 vs 148）、状态（78 vs 76）、方向（48 vs 44）——首版列宽按估算设太紧，`table-layout:fixed` 下内容（nowrap）压出格重叠。修法：按实测自然宽 + 余量重设（时间165/状态90/类型125/标的135/方向52/数量75/预期68，合计 ~710px）+ 类型/标的长资产名（BUIDL@blackrock-buidl 等）加 `white-space:normal + word-break` 断行兜底防重叠 + min-width 880→1010px。CDP 复测：桌面全列 overflow=false 无重叠、备注 489px 理由两行完整显示 truncated=false、首列 contentStart==tableLeft 无左空白、数字列右对齐保留 24-26px 自然间隙；375 备注 300px 完整 + docSW==innerW 无溢出 + SimExec note-cell 保持原样。
- **后续④（业主反馈：表头与值对齐）**：业主「订单历史的列表中，表头标题与值没有对齐，我认为表头标题应该在值单元格的中间。表头标题居中，单元格值为左对齐」→ 修法（作用域限 `[aria-labelledby="sim-orders-title"]`）：删除此前按列设的文字列/数字列对齐规则（`th:nth-child(-n+5),td:nth-child(-n+5){text-align:left}` + `th:nth-child(8),td:nth-child(8){text-align:left}`），替换为**全局** `th { text-align:center }`（表头标题居中对齐于值单元格中线上方）+ `td { text-align:left }`（单元格值统一左对齐，含数字列——业主明确「值左对齐」优先于数字列右对齐的惯例，数量/预期不再右贴）。CDP 精确测量（`document.createRange().selectNodeContents(th)` 取表头文字实际边界，此前 69px「offset」是我探测脚本用 span 估宽的表象、非真实渲染）：8 列 th computed textAlign=center 且**表头文字中心偏移 0px**（与单元格中心完全重合）、td 全 textAlign=left。375 无溢出 + SimExec 流水表 note-cell 未误伤（作用域隔离）。部署：npm run build（新 hash **index-BajggGUF.css / index-DcgC6jxF.js**）+ 原子替换重启 + systemctl active + healthz ok + served bundle == 新 dist（CDP 实测 8 列全居中/全左对齐已部署）。
- **后续⑤（业主反馈：拒单理由右对齐）**：业主「还是这个表格，桌面端显示的问题，拒绝理由内容，需要右对齐」→ 备注列（第 8 列，拒单理由）桌面端右对齐：`@media (min-width:601px)` 内 `td:nth-child(8){text-align:right}`（作用域限 sim-orders-title，特异性 (0,4,2) > 基础 td (0,3,2)，无论序必胜）——长理由文本贴右缘，与数量/预期数字列右缘对齐线一致；表头仍居中、其余 7 列仍左对齐不动；**移动端保持左对齐**（300px 窄列多行中文右对齐撕裂可读性；业主明确「桌面端」）。CDP 实测：桌面 1280 第 8 列 computed right + 其余 td left + 8 th center（内容显示 id=18 完整理由）；375 note 仍 left + docSW==innerW 无溢出。部署：npm run build（新 hash **index-Sh-YR4il.css / index-BS7-6fgp.js**）+ 原子替换重启 + healthz ok + served bundle == 新 dist。

## #85 · 2026-08-16 · 开源同类项目调研 + 回测认知裁决（第 1 原则是策略真盈利）→ D-061

- **参与方**：业主（两问 + 回测核心质疑 + 拍板「你的判断是正确的，走 D#」）、Claude（四路调研 + 诚实分维度裁决 + 三盲区补充 + 落档）
- **议题**：①「在 GitHub 有没有同类开源项目？有没有值得学习引进的部分？」②「如果引进这些功能，我们的系统可以算比开源都优秀吗？」③「关于回测，我并不大迷信，过去的历史并不代表未来的行情，我认为第 1 性原则是这个策略是否真的能盈利，跑 bot 的前提是策略确定性能盈利才能跑 bot，你认为呢？」
- **调研（①）**：四路并行检索——执行层机器人（Hummingbot 策略、CryptoBots、auto-arb，全为 Python+ccxt 自动执行）**代码一个不引**：引代码撞 D-010 无密钥 / D-019 不自动执行 / D-043 不接 LLM 三红线 + Python 栈错配（我们是 Go，引入 = 第二运行时）。扫描器类（algo-arbitrage / crypto-arbitrage-terminal / backpack-basis-trading-monitor / funding-rate-arbitrage-scanner）只有**模式**值得学。个人财务仪表盘（Wealthfolio Rust+Tauri / Maybe Rails 40k★ / Mosaic Go+PG+React 同栈）**栈接近但领域不同**（我们不是记账软件，是决策+记账）。MCP/LLM 工具（funding-rates-mcp、k.i.t.-bot）被 D-043 禁入。**可学 3 模式**：① TWR/MWR 收益率口径（台账/归因 v2 候选）；② 7d 费率窗口统计（7d min/max + 正费率占比，backpack-basis-trading-monitor 借鉴，直映「当前是否处于可交易窗口」）；③ 回测摩擦/滑点/杠杆建模 + 风险指标（Sharpe/maxDD/VaR）。
- **诚实裁决（②）**：分赛道——**决策支持+记账细分无开源对标（领先成立）**；但「比开源都优秀」**不成立**：回测引擎、执行层、社区生态、UI 四维成熟开源项目更强。不跟业主高估对齐（practices 已有教训：诚实是底线）。
- **认知裁决（③ → D-061）**：Claude 同意 D-058「盈利验证优先」排序（业主第 1 原则 = 策略真能盈利，本地模拟盘验证、确认盈利才跑 bot，与 D-058/对话 #77-#79 同源），并补充三个盲区，业主确认「走 D#」：
  ① **回测能证伪不能证真**——历史高费率时段也覆盖不了摩擦 = 结构性否定可信；回测赚钱对未来无预测力（过拟合 + 行情结构变化 + 摩擦低估）→ 回测从「可学模式」降级为**可选证伪自检**（只做「门禁条件回放」：≥15% 高费率过滤条件回放历史，验证门禁会不会正确触发、扣摩擦是否为正；非必做不阻塞阶段 0）。
  ② **判定门① 盲区：当前无机会环境会误判策略失败**——当前 funding 4–7% vs 15–30% 窗口档不存在，30 天前向 paper 大概率 ≈ 零单 ≈ 0 PnL，测的是**环境**不是**策略**；无机会时「零单」本身就是正确输出（宁缺毋滥 D-019）→ 判定门① 加**「环境-策略分离」条款**。
  ③ **前向验证信号薄**——诚实基线 3.2–3.7% ≈ 0.3%/月，摩擦假设（0.3% 已核实普通主户，D-046）与窗口代表性主导结论 → 测量时记录环境条件（当期 funding 中位数/可交易面/有无窗口档）随结果留档。
- **施工（D-061，docs-only 零代码）**：decisions.md 追加 **D-058 补充条款**——① 回测降级可选证伪自检（门禁条件回放，证伪可信证真无预测力，修订 D-036 边界）；② 判定门① 环境-策略分离（无窗口档「零单 = 正确输出」，禁止读成策略失败）；③ 测量口径补充（摩擦/窗口代表性/环境条件留档）；④ 7d 费率窗口统计 + TWR/MWR 列候选另立 D#。零执行门禁/规则/阈值/D-016/MinSpread/CarryMinSpread/白名单改动；不赌（D-019）不接 LLM（D-043）。
- **决策号**：D-061。

## #86 · 2026-08-16 · 所有决定引进的方案落地（方案一先行）+ 判定门① 自欺盲区 → D-062/D-063

- **参与方**：业主（「所有决定引进的方案，都要落地，而且要先落文档，确定引进方案，自我审计后开始执行，包括阶段0，具体先做那1个，由你决定」+ 中途补充「更大的盲区：判定门① 自己也会骗人，如果有条件的话，最好也一起解决掉」）、Claude（排定施工序 + 施工 + 防自欺层）
- **议题**：① D-061 列候选（TWR/MWR + 7d 费率窗口统计 + 门禁条件回放）全部落地；② 判定门① 测量数据面自己会骗人——快照缺口/数据损坏会让 TWR 静默失真但 gate 照判 PASS/FAIL。
- **决策（①）**：三个方案按依赖序落地——**方案一（阶段 0 判定门① 测量引擎 + TWR/MWR）先行**（直映 D-058「运行期定跨窗口 paper 收益测量口径」，最基础）→ 方案二 7d 费率窗口统计（可交易窗口判据）→ 方案三 门禁条件回放（可选自检非主线）。每个：落 D# → 定方案 → A–F 自审 → 施工。
- **施工（D-062 方案一）**：migration 0011 `sim_equity_snapshots`（ts PK，8h tick 落点）+ driver.snapshotEquity 每 settle tick 持久化 + `internal/sim/return.go` 纯函数（TwrAnnualized：无窗口内外部流 = 期初期末简单年化，有流 = Dietz 链乘分段；MwrAnnualized：IRR 二分收敛；Annualize：(1+r)^(365/days)−1；快照 <2 或 days≤0 → ErrInsufficientData/ErrDataAnomaly 不编造）+ `GetPerformanceReport` RPC #8（窗口 = 最近 30 天，TWR/MWR 年化 + 判定门① 判定 + 环境统计 funding 中位数/max/高费率时段/可交易面 + 快照覆盖/期望）+ EvaluateGate 判定（判定线 = GateBaselineHigh 3.7 + Friction 0.3 = 4.0%，pending/pass/watch/fail/env_no_window/data_anomaly 六态；零成交 → env_no_window 非失败（D-061 环境-策略分离）；拒单 >0 → watch「有机会未进场」；高费率时段/小样本 PASS 附加警示）+ 前端 SimExec PerformanceZone（判定门① 徽标 + TWR/MWR + 环境瓦片 + 快照覆盖/期望 + 判定线说明）；对抗锚点（删 snapshotEquity 写入必红等）。
- **施工（D-063 防自欺层，业主中途补充盲区）**：判定门① 可信度自检——**SnapshotCoverage**（8h tick 基线 3 份/天，<90% → 判定不采信 DATA_ANOMALY）+ **ValidateSnapshotIntegrity**（ts 单调 + equity≈cash+market_value 恒等式，损坏任何时候不采信）+ **GateTrustQualifier**（窗口未满 <30 天不误判数据坏；部分缺口附加警示不覆盖判定）+ **单位统一 ×100**（纯函数返回小数 0.708、gate 阈值与 RPC 字段为百分点点数 4.0——审计发现此前不经换算 0.7 vs 4.0 恒 FAIL，是真实的「判定门自己骗人」单位错配形态，编排层 ×100 一次共用同口径）+ PASS 自辩 caveat（环境红利/小样本明示）。测试锁定 TWR/MWR ≈70.84 锁死 ×100。
- **部署（对话 #86）**：build → bin 原子替换 → systemctl restart（sudo 密码业主提供，此前 systemctl 交互认证受阻）→ 验 healthz ok（pending_migrations 消除，migration 0011 应用）+ psql sim_equity_snapshots 表存在（0 行，首 8h tick 未跑，符合预期）+ GetPerformanceReport RPC 返回 status=pending（「数据不足（窗口未满或快照 < 2），前向验证进行中」）+ served bundle == 新 dist（发现 bin 比 dist 早 28min 导致 396B 漂移 → 重建二进制消除，BUNDLE MATCH ✓）+ 进程 active 26s。
- **决策号**：D-062、D-063。

## #87 · 2026-08-16 · 方案二 7d 费率窗口统计落地（D-061 候选之二）→ D-064

- **参与方**：Claude（排定序之二施工，业主「所有决定引进的方案都要落地」延续）
- **议题**：D-064 方案二 7d 费率窗口统计——「当前是否处于可交易窗口」判据（backpack-basis-trading-monitor 借鉴，D-061 列候选之二）。
- **施工（D-064）**：`internal/dashboard/windowstats.go` 纯函数——具名常量 FundingWindowDays=7 / WindowTierHigh=15（D-016 15% 档同源）/ WindowPositiveShare=0.9 / WindowWatchShare=0.5 / WindowMinSamples=3（改走 D#）+ `ComputeFundingWindowStats`（min/max/mean/正费率占比；空数据 → not「无数据」不编造 practices #7；样本<3 →「样本过少仅供参考」诚实标注）+ `ClassifyFundingWindow` 四类 high/tradable/watch/not（负均值/占比<50% → not，宁缺毋滥 D-019）；`windowstats_rpc.go` `ListFundingWindowStats` RPC（dashboard 域，QueryFacts Kind=funding From=now−7d 零新迁移零写路径 + overall + per_pair 均值降序前 50 + 注入时钟 s.Now）+ proto ListFundingWindowStats + 对抗锚点测试（高费率档优先/占比判据/空守卫删必红）+ RPC 测试（窗口边界过滤 / 分组排序 / 空窗口，fakeStore 时钟适配——fake 未设 To 兜底 time.Now()，测试 facts 须取 t0 附近真实时刻）。前端 `FundingWindowZone.tsx`（机会实算卡同域只读证据面：overall 徽标 + 均值/最低/最高/正占比/样本瓦片 + 逐 venue·symbol 表）+ hooks useSnapshot 第 9 路 RPC + `windowDefaults.ts` 兜底常量（**hooks.ts 463→449 行防 450 check-lines 硬阻断拆分**）+ style.css。
- **部署实测（对话 #87）**：npm run build + go build → systemctl restart（sudo 密码业主提供沿用）→ healthz ok + **served bundle == dist（JS+CSS md5 逐字节一致，比上次只比对 hash 名更严）** + **ListFundingWindowStats RPC 返回真实数据**：overall tradable（473 份读数 / 正占比 91% / 均值 5.19%，note 自明「正费率占比 91%（≥90%）且均值 5.19% ≥ 0」）；per_pair 6 行均值降序 okx/ETH（100%，9.84%）→ okx/BTC（100%，9.31%）→ binance/BTC（100%，4.44%）→ binance/ETH（97%，3.66%）→ okx/TRX（92%，3.04%）→ binance/TRX（watch 57%，0.73%）；与 psql 实查 funding facts 7d 聚合完全一致。当前环境 = tradable 非 high（未达 15% 档，诚实基线区，机会实算卡/判定门① 环境判据同口径呼应）。
- **决策号**：D-064。方案三 门禁条件回放为末位可延后（D-061 排定序），阶段 0 运行态观察优先。

## #88 · 2026-08-17 · 方案三 回放证伪门禁落地（D-061 候选之三）+ 业主纠偏「做成门禁」→ D-065 修订

- **参与方**：业主（纠偏「不做可选，是每个策略都自动做，做成门禁」）、Claude（重设计 + 施工 + 部署）
- **议题**：D-061 候选之三 门禁条件回放落地。原 D-065 设计为「可选证伪自检 / dashboard 域只读证据面」；收尾时业主纠偏——**回放判据不做可选，每个策略都自动做，做成门禁**。
- **决策（D-065 修订）**：回放判据升格为 **sim 域订单管线强制门禁**：纯函数迁 `internal/sim/replay.go`（泛化 tier/friction 参数；`replayGateCfgs` 表 = 每策略自有高费率档 funding 15%/0.3%、carry 8%/0.3%、repo 5%/0（OTC 无 taker 费，防单读数错杀），值锚 D-016/D-021/D-061①/D-046）；`Signal` +ReplayVerdict/ReplayNote（Driver buildSignal 对每个建单信号强制预计算，保 SignalToOrder 纯函数无 I/O）+ `RiskReplay=REPLAY_VETO` 门禁（falsified 结构性证伪 / watch 净不抵稳定币基档 4.5% → 拒单；pass=证伪未发生非收益预测 / no_window=D-061② 环境无窗口 → 放行）；simapi `GetReplayState` RPC #9 只读证据面（P4 可检查性：门禁休眠也可见）+ SimExec ReplayGateZone 卡。
- **施工**：`sim/replay.go`（ComputeReplay/OverallReplay/ReplayKindConfig/Driver.replayGate）+ `order.go`（Signal 字段 + RiskReplay 门禁）+ `driver.go`（buildSignal 强制回放填充 + settleFactKind 读 SSOT 表）+ `simapi/replaystate.go`（GetReplayState：每 kind 历史分组逐对回放 → overall）+ proto #9 RPC + 前端 ReplayGateZone（自包含拉取防 hooks.ts 顶 450）+ 对抗锚点测试（删门禁分支/删摊摩擦/删 no_window 守卫/删 buildSignal 填充/删表行必红）+ simapi 测试 + repo 摩擦=0 防错杀测试。
- **部署实测（对话 #88）**：npm run build + go build → systemctl restart（sudo 密码业主提供沿用）→ healthz ok + **served bundle == dist（md5 逐字节一致，JS+CSS）** + **GetReplayState 真实数据：三策略均 no_window**（funding 4505 样本、max 10.95%、无 ≥15% 读数；carry 105 样本；repo 48 样本；与 psql 实查完全一致）= D-061② 门禁休眠正确输出（历史无高费率档，门禁不误判不空转）。
- **决策号**：D-065（修订块）。**D-061 方案一/二/三全数落地毕**；下一步 = 阶段 0 运行态观察。

## #89 · 2026-08-17 · SimExec 卡序重排：模拟持仓/订单历史上提（业主指令）

- **参与方**：业主（「模拟持仓/订单历史，这两个卡片提上来，放到模拟账户与判定门① 阶段 0 测量之间」）、Claude（重排 + 部署）
- **议题**：SimExec 面板卡片顺序。原序 = 模拟账户 → 判定门① 测量 → 回放证伪门禁 → 模拟持仓 → 订单历史。业主要求持仓/订单历史上提。
- **决策**：新序 = **模拟账户 → 模拟持仓 → 订单历史 → 判定门① 测量 → 回放证伪门禁 → 测试网账户 → 对账报告**。持仓/订单历史 = 「第一眼」操作面（看仓 + 看拒单负样本），紧跟账户对账；判定门① 测量与回放门禁置后（只读观察面，非高频操作）。
- **施工**：`SimExec.tsx` 仅 JSX 渲染序调整（PositionZone + OrderHistoryZone 移至 AccountZone 与 PerformanceZone 之间）；零后端/RPC/门禁/样式改动。
- **部署实测**：npm run build → go build → systemctl restart → healthz ok + served bundle == dist（md5 逐字节一致）。
- **决策号**：纯前端呈现层，无 D#（布局演进延续 D-048/D-050/D-052/D-053 既有线）。

## #90 · 2026-08-17 · facts.md 快照无界增长根治——快照段封顶 + 规则触发节流（D-066，STATE 待决策落定）

- **参与方**：业主（「先把方案给 arb，仍然回来处理 arbcn 中同样的这个问题」——同型问题 = arb audit.pb / arbcn facts.md 快照，均为无界增长）、Claude（诊断 + 定案 + 施工 + 部署）
- **议题**：facts.md 快照无界增长（STATE 待决策，发现时 1564 行/40 份 → 施工前已 **3404 行/90 份**）。根因：exporter 每次导出（boot + 24h 定时 + **每次规则 armed→active**）追加整份新快照、旧快照标「已过期」不删除 → **T2 级时间序列历史写进 T1 事实库**；规则触发是主力（90 份/约 4 天 ≈ 20 次/天）。快照是 bulk 镜像（每次 ~30 行高度重复），与手工事实行性质不同；全库无代码读 facts.md（唯一读者 = 人 + git）。
- **决策（D-066）**：①**快照段封顶 `maxSnapshots=5`**——段内只留最近 5 份（现行 + 4 份已过期），最旧整体移除；**历史由 git 保留**（P2「历史机械滚出活跃层（git 保留，token 不付）」，facts.md 每次 check-out 提交即归档），**不另写 LOG.md**（P3：同一事实两份）；②**规则触发导出节流 `ruleTriggerThrottle=10min`**——距上次成功导出 <10min 的规则触发合并跳过（boot + 24h 定时不受节流）；③**节头稳定化**——「## 监控快照」节头重置到段顶（历史演进中被新快照顶到段中部，结构漂移顺手修正）；④**D-028「已过期不删除」规则适用范围修订**——手工事实行（少量、带 D# 锚点）保留「不删除」，机器快照段（bulk 镜像）改封顶淘汰。
- **施工**：`exporter.go`（maxSnapshots/ruleTriggerThrottle 常量 + Run 节流判断 + writeSection 封顶&节头归一 + stripSectionHeader/truncateSnapshots 纯函数 + sectionHeader 文案）+ `exporter_test.go`（既有 6 测试语义不变；新增对抗 ×2：**删 truncateSnapshots → 第 6 份快照残留必红** / **删节流判断 → 5min 内二次触发即出第二快照必红**，均已实测红）。§10 全绿（build/vet/全量 test 含 pgstore 真库）。
- **部署实测**：go build → systemctl restart → healthz ok；**boot 导出自愈：facts.md 90 份/3404 行 → 5 份/268 行**，手工事实段 5 节 + 7 行事实逐字保留、节头位于段顶、endMarker 归一独立成行。
- **决策号**：D-066。

## #91 · 2026-08-17 · 跨项目复盘：arb 是否需加模拟域？（业主问，judgment 修正早先引进建议）

- **参与方**：业主（「arb 项目已经进了真金阶段了，还有必要加一个模拟域吗？」）、Claude（判断）
- **议题**：早先 arbcn→arb 引进建议把「模拟执行域 + 盈利验证先行（D-058）」列为第一优先；业主质疑——arb 已真金，加模拟域是否必要。
- **决策（Claude 唯一决策者判断）**：**arb 不建完整模拟域**。理由：①模拟域在 arbcn 的核心价值 = 真金前「盈利验证先行」（D-058），arb **已跨过这道门**（执行机制已用真钱验证）——搬整个 paper 域解决的是已不存在的问题；②arb 有比模拟更好的东西：**真实成交校准数据**（D-096 order_leg_fill 落真金滑点/swap/commission，17-audit 归因闭环首次可执行）——校准评估器用现实而非模拟假设；③真正值得引的是**窄机制**：回放证伪门禁（arbcn D-065，新策略/配置启用前回放历史高费率窗口防真金误操作，读现有数据成本低）+ 判定门① 前向验证（只对新策略，走小步真金冒烟 D-058 阶段 A ~1万，非全量模拟域）；④成本收益：完整模拟域 = arbcn 一串 D#（D-055/056/065 等），对已有真实校准数据的真金系统不划算。
- **决策号**：无 arb 侧 D#（判断修正早先引进建议优先级；如需落 arb 文档另行登记）。

## #92 · 2026-08-17 · 跨项目审计纠偏 + arb D-097 引进方案落档（业主指令「先重新审计确认真优，再写方案文档」）

- **参与方**：业主（「等等，我觉得重新审计一下，arbcn 优过 arb 的地方，确认真的优过后，再帮 arb 写方案文档」——刹停早先"直接落档"冲动）、Claude（三路并行审计 + arb 侧落档）
- **议题**：早先建议「arbcn 前端更好更简洁，迁移给 arb + 还有可以引进给 arb 的两项」→ 业主要求先审计确认真优再写文档。
- **审计方法**（arb practices §8 已落「跨项目借鉴纪律」）：三路并行 agent（领域机制 / 前端 / 工程）各走 ①证据 file:line ②对抗核查（arb 是否已有等价物）③反向核查（arb 实际强在哪）。**结果证伪了多数"想当然的优"**——arbcn 前端"整体更好"只在前端通用件 + 设计原则层面成立，工程整体质量 arb 更强。
- **审计确认真优 14 项 → arb D-097**：领域机制 6（回放证伪门禁窄版 / swap 方向快照+翻转检测 / 经验库签名字典 / 瞬时 vs 30 日均值标注 / TWR-MWR 测量原则 / 7d 费率窗口）+ 前端 5（Collapse 受控 details / Chip 语义 tone 徽标 / scroll-cap 高度封顶 / 第一眼信息架构原则 / 375px 真机测试纪律）+ 工程 3（pre-commit「业务代码必须随 STATE.md 更新」门禁 / 诚实标注制度化 / 交接负载方向校验字段）。
- **明确不引 6 类**（反向核查证伪）：WS-SSE（arb constraints 硬禁 WebSocket）、现金账本（broker 权威账本）、动态资本路由、毕业门禁框架、完整 UI 迁移（引窄机制）、零密钥（arb 必须持 broker 凭证，D-097 记录）。
- **arbcn 侧自省**（审计副产品，arbcn 自身短板，指向 arb D-097 证据）：arb 的 react-table 引擎 / i18n / Zustand 薄 store / CI / migration 自愈双轨 / tech-debt 台账 / govulncheck / Playwright 均强于 arbcn；arbcn 前端 hooks.ts/style.css monolith（vs arb 薄 CSS）是待偿债。不另行重复落 arbcn 文档（P3 单一真相源，指向 arb 侧）。
- **落档**：arb decisions.md **D-097**（综合方案 A 机制①-⑥ / B 前端五件 / C 工程三项 / 明确不引 6 类 / 优先级 A2→B/C→A1→A3-A6）+ arb STATE（最后更新 / 当前施工 / 阻塞待决策 三处）+ arb practices §8。arb commit **b99106e**，未 push（arb 本地领先 origin 183 commit 是其自身节奏，D-096 亦未推，不擅推）。
- **决策号**：arb D-097（arbcn 侧无 D#）。每项施工另立独立 D#（A2→D-098 起），**待业主授权逐项施工**。

## #93 · 2026-08-17 · 补 arbcn 自身缺失（前端 monolith 拆分 + CI，D-067）+ 机会核实（当前无机会）

- **参与方**：业主（「回到本项目 arbcn，先补上本项目的缺失，然后确认一下，arbcn 目前都没有机会吧？」）、Claude（施工 + 核实）
- **议题**：业主指令返回 arbcn 补缺。缺失 = 对话 #92 审计确认的 arbcn 自身短板（D-097 副产品）：前端 monolith（hooks.ts 450 行卡 check-lines 硬线 + style.css 1248 行无门禁）+ 无 CI。范围经 AskUserQuestion 定夺 = **全拆（hooks + CSS + CI）**。
- **施工（D-067，A→B→C 三工作流，本地复刻 CI 全绿）**：**A hooks.ts → hooks/ 8 文件**（6 hook 零依赖整块搬 + shared + barrel，消费方 import 路径不变）；**B style.css → styles/ 12 文件**（按原 1248 行顺序切分，main.tsx 按原顺序 import，**产物 CSS md5 与拆分前锚点逐字节一致 `23ea63ef2a9cde1176bb5fe9aeb54a53` = 零视觉回归锚点**）；**C .github/workflows/ci.yml**（照搬 arb 骨架 + arbcn 适配：npm build 在 go build 前 / go test -race / check-lines，首版不加 govulncheck + postgres）。**CI 复刻暴露 2 个既有测试 data race**（TestRunThrottlesRuleTrigger/TestRunExportsOnTrigger——测试在 Run goroutine 执行期间直接写 x.Now 字段，`go test` 不带 -race 全绿、`-race` 必红）→ atomic.Value 时钟闭包修复。**部署闭环**：go build → systemctl restart arbcn-monitor（sudo）→ healthz ok → served bundle 引用新 hash（index-CuV4QYt8.css == dist 锚点）+ 全量 go test -race 绿。
- **机会核实（全部 live RPC，三重证据）**：① **ListOppCards 13 张卡片全部 breakeven/trap，无 trade**（carry_asset ×5 最高 SUSDE 4.39% < 4.5% base；funding_hedge ×8 其中 3 张 trap 负值、最高 okx-BTC net 3.08% < 15%；repo ×2 breakeven）② **GetReplayState 三策略均 no_window**（funding 4598 样本 max 10.95% < 15% 档，与 D-065 门禁休眠正确输出一致）③ **GetSimAccount equity=100k 空仓**。**Nuance**：ListFundingWindowStats 7d overall = tradable（93% positive share，mean 5.29%）——**环境可行但当前无具体猎物**（「有场子但没猎物」：费率结构支持套利、但当下没有 ≥ 门槛的窗口档；阶段 0 判定门① 继续 env_no_window 是正确输出，非策略失败，D-061② 环境-策略分离）。
- **决策号**：arbcn **D-067**（一个 D# 三个工作流：A hooks / B style.css / C CI，分别 commit，各可独立验收）。
