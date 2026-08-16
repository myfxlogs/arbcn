# STATE — 当前工作状态 + 交接负载

> 开工读 / 收工写。滑动窗口（AGENTS.md §4）：只存进行中 + 近期完成 + 交接负载；最旧 ✅ 移 LOG.md。

## 交接负载

- 现状: **M1/M2-a 追溯深审 + M3-a 施工复审全部闭环已部署**（D-035）；**M3 文档审计完成**（D-036：收敛口径修正 + G1–G5 全落 spec）；**M3-b 细化设计定稿 + 施工交付 + 复审修复 + S4 数据源修正部署验证**（D-037 + D-031 实证修订 + 本 commit：spec §9 S1–S5 全落地并已部署实测）。S1 规则→Signal 驱动（rule.OnActive 携带命中实体 + sim.Driver §3.1.1 映射表，删映射必红）；S2 8h 结算（(symbol,venue) 分组取真实市场 funding，跨 venue 污染必红）；S3 testnet 只读探针 + key 隔离（新包 internal/simtestnet，SIMULATED 缺标记拒绝加载，sim 包零网络由 domains_test 把关）；S4 历史收敛（exchange 历史 collector + 幂等回填 + sim_report 周频统计）——**数据源已修正并部署验证**（fapi.binance.com 满 365d + OKX funding-rate-history ~90d，见 D-031 修订；DB 实测 binance min_ts=2025-08-15）；S5 carry 白名单默认空（宁缺毋滥）。main.go 已接线：boot 历史回填（一次性幂等）+ simDriver（OnActive compose）+ 8h 结算循环 + testnet 探针随 settle tick；H1 洪水去重（8h 桶折叠 + Limit 2M）已含。**M3-c 细化设计定稿 + 施工交付 + 决策层复审修复**（D-038 + spec §10 C1–C5 全落地 5fd8f63；决策层复审发现并修复 repo/carry 二次门禁**恒拒**设计缺口——D-039 kind 分派数据面 + 7 对抗测试，spec §10.3/§10.4 同步）。**D-040 测试网账户区已闭环部署**（业主提问触发，对话 #49）：探针 Run 返回余额快照 → sim_testnet_accounts（migration 0006）→ GetTestnetAccounts RPC → SimExec 账户区；启动即探针一次（不等 8h tick）+ 8h tick 刷新；equity_usd 诚实口径（okx totalEq 精确 / binance 稳定币合计近似）；部署实测两路真实虚拟资金 + **sim_testnet heartbeat 首次登记已随启动探针闭环**（此前的观察项已消除）。**加密交易所出入金通道业主实测通过**（对话 #50）；TRX funding 跨所差异已核实为真实市场分歧非 bug（binance +2.3%~+2.7% 年化 vs okx −3.5%~−3.8%，实测量价一致，负 funding 不触发建单，宁缺毋滥）。**机会盘点（对话 #52）**：面板无机会 = 正确（BTC/ETH funding 4–7% 年化远低 15/20% 门槛，funding 窗口档当前不存在）；真实可交易对冲 = BTC cash-and-carry ~7%（现货+永续空，delta 中性）+ sUSDe 4.44% + 稳定币 3.3–4.2%；全市场扫描实证极端 funding 全在被砸微盘币（ACE −1495% 等，陷阱非机会，验证宁缺毋滥）；**架构确认维持轮询**（对话 #51，8h 结算尺度下事件驱动收益≈0）。**出入金通道全部闭环**：加密交易所（#50）+ **OTC 法币通道 1 万小额业主实测通过**（对话 #53）。**模拟执行演练档已部署**（D-041：funding_drill band [5%,15%)，okx BTC avg_30d=6.66% 重启即触发演练单，供确认→成交→8h 结算端到端冒烟，对话 #55）。**演练单拒单根因已修复**（D-042，对话 #56）：D-041 演练单连续拒单（UNHEDGED，ref 负值）→ 根因 = `pgstore.LatestFacts` WHERE 子句缺括号的 SQL 运算符优先级 bug（多参数查询退化为"只生效符号条件"，fundingHedgeSignal 取到 funding 负值当 ticker 价）；修复每子句加括号 + 对抗测试 TestLatestFactsFilters；另加固引擎 boot 竞态（rule.Config.BootDelay=15s + TestRunBootDelay）+ 修正 migrate 陈旧断言（want 5→6）。部署实测：**sim_orders id=3 = suggested（okx BTC ref 63063.30 spread 6.64% risk_flags={}）**——演练单可确认→成交→8h 结算全链路闭环；拒单 id=1/2 保留为负样本。**SimExec 三处体验修复已部署实测**（对话 #57 后续，业主反馈）：① 拒单提示条加手动 × 关闭入口；② 风险标记 7 个 Risk* 全量补中文徽标（UNHEDGED→未对冲 / SPREAD_LOW→价差过低 / SIZE_OVER→单笔超限 / DAILY_OVER→日额超限 / INVALID_INPUT→输入无效，SPREAD_DRIFT→漂移 / WHITELIST→未白名单保留）；③ 持仓补实时数值（proto +14 cur_price / +15 expected_ann / +16 unrealized_pnl + ListSimPositions 查 ticker/funding 计算 + PositionZone 加 4 列开仓价/当前价/预期年化/实时收益=已结算+未实现）。实测：永续空腿 expected_ann=5.53%（当前 okx funding 年化）、unrealized=+286k，现货多腿 unrealized=−286k，**两腿对冲相消=0（delta 中性 ✓，D-019 不赌原则）**；对抗测试 TestListSimPositionsRealtime 删浮动计算必红（已实测）。sim_orders id=4 拒单 SPREAD_DRIFT 20.08% 为设计内 fail-closed（生成 avg_30d 6.63% vs 确认时刻单点 funding 回落）。**确认下单面板移入监控总览页**（对话 #59，业主反馈）：新组件 ConfirmPanel（告警流卡片下方、触发器上方整宽卡片）只显示「待确认」订单 + 确认按钮（业主决策"简化一些"）；模拟执行 tab 瘦身为持仓/账户/报告（OrderZone 移除）；useSim 提升 App 层共享（ConfirmPanel + SimExec 同源，确认后两处同刷新）；共享展示辅助抽 sim.tsx（SIMULATED 徽标 + kind/side 中文映射，锚点 TestSimExecBadgeRenderable 扩展检查 sim.tsx 定义 + 两面板引用）。已部署实测（新 hash index-DxLmi9bv.js，ConfirmPanel 内容入 bundle）；**端到端验证通过**：re-arm funding_drill（rule_id=223）→ 触发 sim_orders id=5 suggested（ref 63031.8 spread 6.606%）→ ListSimOrders 正常返回（ConfirmPanel 数据源）→ ConfirmSimOrder accepted=true（二次门禁通过，年化变化 16.3%<20%）→ 持仓新增两腿（short expectedAnn=5.98%/long 0）+ 旧腿实时值正确，对冲相消=0。**确认下单卡片布局再调整已部署应用**（对话 #60，业主反馈）：ConfirmPanel 从告警流下方整宽卡改为**与机会面板同行**（`.row-col` 右列 = 告警流上 + 确认下单下，机会面板 `grid-row: span 2` 左列跨两行）；部署应用（新 hash index-DdoEXWry.js served 匹配磁盘 dist，row-col/确认下单 入 bundle）+ 全量 commit 推送 GitHub。**交付闭环教训入 practices #19**（§7.3 自审 ≠ 交付终点：运行态改动 check-out 必须「构建 → 部署应用 → 推送」三步闭环，本地 commit 不代表上线）。**机会面板 defi 矩阵横向滑块已修**（对话 #61，业主提问触发）：根因 = 5 个 DeFi 协议名当矩阵列头（14~16 字符 nowrap）撑宽；修法 MatrixTable 加 colLabel + Opportunity venueLabel 映射（Aave/BlackRock/Ethena/Morpho/Ondo，practices #18 全集对照）；部署应用（index-CJDJqr2V.js served 匹配）。**进化建议引擎 L0 已部署应用**（D-044，对话 #62）：总览页触发器下方新增「进化建议」卡（reject_dist 拒单分布 / defi_anomaly 利率尖峰 / no_order 连续无单 / source_down·stale 停更四信号，中位数×2.0 因子检测，只读证据表面永不自动改规则，action 一律走 D# 人工决策）；部署实测 reject_dist（UNHEDGED ×2/SPREAD_DRIFT ×1）+ source_stale ×3 正确出现，no_order 被近 7 天 filled 单正确抑制，defi_anomaly 因 Aave 12.57% 尖峰已回落至 3.29%（最新截面平稳）正确不报。**carry + repo 完整接入模拟盘已部署应用**（D-045，对话 #63）：结算数据面按腿 kind 分派（settleFactKind：funding_hedge→funding / carry_asset→defi_rate / repo→reverse_repo，settleOnce 按 (kind,symbol,venue) 分组 + SettleFunding 带 kind 维）——此前 settleOnce 只查 funding 事实，carry/repo 腿"建了仓也永不生息"；repoSignal 落单 venue/symbol 取事实真实值（sina/GC001，不再硬编码 domestic/GC001）；carry 独立低门槛 CarryMinSpread（默认 1%，纠正 funding 摩擦假设误用于持有生息的口径错配，非放宽门禁造数据——repo 5% 时点意图/funding 15% 不变）；生产 env 新建 /etc/arbcn/arbcn-monitor.env（此前 systemd `EnvironmentFile=-` 指向但文件缺失 = 全默认运行）注入 `ARBCN_SIM_CARRY_WHITELIST=SUSDE,USDE,BUIDL,STEAKUSDC,USDY`。对抗测试 3 个新锚点（删分派/改回硬编码/删分档必红已实证）+ config_test 扩展；全量测试/vet/-race 绿；部署重启实测 active + 新二进制 inode 匹配 + served bundle 一致。**诚实标注**：carry 单是否真触发取决于 defi 池出现 ≥0.5%/h 变动（不造数据）；repo 当前 0.865% < 5% 会被 SPREAD_LOW 拒（负样本，符合时点逆回购意图）。
- 方向校验: 与 AGENTS.md §1 一致 —— 不赌原则（D-019）+ 收益最大×路径最短（D-020）+ 加密三档（D-021）+ 敞口知情（D-023）+ 无密钥铁律（D-010/§13，M3-a 复审复核：sim 包零网络零密钥）。**进化建议引擎 L0（D-044）守住边界**：只读证据表面，永不自动改规则（决策在环，D-019/D-043 同源）。**D-045 carry+repo 接入守住口径**：carry/repo 均 D-021 已定义档位（非新方向）；结算仍用真实市场公开事实（非 testnet，D-037 同口径）；CarryMinSpread=1% 是纠正 funding 摩擦假设误用于持有生息的口径错配，非为数据放宽门禁。
- 施工表:
  | 子任务 | 状态 | 锚点 |
  |--------|------|------|
  | 脚手架 + 协议 + 门禁 | ✅ | 5bea136 / D-001~D-005 |
  | charter 正式化 ~ 不赌原则 | ✅ | D-008~D-019 |
  | 优先级反转：监控先行 | ✅ | D-020 |
  | 加密三档结构 | ✅ | D-021 |
  | 国内 vs 加密对比 + 敞口知情 | ✅ | D-023 |
  | 方案再审计：动态路由 + 阶梯三档 + 出入金设计 | ✅ | D-024 |
  | 保本凸性档（CPPI 式） | ✅ | D-025 |
  | 档位定稿（保本凸性 50% + 稳定币 50%） | ✅ | D-026 |
  | 第 3 轮审计：期权动态预算 + 自托管腿 | ✅ | D-027 |
  | 市场事实知识库 facts.md + 数据闭环 | ✅ | D-028 |
  | 监控系统架构设计 | ✅ | D-029（02-monitor-architecture.md） |
  | Go 裁决 + 客户端形态 + 三问确认 | ✅ | D-030 |
  | **M1-a 脚手架 → 施工 agent #1** | ✅ | e999f8e |
  | **M1-b Fact 模型+存储层 → 施工 agent #2** | ✅ | e04aa4b |
  | **M1-c Exchange collector → 施工 agent #3** | ✅ | c1accb2 |
  | **M1-d DeFiRates/Domestic/Calendar/IV → 施工 agent #4** | ✅ | c8b912d |
  | **M1-e 规则引擎+对抗测试 → 施工 agent #5** | ✅ | 505d2af |
  | **M1-f Alerter+元监控 → 施工 agent #6** | ✅ | 88dbf9e |
  | **M1-g Web 仪表盘最小集 → 施工 agent #7** | ✅ | bb8c399 |
  | **M1-h 全链路联调+A-F 自审 → 施工 agent #8** | ✅ | 6cefbea |
  | **M1 验收（决策层）** | ✅ | dialogue #29 |
  | **M1-i SMTP 降级补丁 → 施工 agent #9** | ✅ | 0d740b7 |
  | 部署 systemd 常驻（mluser 运行） | ✅ | 2026-08-15 实测 active |
  | 加密交易所出入金通道验证 | ✅ | 业主实测通过（2026-08-16，dialogue #50） |
  | 出入金通道验证（1 万小额 OTC 法币通道） | ✅ | 业主实测通过（2026-08-16，dialogue #53） |
  | **M2-a 后端：3 RPC（ListUnacked/AckAll/ListSourceHealth）+ 调度去重 → 施工 agent #1** | ✅ | e26eea9 |
  | **M2-a 前端：铃铛通知中心 + freshness 徽标 → 施工 agent #2** | ✅ | 2afac33 |
  | **i18n：页面硬编码英文 → 中文（前后端一致 ruleLabel/消息模板）** | ✅ | 34ae607 |
  | **M2-b：RMB 折算 + facts.md 自动导出 + 台账 → 施工 agent** | ✅ | 复审修 F1/F2 |
  | **M1/M2-a 追溯深审（R1-R6 六路 review + 修复 14 项）** | ✅ | D-035 |
  | **M3-a：订单生成器 + 本地模拟盘回填** | ✅ | D-034/D-035 |
  | **M3-a 复审修复：H1 结算 100× / M1 成交原子 / M3 NaN 门禁 / L1-L3** | ✅ | D-035 |
  | **M3-b S1：规则→Signal 驱动接线（rule.OnActive 携带命中实体 + sim.Driver 映射表）** | ✅ | sim/driver.go + driver_test.go（删映射必红） |
  | **M3-b S2：8h 结算调度（(symbol,venue) 分组取真实 funding）** | ✅ | sim/driver.go settleLoop + TestSettleByVenue |
  | **M3-b S3：testnet 只读探针 + key 隔离（新包 simtestnet，SIMULATED 门控）** | ✅ | internal/simtestnet/ + domains_test |
  | **M3-b S4：历史收敛（exchange 历史 collector + 幂等回填 + sim_report 周频）** | ✅ | collect/exchange/history.go + sim/report.go（部署验证：binance 满 365d / OKX ~90d，D-031 修订） |
  | **M3-b S5：carry 白名单 + 降级（默认空宁缺毋滥）** | ✅ | sim/config.go CarryWhitelist |
  | **M3-c 细化设计定稿** | ✅ | D-038 + spec §10（C1–C5，含 proto 全文/RPC/门禁/锚点） |
  | **M3-c C1：SimService proto + 生成物** | ✅ | proto/arbcn/sim/v1/sim.proto + buf.gen.sim.yaml（独立域；dashboardv1 生成物 byte-identical 未动） |
  | **M3-c C2：SPREAD_DRIFT 二次门禁** | ✅ | sim/order.go RiskSpreadDrift + sim/confirm.go ConfirmDriftCheck（G5 口径 + fail-closed） |
  | **M3-c C3：确认成交流（ConfirmSimOrder 唯一写路径）** | ✅ | simapi/service.go + pgstore Accept/RejectSimOrder（suggested 守卫原子） |
  | **M3-c C4：模拟执行 UI tab** | ✅ | App.tsx 第 4 tab + SimExec.tsx（SIMULATED 徽标 + 即期 RMB） |
  | **M3-c C5：可检查性 + main.go 接线 + 验收** | ✅ | domains_test simapi grep 断言 + mux 接线 + go vet/test -race |
  | **M3-c 复审修复：repo/carry 二次门禁恒拒（D-039 kind 分派）** | ✅ | D-039 + spec §10.3/§10.4 + simapi/confirmDrift + 7 对抗测试（删分派必红） |
  | **M3-c 部署验收（spec §10.6）** | ✅ | 343f6a6 部署实测：重启 healthz ok + SimService 2 RPC 200 + dashboard 零回归（ListFacts 真实数据）+ 前端 index 200；确认流单测 8 场景全覆盖（真实 suggested 订单端到端冒烟依赖规则命中，库空时以单测+连通冒烟替代，诚实标注） |
  | **S3 testnet 探针启用（业主 key）** | ✅ | /etc/arbcn/arbcn-sim.env（mluser:mluser 0600，SIMULATED+全量 key）；binance/okx 连通 curl 实测 200；OKX ts ISO 格式探针 bug 修复（50102，practices #12）；首次 heartbeat 待 8h tick |
  | **D-040 测试网账户区（探针余额持久化 + RPC + UI）** | ✅ | migration 0006 sim_testnet_accounts + store 两方法 + probe Run 返回快照（binance 稳定币近似/okx totalEq）+ GetTestnetAccounts RPC + main 启动探针持久化 + SimExec 账户区（诚实口径标注）；部署实测两路真实虚拟资金 + 启动探针 heartbeat 登记（对话 #49） |
  | 告警流高度封顶修复（触发器脱节） | ✅ | web/src/style.css .timeline max-height:min(60vh,480px)+overflow-y:auto；部署实测（对话 #54） |
  | **D-041 模拟盘 funding_drill 演练档** | ✅ | defaults.go 规则（band avg_30d>5&&<15, Info）+ driver 映射 + 对抗测试 + spec §3.1.1；okx BTC avg_30d=6.66% 部署后即触发（对话 #55） |
  | **D-042 演练单拒单根因修复（LatestFacts SQL 优先级 + boot 竞态加固）** | ✅ | dashboard.go 括号修复 + TestLatestFactsFilters（删括号必红）+ Config.BootDelay=15s + TestRunBootDelay（删 sleep 必红）+ migrate 陈旧断言修正；部署实测 sim_orders id=3=suggested（对话 #56） |
  | **SimExec 三处体验修复（提示条关闭 / 风险标记中文 / 持仓实时数值）** | ✅ | SimExec.tsx riskLabel 7 全量中文 + banner-close 关闭按钮 + PositionZone 4 列；proto +14/+15/+16 + codegen + ListSimPositions 实时字段 + TestListSimPositionsRealtime（删计算必红）；部署实测对冲相消=0（对话 #57 后续） |
  | **确认下单面板移入总览页（ConfirmPanel + sim.tsx 共享 + useSim 提升）** | ✅ | ConfirmPanel.tsx（待确认订单+确认按钮，简化 6 列）+ sim.tsx（徽标/文案共享）+ App.tsx 布局（告警流下方）+ SimExec 瘦身（props）+ TestSimExecBadgeRenderable 扩展（sim.tsx 定义 + 两面板引用）；部署实测 hash 匹配 + 内容入 bundle（对话 #59） |
  | **确认下单卡片布局调整（与机会面板同行，.row-col 右列）** | ✅ | App.tsx 双栏改造（机会面板 grid-row span 2 + `.row-col` = 告警流上/确认下单下）+ style.css .row-col flex；部署应用（index-DdoEXWry.js served 匹配）+ 推送 GitHub（对话 #60） |
  | **进化建议引擎 L0（D-044：四信号只读证据表面）** | ✅ | proto Insight/ListInsights + buf generate + `internal/dashboard/insights.go`（rejectDistribution/defiAnomalies/noOrderHint/median 纯函数 + ListInsights RPC 编排，中位数×2.0 因子、NaN 守卫 practices #7）+ insights_test.go 对抗（删判定/删计数必红已实证）+ fakeStore ListSimOrders 真语义 + 前端「进化建议」卡（触发器下方整宽卡，severity 色 + 类别中文 + actions 走 D#）；部署实测（index-CtjddyB-.js served 匹配，reject_dist UNHEDGED ×2/SPREAD_DRIFT ×1 + source_stale ×3 正确，no_order 被近 7 天 filled 单抑制，defi_anomaly 因最新截面平稳正确不报） |
  | **D-045 carry + repo 完整接入模拟盘（结算分派 + 门槛分档 + venue 对齐）** | ✅ | driver.go `settleFactKind`（funding→funding/carry→defi_rate/repo→reverse_repo）+ settleOnce 按 (kind,symbol,venue) 分组分派 + backfill.go SettleFunding 带 kind + config.go CarryMinSpread（默认 1%/env/NaN 拒载）+ order.go carry 门槛分档 + repoSignal venue/symbol 取事实真实值 + 生产 env 白名单（SUSDE,USDE,BUIDL,STEAKUSDC,USDY）；对抗测试 TestSettleDispatchByKind/TestSettleRepoUsesRealVenue/TestCarryUsesCarryMinSpread（删分派/改回硬编码/删分档必红已实证）+ config_test 扩展 + TestDriverRepoBuildsOrder 事实带真实 venue；部署实测（新二进制 inode 匹配 + 启动干净 + bundle 一致，对话 #63） |
- 阻塞/待决策:
  - 无阻塞。TRX 独立处置（业主自定，费率转正触发器已入监控规格）。
  - SMTP 授权码待办**已移除**（D-033：业主不做邮件推送，浏览器铃铛为主通道）。
  - R3-L3 遗留：calendar collector 的 thursday 周记语义待业主决策（低危，接受待决）。
  - **M3-a 已验收（D-035）**；**M3-b 已施工交付**（D-037 + 本 commit，S1–S5 全落地，见施工表）。
  - **S3 已启用**（业主 key 到位，dialogue #48）：/etc/arbcn/arbcn-sim.env（mluser:mluser 0600）；binance/okx 连通实测 200；探针 8h tick 跑，**首次 heartbeat 登记待下一个 tick**（ListSourceHealth 应见 sim_testnet_binance/okx）。服务以 mluser 运行（key 文件属主为 mluser 非 root，运行环境实测修正）。
  - **M3-b 交付注**：sim 已接线进 main.go（backfill / simDriver / OnActive compose / 8h 结算循环 / 探针随 tick）；结算数据源 = 真实市场公开 funding（D-037 裁决），testnet 费率不参与结算只做 key 隔离验证。sim 包保持零网络零密钥（domains_test + TestNoNetworkImports 把关）。
  - **M3-c 交付注**：SimService 独立域已接线 main.go（st 非 nil 即挂载，sim 配置缺失降级：GetSimReport 返回未启用说明，D-032 同口径）。**repo/carry 二次门禁恒拒已由 D-039 裁决修复**：数据面按 kind 分派权威源（repo→reverse_repo 利率、carry→defi_rate 年化、funding_hedge→ticker/funding），权威源查不到仍 fail-closed 拒（不放宽），ConfirmDriftCheck 签名不变。
  - **D-031 实证修订**：data-api.binance.vision 不镜像 /fapi/*（404）；历史回填源回落 fapi.binance.com（部署机直连 200、满 365d）；OKX 历史端点为 funding-rate-history（funding-history 404），仅保留 ~90d（OKX 部分覆盖，sim_report/avg_30d 受窗口限制，已知 degrade）。
  - 待决策观察（不阻塞）：repo 信号经 SignalToOrder 时仍受 5% 门槛（SPREAD_LOW）约束——平时逆回购 2-4% 会被拒单，仅季末/年末上冲 ≥5% 时放行；与"时点逆回购"策略意图一致（宁缺毋滥），但若业主希望 repo 绕过价差门槛须走 D# 调整。D-045 已确认 repo 保持 5%（意图不变）；carry 改用独立 CarryMinSpread=1% 已生效。
  - D-045 交付注：carry 白名单已生产配置（SUSDE/USDE/BUIDL/STEAKUSDC/USDY 大写，config.go 不做大小写归一）；carry 单真正触发依赖 defi_large_tier_change 命中白名单标的（≥0.5%/h 变动），不造数据；repo 因当前 0.865%<5% 仍 SPREAD_LOW 拒（负样本，设计内）。
- 下一步: **carry + repo 已完整接入模拟盘**（D-045，对话 #63）：结算按 kind 分派数据面 + carry 白名单生产配置 + CarryMinSpread=1% + repo venue 对齐，已部署应用。下一可行动作 = 观察 defi_large_tier_change 是否命中白名单标的（SUSDE/USDE/BUIDL/STEAKUSDC/USDY 之一出现 ≥0.5%/h 变动）→ 触发 carry suggested 单 → 可确认→成交→按 defi_rate 8h 结算全链路；同时业主可在监控总览页进化建议卡查看 L0 证据候选（reject_dist/source_stale 当前可见）并按候选决定是否走 D# 调整。L1 候选（数据累积后）：窗口内 max 冲高检测 / 阈值自适应统计自校准 / 拒单-阈值联动归因。
- 清扫上翻: 本次 review 教训入 practices.md #6-#10（刻度统一/NaN 门禁/状态+从属行原子/信任边界标注/时钟注入覆盖全路径）+ #11（统计效力）+ #12（数据源端点必须部署机实测）+ **#13（门禁/折算数据面按实体类型分派，D-039 教训）**；**L0 进化建议引擎入 decisions.md D-044**（四信号只读证据表面 + dashboard 域 RPC + 按需计算 + 中位数×因子 + 动作一律走 D#；L1/L2 候选留给数据累积后）；**洞察/建议引擎边界教训入 practices #20**（只读证据表面永不自动改规则——优化方向由数据提示、优化动作由决策层拍板，"证据面"与"执行面"分离）；对话落 dialogue.md #62（半自动增强确认 + 决策层把关目标修正 + 施工 + 部署实测）；dialogue.md 轮转 #8~#53 至 LOG.md；追溯深审 + M3-a 复审结论入 decisions.md D-035；M3 文档审计结论入 D-036（收敛口径修正 + G1–G5）；M3-b 细化设计入 D-037（spec §9 + 结算数据源裁决）；S4 数据源实证修订入 D-031（data-api 前提否定 + fapi 回落 + OKX 端点修正）；M3-c 细化设计入 D-038（spec §10 C1–C5）；**M3-c 复审修复（repo/carry 恒拒）入 D-039**；S3 启用 + OKX 探针 ts 格式教训入 practices #12 + dialogue #48；**D-040 测试网账户区入 D-040**；跨数据源同名数值口径标注教训入 practices #14；对话落 dialogue.md #39~#55（含 #51 轮询 vs 事件驱动架构确认、#52 机会盘点 + 微盘陷阱实证、#53 OTC 法币通道实测通过、#54 告警流高度封顶修复、#55 模拟执行无机会核查 + 演练档选型）；**D-041 演练档入 decisions.md**（funding_drill band [5%,15%) 设计 + 告警 Info 级 + 复用 fundingHedgeSignal + 对抗测试锚点）；**D-042 拒单根因修复 + boot 加固入 decisions.md**（SQL 优先级 bug 根因 + TestLatestFactsFilters + BootDelay + TestRunBootDelay + migrate 修正）；**经验教训入 practices #15**（/proc/PID/exe 与磁盘文件对比须用 `stat -L`，procfs 伪 inode 曾误导"旧二进制"假象；SQL where 拼接每子句必须加括号防 AND/OR 优先级陷阱）；对话落 dialogue.md #56/#57/#58；**前端二次确认按钮 bug 教训入 practices #17**（disabled 绑 pending 会禁掉第二次点击，须绑 busy）；**SimExec 三处体验修复（提示条关闭入口 / 风险标记中文全量映射 / 持仓实时数值）入 dialogue #58**（口径：实时收益=已结算+未实现、预期收益=预期年化%、提示条手动 × 关闭；unrealized 只依已确认实时价计算，行情缺失不编造浮动）；**教训入 practices #18**（前端枚举值映射表必须对照后端常量全集，别只映射见过的——riskLabel 曾漏 5/7 个 Risk 常量显示英文原文）；**确认下单面板移入总览页入 dialogue #59**（业主决策：简化只留待确认+确认按钮；sim tab 保留持仓/账户/报告；useSim 提升 App 层共享 + sim.tsx 抽共享，锚点扩展）。**确认下单卡片布局调整（与机会面板同行）+ 交付闭环补课入 dialogue #60**（端到端验证：订单 id=5 suggested→确认成交，持仓对冲相消=0）；**教训入 practices #19**（§7.3 自审 ≠ 交付终点：运行态改动 check-out 必须「构建 → 部署应用 → 推送」三步闭环，本地 commit 不代表上线——业主"提交推送应用"触发本次补课）。**业主三问答入 dialogue #61**（defi_rate 五项 = DeFi 协议资金池非交易所、Aave USDC 12.57% 异常值、矩阵滑块根因修复、实盘操作流程、LLM 接入裁决=不建议）；**LLM 方向锚定落 D-043**（暂不接入，未来解读用模板叙述；业主认可，方向记录在案）。**D-045 carry+repo 接入入 decisions.md**（结算数据面按腿 kind 分派 + repoSignal venue 真实值 + CarryMinSpread 门槛分档 + 白名单显式配置；1% = 纠正口径错配非放宽门禁）；**教训入 practices #21**（施工前必须先对设计文档做 A–F 自审——自行施工/分发施工都须，自审是施工前置门禁不是汇报材料，不待业主追问；AGENTS.md §7.3 已加「两时点强制」条款）+ **practices #22**（结算/折算数据面按实体 kind 分派 + 落单 venue/symbol 取事实真实值勿硬编码，practices #13 结算侧延伸）；**部署发现**：/etc/arbcn/arbcn-monitor.env 此前不存在（systemd `EnvironmentFile=-` 全默认运行），本次新建仅含白名单一行；对话落 dialogue.md #63（全面模拟决策 + 施工前自审纪律确认）。
