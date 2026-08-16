# 对话纪要

> 有痕可阅。每次实质对话追加一段。超 450 行 → 最旧段移 LOG.md（留索引行）。
> 全量原文在各工具自身 session 文件中（人类可读），此处只落实质纪要。

## #1 · 2026-08-15 · 无损交接方案设计与角色确定
- **参与方**：业主（人类）、Claude
- **议题**：多 agent 协同 + 无损交接方案；角色确定；对话落文档。
- **业主需求**：① 多 agent（任何 agent）协同，需无损交接；② 换 agent 不得偏方向 / 改实现 / 违约束；③ 交接要省 token；④ 角色：Claude 第一负责人 / 唯一决策者，其他 agent 施工方，业主只提需求（可能错误）；⑤ 全部对话落文档，有迹可查、有痕可阅。
- **Claude 结论**：以六条第一性原则（P1–P6）推导交接方案 —— git 纯文本唯一共享通道 + T0/T1/T2 分层 + 私有记忆清扫仪式 + 交接负载固定格式 + 门禁化 + 对话纪要。已落 AGENTS.md / decisions.md / STATE.md。
- **决策号**：D-001~D-006。
- **待办**：业主给出 arbcn 领域需求 → charter 正式化（D-006）。

## #2 · 2026-08-15 · 业务需求接收：20万国内套利（讨论阶段）
- **参与方**：业主、Claude
- **议题**：arbcn 业务需求（故事背景）。
- **业主需求**：本金 20 万 RMB，国内，多币种配置可选，目标尽快做到"无风险套利（含套息）"，需要方案。
- **Claude 结论**：纠偏——①零售市场无"无风险套利"；②套息是带敞口交易、非套利；③国内合法无风险收益 2026 现实 1–2%；④加密资金费率套息被 2026-02 八部门 42 号文明令禁止、损失自担、法律不保护；⑤跨境套息受资本管制限制。方案 = 三层组合（应急池/无风险底仓/套利增强/机会型），合法+保本预期 ~2–3.5%/年。
- **决策号**：D-007（国内合法路径收益边界约束）。
- **待办**：业主回答风险锚点 / 合规边界 / 软件形态 → 正式化 charter + 方案文档。

## #3 · 2026-08-15 · 三锚点收敛 + 决策层裁决（charter 正式化）
- **参与方**：业主、Claude（决策层模型）
- **议题**：业主三锚点回答后的决策裁决。
- **业主选择**：①接受跨境/加密合规风险；②中风险 5–8%/年；③决策监控+记账（不自动执行）。
- **Claude 裁决**：目标 5–8%/回撤 ≤8% 锚定，但诚实按年份分布交付（差年 2.5–3.5 / 平年 3.5–5 / 好年 5.5–7）；五层组合 S0–S4 + 阶段门禁；USD 套息走境内合法通道（美元定存 3.0–3.6%、QDII 卫星），汇率逆风分 3 批建仓；加密层 D-013 门禁——当前 funding 5–11% 未达 15% 激活线，监控中不激活。
- **决策号**：D-008（charter）~ D-013（S4 门禁）。
- **待办**：业主确认 charter → Phase 0 开工（台账 + S0/S1 建仓清单）。

## #4 · 2026-08-15 · 门禁误拦与修复（D-014）
- **事件**：charter 正式化提交被 pre-commit 以"AGENTS.md 有删除行"拦截。
- **裁决**：守卫范围收窄为历史标记行（计数制）；AGENTS.md 移出守卫名单（变更控制 = decisions.md + git）。协议自证有效：门禁拦住 → 决策层修复 → 历史无损。
- **决策号**：D-014。

## #5 · 2026-08-15 · 业主质疑最优解 → 方案自审计（D-015/D-016）
- **参与方**：业主、Claude
- **议题**：业主问"方案是否为最优解、是否有优化空间"。
- **审计过程**：提出 6 项优化提案，其中 2 项被 2026 数据当场证伪——北交所打新（稳中 1 手需冻结 500 万+，20 万陪跑）与 QDII-LOF 溢价（限购 10–100 元/日 + 2026-08-07 退市新规）。审计协议 = 提案先核实再修订，证伪即弃。
- **结论**：2026 国内零售真套利通道结构性枯竭（仅剩逆回购时点 + 存款利差）。修订：S1 存单→民营定期（2.3%）；S2 25%→15% + 6.6 加仓线；S3 引擎重写 + 改革触发器；S4 门禁 20% + 首轮 10%；基线修订 2.5–3.5%（差年 2.5–3.5 / 平年 3–4.5 / 好年 4.5–6）。
- **决策号**：D-015（组合修订）、D-016（S4 门禁修订）。
- **待办**：业主确认修订后 charter → Phase 0。

## #6 · 2026-08-15 · 业主渠道偏好：银行产品为主（D-017）
- **参与方**：业主、Claude
- **议题**：业主提出"更多偏向走银行产品"。
- **业主输入**：渠道偏好——银行产品优先，交易所渠道少用。
- **Claude 裁决**：银行 80%（现金管理理财/民营定期/美元理财 3.37–4%/固收+）+ 证券账户最小化 15%（仅时点逆回购+双低转债）+ 加密门禁 5%。结构性存款不配（1.7–2.2% < 民营定期 2.15%）。基线修订 2.3–2.5%。代价已定价：上限压缩 ~1%/年；纯银行备选版上限 2.0–2.3%。
- **决策号**：D-017。

## #7 · 2026-08-15 · 外部方案（千问）评估（D-018）
- **参与方**：业主、Claude（千问方案作为外部输入）
- **议题**：业主引入千问 AI 的"20 万人民币境内美元/人民币对冲理财"方案，要求评估。
- **核实与裁决**：逐条核实 7 项主张——华夏全对冲互认基金真实（采纳入产品雷达）；民生/中信美元定存利率吻合（采纳）；工行"0.8%"错误（实际 2.8%，差 240 元非 1,840）；两得宝 R5/C5 准入门槛与业主画像冲突（拒绝）；"美联储下半年降息"前提与 2026-08 加息预期矛盾（拒绝短存逻辑）；压力测试年息套 6 个月场景夸大 ~2×；购汇用途申报建议不合规（纠正）。吸收 1 个工具，拒绝 3 项结构。
- **决策号**：D-018。

## #8 · 2026-08-15 · 千问方案完整收益重算（推敲求证）
- **参与方**：业主、Claude
- **议题**：业主指出"千问方案不止 2.5% 年化"，要求推敲求证。
- **重算结论**：修正后千问方案真实年化——升值 3%: 0.7–1.3% / 不变: 2.2–2.5% / 贬值 3%: 3.8–4.5%；概率加权期望 ~2.2%（与我方案 A ~2.3% 几乎相同）。关键修正：华夏基金预期 3–4.5%→1.5–2.5%（成立以来年化 1.45%、2024 年 −2.11%）；两得宝收益成立但 R5/C5 门槛卡死。
- **裁决路径**：本质 = 美元敞口旋钮（千问 50% 裸美元 vs 我 20%）。提供三档：20%（保守）/ 30%（折中）/ 40%（千问等效），待业主拍板后落 D-019。
- **待办**：业主定旋钮 → D-019 → Phase 0。

## #9 · 2026-08-15 · 业主定套利第一性原则"不赌"（D-019）
- **参与方**：业主、Claude
- **议题**：业主指出"套利交易的第一性原则，不应该赌"——以此裁定美元敞口旋钮。
- **裁决**：原则入 charter。重审结果：裸美元敞口（千问 50%、我方 20%）均为赌注，清零；S2 重构为华夏全对冲互认基金 15%（对冲后纯息差）；S4 加密 cash-and-carry 重新定位为"方向中性套息"（经济上无赌，门禁管合规）；S1 45%；新增"对冲成本窗口"监控项。基线 2.3–2.5% 不变，方差下降。
- **决策号**：D-019。charter 至此完整（D-008~D-019），Phase 0 开工。

## #10 · 2026-08-15 · 门禁拦截施工表压缩（协议自证 #2）
- **事件**：D-019 提交时 pre-commit 拦截——STATE.md 历史标记净减少（-15 +11）：施工表压缩时旧 ✅ 行未按 §4 滑动窗口轮转到 LOG.md。
- **修复**：被压缩行归档至 LOG.md（轮转 #1），同 commit 完成。门禁按设计拦住协议作者本人——轮转规则（AGENTS.md §4）必须机械执行，不能图省事删行。
- **决策号**：无新增（协议既有规则执行）。

## #11 · 2026-08-15 · 优先级反转：监控先行 + 加密为主（D-020~D-022）
- **参与方**：业主、Claude
- **议题**：业主指出核心是信息监控收集、及时套利机会发现；"80% 走合规加密可行"；已持有 TRX 质押（~5 TRX/天）；优化原则 = 收益最大 × 路径最短。
- **核实**：TRX 链上基础 3.2%，组合优化 6–10%；稳定币头条 8–10% 为小额诱饵档，大额真实 1.6–2.2%，正确产品（定期档/DeFi）4.5–6%。
- **裁决**：系统交付反转（监控 v1 为第一交付，只读公开 API 无密钥）；加密三档（稳定币 60% + 门禁套利 15% + 国内底仓 25%），新基线 3–4%、好年 4.5–6%；出入金 1 万小额先行验证；TRX 持有层单列并纳入监控优化。
- **决策号**：D-020（优先级反转）、D-021（加密三档）、D-022（监控 v1 规格）。
- **下一步**：Claude 出监控 v1 技术规格 → 施工 agent 实现；业主执行出入金通道验证。

## #12 · 2026-08-15 · 国内 vs 加密 收益风险对比（D-023）
- **参与方**：业主、Claude
- **议题**：业主要求"国内产品 VS 加密产品，收益 VS 风险"对比。
- **核心发现**：加密收益全为 USD/币计价——稳定币 4.5–6% 折算人民币后（2026 升值 3%）净 1.5–3%，与国内打平；加密方向 = 重开美元旋钮至 60–75%。业主已有 TRX ≈ 总资产 20–30% 币敞口。
- **裁决**：知情敞口条款 + 推荐混合 60/40（名义 3.1%，三情景 1.8–4.5%）+ 监控新增 RMB 计价净收益折算。75/25 为业主可选收益最大化档。
- **决策号**：D-023。
- **待办**：业主选 60/40 或 75/25 → 监控 v1 规格施工。

## #13 · 2026-08-15 · 方向澄清 + 方案再审计（D-024）
- **参与方**：业主、Claude
- **议题**：业主澄清 TRX 仅为持仓例子非方向（降格为现有资产优化项）；要求找更多更优方向 + 再审计最优性。
- **核实**：Babylon BTC 质押证伪（基准 0.03–1%，BABY 代币 −92%）；Launchpool 2026 Q1 实测 BNB 池 19–22%、稳定币池份额仅 10–15%。
- **审计结论**：静态比例非最优 → 动态资本路由；收益阶梯三档（保守/均衡/进取）；出入金通道设计独立模块；托管阶梯；国内层移除华夏对冲；监控新增计价币种陷阱识别。
- **决策号**：D-024。
- **待办**：业主选风险档（保守/均衡/进取）→ 监控 v1 规格施工。

## #14 · 2026-08-15 · 保本对冲档（D-025）
- **参与方**：业主、Claude
- **议题**：业主提出"另有 1 个保本方案，对冲"。
- **裁决**：阶梯新增保本凸性档（CPPI 式：19 万无风险本金 + 利息 ~4,100 元买期权，本金 100% 保护，牛市凸性上不封顶）。变体：稳定币保本层/保本鲨鱼鳍/TRX 看跌保护。最终阶梯四档：保守 / 保本凸性 / 均衡 / 进取。
- **决策号**：D-025。
- **待办**：业主选档位（或组合）→ D-026 定稿 → 监控 v1 规格施工。

## #15 · 2026-08-15 · 资金费率机制澄清 + TRX 负费率发现
- **参与方**：业主、Claude
- **议题**：业主问"持有合约是否可收资金费"；并指出其 TRX 持仓 = 单边做多（APR + 币价上涨）。
- **核实**：资金费每 8 小时多空互换；正费率（BTC/ETH 当前 +5~11%）空收多付，负费率反之。TRX 当前**负费率**（Perpfinder ~−12% 年化，曾 −46%）——收钱的是多头，业主直觉方向恰好相反。
- **结论**：①TRX 现货+空单对冲当前要付 ~12%/年，无免费套息；②开多收 12% = 单边多上加杠杆，违反 D-019；③负费率 = 市场看空信号，现货持仓承压（APR 3.2% 可能被币价侵蚀——计价币种陷阱活案例）；④正解 = 主流币正费率 cash-and-carry（已在门禁层）+ 新增监控项**跨所资金费率差套利**（同币跨所多空对冲吃费率差，方向中性）；⑤TRX 费率转正 = 监控触发器（套息窗口 + 现货顺风信号）。
- **待办**：并入监控 v1 规格；TRX 处置（持有/减仓/等触发器）业主自定。

## #16 · 2026-08-15 · 档位定稿（D-026 决策层裁定）
- **参与方**：业主、Claude
- **议题**：业主确认 TRX 不在 20 万内；档位倾向保本凸性组合，委托 Claude 专业性裁定。
- **裁定**：保本凸性 50%（10 万，民营定期 5 万 + 灵活保本 4.5 万，利息 1,000 元季滚 BTC/ETH 看涨期权）+ 稳定币基档 50%（10 万，2–3 所定期 4.5–6%，分 3 批）+ 动态路由增强。均衡卫星推迟（脱锚与崩盘同源）。地板 +2.15%，基准 ~3.2–3.7%，牛市 8–10%+。
- **决策号**：D-026。
- **待办**：监控 v1 技术规格施工（下一项）；业主执行出入金验证（1 万小额先行）。

## #17 · 2026-08-15 · 账户现状更新（执行路径缩短）
- **参与方**：业主、Claude
- **议题**：业主披露账户现状——Binance ✓ OKX ✓ 国泰君安 ✓，富民 ✗。
- **更新**：稳定币半仓与期权腿零新增账户（Binance Earn 定期 + OKX + 加密期权）；时点逆回购走国泰君安（已有）；民营定期需富民线上开户（5 分钟，2.15%，存款保险全保，备选现有银行 1.1–1.9%）。出入金 1 万先行验证保留（10 万规模 ≠ 日常小额）。执行路径从 6 步新增成本降为"一个 5 分钟线上开户"。
- **决策号**：无新增（账户状态更新，非决策）。

## #18 · 2026-08-15 · 第 3 轮审计（D-027）
- **参与方**：业主、Claude
- **议题**：业主第三次要求审计最优性（"目前"口径），并确认方案随监控与知识库演进为常态。
- **发现**：①期权预算措辞歧义（"季滚"可读作 1,000/季，超年利息）→ 定死 1,000/年 + 动态预算（随 IV）；②自托管腿在 D-026 执行版丢失 → 补回（7 万 CEX + 3 万自托管）；③地板来源认知修正（组合地板主要来自 B 半 + A 半本金）。
- **裁决**：50/50 保持；40/60 变体供选；方案演进机制 = 路由器 + 触发器 + D# 增量（D-008~D-027 链条即演进证据）。
- **决策号**：D-027。

## #19 · 2026-08-15 · 真实数据验证需求 + 知识库补建（D-028）
- **参与方**：业主、Claude
- **议题**：业主问——方案目前是否最优，还是需接入真实数据？是否缺知识库？
- **结论**：方案架构对当前 regime 最优，执行决策需活数据（6 个数据点，已内建于监控先行）；知识体系缺事实层 → 新建 facts.md（市场事实库，含今天全部核实事实 + 时点/来源/状态 + 过期不删规则）；数据闭环 = 监控采集 → facts.md → 触发器 → 新 D#。
- **决策号**：D-028。

## #20 · 2026-08-15 · 监控系统架构设计（D-029）
- **参与方**：业主、Claude
- **议题**：程序化设计——方法/架构/语言/数据库等。
- **设计**：采集→归一→规则→告警→留痕 管线 + 决策仪表盘；Go+PG+React/TS+ConnectRPC 单二进制；五大设计决策（插件化/事实模型/声明式规则/无密钥/告警状态机）；M1→M2→M3 分期；元监控；邮件告警默认。
- **决策号**：D-029；规格 docs/design/02-monitor-architecture.md。
- **待确认**：①告警通道（邮件+Server酱 默认）；②部署位置（本机 systemd）；③里程碑顺序。

## #21 · 2026-08-15 · Go vs Rust 裁决 + 三问确认（D-030）
- **参与方**：业主、Claude
- **议题**：业主问 Go 与 Rust 之选；三问答复（web 先用着/部署同意/里程碑同意）。
- **裁决**：语言维持 Go（瓶颈是 I/O 非 CPU，Go 超需求两个数量级；真实约束 = 开发速度 + 施工出错率 + 交接成本；高频执行层若出现另立 D# 重议 Rust）；客户端 = web 先行 + PWA 后置 + 原生 App 挂起；部署与里程碑确认。
- **决策号**：D-030。M1 施工任务清单（8 项）入 STATE，M1-a 派发首个施工 agent。

## #22 · 2026-08-15 · M1-a 交付复审（决策层）
- **参与方**：施工 agent #1、Claude
- **交付**：M1-a 脚手架（e999f8e + 9877a1c）——go.mod、arbcn-postgres 容器、4 表 DDL、config、/healthz+slog、systemd 样例、race 测试；验收实测通过（含 docker 真机验收）；A–F 自审全过。
- **决策层复审（5 项裁决）**：①healthz 200/503 degraded 语义 ✅ 批准（正是 §10 元监控所需）；②PG 不可达只 warn 不崩 ✅ 批准（监控自身可用性优先，degraded 状态由 M1-f 告警）；③迁移用 docker-entrypoint 首次建库 ✅ 接受，**版本化迁移工具列入 M1-b 任务项**（schema 将演进）；④DDL CHECK 约束 + alerts ts 索引 ✅ 批准（值域约束 = P4 可检查性）；⑤dialogue 留决策者落档 ✅ 分工正确（本条目即落档）。
- **下一步**：M1-b 派发（Fact 模型 + 存储层接口 + PG 实现 + 版本化迁移）。

## #23 · 2026-08-15 · M1-b 交付复审（决策层）
- **参与方**：施工 agent #2、Claude
- **交付**：M1-b（e04aa4b + cc85918）——Fact 模型+校验、Store 纯接口、PG 实现（CopyFrom 批量/时间窗/规则与触发器读写）、schema_migrations 版本化迁移、迁移幂等与回滚测试、race 全过、真库 5/5。
- **决策层复审（5 项裁决）**：①ticker Kind 纳入 ✅（§5 明列采集 ticker，且价格展示/RMB 折算需要）；②Unit 仅常量表不拒绝 ✅（避免阻塞 M1-d）；③迁移失败 fail-fast、PG 不可达跳过 ✅——补充：**M1-f 元监控须覆盖"未应用迁移"degraded 状态**（PG 恢复后进程补迁移依赖重启）；④InsertFacts 无幂等去重 ✅ 接受（窗口均值规则可容忍；若未来规则需精确去重另立 D#）；⑤卷重建实测被权限拦截 ✅ 接受替代验证（一次性库 + compose config 校验），真机卷重建留待需要时人工放行。
- **下一步**：M1-c 派发（Collector 框架 + Exchange 采集；注意境内访问 Binance/OKX 公开 API 可能受限——实测失败则带证据回报，不自决加代理）。

## #24 · 2026-08-15 · M1-c 交付复审（决策层）
- **参与方**：施工 agent #3、Claude
- **交付**：M1-c（c1accb2 + e2ab7cf）——Collector 接口+注册表、调度器（独立 goroutine/间隔/抖动/退避/优雅退出）、Binance/OKX 四源（funding 年化折算+ticker）、7 个离线 fixture、调度器假源测试、突变测试。真机实测**通过**（无墙风险，无代理无降级）：BTC 7.01%/ETH 5.42%/TRX +5.46%（Binance）、BTC 8.63%（OKX）。
- **决策层复审**：①命名 ARBCN_COLLECT_SOURCES（间隔+启停一变量）✅ 批准（更简）；②活数据回流事实库——TRX funding 旧快照 −12% 被今日实测 +5.46% 取代（facts.md 更新，D-028 闭环第一圈）；③`-tags=live` 冒烟入口 ✅ 保留（常规测试不触发外网）。
- **下一步**：M1-d 派发（DeFiRates/ Domestic/FX/Calendar/IV + 人工录入降级通道）。

## #25 · 2026-08-15 · M1-d 交付复审 + Binance 451 裁决（D-031）
- **参与方**：施工 agent #4、Claude
- **交付**：M1-d（c8b912d + 43e5bb5）——DeFiRates（DefiLlama 5 池）、Domestic（逆回购新浪 + BOC 挂牌爬取）、FX（USDCNH）、Calendar（规则+人工表）、Manual（人工录入端点）、OptionsIV（Deribit DVOL）；5 源真机全通；突变测试；race 全过。活数据：USDCNH 6.7443 / GC001 0.865% / BTC IV 34.82 / ETH IV 47.27。
- **复审**：5 项全批（deposit_rate Kind / bank_rate 1h 礼貌频 / 新浪代东财 / calendar venue 值域 / x-text 直依赖）。Binance fapi 451 地域封锁 → D-031 处置（data-api.binance.vision 试修折入 M1-h；补 Bybit/HTX；Earn 利率人工补录；代理需另立 D#）。
- **下一步**：M1-e 派发（规则引擎 + 状态机 + 对抗测试）。

## #26 · 2026-08-15 · M1-e 交付复审（决策层）
- **参与方**：施工 agent #5、Claude
- **交付**：M1-e（505d2af）——声明式 Cond 文法（avg/last/p25/chg + 窗口前移 @ + 缩放聚合 + 逻辑组合）、状态机（armed→active→resolved，转变才写 alerts）、首版 10 规则全落 + 幂等 Seed、每规则独立调度、对抗测试（删状态机关键行实测必红 + 生命周期全路径）、advisory 迁移锁修复（并行测试撞车）。race 全过。
- **决策层复审（5 项全批）**：①阶梯陷阱用条件内显式 scope 聚合（更简等价）✅，数据源 = binance_ear 人工补录；②逆回购时点简化为 last_24h<=1 近似 ✅；③**元监控心跳契约定稿**（M1-f 必须遵守：kind=heartbeat、value=错过窗口数、发射方 = 独立定时器持续发射且值随停摆增长、阈值 >2）✅；④计价币种陷阱白名单 ETH/WBTC/BTC/WETH + chg ±0.5% ✅；⑤funding 预警 BTC,ETH 逐实体、TRX 不混入 ✅。
- **下一步**：M1-f 派发（SMTP Alerter + 元监控，含未应用迁移 degraded 状态）。

## #27 · 2026-08-15 · M1-f 交付复审（决策层）
- **参与方**：施工 agent #6、Claude
- **交付**：M1-f（88dbf9e + 12d7df1）——SMTP Alerter（假服务器全路径测试 + 失败重试 + at-least-once）、心跳发射方（独立定时器，停摆仍发射值增长，契约合规）、迁移 degraded（healthz 503 + reason 字段）、store 扩展、race 全过、三处删行必红实测。
- **决策层复审（5 项全批）**：①STARTTLS/AUTH 真实联测留给业主凭据后人工验证，AUTH LOGIN 不预加（先核实再采纳）✅；②config.AlertEmail 遗留占位 → **M1-h 删除并改 SMTP.Configured() 门控** ✅；③心跳发射 30s 默认 ✅；④healthz reason 字段增量 ✅；⑤at-least-once 投递语义 ✅（重发优于漏报）。
- **下一步**：M1-g 派发（Web 仪表盘最小集：机会面板/触发器/告警流；ConnectRPC 服务 + React）。

## #28 · 2026-08-15 · M1-g 交付复审（决策层）
- **参与方**：施工 agent #7、Claude
- **交付**：M1-g（bb8c399 + 818474f）——proto 5 方法（buf lint 过）、Go 服务（分页钳制/错误映射/Health 同源）、React 三视图（funding 矩阵/利率矩阵/IV/逆回购/倒计时 StatTile、触发器徽标、告警流+ack）、60s 轮询、深/浅主题；npm build + race + 真传输冒烟全过。
- **决策层复审（5 项全批）**：①Store 接口 4 方法扩展（最小缺口，同 M1-f 模式）✅；②事实快照视图未做 ✅ 接受（M2 可并入机会面板）；③生成代码超限豁免 ✅（机器生成，源 91 行，同 arb proto 豁免先例）；④未评估规则投影 armed ✅（与引擎初态语义一致）；⑤web connect v2 + Go v1.20 跨版本 ✅（已冒烟验证）。
- **下一步**：M1-h 派发（全链路接线联调：main.go 总装 + config.AlertEmail 清理 + Binance 换域试修 + check-lines 门禁脚本 + 30 分钟实测）。

## #29 · 2026-08-15 · M1-h 交付复审 + M1 验收（决策层）
- **参与方**：施工 agent #8、Claude
- **交付**：M1-h（6cefbea + c9cdb91）——main.go 总装（10 源 + 心跳 + 规则 + 告警 + 仪表盘单端口）、webui 嵌入、check-lines 门禁接入 pre-commit、30 分钟实测（1,038 行 facts / 9 kind / 状态机 armed→active→resolved 全路径实测 / healthz 200 / 首页 200 / SIGTERM 干净退出 / 心跳值 ≤1.01 / defi 超时退避自愈）。
- **决策层复审**：①armed 投影不落库 ✅（dialogue #28 裁决，双证据覆盖）；②SMTP 配置非法退出 → **D-032 裁决：降级禁用不崩进程**（与 PG 降级同口径）；③D-031 换域结论 ✅——data-api.binance.vision 无 fapi 路径（404 证据），fapi.binance.com 451 为间歇性已恢复 200 → 保留原域，Bybit/HTX 补源留待需要时另派。
- **M1 验收结论**：**通过**（功能全链路达标；挂起项 = SMTP 真实投递待业主授权码、systemd 安装待业主）。
- **下一步**：M1-i 补丁（SMTP 降级行为）→ 部署（业主两动作：SMTP 授权码 + systemd）→ M2。

## #30 · 2026-08-15 · M1-i 收口（决策层）
- **参与方**：施工 agent #9、Claude
- **交付**：M1-i（0d740b7 + bc2dd34）——SMTP 非法配置 → warn + Alerter 禁用 + 进程存活（非法/合法配置双进程实测）；端口校验补强（Go SplitHostPort 不拦 h:abc/h:99999 的坑）；race 全过。
- **决策层裁决**：①D-032"SMTP 状态入元监控 degraded 面"条款 → **并入 M2**（需动 healthz.go，超出本补丁范围；warn 日志暂作信号）；②smtp_configured 日志字段语义（非法时仍 true）→ M2 顺带修正。
- **状态**：**M1 全部收口**（a~i + 验收）。剩业主两动作：SMTP 授权码 + systemd 安装。M2 规格为下一项。

## #31 · 2026-08-15 · M2 规格定稿 + 通知通道变更（业主）
- **参与方**：业主、Claude
- **议题**：接下来做什么 / 是否需要币安 API key / 告警通知方式。
- **需求**：业主问"给币安 API key?"→ Claude 澄清系统为无密钥设计（公开只读 API + 资金动作永远人工，D-010/§13），业主**无需任何 key**；业主确认**不做 SMTP 邮件推送**，改为页面小铃铛点开看通知（浏览器内通知中心）。
- **结论**：① 通知通道 = 浏览器铃铛为主，SMTP 降级可选，业主 SMTP 授权码待办移除；② M2 拆 M2-a（通知中心 + freshness 徽标 + 事实去重）/ M2-b（RMB 折算 + facts.md 自动导出 + 台账）；③ systemd 部署完成（mluser 运行）；④ 今日实证：周六闭市 fx/repo 报价冻结但采集器健康，心跳用轮询时刻不误报，真实缺口 = 展示层无法区分"闭市/源死" + 周末重复事实噪声。
- **决策号**：D-033。

## #32 · 2026-08-15 · 仪表盘布局：告警流与机会面板同行（业主）
- **参与方**：业主、Claude
- **议题**：仪表盘布局调整。
- **需求**：业主提出"告警流，我想放到跟机会面板同一行"。
- **结论**：App.tsx 用 `.row` 双栏 grid（机会面板 2fr + 告警流 1fr），窄屏 ≤860px 回退单栏；触发器保持独立一行。web 重建 + 二进制更新（22.0M）→ SIGKILL 触发 systemd `Restart=on-failure` 自动拉新（sudo 不可用时无密码重启路径，进程无本地状态故安全）。
- **决策号**：无（纯布局，无方向变更）。

## #33 · 2026-08-15 · M2-a 后端交付（施工 agent #1）
- **参与方**：施工 agent #1、Claude
- **交付**：M2-a 后端——① proto 3 新 RPC（ListUnacked/AckAll/ListSourceHealth）+ buf generate 双份入仓（Go→internal/dashboard/gen、TS→web/src/gen），buf lint 过；② Store 接口最小扩展 ListUnacked/AckAll（pgstore 实现：未读 JOIN rules 降序、单条 UPDATE 原子全清返 RowsAffected），DashboardService 复用 LatestFacts 取 heartbeat lastOK（value×interval 反推）与该源 kind 最新 fact；③ dashboard.New 签名扩展接收 []SourceInfo（main.go 从 collect.Named 投影），main.go Scheduler 开启 Dedup。
- **对抗测试**：ListUnacked/AckAll/ListSourceHealth + dedup 全覆盖；删关键行实测必红（去重 continue 分支 / down 判定分支 / acked WHERE 过滤 ×3 处）。go vet + go test -race 全过（含真库 pgstore/rule 集成）、buf lint 过、check-lines 过、`go build -o bin/arbcn` 成功。
- **决策层裁决点**：无偏离。附带修一处既有坑（pgstore/dashboard_test.go 用 `!=` 比 time.Time，PG TZ=+0800 时恒真误报，HEAD 已复现）——改用 .Equal，心得入 practices.md。
- **下一步**：M2-a 前端施工（铃铛 + freshness 徽标，3 新 RPC 客户端已就绪）。

## #34 · 2026-08-15 · M2-a 前端交付（施工 agent #2）
- **参与方**：施工 agent #2、Claude
- **交付**：M2-a 前端（本提交）——① 铃铛通知中心：header 铃铛 + 未读红徽标（>99 显示 99+），下拉抽屉未读告警列表（LevelChip + 规则名 + 相对时间 + 逐条 ✓确认）+ 底部"全部标记已读"按钮，空态"暂无新通知"，移动端抽屉全屏；② freshness 状态点：机会面板 funding/defi 矩阵格 + IV/repo/日历瓦片加 live/stale/down 色点，悬停 tooltip"最近更新 X 前 · 源间隔 Y · 状态 Z"（stale=市场闭市/冻结、down=采集器失联）；`sourceForTile(kind,venue)` 纯函数（funding→`${venue}_funding`，其余固定映射 defi_rates/deribit_iv/repo/calendar/fx/bank_rate，映射不到 null 静默）；③ 轮询并入 useSnapshot 六 RPC 并行，无新定时器；ackAlert/ackAll 本地即时更新未读数不重拉。
- **验证**：`npm run build`（tsc + vite）过；check-lines 过；`go build -o bin/arbcn ./cmd/arbcn` 成功（dist 嵌入 22.5M）；**真传输冒烟**（临时实例 :50053 不触 systemd）——ListUnacked 返回未读 + total、ListSourceHealth 返回 10 源（binance_funding/okx_funding/defi_rates/deribit_iv/repo/calendar/fx/bank_rate 与 sourceForTile 映射逐名对上；周六 repo/fx/deribit_iv 判 stale = 闭市/冻结，印证 D-033 实证）、AckAll ackedCount=2 后 ListUnacked 归零。
- **决策层注意**：当前 systemd 二进制（19:40 启动）早于 M2-a 后端，新 RPC 404；重建 bin/arbcn + 重启服务即带新功能（前端 dist 已嵌入，无需单独部署 web）。冒烟中的 AckAll 已把生产 2 条 resolved 心跳告警标记已读（仅读取状态，无资金面影响）。
- **下一步**：M2-b 施工（RMB 折算 + facts.md 自动导出 + 台账）。

## #35 · 2026-08-15 · i18n 清理交付 + 部署（Claude 直接施工）
- **参与方**：Claude
- **交付**：页面硬编码英文全部中文化——① 前端 format.ts 新增 statusText/reasonText/stateText/levelText/ruleLabel 五个映射（健康状态、降级原因、触发状态机、告警级别、规则名）；② Chip 徽标（LevelChip/StateChip）、铃铛抽屉规则名、告警流/触发器规则名、健康 chip、机会面板标题全部走映射；③ 后端 rule/state.go 消息模板改中文（`ruleLabel(name)+" 触发: …"` / `" 已解除"`）与前端 ruleLabel 语义对齐（默认回退原名）；④ `npm run build` + `go build` 后 SIGKILL 触发 systemd 重启（新 PID 2271002），healthz ok，新 dist（index-DxayK8R_.js）已托管。
- **实测**：源健康 10 源判定正确（crypto 源 live；repo/fx/deribit_iv 周六闭市 stale）；DB 内最新心跳告警 `采集器心跳 已解除`（中文模板已生效，旧英文行=历史数据）；12:00 曾有一次心跳延迟尖峰（binance_ticker 19.5s）12:07 自愈，现全部 live。
- **决策层注意**：① 遗留两条未确认心跳告警在铃铛中（11:43/12:00 抖动期间产生的 active + 最新 resolved），属真实数据，业主可 ack；② **方向级待决策**：业主已提议「提升到自动交易（人工确认）+ 模拟账号验证，尤其加密层」——触及无密钥铁律（D-010/§13）与「决策监控、不自动执行」形态，须 讨论→决定→执行 后落 D#，未决前不动手。
- **下一步**：M2-b 施工（RMB 折算 + facts.md 导出 + 台账，03-m2-spec.md §4–§6）。

## #36 · 2026-08-15 · M3 模拟执行验证：方向批准 + spec 讨论稿（业主 · 决策层）
- **参与方**：业主、Claude
- **议题**：业主提出的方向变更「提升到自动交易（手动确认）+ 模拟账号验证，尤其加密层」。
- **需求**：业主确认「模拟账号验证理论」意图；Claude 评估——①方向正确（理论需实证，先核实再采纳 D-028）；②无密钥铁律（D-010）是红线，真金执行永不留 key；③不赌（D-019）要求执行器只建议套利/息差类，无对冲拒单；④形态升级须独立里程碑 M3 不掺 M2。
- **结论**（D-034）：方向批准；testnet key 豁免条款（隔离配置 + `SIMULATED` 显式标记 + 只读，真金路径零密钥不变）；加密模拟盘 Binance Testnet + OKX Demo **都接**；先 M2-b（RMB 对账基准）后 M3；M3 拆三步 a 订单生成器+本地模拟盘回填 → b testnet 只读 + 息差收敛 → c 一键确认 UI + 风险门禁闭环；真金执行不在 M3。**细化 spec（docs/design/04-m3-spec.md）已写，动工前须决策层确认**（D-034 ⑦）。
- **风险门禁初始数值**（spec §4，业主可调）：对冲缺腿拒单 / 预期年化 <5% 拒单 / 单笔 >模拟资金 20% 拒单 / 日累计 >50% 拒单 / carry_asset 白名单。
- **下一步**：决策层确认 M3 spec → M2-b 施工 → M3-a 派发。

## #37 · 2026-08-15 · M3 风险门禁默认值定稿（业主确认）
- **参与方**：业主、Claude
- **议题**：M3 spec §4 风险门禁初始数值是否按默认定稿。
- **需求**：业主确认**按默认值定稿**。
- **结论**：风险门禁定稿——预期年化 <5% 拒单（`SPREAD_LOW`）/ 单笔 ≤ 模拟资金 20%（`SIZE_OVER`）/ 日累计 ≤ 50%（`DAILY_OVER`）/ carry_asset 白名单 / 无对冲拒单（`UNHEDGED`）。04-m3-spec.md 标记为**已确认**，可派发施工。改动门禁数值须走 D#。
- **决策号**：D-034 补充（门禁数值定稿）。
- **下一步**：M2-b 施工（RMB 折算 + facts.md 导出 + 台账）→ M3-a 派发。

## #38 · 2026-08-15 · M2-b 施工交付 + 负责人复审（F1/F2）· 施工 agent → 决策层
- **参与方**：施工 agent（M2-b）、Claude（负责人复审）
- **交付**：M2-b 三件套——① **RMB 折算**（internal/rmb 纯函数包：RMB 净收益 = USD 收益率 − 年化人民币升值，30d 尾窗，D-023 算例口径；ListFacts RPC 投影 + 汇率缺失降级"汇率不可用"）；② **facts.md 自动导出**（exporter 组件：日定时 + 规则触发事件，begin/end 标记独占段，旧快照标"已过期"不删除，原子写；rule.Config.OnActive 仅在 armed/resolved→active 转变回调）；③ **台账起步**（ledger 表 0004 迁移 + 录入/流水/按 tier 归因三 RPC + 前端三分页：监控总览/事实快照 RMB 视角/出入金台账）。全部门禁过（go vet / go test -race / buf lint / check-lines / npm build）。
- **负责人复审发现（已修复 + 补对抗测试）**：
  - **F1**：ListFacts 快照投影含 heartbeat 内部遥测，与 exporter skipKinds 排除口径不一致 → facts.go 过滤 + TestListFactsExcludesHeartbeat。
  - **F2**：FactsSnapshot 对所有 kind 统一 pct()，fx（price 6.7443）显示"674.43%"、calendar（days 16）显示"16.00%" → format.ts 新增 unit 感知 factValue()，值列按 unit 分支。两条教训入 practices.md #4/#5。
- **规格歧义裁定**：RMB 公式（spec §4 字面"× 当日 USDCNH"对年化收益率荒谬）→ 裁定 = 收益率 − 年化升值（D-023 算例一致，6%−3%=3%），回填 03-m2-spec.md §4。
- **验证**：go vet/build/test -race 全绿；线上部署（SIGKILL 重启，PID 2296654）healthz ok；ListFacts 实测排除 heartbeat、fx/calendar 单位正确；facts.md 生产首写成功（20:44 快照，段标记 + 无 heartbeat）。
- **下一步**：M3-a 派发施工（04-m3-spec.md §3）。

## #39 · 2026-08-15 · 追溯深审 M1/M2-a 全部代码（决策层派 6 路 review agent）· 施工 agent → 决策层
- **参与方**：Claude（决策层）、R1–R6 只读 review agent
- **议题**：施工 agent 上交代码复审（业主指示："所有文件，你做了审计吗？你是负责，你要负责分派agent上交过来的代码复审工作"）。
- **需求**：审全部 M1/M2-a 代码。
- **结论**：6 路并行只读 review（R1-R6）找施工漏检 → 决策层逐条亲手验证（读代码确认）→ 按实际影响定价（高危 5 / 中危 6 / 低危 3 / 接受 12）→ 修复 + 对抗测试锚点 → Batch 1/2/3 全量测试绿（go test -race 全仓 + npm build）。
- **决策号**：D-035。
- **下一步**：M3-a 施工交付复审 → 部署 → 收工落档。

## #40 · 2026-08-15 · M3-a 施工交付复审（独立 review agent）+ 修复 · 施工 agent → 决策层
- **参与方**：Claude（决策层）、复审 agent
- **议题**：M3-a 施工交付（sim 订单生成器 + 本地模拟盘回填）验收复审。
- **需求**：独立复审 → 决策层验证。
- **结论**：确认无密钥铁律（sim 包零网络零密钥纯本地）、不赌六门禁结构对诚实输入有效、RMB 刻度与 R6#1 兼容；但 **H1 结算 PnL 100 倍放大**（pct_annualized 点数当分数费率，缺 ÷100）阻断验收 → 修（Per8hRate/RMBDayEnd ÷100 + 锚点改正确值）；**M1** 成交非原子（新增 store.FillSimOrder 单事务 + 状态守卫）；**M3** NaN 绕过数值门禁（有限性守卫 + INVALID_INPUT）；L1/L2/L3 门禁加固全修；M2/L4 接受（信任边界 + M3-c 加固，spec 标注）。sim/store/pgstore 全量测试绿。
- **决策号**：D-035。
- **下一步**：SIGKILL 部署 → 线上 healthz 验证 → 收工落档 commit。

## #41 · 2026-08-15 · M3 文档审计（业主提问触发）+ 收敛口径修正 · 业主 → 决策层
- **参与方**：业主（甲方）、Claude（决策层）
- **议题**：业主问"M3 的文档审计过没有？是否为最优解？"。
- **需求**：完整规格独立审计 + 是否最优的诚实判断。
- **结论**：
  - **审计情况**：前次复审只审了代码相关章节（§3.2 结算、§4 门禁——H1/M1/M3 即出自此）；完整规格独立审计现补。
  - **是否最优**：分层判断——**机制验证方向 = 最优**（信号→订单→门禁→模拟成交→结算→RMB 对账，全程不触真金，D-010/D-019 落地）；**收敛统计目标 ≠ 最优**——testnet 前向周级小样本 + 费率偏差回答不了统计问题，且 spec"收敛结论是主要交付物"与"❌ 回测/历史回放"自相矛盾。
  - **决策（D-036）**：①收敛口径修正——前向模拟只验证机制，统计结论由历史 funding 数据出（spec 新增 §5.3 历史收敛分析，M3-b 前置小任务，不另立阶段）；②G1 规则→Signal 映射表 + 运行驱动定义（挂钩 rule.Engine.OnActive；M3-a 未接线的根因即此）；③G2 sim_pnl 措辞对齐实现；④G3 模拟资金独立量纲、比例口径报告；⑤G4 理论无摩擦曲线定义；⑥G5 确认价取确认时刻 ref_price + SPREAD_DRIFT 漂移门禁。G1–G5 全部落进 04-m3-spec.md。
- **决策号**：D-036。
- **下一步**：M3-b 细化设计（含 §5.3 历史回填前置任务 + G1 驱动接线方案）。

## #42 · 2026-08-15 · M3-b 细化设计定稿（D-037）· 业主 → 决策层
- **参与方**：业主（甲方）、Claude（决策层）
- **议题**：业主指示"排 M3-b 细化设计"。
- **需求**：细化 M3-b 到可派工施工粒度。
- **结论**：决策层摸清代码面（rule.OnActive 调用点 state.go:37、Scheduler、sim/store/exchange collector/main.go 接线）后定稿 spec §9（S1–S5）。核心裁决（D-037）：①**结算数据源 = 真实市场公开 funding，非 testnet**（testnet 费率有偏差，污染机制验证；testnet 只做 key 隔离验证，D-034 ④ 都接保持仅明确用途）；②S1 驱动 = OnActive 携带命中实体单点改 + sim.Driver 按 §3.1.1 映射组装；③S2 结算按 (symbol,venue) 分组防跨 venue 污染；④S3 testnet 只读 key 门控（缺 key 降级不阻塞）；⑤S4 历史回填落 facts 表（顺带富化 funding_warn 的 avg_30d）+ 周频 sim_report；⑥S5 白名单默认空宁缺毋滥；⑦诚实标注现货腿取永续 ticker（无现货 collector）。spec §5.1/§5.2 数据源修正 + §9 施工细化。
- **决策号**：D-037。
- **下一步**：M3-b 施工派工（spec §9 S1–S5）。

## #43 · 2026-08-15 · M3-b 施工交付（S1–S5 全落地）· 施工 agent → 决策层
- **参与方**：施工 agent（M3-b）、Claude（决策层）
- **交付**：04-m3-spec.md §9 S1–S5 五件套全落地。
  - **S1 规则→Signal 驱动**：rule.OnActive 签名携带 `[]store.EntityHit`（改点仅 state.go 回调映射）；新 sim.Driver 按 §3.1.1 不可变映射表组装 Signal → Generate 落库；funding_*→funding_hedge（SpotPrice/PerpPrice 取 ticker 最新价、FundingAnn=hit.value，诚实标注：系统无现货 collector，腿存在性由门禁把关）、reverse_repo_timing→repo、carry 白名单→carry_asset、其余信息/遥测规则不建单（宁缺毋滥）。删映射 → TestDriverFundingHitCreatesOrder 必红。
  - **S2 8h 结算调度**：store.ListOpenSimPositions 扩展 venue 维度；settleLoop 按 (symbol,venue) 分组取真实市场 funding（LatestFacts）结算，无事实 skip+warn。串 venue → TestSettleByVenue 必红。
  - **S3 testnet 只读 + key 隔离**：新包 internal/simtestnet（key 承载层，与 sim 物理隔离）；/etc/arbcn/arbcn-sim.env SIM_* 加载，SIMULATED=true 缺标记拒绝加载（TestLoadMissingSimulatedMarker 必红）；只读探针（binance_testnet / okx_demo 公共行情 + 账户只读，HMAC 签名），成功经 Heartbeat.Record("sim_testnet_binance"/"sim_testnet_okx")；零下单路径（domains_test：simtestnet 仅 testnet/demo 域、无主网交易域、无 order 端点；sim 包 TestNoNetworkImports/TestNoOrderEndpoints 把关）。缺 key → 降级禁用不阻塞。
  - **S4 历史收敛**：exchange 历史 collector（Binance data-api.binance.vision fundingRate 翻页 + OKX funding-history after 翻页，annualize 折算）；sim.BackfillHistory 幂等编排（QueryFacts 既有 ts → UncoveredFacts 跳过 → InsertFacts，键含 (venue,symbol,ts) 防同刻跨币误判）；sim_report 周频统计纯函数（实际/理论累计、残差分布、半衰期、摩擦后净 vs 5% 门槛）+ markdown 独占段导出。删 annualize/删幂等跳过 → 必红。
  - **S5 白名单 + 降级**：Config.CarryWhitelist（ARBCN_SIM_CARRY_WHITELIST 逗号分隔去重，默认空）；carry 未白名单 → WHITELIST 拒单；sim 配置缺失 → Driver nil、settleLoop/backfill 跳过（warn 不退出，D-032 同口径）。
  - **main.go 接线（§9.7）**：boot 一次性幂等历史回填 → simDriver（nil=降级）→ OnActive compose（factsExporter + simDriver）→ go settleLoop → testnet 探针随 settle tick；sim_testnet 源并入心跳 Track + 仪表盘源健康面。不新增端口/RPC。
- **验证**：go vet / go build / go test -race ./... 全绿（PG 集成测试 ARBCN_TEST_PG_DSN）；对抗测试锚点逐一实测必红（annualize / SIMULATED 门控 / venue 分组 / 映射表 / 幂等键）；无真实 key（探针 = 机制 + 测试，缺 key 降级）。
- **待决策观察**（不阻塞）：repo 信号受 5% 价差门槛约束（平时逆回购 2-4% 拒单，时点上冲放行）——与"时点逆回购"意图一致，若业主欲 repo 绕门槛须走 D#。
- **下一步**：M3-c（一键确认 UI + SPREAD_DRIFT）；并行等业主 testnet key（S3 启用门控）。

## #44 · 2026-08-15 · M3-b S4 回填数据源修复 + 部署验证 · 决策层
- **参与方**：Claude（决策层，独立验证）
- **议题**：M3-b 施工交付后部署，boot 回填失败（双路历史数据源 404），S4 sim_report/avg_30d 退化。
- **根因**（root-cause-first §7.4）：D-031 前提不成立——data-api.binance.vision 不镜像 /fapi/*（fundingRate/klines 均 404，实测）；fapi.binance.com 部署机直连 200、365d 深度可用（451 未复现）；OKX 记错端点为 funding-history（业务 404），正确端点是 funding-rate-history（仅 ~90d 历史）。
- **修复**：history.go 默认源 data-api → fapi.binance.com（BinanceHistoryBaseURL 配置项保留作 geo-block 覆盖）；OKX 路径 funding-history → funding-rate-history；Src 标注同步；history_test fixture 同步改路径。H1 复审配套已在上轮完成（8h 桶去重 + Limit 2M，防实时洪水 96 行/桶虚高）。
- **部署验证**：重启后 journal 报 "funding history backfill complete days=365"；DB 实测——binance funding 3489 条 min_ts=2025-08-15（满 365d）、okx 1080 条 min_ts=2026-05-14（~90d，OKX 保留限制，部分覆盖已标注）；年化值抽查精确（0.00003722×1095×100=4.08%）；avg_30d binance BTC=6.46%（回填前 ~7d 不可靠，现为真实 30d 均值，低于 funding_warn 门槛 15% 不误报）。sim_report 周频文件需 7×8h tick（~56h）后首次渲染。
- **决策号**：D-031 实证修订（data-api 前提否定 + fapi 回落 + OKX 端点修正）。
- **下一步**：M3-c（一键确认 UI + SPREAD_DRIFT）；并行等业主 testnet key（S3 启用门控）。

## #45 · 2026-08-15 · M3-c 细化设计定稿（D-038）· 业主 → 决策层
- **参与方**：业主（甲方）、Claude（决策层）
- **议题**：业主问"M3-c 有没有文档？文档审计过没有？"——核实后启动 M3-c 细化设计。
- **核实结论**：M3-c 有规格文档（spec §6 + §5.1 G5 + §4 二次校验 + D-034⑤ + D-036 G5），但**只到规格层未到施工细化级**（对比 M3-b §9 的 D-037 粒度）；D-036 审计落点清单无 §6，G5 只是口径定义；STATE 曾误引"spec §9.9"（实为 M3-b 的依赖与阻塞小节）。附带修正：STATE 引用错误已随本 dialogue 记录，STATE 下一步字段改指 spec §10。
- **审计发现**（摸清代码面后）：proto 源缺失（M3-c 加 RPC 须新建独立域）；SPREAD_DRIFT 标记/纯函数/拒单数据面全缺；UpdateSimOrderStatus 不改 risk_flags（拒单需新方法）；两步确认（先 confirmed 再 FillSimOrder）有并发重复建腿竞态；持仓 PnL RMB 折算若误用 RMBDayEnd 年化口径会重演 H1 刻度错位；PnL"实时"语义需裁决为已结算累计。
- **决策（D-038）**：①独立 SimService proto 域（arbcn.sim.v1，4 RPC，不动 dashboardv1 生成物）；②SPREAD_DRIFT 二次门禁纯函数 ConfirmDriftCheck（G5 口径 + 有限性 fail-closed）；③确认成交 = store 单事务原子 AcceptSimOrder（suggested→confirmed→filled + 建腿，状态守卫防并发双插），拒单走 RejectSimOrder 追加标记；④持仓 PnL 即期汇率折算（非 RMBDayEnd 年化口径）；⑤PnL 只显示已结算累计 + 最新 funding 标注；⑥可检查性（simapi 无真实端点 grep 断言 + ConfirmSimOrder 唯一写路径 + SIMULATED 徽标固定渲染）。
- **决策号**：D-038（spec §10 C1–C5 施工权威细化，含 proto 定义全文 + RPC 签名 + 对抗测试锚点）。
- **下一步**：M3-c 施工派工（spec §10 C1–C5）；并行等业主 testnet key（S3 启用门控）。

## #46 · 2026-08-15 · M3-c 施工交付（C1–C5）· 施工方 → 决策层
- **参与方**：施工方（M3-c 施工 agent）、Claude（决策层，待验收）
- **议题**：M3-c（04-m3-spec §10 C1–C5）落地交付——SimService 独立域 + SPREAD_DRIFT 二次门禁 + 确认成交流原子化 + 模拟执行 UI tab + 可检查性。
- **交付**：
  - **C1**：`proto/arbcn/sim/v1/sim.proto`（4 RPC 全文按 §10.2）+ buf v2 双模板（buf.gen.yaml 收窄 dashboard 域 + 新增 buf.gen.sim.yaml，input 级 paths 过滤）；生成物 sim.pb.go / simv1connect / sim_pb.ts；dashboardv1 生成物 byte-identical 零回归；check-lines 豁免 internal/simapi/gen/。
  - **C2**：RiskSpreadDrift 标记 + 纯函数 ConfirmDriftCheck（ref 漂移 >2% 或 年化变化 >20% 独立触发；NaN/±Inf/零值 fail-closed 拒；对抗锚点：删漂移比较/删 fail-closed → 必红）。
  - **C3**：store 原子 AcceptSimOrder（suggested→confirmed→filled + INSERT 全腿单事务，WHERE status='suggested' 守卫 + RowsAffected，拦并发双确认）/ RejectSimOrder（rejected + note 覆盖 + risk_flags 去重追加）；simapi 4 RPC（ConfirmSimOrder 唯一写路径，二次门禁数据面 LatestFacts ticker/funding，查不到 → fail-closed 拒；ListSimPositions pnl_rmb=pnl×即期 USDCNH，缺汇率 → 0；GetSimReport 三态）；sim.BuildLegs 抽共享（M3-b 确认流 / M3-c 人工流建腿口径一致）；时钟注入 Now 覆盖全路径。
  - **C4**：App.tsx 第 4 tab「模拟执行」+ SimExec.tsx 三区（建议订单分组 / 模拟持仓即期 RMB / 对账报告入口）+ SIMULATED/「模拟」徽标固定渲染 + vite 代理。
  - **C5**：simapi domains_test（grep 无 account/withdraw/transfer/下单端点/主网域、无 time.Ticker、SimExec.tsx 含 SIMULATED/模拟）；main.go 接线（st 非 nil 即挂载，sim 配置缺失降级不退出）；go vet + go test -race ./... 全绿；前端 tsc + vite build 过。
- **已知设计问题（须 Claude 裁决，施工方不自行决定）**：repo 订单（venue=domestic, symbol=GC001）无 ticker/funding 事实 → ConfirmSimOrder 二次门禁查不到数据 → **fail-closed 恒拒单**（§10.3「查不到 ticker/funding → 拒」的直接推论）。当前行为符合 spec，但 repo 确认流永远走拒单路径——是否允许 repo 类订单跳过二次门禁（repo 无市场行情可查，天然无方向敞口）需决策层裁决（D# 或本 dialogue）。
- **决策号**：无新 D#（沿用 D-038 施工权威；repo 门禁例外待裁决）。
- **验证**：go build / go vet / go test -race ./... 全绿（PG 集成 ARBCN_TEST_PG_DSN）；前端 npm run build 过；对抗锚点（suggested 守卫 / 漂移比较 / SIMULATED 渲染 / 真实端点 grep）逐一覆盖。
- **下一步**：①部署验收（spec §10.6 清单：重启 healthz ok → tab 打开 → 确认流冒烟）；②repo 二次门禁设计裁决；③等业主 testnet key 启用 S3。

## #47 · 2026-08-15 · M3-c 决策层复审 + repo/carry 恒拒裁决 · 决策层
- **参与方**：Claude（决策层，先核实再采纳 D-028）
- **议题**：M3-c 施工交付（#46）复审验收 + 裁决 #46 挂起的"repo 二次门禁恒拒"设计问题。
- **复审结论（逐项核实，非采信交付报告）**：
  - git 交付面：5 commit（83742ba..5fd8f63）结构清晰、工作树 clean、dashboardv1 生成物零回归（git diff 空）。
  - C1 proto：符合 §10.2（4 RPC / SimOrder / SimPosition，pnl_rmb 即期口径注释明确，独立域互不依赖）。
  - C2 ConfirmDriftCheck：2%/20% 独立触发 + genRef==0/genSpread==0/NaN/±Inf fail-closed（防 0/0=NaN 绕过 practices #7）——超规格要求。
  - C3 原子成交：AcceptSimOrder 单事务两步 UPDATE（suggested→confirmed→filled）各带 RowsAffected 守卫，任一 0 → 整体回滚（无"已确认未成交"悬挂）；RejectSimOrder 原子 + risk_flags 去重追加 + flags 空拒绝调用。
  - C4 UI：SIMULATED 徽标贯穿、确认按钮仅 suggested + 二次点击防误点、PnL 即期折算 + 汇率缺失标 USD 原值、risk_flags 中文徽标、无真金按钮。
  - C5 可检查性：grep 锚点（零真实端点 / 零自动确认定时器 / SIMULATED 渲染）+ ConfirmSimOrder 唯一写路径成立。
  - 基线：`go test ./...`（含 PG 集成）+ `go vet` 全绿。
- **裁决（D-039）**：二次门禁数据面**按 kind 分派权威源**，而非硬编码 ticker/funding 双查（那是 funding_hedge 语义；repo 无 ticker、carry 无 funding → 恒拒 = 确认流功能残缺，非从严）。`ConfirmDriftCheck` 纯函数签名**不变**（spec §10.3 锚点稳定）：
  - repo：ref=面值锚（漂移恒 0）；spread=`KindReverseRepo` 当日利率（与生成侧 repoSignal 同权威源）；查不到 → fail-closed 拒。真实漂移风险=利率变化 >20% 拒。
  - carry：spread=`KindDefiRate` 年化（权威源）；查不到 → fail-closed 拒；ref=ticker 有则查、无则面值锚漂移恒 0（稳定币无方向风险）。
  - fail-closed 语义不放宽：每类订单权威源查不到 → 拒（宁缺毋滥）。
- **实现**：`simapi.Service.confirmDrift` + 7 新对抗测试（repo accept/reject/fail-closed + carry accept/ticker-drift/spread-reject/fail-closed；删 kind 分派 → accept 断言必红）。spec §10.3/§10.4/C2 表格 + decisions D-039 同步。
- **验证**：全量 `go test ./...` + `go vet ./internal/simapi/... ./internal/sim/... ./cmd/...` 全绿。
- **决策号**：D-039（kind 分派数据面）。
- **下一步**：部署验收（spec §10.6：重启 → healthz → 模拟执行 tab → 确认流冒烟）；等业主 testnet key 启用 S3。

## #48 · 2026-08-15 · 业主提供 testnet key → S3 启用 + OKX 探针 ts 格式 bug 修复 · 决策层
- **参与方**：业主（提供 key）、Claude（决策层）
- **议题**：业主提供 Binance 合约测试网 + OKX 模拟盘 key，启用 S3 testnet 只读探针。
- **处置**：
  - **key 承载**：写入 `/etc/arbcn/arbcn-sim.env`（mluser:mluser 0600，SIMULATED=true + SIM_BINANCE_*/SIM_OKX_* 全量）；目录原不存在已创建。服务以 mluser 运行 → 文件属主用 mluser（非 spec 注释的 root:root，运行环境实测修正）。
  - **连通实测**（先核实再采纳 D-028）：Binance `fapi/v1/time` + `fapi/v2/balance`（HMAC 签名）→ 200，测试网虚拟资金 BTC 0.01/USDT 5000/USDC 5000；OKX `public/time` + `account/balance`（x-simulated-trading:1）→ 200，demo 虚拟资金 BTC 1/OKB 100/USDT 5000/ETH 1。
  - **发现并修复探针 bug**：OKX 签名头 `OK-ACCESS-TIMESTAMP` 必须 **ISO 8601 UTC** 格式，探针原用 Unix 毫秒（从 Binance 惯例照搬）→ 实测 50102 "Timestamp request expired"（本机时钟仅偏 66ms 排除时钟）。修 probe.go ts 格式 + probe_test 加 ISO 正则对抗锚点（改回毫秒 → Record 断言必红）。
  - 部署：build + 重启，启动日志无 key 加载警告（此前 chown root:root 导致 mluser 读不了 → permission denied，已改 mluser:mluser）。
- **已知 degrade**：探针随 8h settle tick 跑，首次 heartbeat 登记待下一个 tick（key 连通已由 curl 实测证明，自动化登记等自然 tick）。
- **决策号**：无新 D#（探针实现修正，practices #12 补协议格式教训）。
- **下一步**：观察 8h tick 探针 heartbeat 登记（ListSourceHealth 应见 sim_testnet_binance/okx freshness）；S3 全链路闭环。

## #49 · 2026-08-16 · 测试网账户区需求确认 + D-040 施工部署 · 业主 → 决策层
- **参与方**：业主（提问/选型）、Claude（决策层，本 feature 直接施工——非大里程碑，D-039 同量级）
- **议题**：业主提问「不显示两个账户的模拟资金和账户信息？」——S3 探针只验证连通（heartbeat），余额 body 丢弃。
- **结论**：业主选型「SimExec tab 加测试网账户区」（对照项：ListSourceHealth 只显示 freshness 不含资金，不满足"账户信息"）。
- **处置（D-040）**：
  - 探针 `Run` 返回余额快照（binance `fapi/v2/balance` 稳定币合计近似；okx `account/balance` totalEq 精确）→ 新表 `sim_testnet_accounts`（0006）→ `GetTestnetAccounts` RPC → SimExec 账户区（SIMULATED 标注 + 权益/别名/资产明细）。
  - main.go 启动即探针一次（不等 8h tick）→ 重启后账户区立即有真实数据。
  - 对抗测试：删解析/删稳定币合计/删 totalEq → 必红（已实测 2 处红）。
  - **部署实测**：binance equity 10000（USDT 5000+USDC 5000，非稳定币如实标 —）；okx equity 80673.55（BTC 1/OKB 100/USDT 5000/ETH 1，与业主核实的虚拟资金一致）；**ListSourceHealth 首次 heartbeat 已登记**（此前待 8h tick 的观察项随启动探针顺带闭环）；ListFacts 零回归。
- **决策号**：D-040。
- **下一步**：S3 全链路闭环（探针 + 账户区 + 结算正交）；出入金通道验证（业主）。

## #50 · 2026-08-16 · 加密交易所出入金实测通过 + TRX funding 跨所差异核实 · 业主 → 决策层
- **参与方**：业主（实测确认 + 提问）、Claude（决策层，先核实再采纳 D-028）
- **议题**：① 出入金通道；② 机会面板 TRX 在 binance 显示 2.27%、okx 显示 −3.78%，业主质疑是否 bug。
- **结论**：
  - ① 两个加密交易所出入金**实测通过**（施工表「加密交易所出入金通道验证」✅）；**OTC 法币通道（1 万小额）仍待验证**。
  - ② TRX 跨所差异 = **真实市场分歧，非管线 bug**。核实证据：
    - 实时 curl 实测（部署机）：binance TRXUSDT `lastFundingRate` +0.002472%/8h → 年化 +2.71%（面板 2.27% 为上次轮询值）；okx TRX-USDT-SWAP `fundingRate` −0.003170%/8h → 年化 −3.47%（面板 −3.78% 为上次轮询值）。
    - DB facts（ListFacts）：binance TRX 2.269935 / okx TRX −3.784851，同为 16:09 轮询，与面板一致。
    - 年化系数核对（`annualize = rate×8760/interval×100`，8h→×1095）正确。
    - 根因：funding 由各所**内部盘口溢价**决定（永续价 vs 指数价），binance TRX 永续溢价（多付空）/ okx 折价（空付多）→ 同标的跨所真实分歧；瞬时且会持续数日，不会收敛同一值。
- **系统行为确认**：`funding_warn/critical` 仅 BTC,ETH 且条件 `avg_30d > 15/20`（正阈值）；`trx_funding_positive` 条件 `avg_24h > 0`——okx 负 funding 不满足 → **不建单**（宁缺毋滥）；binance 正 funding 触发费率转正 → 经典 carry（现货多吃质押 + 永续空收 funding）。负 funding 反向机会（现货空 + 永续多）操作不便，系统正确地不追逐。
- **决策号**：无新 D#（数据核实，无设计变更）。
- **下一步**：OTC 法币通道验证（1 万小额，业主）。

## #51 · 2026-08-16 · 监控架构：轮询 vs 事件驱动 · 业主提问 → 决策层
- **参与方**：业主（提问）、Claude（决策层，第一性原理推导）
- **议题**：机会监控系统应「轮询」还是「事件驱动」拉取。
- **结论（维持轮询，不改）**：
  - 第一性推导：机会时间尺度 = **8h funding 结算 epoch**（结算每 8h 才变，事件驱动的高实时性无价值）；数据源全为**拉取型 API**（binance/okx/public REST，无推送），推送仅存在于下单链路——而系统**不自动执行**（决策监控 + 人工下单，D-008），事件驱动收益接近零。
  - 事件驱动唯一成立条件：①真实推送源（websocket）②延迟敏感执行（ms 级）③千级以上 symbol。三者当前均不满足。
  - 现状正确：规则 OnActive 采样触发 + collector 轮询 + web 60s 轮询 = 自愈、简单、省 token；8h 结算循环在 M3 已按 (symbol,venue) 分组对接真实 funding（D-037）。
  - **结论**：保持轮询；事件驱动留作「若未来做自动执行/接入 ws 推送」的升级路径（记入 practices 级认知，非本次改代码）。
- **决策号**：无新 D#（确认既有架构 D-029，无设计变更）。
- **下一步**：无（架构确认，无代码改动）。

## #52 · 2026-08-16 · 当前机会盘点：可交易机会 + 加密对冲 · 业主提问 → 决策层
- **参与方**：业主（提问）、Claude（决策层，先核实再采纳 D-028）
- **议题**：①目前是否无机会可交易？②加密平台是否有对冲机会？
- **结论**：
  - ①「无机会」分两层：**机会面板层正确**——BTC funding 年化 5.8%(binance)/6.9%(okx)、ETH 6.5%/4.0%，全个位数，远低于 funding_warn/critical 门槛（avg_30d > 15/20%），funding 窗口档（15–30%）当前不存在，面板不显示 = 设计正确（宁缺毋滥）；**真实可交易层始终有常态对冲息差**：BTC cash-and-carry（现货买+永续空，吃 funding）~7% delta 中性、sUSDe 4.44%（Ethena 自带对冲）、稳定币生息 3.3–4.2%。
  - ②加密平台对冲机会：有。funding carry（现货 long + 永续 short）现在是 BTC ~7% 最厚，delta 中性，但扣 0.2–0.3% 双边摩擦后约 2 周回本才开始纯 carry；因 <15% 门槛，系统不自动建议，要动需业主手动决定（或走 D# 调门槛）。
  - **关键全市场扫描发现**（本会话新增实证）：binance 532 个合约中极端 funding 全在**被砸的微盘币**（ACE −1495%、COW −1440%、WAL −733% 年化）——是空头砸穿的永续深贴水陷阱（深度薄/滑点大/basis 随时 blowout），非可捕获稳定 carry；追这些违反不赌原则 D-019。**实证验证了系统只跟踪 BTC/ETH/TRX 主流的正确性**（宁缺毋滥挡的正是这种）。
  - 诚实定位：当前 4.4–7% 对冲息差高于诚实基线（D-026 3.2–3.7%）、低于 spec 档位预期（sUSDe 8–12% / funding 15–30%）= 市场平静期（BTC IV 34.9），非设计错误。
- **决策号**：无新 D#（数据盘点 + 架构确认，无设计变更）。
- **下一步**：无代码改动。业主如要动 BTC carry 或 sUSDe，另行测算执行成本与持仓周期。

## #53 · 2026-08-16 · OTC 法币通道实测通过（出入金通道全部闭环）· 业主 → 决策层
- **参与方**：业主（实测确认）、Claude（决策层）
- **议题**：出入金通道验证最后一项——OTC 法币通道（1 万小额）是否打通。
- **结论**：业主实测通过 ✅。至此出入金三向全闭环：加密交易所出入金（#50）+ **OTC 法币通道（1 万小额）**。
- **影响**：施工表「出入金通道验证（1 万小额 OTC 法币通道）」⬜→✅；资金进出双向可行，实际建仓的前置条件全部就绪（机会监控 + 模拟盘 + 出入金通道）。
- **决策号**：无新 D#（业主实测确认，无设计变更）。
- **下一步**：业主决定是否开始实际建仓（首笔可从小额起，对照机会面板/台账执行）；若建仓则台账记 ledger（channel/amount/fee/tier）。

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
