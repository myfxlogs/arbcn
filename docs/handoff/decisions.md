# 决策记录（ADR-lite）

> 格式：背景 / 决策 / 理由。只追加不删改；超 ~350 行最旧归档到 decisions-archive.md（留索引行）。

## D-001 角色结构：Claude 唯一决策者
- **背景**：多 agent 协同需明确决策权归属，否则施工 agent 会"自由发挥"改变项目方向。
- **决策**：Claude = 第一负责人 / 唯一决策者；其他 agent = 施工方（只实现、不决策）；人类 = 甲方业主（只提需求，可能错误，由 Claude 把关纠错）。
- **理由**：单一决策者 + 可检查门禁 = 方向与方法不漂移（P4+P5）。

## D-002 交接通道：git 跟踪纯文本 = 唯一共享通道
- **背景**：各 agent 私有记忆互不可见，私有记忆里的项目事实会随 agent 离开而丢失。
- **决策**：项目事实只存仓库内 git 跟踪纯文本文档；任何 agent 私有记忆 = 一次性缓存，收工必须清扫上翻（§6.2）。
- **理由**：P1 通道不变量 —— 交接只能走双方都读得到的通道。

## D-003 文件集：精简起步
- **背景**：文件越多，SSOT 决策点越多，token 越贵。
- **决策**：起步只建 AGENTS.md + STATE.md + decisions.md + practices.md + dialogue.md + LOG.md + docs/README.md + 入口壳 + pre-commit hook。code-map / constraints / design 等项目长大（T1 超 ~450 行）再拆。
- **理由**：P3 SSOT + P2 成本恒定。

## D-004 清扫仪式：pre-commit hook 机械强制
- **背景**：靠"记得"执行纪律必被违反（P4）。
- **决策**：scripts/hooks/pre-commit 强制 —— ①业务代码提交必须伴随 STATE.md 更新；②保护文件历史标记删除拦截（归档轮转豁免）。禁 --no-verify。
- **理由**：无法机械检查的规则必然被违反。

## D-005 对话纪要：dialogue.md（T1）
- **背景**：业主要求全部对话"有迹可查、有痕可阅"，但全量原文违反 token 经济。
- **决策**：每次实质对话落一段结构化纪要（日期/参与方/议题/需求/结论/决策号）到 docs/handoff/dialogue.md；全量原文由各工具 session 文件兜底（人类可读）。
- **理由**：P2 增量优先 + 实质无损（推理链完整可追溯）。

## D-006 charter 暂定锚，待业主确认
- **背景**：arbcn 领域尚未由业主明确，但方向必须有锚（P5）。
- **决策**：暂按 arb 同域（跨平台跨经纪商套利）立锚，标注"待业主确认"；业主给需求后以新 D# 正式化。
- **理由**：有锚可改，无锚漂移。

## D-007 约束：国内合法路径的无风险收益边界（1–2%）
- **背景**：业主目标"无风险套利（含套息）"，但 2026 年国内现实：无风险利率整体 1–2%（逆回购 1.5-2.5%、货基破 1%、大额存单 3 年 1.55-2.1%、储蓄国债 1.63-1.7%）；加密资金费率套息被八部门 42 号文（2026-02）明令禁止，个人投资 = 民事无效、损失自担；跨境套息受资本管制（5 万美元/年）限制。
- **决策**：arbcn 合法路径 = 无风险底仓 + 低风险套利增强组合（预期 ~2–3.5%/年）；不做国内不合法的加密/衍生品套息；"套息"定义为带敞口交易，不并入"无风险"口径。
- **理由**："高收益 + 无风险 + 国内合法"三者不可兼得（三角不可能），方案必须显式二选一；业主坚持高收益套息则须明确接受合规风险。

## D-008 charter 正式化（取代 D-006 暂定锚）
- **背景**：业主给出需求（20 万 RMB 国内套利、多币种配置）并完成三锚点选择。
- **决策**：arbcn = 个人资金运营决策系统；目标长期年化 5–8%（市场条件允许时）、最大回撤 ≤8%（中风险）、当前诚实基线 2.5–4%；路径 = 无风险核心先行 → 合法 USD 套息 → 低风险增强 → 加密套息（门禁后置）；形态 = 决策监控+记账；合规 = 国内合法优先，灰色层业主知情承担。
- **理由**：P5 方向锚必须真实化；5–8% 与当前市场条件（funding 5–11%、人民币升值、国内 1–2%）存在真实差距，目标定为"条件允许时"并诚实按年份分布交付。

## D-009 组合架构：五层 S0–S4 + 阶段门禁
- **背景**：业主选"接受合规风险 + 中风险 5–8%"。
- **决策**：S0 应急 10%（货基）/ S1 无风险底仓 40%（逆回购+大额存单+储蓄国债）/ S2 多币种套息 25%（境内美元定存 3.0–3.6% + QDII 美债）/ S3 低风险增强 20%（转债打新/双低/折溢价）/ S4 加密套息 5%（弹性 0–15%，门禁激活）。
- **理由**：风险自内向外逐层递增，每层硬帽；无风险核心占比 50% 保证最坏情况本金安全垫。

## D-010 软件范围：决策监控 + 记账（不自动执行）
- **背景**：业主定形态为决策监控+记账。
- **决策**：系统 = 台账/归因 + 收益日历 + 机会雷达（资金费率/折溢价/汇率）+ 风险仪表（回撤 ≤8% 硬线）+ 合规提示（购汇额度/42 号文）。人工下单，系统只出决策依据。
- **理由**：国内券商/银行自动化接口受限 + 灰色层人工留痕是风控要求；监控先行跑通业务。

## D-011 技术栈：Go + React/TS + PostgreSQL + ConnectRPC
- **背景**：arbcn 是全新项目，技术栈由决策层定。
- **决策**：Go core + React/TS web + PostgreSQL + ConnectRPC 单二进制（复用 arb 已验证架构模式，不复制代码）。
- **理由**：该栈在 arb 已验证（单二进制部署、内嵌静态文件、流式监控）；PG 适合台账/时序 P&L。

## D-012 USD 套息通道优先级 + 汇率逆风分步建仓
- **背景**：2026-08 核实：境内美元定存 3.0–3.6%；QDII 美债基金人民币份额 YTD −1.5%~−3.4%；USD/CNH 6.74–6.76 且人民币 2026 已升逾 3%，机构看下半年 6.7–7.0 先升后稳。
- **决策**：S2 主用境内美元定存（5 万美元购汇额度内，20 万 RMB ≈ 2.96 万美元），QDII 作卫星；**分 3 批建仓摊汇率**，不做一次性换汇。
- **理由**：票息确定但汇差是当前真实逆风；分步降低择时风险。

## D-013 S4 加密套息激活门禁（当前监控中，不激活）
- **背景**：业主接受加密合规风险，但 2026-08 funding 年化仅 5–11%（情绪中性），扣 OTC 摩擦/交易所/稳定币风险后净 4–9%，不划算。
- **决策**：S4 激活须同时满足——①30 日滚动资金费率年化 >15%；②OTC/离岸合规风险清单通过；③额度 ≤ 总额 20%；④以新 D# 记录激活。当前不满足，**只监控不激活**。
- **理由**：拿法律风险（42 号文：民事无效、损失自担）换 4–9% 净收益不划算；门禁把"接受风险"变成"条件触发"，而非常驻敞口。

## D-014 pre-commit 守卫细化：只拦历史标记行，AGENTS.md 移出删除守卫
- **背景**：charter 正式化（D-008）替换 AGENTS.md §1 暂定锚段落时，pre-commit 以"保护文件有删除行"拦截——但 charter 是活文档，内容更新合法，旧文本已由 D-006 + git 保留。门禁不得阻止合法内容更新，否则要么逼人 --no-verify（更糟），要么项目停摆（P4：门禁须机械可查且不误伤）。
- **决策**：① 删除守卫收窄为**只拦历史标记行**（✅/⬜/🔄/决策头/纪要头），**计数制**（标记净减少才拦；🔄→✅ 状态流转、charter 替换放行）；② AGENTS.md 移出守卫名单（变更控制 = decisions.md + git）；③（执行中修正）轮转豁免条件：LOG.md **有新增行**即成立（原实现只认新增文件 `^A`，误拦 LOG.md 修改场景）。
- **理由**：守卫保护的对象是"历史痕迹静默消失"，不是"内容变更"；计数制在机械性与误伤率之间取最优。

## D-015 方案自审计：组合修订（业主要求审计的响应）
- **背景**：业主质疑方案是否最优解，决策层执行自审计。审计中两项"优化"提案被 2026 数据当场证伪（北交所打新需冻结 500 万+现金才稳中 1 手，20 万陪跑；QDII-LOF 限购压至 10–100 元/日且 2026-08-07 新规最晚 2027 年底退市）——审计协议 = 提案先核实再修订，证伪即弃。
- **决策**：① S1 大额存单（现行 30 万起购，20 万够不到）→ 民营银行定期（富民 3 年 2.15%，存款保险 50 万内全保）+ 储蓄国债 + 时点逆回购，S1 预期 2.0→2.3%；② S2 USD 25%→15%，设 USD/CNH < 6.6 停止加仓线（票息差 1.3% − 预期升值 1–3% ≈ 0，EV 偏薄）；③ S3 引擎重写：打新/LOF 标注"2026 结构性关闭"，核心 = 双低转债（明确带敞口），新增"改革触发器"监测（北交所市值配售改革、大额存单 20 万新规落地 → 打新/存单通道重启）；④ charter 诚实基线 2.5–4% → 2.5–3.5%。
- **理由**：2026 国内零售真套利通道结构性枯竭（仅剩逆回购时点 + 存款利差），最优解 = 无风险层效率最大化 + 条件引擎门禁化；审计方向是求真，不是自我美化。

## D-016 S4 门禁修订（修订 D-013）
- **背景**：D-013 设 15% 激活线未计入 3–4% 往返摩擦（OTC 溢价 + 冻卡 + 对手方）；8% 回撤线对 S4 无效（跳空归零型风险，非回撤型）。
- **决策**：① 激活阈值 15%→**20%**（30 日滚动资金费率年化）；② 首轮上限 20%→**10%**（尾部贡献 ~−5–10% 可存活）；③ 8% 回撤线只适用 S0–S3，S4 尾部风险单独设上限并在激活 D# 中显式记录。
- **理由**：摩擦摊销后 15% 阈值 EV 为负；跳空风险无法用回撤线防护，必须用权重上限防。

## D-017 渠道结构修订：银行产品为主（80%），业主渠道偏好
- **背景**：业主提出渠道偏好——"更多偏向走银行产品"。核实 2026-08 银行系盘面：美元现金管理理财均值 3.37%（九成 3–3.5%）、美元封闭式理财 ~4%（建行）、美元定存 3.2–3.6%、民营定期 2.15%、固收+理财中位 2.35%（头部 3.03%）、现金管理类 1.13–1.25%、结构性存款实际 1.7–2.2%（<民营定期，不配）。
- **决策**：渠道结构 = 银行 80%（S0 现金管理理财 10% / S1 民营定期+国债 40% / S2 美元理财+定存 20% / S3 固收+理财 10%）+ 证券账户 15%（时点逆回购+双低转债）+ 加密门禁 5%。基线修订 2.3–2.5%（差年 2.2–2.5 / 平年 2.5–3.0 / 好年 3.0–3.5）。若业主坚持零证券账户 → 纯银行版上限 2.0–2.3%，唯一增强杠杆只剩 S4。
- **理由**：渠道偏好是业主约束输入，按角色必须采纳；但其代价须显式定价（上限压缩 ~1%/年，5–8% 目标只剩双低大年 + S4 两条路）。美元理财是银行原生"收益高地"，把"套息"留在银行通道内完成。

## D-018 外部方案（千问）评估结论：吸收互认基金工具，拒绝其余
- **背景**：业主引入千问 AI 的对冲理财方案（25% 货基短债 / 40% 美元定存裸持 / 25% 华夏全对冲互认基金 / 10% 中行两得宝），要求评估。按协议逐条核实。
- **核实结果**：① 华夏精选人民币投资级别收益基金（968201/968202）真实——2026-05-11 开售，双资产全对冲（点心债+美元投资级债，基金层面汇率对冲，人民币敞口 ≥70%），2025 +4.28% / 2024 −2.11% / 近 1 年 +1.73%，回撤 −2.92%，A- 评级；② 两得宝在售但 R5/C5 激进型准入（2021 起），与业主中风险画像冲突；③ 工行美元定存实际 2.8%（5000 美元以上），千问称 0.8% 错误，"差 1,840 元"虚报（真实差 240 元）；④ 千问压力测试用年息套 6 个月场景，收益夸大 ~2×；⑤ 千问"美联储下半年降息"前提与 2026-08 加息预期升温矛盾；⑥ 购汇用途申报建议不合规。
- **决策**：吸收——华夏基金纳入产品雷达（S3 银行子层候选，全对冲拿美元债收益不吃汇率风险）；拒绝——50% 裸美元敞口（违反 D-015）、两得宝（画像冲突）、6 个月短存逻辑（加息预期下锁 1 年更优）。
- **理由**：自然对冲概念成立但敞口定价错位；工具层发现优于结构层方案。

## D-019 套利第一性原则"不赌"入 charter + S2 重构
- **背景**：业主定原则——"套利交易的第一性原则，不应该赌"。以此重审全方案：千问 50% 裸美元（赌）、我方 S2 裸美元 20%（也是赌，票息差是汇率风险的补偿）、华夏全对冲（不赌）、S4 加密 cash-and-carry（方向中性，经济上最纯套利）、时点逆回购/存款利差（不赌）。
- **决策**：① "不赌"原则写入 charter：不做无对冲的方向性敞口；② S2 裸美元清零，重构为华夏全对冲互认基金 15%（对冲后纯息差 1.5–2.5%）；③ S4 重新定位为"方向中性套息"（经济上无赌，法律灰色由门禁管）；④ S1 40→45%，新增监控项"对冲成本窗口"（远期贴水 vs 利差，贴水压缩时加仓对冲产品）；⑤ 若未来有真实用汇需求（留学/移民），另立"用途性配置"层，不进投资层。
- **理由**：套利收益来自价差收敛，不来自方向判断；裸敞口的收益是风险补偿，不是套利。合法性悖论：合法+带赌（裸美元）被原则淘汰，灰色+不赌（S4 方向中性）在门禁内保留——原则裁定的是经济性质，合规性由 D-013/D-016 另管。

## D-020 优先级反转：机会监控为第一交付
- **背景**：业主指出"现在不是施工的问题，是信息监控、收集的问题，是及时发现套利套息机会的问题"；优化原则 = 收益最大 × 实现路径最短。
- **决策**：系统交付顺序反转：①机会监控 v1（资金费率/稳定币真实利率/质押 APR/基差/逆回购时点/对冲成本窗口——全部公开只读数据，无账户密钥）→ ②台账/归因 v2 → ③风险仪表 v3。机会雷达从 Phase B 提为 MVP。
- **理由**：机会发现是收益的先行条件；只读公开 API 无需券商/交易所对接 = 最短实现路径。

## D-021 加密为主方向：三档结构（业主 80% 加密方向）
- **背景**：业主"80% 走合规加密可行" + 已持有 TRX 质押（~5 TRX/天，隐含 4–5 万 TRX）。核实：稳定币头条 8–10% 为 200–300 USDT 小档诱饵，大额实际 1.6–2.2%；大额可用 = Binance Earn 定期 4.5–5.8% / Bybit 4.8–5.5% / OKX 5.0% / Aave 4.67% / Morpho 4–6.5% / 代币化美债 3.4–5%。TRX 链上基础 3.2%，能量/投票组合 6–10%，活动档 20%+ 促销。
- **决策**：加密三档——①稳定币收益层 60%（不赌，4.5–6%，场所分散 ≥2 所，定期档非阶梯活期）；②门禁套利层 15%（cash-and-carry，D-016 监控触发，未激活时资金并入①）；③国内底仓 25%（现金管理+逆回购时点+民营定期，保底+冻卡回旋）。TRX 持有层单列（业主自选方向敞口，不计套利口径，APR 优化属监控任务）。**出入金先行验证**：1 万小额 OTC → 冻卡观察 → 分 3–4 批入金。新基线 3–4%，好年 4.5–6%。严格说境内无"合规加密"通道（42 号文），"合规"= 主流场所+保守策略+全程留痕。
- **理由**：收益/路径排序——稳定币金额档（4.5–6%）> TRX 优化（3.2→6–10%）> cash-and-carry 窗口 > 国内层；阶梯利率陷阱必须以"金额档真实利率"原则规避。

## D-022 监控系统 v1 规格（监控先行）
- **背景**：D-020 优先级反转后的第一施工项。
- **决策**：v1 范围（全部公开只读数据）：①资金费率聚合（主流币 × 主要所，30 日滚动，告警阈值 15%/20%）；②稳定币金额档真实利率（按业主资金档，非头条档，含阶梯结构识别）；③质押 APR（TRX 链上/能量组合/平台档，含业主持仓日收益追踪）；④基差（季度合约 vs 现货）；⑤国内层（逆回购时点日历/银行利率雷达/对冲成本窗口）；⑥出入金通道台账（OTC 记录/冻卡标记）。告警推送。台账/归因并入 v2。
- **理由**：机会先行；公开 API 无密钥 = 最短路径；金额档原则规避宣传陷阱。

## D-023 加密层的美元敞口定性：知情条款 + 混合比例裁决
- **背景**：业主要求"国内产品 VS 加密产品，收益 VS 风险"对比。对比暴露关键事实：加密产品收益全部为 USD/币计价——稳定币 4.5–6% 折算人民币后，升值 3% 情景净 1.5–3%（与国内打平），贬值情景 7.5–9%。D-019"不赌"原则在加密层以"稳定币"名义重新遇到币种敞口问题。
- **决策**：①知情敞口条款——加密层 = 业主知情的美元敞口选择，敞口比例显式落决策记录；②推荐**混合 60/40**（加密 60 + 国内 40：名义 ~3.1%，三情景 1.8–4.5%）；加密 75/25 为业主可选的收益最大化档（敞口 60–75%）；③监控 v1 必测项新增 **RMB 计价净收益**（加密收益按当日汇率自动折算，杜绝虚高美元数字）；④总资产口径：业主已有 TRX 持仓（~5 TRX/天）计入敞口统计。
- **理由**：币种敞口是加密收益的隐性成本；知情 + 显式比例 + 自动折算，是把"80% 加密"从口号变成可管理决策的三件套。

## D-024 方案再审计：动态资本路由 + 收益阶梯三档 + 出入金设计
- **背景**：业主澄清 TRX 仅为持仓例子（非方向，降格为"现有资产优化项"，3.2%→6–10% 照做）；要求找更多更优方向 + 再审计最优性。
- **核实**：① Babylon BTC 质押证伪——散户基准 0.03–1%，收益以 BABY 代币发放（自高点 −92%），排除；② Binance Launchpool 2026 Q1 实测 BNB 池平均 19–22%（单期 6–45%），但稳定币池仅占奖励 10–15%——高收益与 BNB 币价风险绑定，归入持有层加成。
- **审计发现（6 项）**：①静态比例非最优 → **动态资本路由**（基档驻留 + 按实时利差路由进 funding 窗口/sUSDe 类/时点逆回购/Launchpool 日历）；②收益阶梯三档：保守（稳定币 4.5–6%）/ 均衡（+sUSDe 类 8–12% 占加密层 25–30%）/ 进取（+funding 动态 15–30% 窗口）；③出入金通道设计缺失（最大操作风险）→ 独立模块：OTC 溢价监测 + 多通道 + 小额多笔 + 冻卡预案；④托管阶梯：2–3 所 + 自托管 10–20% + 代币化美债尾险分散；⑤国内层简化：移除华夏对冲（敞口已知情接受，2.6% 对冲费白付），改为现金管理 10% + 民营定期 10–15% + 时点逆回购 10–15%；⑥监控新增"计价币种陷阱"识别（以非稳定币计价的收益 ≠ 真实收益）。
- **理由**：收益最大 = 阶梯内嵌 + 动态路由（基档不闲置、尖峰不缺席）；路径最短 = 现有账户 + 公开 API；审计原则 = 先核实再采纳（本轮再次证伪一个方向）。

## D-025 保本凸性档（业主提出"保本+对冲"方案）
- **背景**：业主提出"另有 1 个保本方案，对冲"。
- **决策**：收益阶梯新增**保本凸性档**（CPPI 式）：19 万本金在国内无风险层（民营定期+时点逆回购 ~2.15%，存款保险全保），年利息 ~4,100 元买 BTC/ETH 看涨期权（凸性敞口）；风险上限 = 全部利息，本金 100% 保护。熊/平市：期权归零，仍得 ~2.15%；牛市：凸性参与，上不封顶。变体：①稳定币保本层（5%→1 万利息买期权，含美元敞口，D-023 适用）；②交易所保本鲨鱼鳍（0.5–8% 本金保护型结构化）；③已持 TRX 看跌保护（ATM put ~8–15%/年，仅极端市况用）。
- **理由**：与"不赌"原则完全兼容——本金零风险，唯一风险 = 利息；期权凸性参与加密牛市。最终阶梯四档：保守 / 保本凸性 / 均衡 / 进取。

## D-026 档位定稿（决策层裁定）：保本凸性 50% + 稳定币基档 50% + 动态路由
- **背景**：业主确认 TRX 不在 20 万之内（独立处置）；档位倾向保本凸性组合，委托 Claude 以专业性裁定。
- **决策**：① A 半（10 万）保本凸性：民营定期 5 万（2.15%）+ 现金管理/时点逆回购 4.5 万（灵活保本），年利息 ~2,150 元 → 1,000 元季滚 BTC/ETH 看涨期权（凸性）+ 1,150 元留存（地板 ~1.15%）；② B 半（10 万）稳定币金额档：2–3 所定期档 4.5–6%（场所分散），分 3 批入金；③ 动态路由增强（零静态成本）：季末逆回购时点 / funding>20% cash-and-carry / Launchpool 稳定币池 / 跨所费率差；④ 均衡卫星（sUSDe 类）推迟——3 个月无事故运营后新 D# 评估；⑤ 保本鲨鱼鳍作为期权操作备选工具。
- **数字**：地板 +2.15%（最坏情景仍正）；基准 ~3.2–3.7%（平汇率）；贬值 3% ~5.2%；加密牛市 8–10%+；正常情景回撤 ~0。5–8% 目标保持条件性。
- **理由**：纯保本凸性期望过低（期权 theta 拖累，横盘年 ~0%），必须与收益档组合——稳定币档买期望，凸性档买右尾；均衡档脱锚风险与期权赔付同源（加密崩盘时双双失效），推迟为条件卫星；组合优化目标是地板 + 右尾，符合业主"保本优先 + 弹性参与"的真实偏好。

## D-027 第 3 轮审计：期权预算动态化 + 自托管腿补齐（修订 D-026）
- **背景**：业主第三次要求审计最优性，并确认"监控变化 → 知识库扩张 → 方案演进"为常态。
- **发现**：① D-026"1,000 元季滚"措辞歧义（可读作 1,000/季 = 4,000/年 > 年利息 2,150，破坏保本）——澄清为 **1,000/年（≈250/季）**，并引入**动态期权预算**（低 IV 季加至 1,500–2,000/年，高 IV 季减至 500–1,000，用留存余额加购，不动本金），新增监控项 BTC/ETH IV；② D-024 的"自托管 10–20%"未带入 D-026 执行版——稳定币 10 万改为 **7 万 CEX（Binance 4 + OKX 3）+ 3 万自托管**（硬件钱包 → Morpho/Aave 4.5–6.5%，备选 BUIDL 3.4%），应对 42 号文执法尾部 + 单所托管尾部；③ 地板来源认知修正：组合地板主要由 B 半 + A 半本金提供，留存仅支撑 A 半地板，1,000 基线不变（地板 +2.15% 承诺保持）。
- **决策**：50/50 保持裁定；40/60 变体（B 半 60%，期望 +0.5%，地板 ~1.5%）供业主选择。
- **理由**：凸性要便宜时买（IV 定价）；自托管是场所风险的唯一分散；审计的职责 = 把执行细节推回决策文本的意图。

## D-028 市场事实知识库（facts.md）+ 数据闭环
- **背景**：业主问——方案是最优还是需接入真实数据验证？是否缺知识库？
- **决策**：①方案**架构**在当前市场 regime 下最优（50/50 结构对 regime 稳健），但**执行决策必须接入真实数据**（6 个活数据点：CEX 定期档利率 / funding 30 日滚动 / TRX funding 正负 / BTC·ETH IV / 逆回购时点 / USD·CNH）——载体 = 监控 v1 + 事实库，此即 D-020"监控先行"的本意；②新建 `docs/handoff/facts.md`（T1 事实层）：每事实带 值/核实时间/来源/状态，更新规则 = 新数据到 → 旧事实标"已过期"不删除；③数据闭环：监控采集 → facts.md 更新 → 阈值触发器比对 → 决策层新 D# → 执行；④知识体系三层完备：事实层（facts.md）+ 决策层（decisions.md）+ 方法层（practices.md）；⑤决策前先查库，库中无 → 先核实后写入（省 token + 防过期数据）。
- **理由**：事实恰好存一处（P3）；带时点与状态防过期（P4）；知识库 = "监控变化 → 知识扩张 → 方案演进"的落盘载体。

## D-029 监控系统架构：采集→归一→规则→告警→留痕 管线
- **背景**：业主要求程序化设计（方法/架构/语言/数据库）。
- **决策**：①栈（D-011 重申）：Go 核心 + PostgreSQL（独立容器 arbcn-postgres:5434，存储层接口封装）+ React 19/TS + ConnectRPC 单二进制 :50052 + log/slog + 自实现调度；②五大设计决策：Collector 插件化（知识库扩张 = 加 collector 不改核心）/ 统一事实模型 Fact{kind,venue,symbol,value,ts,src} / 规则声明式（配置行非代码）/ 无密钥硬边界（公开只读 API，资金动作永远人工）/ 告警状态机（armed→active→resolved 状态转变才推送）；③数据源：交易所公开 API / DefiLlama / 东财·新浪公开行情 / 日历规则+人工表 / Deribit·OKX 公开 IV（受阻降级人工）；④交付：M1 核心（collector+Fact+规则+告警+最小仪表盘）→ M2 闭环（RMB 折算 + facts.md 自动导出 + 台账起步）→ M3 增强（跨所费率差 + 出入金台账 + IV 期权预算）；⑤可靠性：systemd + /healthz + 元监控（监控死了本身就是 critical 告警）；⑥告警通道：邮件 SMTP 默认 + 微信 Server酱 可选（国内 Telegram 不可达）；⑦明确不做：交易执行/密钥/回测/多用户/移动端。
- **规格**：docs/design/02-monitor-architecture.md（施工权威文档）。
- **理由**：管线化 = 每个环节可独立测试与替换；插件化 + 声明式 = 知识库扩张不引发重写；无密钥 = 合规姿态与安全面双收。

## D-030 语言裁决维持 Go + 客户端形态（web 先行 / PWA 后置）
- **背景**：业主问"为什么选 Go，Rust 不是更高效吗"；答复三问：①客户端 = 自建移动 app 或 web 先用着；②部署同意；③里程碑同意。
- **决策**：①语言维持 Go——Rust 性能更高效属实，但本系统瓶颈是网络 I/O（1–5 分钟轮询 + 规则求值）非 CPU，Go 性能超出需求约两个数量级；真实约束 = 开发速度、施工 agent 出错率、多 agent 交接成本（arb 同栈模式复用）。Rust 借用检查/异步复杂度使开发慢 2–3 倍且交换不到可感知收益。未来若出现微秒级高频执行层需求，另立 D# 重议 Rust。②客户端形态：web 仪表盘先行（M1，浏览器）→ PWA 化（M2，手机主屏可装+推送，零第二代码库）→ 原生 App 挂起（触发条件另立 D#）。③部署本机 systemd、里程碑 M1→M2→M3 确认通过。
- **理由**：选语言对齐瓶颈而非性能信仰；个人工具客户端 80/20 = PWA。

## D-031 Binance API 地域封锁（451）处置策略
- **背景**：M1-d 复测时 fapi.binance.com 返回 451（地域封锁，curl 直连同），M1-c 实测时尚通——间歇性/新上封锁。
- **决策**：修复顺序——①换公开行情专用域 data-api.binance.vision（官方公开数据域，折入 M1-h 联调试修）；②失败则补 Bybit/HTX funding 源（OKX 已在，费率监控底线 = ≥2 所，跨所费率差不依赖 Binance）；③Binance Earn 定期利率改人工补录（manual 通道每周）；④代理为最后手段，另立 D# 才可上。
- **实证修订（2026-08-15 M3-b S4 部署验证）**：①前提不成立——data-api.binance.vision **不镜像 /fapi/\***（fundingRate/klines 均 404，api/v3 现货域正常）；且 fapi.binance.com 在部署机直连 200、365d fundingRate 历史深度可用（**451 封锁在部署机未复现**，D-031 判定为间歇性）。**修订**：历史 funding 回填数据源默认取 `fapi.binance.com`（`BinanceHistoryBaseURL` 配置项保留，供未来 geo-block 覆盖）；实时 collector 本就是 fapi 域，实时+历史统一为同一数据源。**OKX 端点修正**：历史 funding 正确端点是 `/api/v5/public/funding-rate-history`（`funding-history` 返回业务 404）；OKX 仅保留 ~90d 历史（365d 窗口部分覆盖，回填实测 min_ts=2026-05-14，已文档标注 degrade）。
- **理由**：公开数据可用性波动是常态（M1-c 实测通、M1-d 实测 451），采集层必须多源容错；无密钥原则下换域/补源都是低风险动作，代理涉及合规姿态变化需显式决策。数据源端点假设一律部署机实测后采纳（先核实再采纳 D-028）。

## D-032 SMTP 配置非法降级运行（修订 M1-h 行为）
- **背景**：M1-h 回报——SMTP 配置了但非法时 Alerter 校验失败 → 进程退出 → systemd 重启循环。与"监控自身可用性优先"（dialogue #22：PG 不可达只 warn 不崩）矛盾。
- **决策**：SMTP 配置非法 → Alerter **降级禁用 + warn 日志，不退出进程**；告警行留在 alerts 表排队（配置修正后补投）；"SMTP 未配置/非法"列入元监控 degraded 面。
- **理由**：邮件投递失败不该拖垮监控主循环——错过行情窗口的代价 > 邮件晚到；与 PG 降级裁决同口径。

## D-033 通知通道变更为浏览器铃铛 + M2 范围定稿（M2-a/M2-b）
- **背景**：M1 收口后业主问"接下来做什么，给币安 API key?"——澄清系统为无密钥设计（公开只读 API，资金动作永远人工），业主**无需任何 API key**；随后业主确认不做 SMTP 邮件推送，改为"页面做个小铃铛，点开看通知"（浏览器内通知中心），要求 M2 规格定稿。
- **决策**：
  ① 通知通道：**浏览器铃铛通知中心为主通道**（未读计数徽标 + 抽屉告警列表 + 逐条/全部已读）；SMTP 实现保留但降级为可选通道（不申请授权码、不做真实投递验证）；**业主 SMTP 授权码待办移除**。系统永不接入交易/API key（无密钥铁律重申，D-010/§13）。
  ② M2 拆两阶段：**M2-a = 通知中心 + 源 freshness 徽标 + 连续重复事实去重**（后两项为今日生产实证：周六闭市 fx/repo 报价冻结但采集器健康，心跳用轮询时刻故元监控不误报，真实缺口 = 展示层无法区分"闭市"与"源死"，且周末产生大量相同 (value,ts) 噪声行）；**M2-b = RMB 折算 + facts.md 自动导出 + 台账起步**（D-029 原定）。
  ③ D-032 遗留两项随通道变更修订：SMTP 状态入 degraded 面 → 改为"通知中心可用性（alerts 表可达）"与"源 down 状态"入 degraded 面；smtp_configured 语义修正 → SMTP 为可选通道，保留 warn 即可，不再强制。
  ④ 部署：systemd 常驻已完成（mluser 运行，unit 模板同步）。下一步 = M2-a 派发施工。
- **理由**：通知要触达"用户看得见的地方"——页面内铃铛打开即见、零外部依赖、零授权码摩擦，符合单人工具定位（D-030 客户端形态）；freshness 徽标把"闭市 vs 源死"编码为可机械检查的状态，直接服务 D-028 数据闭环；去重降周末噪声，相同 (value,ts) 无信息量、跳过为纯收益。

## D-034 M3 模拟执行验证：方向批准 + testnet key 豁免条款（先细化设计后动工）
- **背景**：M2-a 交付后业主提出方向变更——「提升到自动交易（手动确认），毕竟这些都是理论上的，可以用模拟账号做验证，特别是加密这一块」。业主澄清"自动交易"实为**信号→建议订单→人工一键确认**的混合形态。
- **决策**：
  ① **方向批准**：监控信号长期停留"理论"未经实证，模拟账号验证 = 先核实再采纳（D-028）落地，正确。但**范围升级须独立里程碑 M3**，不掺 M2。
  ② **testnet key 豁免条款（修订 D-010 适用范围）**：模拟盘 testnet key **允许**，硬性条件——只连模拟节点（Binance Testnet / OKX Demo）；独立配置文件与真钱路径**物理隔离**；配置项显式标记 `SIMULATED`；**真金执行路径维持零密钥不变**。
  ③ **不赌原则（D-019）重申**：执行器只允许建议套利/息差类动作；无对冲的方向性建议**拒单**。风险门禁（单笔/日最大、未对冲拒单、超阈值拒绝）内建于订单生成器。
  ④ **加密模拟盘选型**：**Binance Testnet + OKX Demo 都接**（业主定），交叉验证不同交易所执行差异。
  ⑤ **M3 三步**：M3-a 订单生成器（信号→建议订单，本地模拟盘回填）→ M3-b 模拟对账（testnet 只读接入 + 模拟持仓跑息差收敛）→ M3-c 一键确认 UI + 风险门禁闭环。真金执行**不在** M3 范围。
  ⑥ **顺序**：**M2-b 先**（RMB 折算 + facts.md 导出 + 台账），给 M3 提供 RMB 对账基准；M3 在 M2-b 后开工。
  ⑦ **细化设计先于动工**：本方向先写详细 spec（docs/design/04-m3-spec.md：风险门禁口径、对账基准、testnet 隔离实现、验收标准），**动工前再确认一次**。
- **理由**：业主判断"理论需实证"正确且是套利系统的必经之路；但自动执行是所有风险源中最大的，模拟盘是唯一既验证理论又不触真金风险的路径；无密钥铁律保护的是真钱，testnet 隔离豁免不扩大真钱风险面；先 spec 后施工保持 讨论→决定→执行 顺序不跳。

## D-035 M1/M2-a 追溯深审 + M3-a 施工复审（2026-08-15）
- **背景**：业主指示——"所有文件，你做了审计吗？你要负责分派 agent 上交过来的代码复审工作"。决策层派 6 路只读 review agent（R1–R6）并行审全部 M1/M2-a 代码，逐条亲手验证（读代码确认）按实际影响定价；M3-a 施工交付另派独立 review agent 复审。
- **追溯深审修复（M1/M2-a，全部闭环）**：
  - **高危 5**：R6#1 rmb 刻度混用（`AnnualizedRMBAppreciation` 改回返回百分点点数）；R1#1 dedup 吞 sink 失败（next 成功后统一记 last，失败不推进）；R2#1 Seed 覆盖 DB 人工编辑（`ON CONFLICT DO NOTHING` + 回退 SELECT）；R3-H1 fx `parseQuote` 越界（<18 拒）；R5#1 healthz 启动失败谎报（`BootErr` → 503）。
  - **中危 6**：R4#1 heartbeat 泄漏进快照投影（ListFacts 排除）；R4#2 stale 阈值抖动（2×interval，±10% 抖动余量）；R5#2 pre-commit 整文件删除绕过守卫；R6#2 ack 竞态（前端 `ackVersion` 丢弃在途旧 poll）；R3-M2 sched 超时 10s→35s（≥ collector http client 超时）；R3-M3 manual 单位口径（`kindDefaultUnit` 缺省填充 + 冲突 400）。
  - **低危 3**：R5#6 http.Server 读写超时；R6#4/R4#3 ledger 金额校验（Amount==0 / FeeRate 有限拒绝）；R4#7 ListSourceHealth 时钟注入（`s.now()` + 测试注入）。
  - **接受/记录 12**：R1#3 venue 跨源、R4#4 SMTP TLS、R4#5 degraded 面欠账、R4#6 fx 序列、R4#9 heartbeat 内存、R5#4 maxFacts、R5#5 chg+offset、R5#7 cwd、R5#8 dist、R3-L1 bankrate、R3-L2 manual 无鉴权、R3-L3 thursday（待业主决策）。
- **M3-a 施工复审（独立 review agent，修复后达验收线）**：
  - **H1 修复（阻断）**：结算 PnL **100 倍放大**——`Per8hRate` 与 `RMBDayEnd` 把 pct_annualized 百分点点数当分数费率（缺 ÷100）；模拟盈亏失真，M3 验证目标失效。修：两处 ÷100 + 锚点测试改正确数值 + spec 标注刻度约定。
  - **M1 修复**：`ConfirmAndFill` 非原子（先置 filled 后逐条建腿，插腿失败留"filled 但缺腿"半对冲裸敞口 = D-019 违反 + 不可自愈）→ store 新增 `FillSimOrder` 单事务（`WHERE status='confirmed'` RowsAffected 守卫 + INSERT 全腿原子），顺带消除并发双插（L6）。
  - **M3 修复**：NaN/±Inf 绕过数值门禁（Go `NaN<x` 恒 false）→ `SignalToOrder` 有限性守卫 + 新标记 `INVALID_INPUT`；同批 L1 未知 kind 拒单、L2 负价差不被 FundingAnn 掩盖、L3 负日累计拒单（门禁加固）。
  - **M2/L4 接受**：`CarryWhite` 是驱动层信任边界，非可验证门禁（M3-b 接 testnet 前落显式白名单配置，spec 已标注）；迁移 CHECK/状态机跳转留 M3-c 加固。
  - **合规复核通过**：sim 包零网络、零密钥、纯本地（D-010 无密钥铁律）；六道风险门禁结构对诚实输入有效；RMB 刻度与 R6#1 兼容。
- **验证**：全量 `go test -race ./internal/...` 绿 + `go vet` + `npm run build` 绿 + PG 集成测试（含 FillSimOrder 原子性/状态守卫 4 场景）；线上部署（SIGKILL 重启，PID 2328862）healthz ok + migration 0005 applied。
- **结论**：M1/M2-a 追溯深审全部闭环（任务 #11/#12/#13/#14）；M3-a 修复后进入部署验证（任务 #15）。

## D-036 M3 文档审计：收敛口径修正 + 规格缺口修补（G1–G5）（2026-08-15）
- **背景**：业主问"M3 的文档审计过没有？是否为最优解？"。前次复审只审了代码相关章节（§3.2 结算、§4 门禁——H1/M1/M3 即出自此）；完整规格独立审计现补。审出：方向对（机制验证 = 最优解），但**收敛统计目标用 testnet 前向周级小样本回答不了**（统计问题用错工具），且规格有 5 处缺口。
- **决策**：
  ① **收敛口径修正（核心）**：M3-b 前向模拟**只验证机制**收敛（结算管线 + 双边价差行为观察）；**统计性结论**（是否真收敛/收敛速度/残差分布）由历史数据出——spec 新增 §5.3 历史收敛分析（Binance funding 历史 `data-api.binance.vision/fapi/v1/fundingRate`（D-031 公开域）+ OKX `/api/v5/public/funding-history` + 现货/永续价差历史，公开只读无密钥 D-010），为 M3-b 前置小任务，不另立阶段。§8"不做"修正：排除的是**交易策略回测 / 顺单历史回放**；历史数据回填 + 收敛统计是数据基础，须做。原 spec"收敛结论是 M3 主要交付物"与"❌ 回测引擎/历史数据回放"自相矛盾——本决策消除该矛盾。
  ② **G1 规则→Signal 映射与驱动**（最大缺口，M3-b 集成前必补）：spec 新增 §3.1.1 映射表——`funding_warn`/`funding_critical`/`trx_funding_positive` → `funding_hedge`，`reverse_repo_timing` → `repo`，白名单生息资产 → `carry_asset`；`defi_large_tier_change`/`ladder_trap`/`iv_opportunity`/`usdcnh_buy_line`/`collector_heartbeat` **不产生模拟单**（信息类/IV 非 M3 范围/遥测，宁缺毋滥）；映射为 sim 包内不可变常量表 + 对抗测试（未知规则 → 不建单）。§3.1.2 运行驱动：挂钩 `rule.Engine.OnActive`（armed→active 转变 = 机会出现时刻，避免持续满足重复建单噪声），OnActive 扩展携带命中实体列表（引擎本就逐实体聚合，自然小改）；流程 OnActive→查 LatestFacts→组装 Signal→SignalToOrder→InsertSimOrder；结算 8h 调度复用 collect.Scheduler 骨架。M3-a 交付成零侵入纯库、未接线的根因即此。
  ③ **G2**：`sim_pnl` 措辞对齐实现——迁移 0005 是 `sim_positions.pnl` 列，文档改口径（P3 单一真相源）。
  ④ **G3**：模拟资金基数（默认 100_000 模拟 USD）是**独立模拟量纲**，不映射真实组合规模；对账报告比例口径（PnL/Capital %）为主、绝对模拟值为次，明示不映射。
  ⑤ **G4**：定义理论无摩擦曲线——cash-and-carry 定价（永续价收敛现货、价差→0）、理想 funding 累计 = 预期年化 × 名义 × 天数 ÷ 365（每 8h 结算）、摩擦模型 = 手续费 + 滑点 + 现货腿资金占用。
  ⑥ **G5**：M3-c 确认成交价取**确认时刻最新 ref_price**（非生成时）；生成→确认窗口 ref_price 漂移 > 2%（或预期年化变化 > 20%）→ 二次门禁拒单（新标记 `SPREAD_DRIFT`，M3-c 实现）。
- **理由**：先核实再采纳（D-028）要求验证目标与方法匹配——统计结论只能来自有统计效力的数据（历史），机制验证才适合前向模拟；规格缺口不补，M3-b/c 施工会在集成时才发现"驱动没定义"（M3-a 已验证过这个坑）；P3 要求文档与实现口径一致。
- **结论**：M3 方向保持（机制验证 = 最优解）；收敛验证改为**前向验证机制 + 历史验证统计**双证据；G1–G5 全部落进 04-m3-spec.md（§0.1/§1.2/§3.1.1/§3.1.2/§3.2/§5.2/§5.3/§8）。STATE 施工表 M3-b 行更新：细化设计含 §5.3 历史回填前置任务。

## D-037 M3-b 细化设计定稿（施工权威 spec §9 + 结算数据源裁决）（2026-08-15）
- **背景**：D-034 ⑦"先细化设计后动工，动工前再确认一次"；D-036 后业主指示"排 M3-b 细化设计"。决策层摸清代码面（rule.OnActive 调用点 state.go:37、collect.Scheduler、sim 现有接口、store sim 接口、exchange collector、main.go startPipeline 接线）后定稿。
- **决策**：
  ① **结算数据源裁决**：模拟结算 funding = **真实市场公开 funding**（既有 binance_funding/okx_funding 采集，无 key），**不是 testnet**——testnet 费率有偏差（spec §5.1 自标），喂结算污染机制验证数字；testnet 只做 §2 key 隔离机制验证。D-034 ④"Binance Testnet + OKX Demo 都接"保持，仅明确用途。
  ② **S1 规则→Signal 驱动（G1 落地）**：`rule.Config.OnActive` 签名改 `func(ctx, r, entities []store.EntityHit)`（改点仅 state.go:37，matches 已在作用域；exporter.OnRuleActive 同步改忽略 entities）；新 `sim.Driver.OnRuleActive` 按 §3.1.1 映射表组装 Signal → Generate 落库；未知规则不建单（宁缺毋滥）；单次激活一单（OnActive 仅 armed→active 转变）。
  ③ **S2 8h 结算**：store.ListOpenSimPositions 扩展 venue 过滤 + sim.SettleFunding 扩展 (symbol,venue)——按 (symbol,venue) 分组结算，防 BTC@binance / BTC@okx 互相污染；结算值取 LatestFacts 真实 funding。
  ④ **S3 testnet 只读 + key 隔离（key 门控）**：SIM_* 独立文件 + `SIMULATED=true` 强校验（缺标记拒加载）；只读探针经 Heartbeat.Record 登记进 ListSourceHealth（复用 M2-a freshness 面）；零下单路径（domains_test 断言无主网交易域/下单域）；缺 key → 降级禁用，不阻塞 S1/S2/S4/S5。
  ⑤ **S4 历史收敛分析**：历史回填直接落 **facts 表**（kind=funding + 真实 ts；binance.vision `/fapi/v1/fundingRate` 翻页 + OKX `/funding-history` 分页；默认 365d `ARBCN_SIM_HISTORY_DAYS`；QueryFacts 幂等跳过已覆盖时段）；顺带让 funding_warn 的 avg_30d 立即有真实回溯（此前仅 1–3 天实时）；周频 sim_report（实际 vs 理论累计、残差分布、收敛半衰期、摩擦后净收益 vs 5% 门槛）。
  ⑥ **S5 白名单 + 降级**：Config.CarryWhitelist（`ARBCN_SIM_CARRY_WHITELIST`，默认空 = carry 被 WHITELIST 拒单直到显式配置，宁缺毋滥，M3-a 复审 M2 接受项落地）；sim 配置缺失 → 降级禁用（D-032 同口径 warn 不退出）。
  ⑦ **诚实标注**：系统无现货 collector，ticker 即永续价；funding_hedge 现货腿价取 ticker（basis/现货腿差留真实执行层），M3 只验证 funding 机制。
- **理由**：先核实再采纳（D-028）——设计全部锚定现有代码面（OnActive 单点改、复用 Scheduler/Heartbeat/facts 管线/domains_test 模式），无新架构发明；结算数据源选真实市场是 D-036"前向只证机制"的直接推论（数字诚实才有验证效力）；历史回填落 facts 表 = 单一机制双收益（收敛数据 + 规则回溯富化）。
- **结论**：M3-b 拆 S1–S5，spec 新增 §9 施工权威细化 + §5.1/§5.2 数据源修正；施工派工即按 spec §9。M3-c 后置（D-034 ⑤ 顺序不变）。testnet key 由业主提供，缺失 S3 降级不阻塞核心。

## D-038 M3-c 细化设计定稿（施工权威 spec §10 + 确认流裁决）（2026-08-15）
- **背景**：D-037 后 M3-b 全闭环（施工 + 复审 + S4 数据源修正部署验证）。业主问"M3-c 有没有文档 / 文档审计过没有"——核实：M3-c 文档只在 D-036 扫过规格缺口层（G5 口径），未到施工细化级（D-036 落点清单无 §6；STATE 曾误引不存在的 §9.9）。决策层摸清代码面（store sim 接口 / sim 包 ConfirmAndFill/状态机 / dashboard 10 RPC / 前端 3 tab / proto 工具链）后定稿 spec §10（C1–C5）。
- **决策**：
  ① **独立 SimService proto 域**（arbcn.sim.v1，新 `proto/` 目录 + buf.yaml）：4 RPC（ListSimOrders / ConfirmSimOrder / ListSimPositions / GetSimReport）。**不动 dashboardv1 生成物**——其 .proto 源缺失（只有 dashboard.pb.go / dashboard_pb.ts）且无 sim 域，硬改有反推风险；独立域零回归。工具链已验：protoc / protoc-gen-go / buf 在，protoc-gen-es 在 web devDeps，connect-go v1.20 / protobuf v1.36.11 对齐现有生成物。
  ② **SPREAD_DRIFT 二次门禁（G5 落地）**：新 `RiskSpreadDrift` 标记 + 纯函数 `ConfirmDriftCheck(genRef, genSpread, curRef, curSpread)`——ref 漂移 >2% 或 年化变化 >20% 各自独立触发拒单；**有限性 fail-closed**（确认重查价 NaN/零 → 拒，practices #7）；数据面 = 确认时刻 LatestFacts(ticker/funding) 重查，查不到 → 拒（从严）。
  ③ **确认成交 = store 层单事务原子 `AcceptSimOrder`**（suggested→confirmed→filled + INSERT 全腿，WHERE status='suggested' 守卫，RowsAffected 拦并发双确认）——替代"先置 confirmed 再 ConfirmAndFill"两步（practices #8 原子性：防"已确认未成交"悬挂 + 并发重复建腿）；confirmed 是事务内中间态，外部只见 suggested/filled。拒单走新 store 方法 `RejectSimOrder`（原子置 rejected + risk_flags 追加 SPREAD_DRIFT）。
  ④ **RMB 口径区分**：持仓 PnL = 模拟 USD 绝对金额 → **即期汇率折算**（USDCNH 事实直接乘）；**非** RMBDayEnd 年化口径（那是费率折算，H1 刻度线）。汇率缺失 → 显示 USD 原值 + 标注。
  ⑤ **PnL 只显示已结算累计**（settleOnce 每 8h）+ 最新 funding 年化标注；不做未结算实时估算（范围蔓延，M3 只验证机制）。
  ⑥ **可检查性**：domains_test 增 simapi 包无真实账户/下单端点（grep 断言）；ConfirmSimOrder 是唯一写路径（无自动确认定时器）；SIMULATED 徽标前端固定渲染（可 grep）。expired 状态默认不触发（避免时间窗复杂度）。
- **理由**：先核实再采纳（D-028）——设计全部锚定现有代码面（复用 store sim 接口 / sim legs 组装 / rmb 包 / dashboard RPC 模式 / domains_test 模式），无新架构发明；proto 独立域规避源缺失的回归风险；原子确认是 M3-a 复审 M1（成交原子）的同类不变量在人工流上的延续；RMB 即期口径避免 H1 刻度错位重演。
- **结论**：M3-c 拆 C1–C5，spec 新增 §10 施工权威细化（含 proto 定义全文 + RPC 签名 + 门禁口径 + 对抗测试锚点）；施工派工即按 spec §10。D-036 G5 从"口径定义"升格为"可施工规格"。

## D-039 二次门禁数据面 kind 分派（repo/carry 恒拒修复）（2026-08-15）
- **背景**：M3-c 施工交付后决策层复审（先核实再采纳 D-028），发现 spec §10.3 的二次门禁数据面存在**设计缺口**：§10.3「确认时刻 LatestFacts(ticker/funding)，查不到 → fail-closed 拒」是 funding_hedge 语义，但 handler 层对**所有 kind 硬编码双查 ticker/funding**——repo 无 ticker（面值锚 100，无价格行情）、carry 无 funding（稳定币生息无资金费率），导致 **repo/carry 确认恒拒**（M3-c UI 对这两类订单不可用）。施工 agent 未自行决定，显式挂起（对话 #46）。
- **决策**：`ConfirmDriftCheck` 纯函数签名不变（§10.3 锚点稳定）；数据面改为**按 kind 选权威数据源**（`Service.confirmDrift`，service.go）：
  ① **funding_hedge**：ticker → curRef、funding → curSpread，双查缺一 fail-closed 拒（原样）。
  ② **repo**：ref = 面值锚（curRef=genRef，漂移恒 0）；spread = `KindReverseRepo` 当日逆回购利率（与生成侧 repoSignal 同权威源）；查不到 → fail-closed 拒。repo 真实漂移风险 = 利率变化（5→6.5 即 +30% >20% 拒）。
  ③ **carry_asset**：spread = `KindDefiRate` 生息年化（权威源）；查不到 → fail-closed 拒。ref = ticker 有则查（稳定币现价漂移 >2% 拒），无 → curRef=genRef（面值锚 1.0 漂移恒 0，跳过 ref 检查——稳定币无方向风险，核心漂移是生息年化，白名单已对冲）。
  ④ 未知 kind → fail-closed 拒（与 SignalToOrder L1 同口径）。
- **理由**：fail-closed 语义**保持不放宽**——每类订单的**权威源**查不到 → 拒（宁缺毋滥，与生成侧同口径），只是把"权威源"从"ticker/funding"修正为按 kind 的正确事实源（reverse_repo / defi_rate 本就存在且是生成侧权威）。repo/carry 恒拒 = 确认流功能残缺，不是"从严"；不赌原则（D-019）要求 repo 天然无方向敞口，其唯一漂移风险（锁定利率变化）已由 reverse_repo 检查覆盖。
- **结论**：spec §10.3/§10.4/C2 表格同步更新；service.go `confirmDrift` + 7 个新对抗测试（repo accept/reject/fail-closed + carry accept/ticker-drift/spread-reject/fail-closed，删 kind 分派必红）。全量测试 + vet 绿。

## D-040 SimExec 测试网账户区（探针余额持久化 + RPC + UI）（2026-08-16）
- **背景**：业主提问「不显示两个账户的模拟资金和账户信息？」——S3 探针只做连通性验证（余额 body 丢弃、仅 Record heartbeat），测试网账户区数据面缺失。业主选型「SimExec tab 加测试网账户区」（对话 #49）。
- **决策**：
  ① **探针 Run 返回余额快照**（`probe.Run(ctx) ([]store.TestnetAccount, error)`）：成功路解析余额返回 + 照旧 Record；失败路仍返回聚合错误（独立判断语义保持），调用方按快照持久化、按错误 warn（D-032 同口径）。零下单路径不变。
  ② **快照持久化**：新表 `sim_testnet_accounts`（migration 0006，source 主键 upsert，details JSONB）→ `store.UpsertTestnetAccount`/`ListTestnetAccounts`。
  ③ **equity_usd 口径因 source 而异（诚实标注，前端明示）**：OKX = `totalEq`（交易所精确折算）；binance = 稳定币（USDT/USDC/BUSD/FDUSD）合计**近似**（无行情折算非稳定币，非全量净值）。不做多一层的行情估值查询——测试网账户区用途是"目测虚拟资金"，稳定币合计 + 原值余额已够；全量净值若要精确是另一数据面（未来 D#）。
  ④ **接线**：main.go 启动即探针一次（不等 8h tick，账户区立即有数据）+ 8h tick 刷新；sim.proto 加 `GetTestnetAccounts` RPC（独立域 codegen）；SimExec.tsx 加测试网账户区（SIMULATED 标注 + 每账户卡：权益/别名/更新时间/资产明细表）。
- **理由**：与 S3 定位一致——testnet 只做 key 隔离 + 连通验证；余额展示是连通验证的自然延伸（数据已返回，只是此前丢弃）。诚实口径优先：binance 无行情估值就不编造全量净值（宁可标"近似"）。
- **结论**：migration 0006 + store 两方法 + probe 解析（binance 稳定币近似 / okx totalEq）+ GetTestnetAccounts RPC + main.go 启动探针持久化 + SimExec 账户区 + 对抗测试（删解析/删合计必红）。部署实测：两路真实虚拟资金返回（binance equity 10000=USDT+USDC；okx totalEq 80673.55，BTC 1/OKB 100/USDT 5000/ETH 1）；ListSourceHealth 首次 heartbeat 已登记；全量测试 + vet 绿。

## D-041 模拟盘 funding_drill 演练档（业主选型 · 对话 #54/#55）（2026-08-16）
- **背景**：业主问「模拟执行一直没有开仓机会？」——核查确认这是设计内行为（sim_orders=0、funding_warn/critical 从未激活；BTC/ETH funding avg_30d 实时 3.4–6.7% 远低于 15%/20% 门槛；SPREAD_LOW 5% + carry 白名单空 = 双重宁缺毋滥）。业主经三选一选型：**「降门槛演练档（funding ≥5%）」**——让当前真实可交易的 BTC cash-and-carry ~7% 进入模拟盘，补上部署时缺的「真实 suggested 订单端到端冒烟」（确认→成交→8h 结算）。
- **决策**：
  ① **新规则 `funding_drill`**（defaults.go）：`kind=funding, symbol=BTC,ETH, cond=avg_30d > 5 && avg_30d < 15, level=Info, interval=300`。band 下限 5% = 跨过 SignalToOrder SPREAD_LOW 门禁；上限 15% = funding_warn 门槛，避免与真实窗口档重复出单（drill 在 ≥15% 自动 resolve、warn 接手）。上限 15 与 funding_warn 同库同源（rules 表，改一须改二，注释明示）。不新建平行机制（沿用 S1 规则→Signal 驱动，A 原则复用）。
  ② **映射**：`signalMappers["funding_drill"] = fundingHedgeSignal`（driver.go），复用 funding_hedge 组装（ticker 价即双腿，UNHEDGED/SPREAD_LOW/SPREAD_DRIFT 门禁全走）。
  ③ **告警耦合接受**：armed→active 转变必写 alerts 行（引擎无静默机制）→ funding_drill 触发发 **Info 级**「资金费率演练档」告警，视为特性（"演练机会出现"提示），不打扰（Warn/Critical 保留给 15/20 档）。
  ④ **不改 SPREAD_LOW/MinSpread**：5% 门槛保留；carry（sUSDe 4.34%）仍拒单（业主未选 carry 档，保持宁缺毋滥）。
- **理由**：业主明确要「真实演练全链路」而非保持空态；band 设计把演练档与真实窗口档无重叠解耦（Cond 层表达，DB 规则表单一真相源）；复用现有 fundingHedgeSignal + 门禁 = 最小改动、零架构漂移。诚实边界：演练单基于真实市场公开 funding（D-037 裁决），仍是 SIMULATED 不接真实资金。
- **结论**：defaults.go（规则+label）+ driver.go（映射）+ 对抗测试（TestDriverFundingDrillCreatesOrder 删映射必红；TestEachDefaultFiresOnSyntheticFacts 加演练档正例；integration_test Seed 10→11）+ spec §3.1.1 表。部署实测预期：okx BTC avg_30d=6.66% >5% → 重启后 funding_drill 激活 → sim 落 funding_hedge 建议单（BTC@okx），业主可确认成交走通全链路。全量测试 + vet 绿。

## D-042 演练单拒单根因修复：LatestFacts SQL 优先级 bug + 引擎 boot 竞态加固（2026-08-16）
- **背景**：D-041 funding_drill 部署后演练单**拒单**（sim_orders id=1/2 rejected，UNHEDGED，ref_price=-0.16/-0.23 负值）——本应「重启即触发建议单」，却连续两次被拒。逐层排查（xmin 事务序证实数据先落库、反汇编 + RPC 组合查询、inode 对比）最终锁定**非数据竞态、非旧二进制**：`pgstore.LatestFacts` 的 WHERE 子句**每条件缺括号**。
- **决策**：
  ① **根因 = SQL 运算符优先级**：`where := []string{"$1 = '' OR kind = $1", "$2 = '' OR venue = $2", "$3 = '' OR symbol = $3"}` 用 `AND` join 但**每个子句未加括号** → 实际求值为 `$1='' OR kind=$1 AND $2='' OR venue=$2 AND $3='' OR symbol=$3`，`AND` 优先级高于 `OR` → 多参数组合下退化为「只生效符号条件」（DB 实证：`ticker/okx/BTC` 返回 funding+iv+ticker 五行，首行 = funding@binance 负值）。`fundingHedgeSignal` 取 `fs[0]` → 拿到 funding 负值当 ticker 价 → `SpotPrice ≤ 0` → UNHEDGED 拒单。**修复**：每子句加括号 `($1 = '' OR kind = $1)` 等。
  ② **对抗测试**：`TestLatestFactsFilters`（pgstore 真库）——ticker/okx/BTC 必须只返回 ticker 行 + venue 单独 / symbol 单独 / 全空对称断言；删括号必红（已实证：bug 版返回 4 行 want 1）。
  ③ **boot 竞态加固**：`rule.Config.BootDelay`（默认 0 不改行为）——Scheduler 与 Engine 并行启动，collector 首轮 poll 落库可能晚于引擎首评，数据未到 → 规则首评空跑（funding_drill「重启即触发」不可靠）。main.go 接 15s。对抗测试 `TestRunBootDelay`（删 sleep 必红）。
  ④ **陈旧测试修正**：`migrate_test.go` want 5→6（D-040 加 migration 0006，测试陈旧非回归）+ 表清单补 `sim_testnet_accounts`。
- **理由**：P1 通道不变量——真相只能从可复现证据来；「旧二进制」假象来自 `stat` 不带 `-L` 读 /proc/PID/exe 返回 procfs 伪 inode，**经验教训**：对比 /proc/PID/exe 与磁盘文件须用 `stat -L`（跟随符号链接）。根因是**任何多参数 LatestFacts 调用都受影响**的潜伏缺陷（dashboard 多参查询 / simapi 确认流 / heartbeat 同步也踩），D-041 演练档首次在真实链路暴露。boot 加固为独立防御（虽非本次根因，但同属「重启即触发不可靠」家族）。
- **结论**：dashboard.go 括号修复 + TestLatestFactsFilters 对抗测试 + BootDelay + TestRunBootDelay + migrate 陈旧断言更新。部署实测：**sim_orders id=3 = suggested（okx BTC ref 63063.30 spread 6.64% risk_flags={}）**——演练单可确认→成交→8h 结算全链路闭环；拒单 id=1/2 保留为负样本。全量测试 + vet 绿。

## D-043 暂不接入 LLM（未来解读需求用模板叙述）（2026-08-16）
- **背景**：业主问「本项目有没有必要接入 LLM」。按项目第一性原则推导后给裁决，业主认可并要求落档（对话 #61）。
- **决策**：**暂不接入 LLM**。若未来出现"读盘解读/自然语言问答"需求，采用**模板化叙述**（把结构化事实拼成中文段落，先例 = sim_report 周频 markdown 模板），不引入 LLM 生成。方向锚定：任何 LLM 接入提案须先过本决策（走新 D# 推翻）。
- **理由**：① 违反「先核实再采纳」（D-028）+「可机械检查」（P4）——LLM 输出不可机械验证（幻觉），进不了门禁/对抗测试，等于引入无法核实的信息源；② 无信息增量——决策所需事实已结构化（facts.md / RPC / JSON），LLM 只是同数据降级成自然语言，只加出错面；③ 违背克制姿态——外部 API + 密钥 + 成本 + 网络依赖，与「零外部执行面、人工决策」定位相悖（D-019 不赌、D-010 无密钥铁律同源）。
- **结论**：方向记录在案。本次一并核实并落档 defi_rate 五项身份（`aave-v3/blackrock-buidl/ethena-usde/morpho-blue/ondo-yield-assets` = **DeFi 协议资金池**，DefiLlama yields 数据源，格式 `资产@协议`，非交易所；喂 carry 稳定币生息档 D-021 第二档；当前年化 3.55~12.57%，其中 **Aave USDC 12.57% 为借贷利率瞬时尖峰，异常值，实盘决策须按均值核实 D-028**；这些池仅入事实库，carry 白名单默认空 M3-b §9.6，未显式配置前 carry 订单被 WHITELIST 拒单）。

## D-044 进化建议引擎（L0 只读证据表面）（2026-08-16）
- **背景**：业主「不接 LLM，希望更智能化，随数据采集知识库能支撑更聪明的判断」（对话 #62）。进化回路五环（数据面/执行面/学习面/记忆面/方向面）里缺「半自动增强」环——系统只被动出告警/建议单，不主动把「该决策层关注什么」的**证据候选**摆出来。目标经决策层把关修正为**做「资金运营的领域专家」而非「通用智能」**：证据驱动的收敛，不引入 LLM 式发散（D-043 已锚定不接 LLM）。
- **决策**：
  ① **L0 四信号**（全部只读、可对抗测试）：`reject_dist` 拒单原因分布（risk/info）、`defi_anomaly:*` DeFi 利率异常尖峰（anomaly/warn）、`no_order` 连续无单提示（opportunity/info）、`source_down/stale:*` 数据源停更（data/critical|warn）。
  ② **RPC 归属 dashboard 域**：`ListInsights` 加到 `DashboardService`（已持 sim_orders/facts/sources/sourceHealth 全数据源，不加新包、不加新域）。
  ③ **按需计算（on-demand pull）**：四信号都便宜，不加表、不加后台循环；前端沿用 useSnapshot 的 60s poll（第七个并行 RPC）。
  ④ **异常检测 = 截面中位数 × 因子**（`value > 2.0×median`，样本 ≥3，NaN/±Inf 跳过 practices #7）：不引入 stddev；中位数对利率整体上行（regime shift）稳健——全员上涨时中位数同步上移不误报。
  ⑤ **取每 (venue,symbol) 最新 defi_rate 做截面**（30d 回看窗口）：瞬态尖峰在两轮采集间已回落的场景**不标异常**——L0 报「当前截面离群」，非「历史曾冲高」。已在实盘验证（Aave 12.57% 06:19 尖峰 06:42 已回落至 3.29%，最新截面平稳不误报）。「窗口内 max 冲高」语义留给 L1（数据累积后）。
  ⑥ **动作一律指向 D# 人工决策**：每条 insight 的 actions 只给「核实/评估→走 D#」候选，不落任何自动执行路径。
- **理由**：P4 可机械检查——纯函数 + 对抗测试（删判定/删计数必红）；与 D-019（不赌）/D-028（先核实再采纳）/D-043（不接 LLM）同源。只读证据表面：**决策永远手动**，引擎只把证据候选摆在决策层面前。
- **结论**：`internal/dashboard/insights.go` + `insights_test.go`（对抗）+ proto `Insight/ListInsights` + 前端「进化建议」卡（触发器下方整宽卡）+ docs（practices #20 / dialogue #62）。部署实测：`reject_dist`（UNHEDGED ×2/SPREAD_DRIFT ×1）、`source_stale` ×3（repo/fx/deribit_iv 周末低活跃源）正确出现；`no_order` 被近 7 天 filled 单正确抑制；`defi_anomaly` 因最新截面平稳正确不报。L1 候选：统计自校准（窗口 max 冲高 / 阈值自适应）、L2 归因智能（拒单-阈值联动）——留给数据累积后按本决策边界续。

## D-045 carry + repo 完整接入模拟盘（结算分派 + 门槛分档 + venue 对齐）（2026-08-16）
- **背景**：业主观察模拟盘**只真正跑通 funding_hedge（现货+永续）**——面板看到的都是「预期收益」，其他机会类没在模拟里产生数据。业主问「能否把其他机会都接入模拟」，业主已定：**carry + repo 都接**，carry 白名单 **SUSDE / USDE / BUIDL / STEAKUSDC / USDY（能做的都做上）**。调查定位三个真实缺口（非「门禁太严」）：① 结算数据面只查 `funding` 事实 → carry/repo 腿建仓后永不生息；② repoSignal 落单硬编码 `GC001/domestic`，事实真实存 `sina/GC001` → 结算永 miss；③ `MinSpread=5%`（funding_hedge 摩擦假设）误用于 carry → 当前 defi 利率 3~5% 全被拒。
- **决策**：
  ① **结算数据面按腿 kind 分派**（`settleFactKind(kind)`）：funding_hedge→`fact.KindFunding` / carry_asset→`fact.KindDefiRate` / repo→`fact.KindReverseRepo`；`settleOnce` 分组键 `(kind,symbol,venue)`；`SettleFunding` 加 kind 参数（按 kind 过滤腿）。
  ② **repoSignal 落单 venue/symbol 取事实真实值**（`fs[0].Symbol / fs[0].Venue`，不再硬编码 domestic/GC001）。
  ③ **carry 门槛按 kind 分档**：新增 `CarryMinSpread`（默认 1.0%，env `ARBCN_SIM_CARRY_MIN_SPREAD`），carry 用低门槛；funding_hedge 与 repo 保持 `MinSpread=5%`（repo 时点逆回购意图不变）。**1% 是纠正口径错配（funding 摩擦假设误用于持有生息），非放宽门禁造数据**。
  ④ **carry 白名单显式配置**：生产 env `ARBCN_SIM_CARRY_WHITELIST=SUSDE,USDE,BUIDL,STEAKUSDC,USDY`（事实库符号全集，大写；config.go 只 trim+去重、不做大小写归一）。
- **理由**：practices #13「数据面按实体类型分派」的结算侧漏项（D-039 修了确认二次门禁侧，结算侧还漏着）；「一类实体的门禁/数据面天然通用」= 隐蔽假设坑（#13/#4 同源）。carry/repo 均为 D-021 已定义档位，非新方向；结算仍用真实市场公开事实（非 testnet，D-037 同口径）。
- **结论**：driver.go（settleFactKind + settleOnce 分派 + repoSignal venue 对齐）+ backfill.go（SettleFunding 带 kind）+ config.go（CarryMinSpread）+ order.go（carry 门槛分档）+ 生产 env 白名单。对抗测试 3 个新锚点（删分派/改回硬编码/删分档必红已实证）+ config_test 扩展 + TestDriverRepoBuildsOrder 事实带真实 venue。全量测试/vet/-race 绿；部署重启实测：服务 active + 新 bundle served。**诚实标注**：carry 单是否真触发取决于 defi 池出现 ≥0.5%/h 变动（不造数据）；repo 当前 0.865% < 5% 会被 SPREAD_LOW 拒（负样本，符合时点逆回购意图）。

## D-046 机会实算卡 + 市场结构经验库（投运后系统自算账 · 经验资产自我进化）（2026-08-16）
- **背景**：业主核心追问（对话 #64）：「**投运后我不可能永远带着你，让你帮我算账——系统如何自己解决这个问题？**」并再次关联 LLM 疑问。裁决：我不在场时做的「算账」（瞬时 9.14% vs 30 日均值 4.16% / 0.3% 摩擦需 12 天保本 = 尖峰陷阱）**不是聪明，是公式**——可写成确定性计算机械执行，无需 LLM（D-043 已锚定不接 LLM）。业主进一步定方向：**新情况要「吸收成知识库、成为经验资产、随行情自我进化」**。
- **决策**：
  ① **机会实算卡（Feature 1）**：对每个实时机会（funding_hedge/carry_asset/repo）确定性算账——瞬时年化 / 30 日均值 / 保本持续天数（`f×365/Inst`）/ 扣摩擦净年化（30 日持有口径 `Avg30−f×365/30`）/ 三档判定（坑·打平·可抓）/ 中文模板叙述（D-043 模板化，非 LLM 生成）。纯函数 + on-demand RPC（`ListOppCards`，dashboard 域，镜像 insights.go 模式），无新表无后台循环。**判定基准 = 净年化 vs 稳定币基档 4.5%**（D-021 机会成本）：`RatingGrab` 净年化 > 4.5%（funding 15% 门槛扣摩擦后净 11.35% ≫ 4.5% → 可抓，与 D-016 一致）；`RatingTrap` 瞬时 > 2×均值（尖峰，与 D-044 defi_anomaly 中位数×2 因子同口径）或净年化 ≤ 0 或瞬时 ≤ 0；`RatingBreakeven` 介于之间。**摩擦 = 可配置常量（env `ARBCN_OPP_FRICTION_FUNDING` 默认 0.3）**，业主核实两交易所均普通主户（非 VIP、无抵扣）→ 0.3%（现货 taker 0.1%×2 + 永续 taker 0.05%×2）为**已核实值**（facts.md 落档），后续升 VIP/启用抵扣改 env 不改代码。carry/repo 摩擦 ≈ 0（持有生息无方向摩擦，标注 caveat）。结论对合理摩擦区间（0.1–0.5%）稳健。
  ② **市场结构经验库（Feature 2）**：新表 `knowledge_entries`（signature/venue/symbol/verdict/rationale/source/status/validated_at/validation_note）。**吸收 = 人工 + D#（seed 落盘，git 跟踪）**；**匹配 = 确定性签名纯函数**（`internal/knowledge` 包：`FundingSpikeTrap` / `DefiPoolSpikes` 中位数×2 / `CrossVenueDivergence` minSpread=4.0，可对抗测试）；**呈现 = 只读 `knowledge_match` insight（category=knowledge，severity=info，actions 指向 D#）+ 浏览 RPC `ListKnowledgeEntries`**；**验证 = 后续重匹配 + 人工复核 → 新 D# 更新 verdict/status**。**不落任何自动吸收/自动改规则路径**（practices #20 边界重申）。首版 3 条 seed（git 跟踪）：① `funding:spike_trap`（ETH@okx 9.14% vs 4.16% 尖峰陷阱，D-043/D-016）；② `defi:single_pool_spike`（Aave USDT 12.57% 单池尖峰已回落）；③ `funding:cross_venue_divergence`（TRX binance +2.3% vs okx −3.5%，对话 #50 已核实真实分歧）。
  ③ **实算卡是只读证据，不触碰任何门槛**：D-016 15%/20% 激活线、SignalToOrder 的 MinSpread/CarryMinSpread、carry 白名单**一律不动**。卡只说「这笔账扣摩擦后划不划算」，执行门禁仍由现有规则引擎 + 门禁把关。
- **理由**：映射 D-044「进化回路五环」尚未建成的**学习面/记忆面**（L1/L2 候选）；方向仍属「资金运营领域专家」收敛，非 LLM 发散（D-043）。实算卡 = 确定性公式直对应「扣摩擦算账」本质（投运后无我 在场也能算）；经验库 = 签名纯函数直对应「机械匹配」本质，吸收闭环 = 人工 D#（决策权最高，AGENTS.md §0），系统只匹配与呈现永不自动吸收（P4 可机械检查 + practices #20）。
- **结论**：`internal/dashboard/oppcalc.go`（纯函数）+ `oppcalc_rpc.go`（ListOppCards）+ `internal/knowledge/`（包：签名字典 + 探测器纯函数 + Defaults）+ migrations 0007 `knowledge_entries` + store 两方法 + `internal/dashboard/knowledge.go`（ListKnowledgeEntries + knowledgeMatches 信号 5）+ insights.go 接线 + 前端「机会实算卡」区块（Opportunity.tsx，评级徽标 grab=绿/打平=黄/坑=红 + 瞬时/30日均值/保本天数/净年化瓦片 + 中文叙述 + 摩擦明示）+「市场结构经验库」卡（KnowledgeBoard.tsx）+ Insights catLabel「knowledge→经验」+ hooks（snapshot 加 cards 第 8 路 RPC；useKnowledge 低频加载）。对抗测试 3 组新锚点（删实算公式/删签名匹配/删中位数因子必红已实证）+ 4 处 test fake 补 Store 新方法。全量测试/vet/npm build 绿；部署重启 + migration 0007 自动应用；实测 ListOppCards 返回 ETH@okx 尖峰陷阱卡（当前瞬时 8.82% vs 均值 ~4%）、ListKnowledgeEntries 返回 3 条 seed、ListInsights 出现 knowledge_match。**诚实标注**：经验库首版 3 条 seed 覆盖已知模式，新情况靠后续 D# 逐步吸收——这正是「自我进化」的机制（越用越多），非一次建满。

## D-047 前端第一性原则审计修复：数据层随视图生命周期 + 映射收敛 + 小项清理（2026-08-16）
- **背景**：业主指示「对前端每一个页面做第一性原则审计」（对话 #65）。审计结论：四页 + 共享层的页面组件本身全是干净薄渲染，问题集中在三层——**P0 根因**：数据 hook（useSnapshot/useFactsSnapshot/useSim/useKnowledge）绑 App 根生命周期而非视图生命周期 → 三症状（① 非当前 tab 数据仍 60s 轮询空转；② 顶部「刷新」只 reload useSnapshot，同页 ConfirmPanel/KnowledgeBoard 不动 = 刷新名不副实；③ useSim 无轮询，8h 结算新单在确认面板是旧数据）。**P1 双源映射（P3 违背）**：F1 store.SimKind 枚举两份中文映射文案不同（Opportunity.kindLabel vs sim.kindText）；F2 FactsSnapshot.tsx COVERED_KINDS 复制后端 rmb.CoveredKinds。**P2 小项**：F3 死分支 / F4 pnl_rmb 0 占位启发式 / F5 ledgerDate 归置错位 / F6 fresh-dot 重复 / F7 观察项。业主经 AskUserQuestion 拍板 **P0+P1+P2 全修**。
- **决策**：
  ① **P0 数据层随视图生命周期**：新增 OverviewPage/FactsPage/SimPage 三页面组件分页承载数据 hook，**hook 生命周期 = 视图生命周期**（切 tab 即卸载停止轮询）；App 根仅保留 useSnapshot（驱动 header 健康徽标/铃铛 = 全局 chrome，任何 tab 需要）。useSim 加 60s 轮询（8h 结算新单可被确认面板捕获）。顶部全局刷新 = `useSnapshot.reload` + `refreshKey` 状态递增 → 总览页 useSim/useKnowledge 依赖 refreshKey 联动重载。React hooks 规则（不可条件调用）使「页面组件分页」成为唯一正确形态（App 条件分支放不下 hook）。
  ② **P1 F1 收敛**：kindText（sim.tsx）为 SimKind 中文映射单一来源，Opportunity 删本地 kindLabel 改 import；新增 grep 锚点测试 `TestSimKindLabelCoverage`（镜像 TestSimExecBadgeRenderable，断言 sim.tsx 覆盖 store.SimKind* 三字面量 + Opportunity 不得含 funding_hedge 字面量——删覆盖/加回双源必红）。
  ③ **P1 F2 收敛**：covered 单一真相源**在后端**——proto `FactRmb.covered` + `rmb.Converted.Covered`（Convert 循环单点 `Covered: CoveredKinds[f.Kind]`）+ `toFactRmb` 映射；前端 FactsSnapshot 删 COVERED_KINDS 集合改用 `f.covered`。
  ④ **P2 F4**：proto `ListSimPositionsResponse.fx_available` presence flag（镜像 ListFactsResponse 既有模式），前端 SimExec 删 `every(pnlRmb===0)` 0 占位启发式，改 `fxAvailable` prop——**真零 PnL 不再被误标「USD 原值」**，汇率缺失时标注「USD 原值」是显式信号非启发式推断。
  ⑤ **P2 F3/F5/F6**：删 freshness.ts sourceForTile 的 fx/bank_rate 死分支；ledgerDate 移入 format.ts（归置工具函数）；抽 FreshDot 共享组件（Matrix/StatTile 复用）。
  ⑥ **F7（Alerts/Bell 行重复）保持**：告警时间线 vs 铃铛通知抽屉语境有真差异，合并引入的 prop 膨胀 > 消除的重复，D# 记 rationale。
  ⑦ **设计回归留痕**（§1 首行锚，改动走 D#）：对话 #59 曾把 useSim 提升 App 层共享（「确认后两处同刷新」）。本次下沉后，ConfirmPanel 确认 → 切 sim tab → SimPage 挂载即重拉最新，**天然覆盖原共享动机**，App 层共享不再必要——回归为「数据随视图」，不引入状态提升的隐式常驻。
- **理由**：视图不可见 = 数据不需要 = 空转，数据获取层应与视图生命周期对齐（P4 可检查：hook 卸载即清 interval 可 grep）；映射收敛直接对应 P3 单一真相源；presence flag 复用既有 ListFactsResponse 模式（A 原则复用，不发明新机制）；0-as-missing 反模式二犯（practices #7/#23 同族），缺失信号必须显式 flag。
- **结论**：三页面组件 + hooks 下沉/轮询/refreshKey + proto 两字段（FactRmb.covered / ListSimPositionsResponse.fx_available）+ buf generate + 映射收敛 + 小项清理 + FreshDot。对抗测试 TestSimKindLabelCoverage（红→绿已实证：写测试时 Opportunity 仍持 funding_hedge 字面量必红，删后转绿）。全量 go test（含真库 DSN）/vet/npm build 绿；部署重启（SIGKILL 既有模式）+ 实测 ListFacts covered 18/32、ListSimPositions fx_available=true、served bundle 匹配新 dist（index-c9bH9JKr.js）。
