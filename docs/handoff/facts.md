# 市场事实库（facts.md · T1 知识层）

> **单一事实源（P3）**：市场/合规/渠道事实恰好存于此处；决策前先查库，库中无 → 先核实后写入。
> **每事实**：值 / 核实时间 / 来源 / 状态（现行 · 监控中 · 已过期·被X取代 · 已排除）。
> **更新规则（P4）**：监控新数据到 → 旧事实标"已过期（被新值取代）"，不删除；每事实带 D# 或 dialogue# 锚点。
> **数据闭环（D-028）**：监控采集 → 本库更新 → 触发器比对 → 决策层新 D# → 执行。
> 预算：≤450 行；超量最旧滚动归档 LOG.md。

## 1. 国内利率（2026-08 核实）

| 事实 | 值 | 核实 | 来源 | 状态 |
|------|-----|------|------|------|
| 国内无风险利率带 | 1–2% | 2026-08-15 | 多源综合 | 现行 |
| 国债逆回购平时 / 时点 | 1.5–2.5% / 月末季末冲 5–6%（2026 Q1 末 4.2%） | 2026-08-15 | 券商/媒体 | 监控中 |
| 民营定期：富民 3 年 | 2.15%（存款保险 50 万内全保） | 2026-08-15 | 媒体 | 现行（执行前复核） |
| 民营定期：新网 3/5 年 | 1.9% | 2026-08-15 | 媒体 | 现行 |
| 大额存单门槛 | 现行 30 万起（征求意见稿拟降 20 万，未落地） | 2026-08-15 | 央行/媒体 | 触发器 |
| 大额存单 3 年利率 | 1.55–2.1%（民营更高） | 2026-08-15 | 融360 | 现行 |
| 储蓄国债 | 3 年 1.63% / 5 年 1.70% | 2026-08-15 | 财政部 | 现行 |
| 现金管理类理财 / 货基 | 1.13–1.25% / ~1.05% | 2026-08-15 | 理财周报 | 现行 |
| 固收+理财 | 业绩基准中位 2.35%，头部 3.03% | 2026-08-15 | 理财周报 | 现行 |
| 结构性存款实际兑付 | 1.7–2.2%（< 民营定期，不配） | 2026-08-15 | 上市公司公告 | 已排除 |
| 大行定存 | 1 年 0.95% / 5 年 1.30% | 2026-08-15 | 媒体 | 现行 |
| 华夏全对冲互认基金（968201/968202） | 2025 +4.28% / 2024 −2.11% / 近 1 年 +1.73% / 成立以来年化 ~1.45% / 回撤 −2.92% | 2026-08-15 | 基金公告 | 现行（D-018 采纳） |

## 2. 美元与汇率（2026-08 核实）

| 事实 | 值 | 核实 | 来源 | 状态 |
|------|-----|------|------|------|
| 境内美元定存 | 民生 3.1% / 中信 3.0% / 外资活动 3.2–4.1%（汇丰 3 个月 4.1% 新资金） | 2026-08-15 | 银行公告 | 现行（D-019 后仅作参考） |
| 工行/中行美元定存 | 挂牌 2.8%（5000 美元以上）；以下 0.8–0.85% | 2026-08-15 | 21 财经 | 现行 |
| 美元现金管理理财 | 均值 3.37%（九成 3–3.5%） | 2026-08-15 | 21 财经 | 现行 |
| 美元封闭式理财（建行） | ~4% | 2026-08-15 | 21 财经 | 现行 |
| QDII 美债基金（人民币份额） | 2026 YTD −1.5%~−3.4% | 2026-08-15 | 天天基金 | 现行 |
| USD/CNH | 6.74–6.76；中间价 6.7878；2026 YTD 升值 3.4% | 2026-08-15 | 汇市报道 | 监控中 |
| 机构汇率研判 | H2 看 6.7–7.0，稳步升值、空间受限 | 2026-08-15 | 中金/建行/星展 | 现行 |
| 美联储利率区间 | 4.00–4.25%；9 月加息概率 67.2%（市场定价）；加息预期升温 | 2026-08-15 | 汇市报道 | 现行 |

## 3. 加密市场（2026-08 核实）

| 事实 | 值 | 核实 | 来源 | 状态 |
|------|-----|------|------|------|
| BTC/ETH funding 年化 | +5~11%，情绪中性 | 2026-08-15 | skyemeta/CoinGlass | 监控中（门禁 >20%） |
| BTC/ETH/TRX funding 实测（arbcn live） | **跨所分化**：TRX Binance −1.57% vs OKX +1.71%；BTC Binance 6.84% vs OKX 8.38%（2026-08-15 18:14 快照） | 2026-08-15 | arbcn 监控系统 live 采集 | 现行（M3 跨所费率差素材） |
| TRX funding | 曾负费率 ~−12%（Perpfinder 快照，已过期）；波动极大且跨所分化，以实时监控为准 | 2026-08-15 | Perpfinder（过期）/ arbcn live | 已过期（被 live 取代） |
| Binance API 可达性 | fapi.binance.com 间歇 451；data-api.binance.vision 仅 spot /api/v3（fapi 404）→ 保留原域（D-031） | 2026-08-15 | arbcn 实测 | 现行 |
| CEX 稳定币阶梯陷阱 | 头条 8–10% 仅 200–300 USDT 小档；超额档 1.6–2.2% | 2026-08-15 | BlockBeats | 现行（金额档原则） |
| **交易所执行摩擦（funding_hedge 双开双平）** | **0.3%**（现货 taker 0.1%×2 + 永续 taker 0.05%×2）；**两交易所均普通主户费率（非 VIP、无抵扣），业主核实** | 2026-08-16 | 业主核实（对话 #64，D-046） | **已核实 · 普通主户**（D-046 机会实算卡默认摩擦；后续升 VIP/启用 BNB 抵扣 → 改 env `ARBCN_OPP_FRICTION_FUNDING` 不改代码） |
| **funding 数据源范围（arbcn live）** | **维持 binance + okx 两家，不加所**：两所已覆盖业主可交易面（普通主户已核实）；加第三所不改「流动性币 funding 极少过 15% 门槛」基本面（极端 funding 全在微盘陷阱币，对话 #52 实证）→ 加所 = 负期望投入（新 collector + 部署机端点实测 practices #12 + freshness 故障面）。**值得扩的是标的维度**（SOL/XRP 等，须两所均有现货+永续可对冲 + 实证过门槛，受宁缺毋滥约束）。例外触发 = 跨所费率分歧在流动性标的上反复命中且业主确证可套 → 才考虑加第三所（Bybit，2 所 = 每币 1 分歧对 / 3 所 = 3 对） | 2026-08-16 | 裁决（对话 #67，D-049） | 现行（数据源边界） |
| 稳定币大额档 | Binance Earn 定期 USDT 5.8%/USDC 4.5%；Bybit 4.8–5.5%；OKX USDT 5.0% | 2026-08-15 | StableLens | 现行（执行前复核） |
| DeFi 稳定币 | Aave USDC 4.67%；Morpho 4–6.5%；sDAI 5–8%；sUSDe 8–11.8%（高风险档） | 2026-06/07 | StableLens 类 | 监控中（均衡卫星） |
| 代币化美债 | BUIDL 3.4%；USDY/BENJI 4–5% | 2026-06 | StableLens | 现行（自托管备选） |
| TRX 质押 | 链上基础 3.2%；能量/投票组合 6–10%；活动档 20%+ 促销 | 2026-08-15 | MEXC/SafePal | 监控中（业主持仓优化） |
| Babylon BTC 质押 | 基准 0.03–1%；收益以 BABY 计价（自高点 −92%） | 2026-08-15 | 币安公告/社区 | **已排除**（D-024 陷阱案例） |
| Launchpool | 2026 Q1 BNB 池实测 19–22% APY；稳定币池份额仅 10–15% | 2026 Q1 | traderabyss | 监控中（日历） |
| BTC 价格 | ~$63.5k（8/12） | 2026-08-15 | inflowscan | 监控中 |
| DeFi 风险前例 | 2026-04 Kelp 桥漏洞 → Aave 坏账 1.96 亿美元 | 2026-08-15 | 媒体报道 | 现行（风险基准） |

## 4. 合规与制度（现行）

| 事实 | 值 | 核实 | 来源 | 状态 |
|------|-----|------|------|------|
| 42 号文（银发〔2026〕42 号） | 2026-02-06 八部门；个人投资虚拟货币 = 民事无效、损失自担、法律不保护；废止 237 号文；RWA 可备案合规 | 2026-08-15 | 央行官网 | 现行（灰色层门禁依据） |
| 个人购汇额度 | 5 万美元/年 | 现行 | 外汇局 | 现行 |
| LOF 退市新规 | 2026-08-07 征求意见稿：QDII/商品期货 LOF 最晚 2027-12-31 退市 | 2026-08-15 | 沪深交易所 | 现行（D-015 依据） |
| 存款保险 | 50 万内本息全额偿付 | 现行 | 存款保险条例 | 现行 |
| 两得宝准入 | 在售但 R5/C5 激进型门槛 | 2026-08-15 | 中行官网 | 已排除（画像冲突） |
| 北交所打新 | 中签 0.02–0.03%；稳中 1 手需冻结 500 万+ | 2026-08-15 | 申万/开源研报 | 已排除（D-015） |

## 5. 业主渠道现状（#17）

| 事实 | 状态 |
|------|------|
| Binance 账户 / OKX 账户 / 国泰君安账户 | 已有 ✓ |
| 富民银行 | 未开（线上开户 5 分钟，2.15%） |
| TRX 质押持仓（~5 TRX/天） | 已有，独立于 20 万 |


<!-- ARBCN-EXPORT-BEGIN -->
## 监控快照（arbcn 自动导出 · M2-b §5 / D-028 闭环 · D-066 封顶）

> 机器生成：监控最新值渲染进事实库；新快照到来 → 旧快照标「已过期」，段内只留最近 5 份（历史由 git 保留）。
> 机器可读投影：DashboardService.ListFacts（web 前端事实快照视图）。

### 快照 2026-08-17 00:34（现行）

| 事实 | 值 | 单位 | 采集时刻 | 来源 |
|------|-----|------|---------|------|
| calendar month_end@rule | 14 | days | 2026-08-17 00:34 | rule |
| calendar quarter_end@rule | 44 | days | 2026-08-17 00:34 | rule |
| calendar thursday@rule | 3 | days | 2026-08-17 00:34 | rule |
| defi_rate BUIDL@blackrock-buidl | 3.567 | pct_annualized | 2026-08-17 00:34 | yields.llama.fi/pools pool=b663ca59-c7e6-4435-ae4a-28d339ce6a15 |
| defi_rate STEAKUSDC@morpho-blue | 4.158 | pct_annualized | 2026-08-17 00:34 | yields.llama.fi/pools pool=931ea9be-5f4d-428e-beaf-205fc5b4e2b5 |
| defi_rate SUSDE@ethena-usde | 4.393 | pct_annualized | 2026-08-17 00:34 | yields.llama.fi/pools pool=66985a81-9c51-46ca-9977-42b4fe7bc6df |
| defi_rate USDC@aave-v3 | 3.279 | pct_annualized | 2026-08-17 00:34 | yields.llama.fi/pools pool=aa70268e-4b52-42bf-a116-608b370f9501 |
| defi_rate USDY@ondo-yield-assets | 3.55 | pct_annualized | 2026-08-17 00:34 | yields.llama.fi/pools pool=ac61ee82-2fe4-4f9b-a9cd-7fb33f598859 |
| deposit_rate 一年@boc | 0.95 | pct_annualized | 2026-08-17 00:34 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三个月@boc | 0.65 | pct_annualized | 2026-08-17 00:34 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三年@boc | 1.25 | pct_annualized | 2026-08-17 00:34 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 二年@boc | 1.05 | pct_annualized | 2026-08-17 00:34 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 五年@boc | 1.3 | pct_annualized | 2026-08-17 00:34 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 半年@boc | 0.85 | pct_annualized | 2026-08-17 00:34 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 活期@boc | 0.05 | pct_annualized | 2026-08-17 00:34 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| funding BTC@binance | 5.91 | pct_annualized | 2026-08-17 00:34 | fapi/v1/premiumIndex rate=0.00005397 per8h |
| funding BTC@okx | 6.021 | pct_annualized | 2026-08-17 00:34 | api/v5/public/funding-rate rate=0.0000549854452149 per8h |
| funding ETH@binance | 4.949 | pct_annualized | 2026-08-17 00:34 | fapi/v1/premiumIndex rate=0.00004520 per8h |
| funding ETH@okx | 7.286 | pct_annualized | 2026-08-17 00:34 | api/v5/public/funding-rate rate=0.0000665356945885 per8h |
| funding TRX@binance | 4.078 | pct_annualized | 2026-08-17 00:34 | fapi/v1/premiumIndex rate=0.00003724 per8h |
| funding TRX@okx | 6.176 | pct_annualized | 2026-08-17 00:34 | api/v5/public/funding-rate rate=0.0000564047570080 per8h |
| fx USDCNH@sina | 7.25 | price | 2026-08-16 19:52 | test:sina |
| iv BTC@deribit | 35.3 | pct | 2026-08-16 08:00 | api/v2/public/get_volatility_index_data DVOL |
| iv ETH@deribit | 47.54 | pct | 2026-08-16 08:00 | api/v2/public/get_volatility_index_data DVOL |
| reverse_repo GC001@sina | 0.865 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sh204001 |
| reverse_repo R-001@sina | 0.84 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sz131810 |
| ticker BTC@binance | 6.334e+04 | price | 2026-08-17 00:34 | fapi/v1/ticker/price |
| ticker BTC@okx | 6.333e+04 | price | 2026-08-17 00:34 | api/v5/market/tickers |
| ticker ETH@binance | 1890 | price | 2026-08-17 00:34 | fapi/v1/ticker/price |
| ticker ETH@okx | 1890 | price | 2026-08-17 00:34 | api/v5/market/tickers |
| ticker TRX@binance | 0.3321 | price | 2026-08-17 00:34 | fapi/v1/ticker/price |
| ticker TRX@okx | 0.3321 | price | 2026-08-17 00:34 | api/v5/market/tickers |

### 快照 2026-08-17 00:34（已过期 · 被 2026-08-17 00:34 快照取代）

| 事实 | 值 | 单位 | 采集时刻 | 来源 |
|------|-----|------|---------|------|
| calendar month_end@rule | 14 | days | 2026-08-17 00:34 | rule |
| calendar quarter_end@rule | 44 | days | 2026-08-17 00:34 | rule |
| calendar thursday@rule | 3 | days | 2026-08-17 00:34 | rule |
| defi_rate BUIDL@blackrock-buidl | 3.567 | pct_annualized | 2026-08-17 00:34 | yields.llama.fi/pools pool=b663ca59-c7e6-4435-ae4a-28d339ce6a15 |
| defi_rate STEAKUSDC@morpho-blue | 4.158 | pct_annualized | 2026-08-17 00:34 | yields.llama.fi/pools pool=931ea9be-5f4d-428e-beaf-205fc5b4e2b5 |
| defi_rate SUSDE@ethena-usde | 4.393 | pct_annualized | 2026-08-17 00:34 | yields.llama.fi/pools pool=66985a81-9c51-46ca-9977-42b4fe7bc6df |
| defi_rate USDC@aave-v3 | 3.279 | pct_annualized | 2026-08-17 00:34 | yields.llama.fi/pools pool=aa70268e-4b52-42bf-a116-608b370f9501 |
| defi_rate USDY@ondo-yield-assets | 3.55 | pct_annualized | 2026-08-17 00:34 | yields.llama.fi/pools pool=ac61ee82-2fe4-4f9b-a9cd-7fb33f598859 |
| deposit_rate 一年@boc | 0.95 | pct_annualized | 2026-08-17 00:34 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三个月@boc | 0.65 | pct_annualized | 2026-08-17 00:34 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三年@boc | 1.25 | pct_annualized | 2026-08-17 00:34 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 二年@boc | 1.05 | pct_annualized | 2026-08-17 00:34 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 五年@boc | 1.3 | pct_annualized | 2026-08-17 00:34 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 半年@boc | 0.85 | pct_annualized | 2026-08-17 00:34 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 活期@boc | 0.05 | pct_annualized | 2026-08-17 00:34 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| funding BTC@binance | 5.91 | pct_annualized | 2026-08-17 00:34 | fapi/v1/premiumIndex rate=0.00005397 per8h |
| funding BTC@okx | 6.04 | pct_annualized | 2026-08-17 00:34 | api/v5/public/funding-rate rate=0.0000551611556975 per8h |
| funding ETH@binance | 4.949 | pct_annualized | 2026-08-17 00:34 | fapi/v1/premiumIndex rate=0.00004520 per8h |
| funding ETH@okx | 7.313 | pct_annualized | 2026-08-17 00:34 | api/v5/public/funding-rate rate=0.0000667810973505 per8h |
| funding TRX@binance | 4.078 | pct_annualized | 2026-08-17 00:34 | fapi/v1/premiumIndex rate=0.00003724 per8h |
| funding TRX@okx | 6.157 | pct_annualized | 2026-08-17 00:34 | api/v5/public/funding-rate rate=0.0000562317330609 per8h |
| fx USDCNH@sina | 7.25 | price | 2026-08-16 19:52 | test:sina |
| iv BTC@deribit | 35.3 | pct | 2026-08-16 08:00 | api/v2/public/get_volatility_index_data DVOL |
| iv ETH@deribit | 47.61 | pct | 2026-08-16 08:00 | api/v2/public/get_volatility_index_data DVOL |
| reverse_repo GC001@sina | 0.865 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sh204001 |
| reverse_repo R-001@sina | 0.84 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sz131810 |
| ticker BTC@binance | 6.334e+04 | price | 2026-08-17 00:33 | fapi/v1/ticker/price |
| ticker BTC@okx | 6.334e+04 | price | 2026-08-17 00:34 | api/v5/market/tickers |
| ticker ETH@binance | 1890 | price | 2026-08-17 00:33 | fapi/v1/ticker/price |
| ticker ETH@okx | 1890 | price | 2026-08-17 00:34 | api/v5/market/tickers |
| ticker TRX@binance | 0.3321 | price | 2026-08-17 00:33 | fapi/v1/ticker/price |
| ticker TRX@okx | 0.3321 | price | 2026-08-17 00:34 | api/v5/market/tickers |

### 快照 2026-08-17 00:08（已过期 · 被 2026-08-17 00:34 快照取代）

| 事实 | 值 | 单位 | 采集时刻 | 来源 |
|------|-----|------|---------|------|
| calendar month_end@rule | 14 | days | 2026-08-17 00:08 | rule |
| calendar quarter_end@rule | 44 | days | 2026-08-17 00:08 | rule |
| calendar thursday@rule | 3 | days | 2026-08-17 00:08 | rule |
| defi_rate BUIDL@blackrock-buidl | 3.567 | pct_annualized | 2026-08-17 00:08 | yields.llama.fi/pools pool=b663ca59-c7e6-4435-ae4a-28d339ce6a15 |
| defi_rate STEAKUSDC@morpho-blue | 4.158 | pct_annualized | 2026-08-17 00:08 | yields.llama.fi/pools pool=931ea9be-5f4d-428e-beaf-205fc5b4e2b5 |
| defi_rate SUSDE@ethena-usde | 4.393 | pct_annualized | 2026-08-17 00:08 | yields.llama.fi/pools pool=66985a81-9c51-46ca-9977-42b4fe7bc6df |
| defi_rate USDC@aave-v3 | 3.283 | pct_annualized | 2026-08-17 00:08 | yields.llama.fi/pools pool=aa70268e-4b52-42bf-a116-608b370f9501 |
| defi_rate USDY@ondo-yield-assets | 3.55 | pct_annualized | 2026-08-17 00:08 | yields.llama.fi/pools pool=ac61ee82-2fe4-4f9b-a9cd-7fb33f598859 |
| deposit_rate 一年@boc | 0.95 | pct_annualized | 2026-08-17 00:08 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三个月@boc | 0.65 | pct_annualized | 2026-08-17 00:08 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三年@boc | 1.25 | pct_annualized | 2026-08-17 00:08 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 二年@boc | 1.05 | pct_annualized | 2026-08-17 00:08 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 五年@boc | 1.3 | pct_annualized | 2026-08-17 00:08 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 半年@boc | 0.85 | pct_annualized | 2026-08-17 00:08 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 活期@boc | 0.05 | pct_annualized | 2026-08-17 00:08 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| funding BTC@binance | 5.473 | pct_annualized | 2026-08-17 00:08 | fapi/v1/premiumIndex rate=0.00004998 per8h |
| funding BTC@okx | 6.689 | pct_annualized | 2026-08-17 00:08 | api/v5/public/funding-rate rate=0.0000610831937483 per8h |
| funding ETH@binance | 4.619 | pct_annualized | 2026-08-17 00:08 | fapi/v1/premiumIndex rate=0.00004218 per8h |
| funding ETH@okx | 8.009 | pct_annualized | 2026-08-17 00:08 | api/v5/public/funding-rate rate=0.0000731370944555 per8h |
| funding TRX@binance | 2.979 | pct_annualized | 2026-08-17 00:08 | fapi/v1/premiumIndex rate=0.00002721 per8h |
| funding TRX@okx | 5.751 | pct_annualized | 2026-08-17 00:08 | api/v5/public/funding-rate rate=0.0000525189739501 per8h |
| fx USDCNH@sina | 7.25 | price | 2026-08-16 19:52 | test:sina |
| iv BTC@deribit | 35.27 | pct | 2026-08-16 08:00 | api/v2/public/get_volatility_index_data DVOL |
| iv ETH@deribit | 47.56 | pct | 2026-08-16 08:00 | api/v2/public/get_volatility_index_data DVOL |
| reverse_repo GC001@sina | 0.865 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sh204001 |
| reverse_repo R-001@sina | 0.84 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sz131810 |
| ticker BTC@binance | 6.312e+04 | price | 2026-08-17 00:08 | fapi/v1/ticker/price |
| ticker BTC@okx | 6.311e+04 | price | 2026-08-17 00:08 | api/v5/market/tickers |
| ticker ETH@binance | 1883 | price | 2026-08-17 00:08 | fapi/v1/ticker/price |
| ticker ETH@okx | 1883 | price | 2026-08-17 00:08 | api/v5/market/tickers |
| ticker TRX@binance | 0.3318 | price | 2026-08-17 00:08 | fapi/v1/ticker/price |
| ticker TRX@okx | 0.3318 | price | 2026-08-17 00:08 | api/v5/market/tickers |

### 快照 2026-08-16 23:59（已过期 · 被 2026-08-17 00:08 快照取代）

| 事实 | 值 | 单位 | 采集时刻 | 来源 |
|------|-----|------|---------|------|
| calendar month_end@rule | 15 | days | 2026-08-16 23:59 | rule |
| calendar quarter_end@rule | 45 | days | 2026-08-16 23:59 | rule |
| calendar thursday@rule | 4 | days | 2026-08-16 23:59 | rule |
| defi_rate BUIDL@blackrock-buidl | 3.567 | pct_annualized | 2026-08-16 23:38 | yields.llama.fi/pools pool=b663ca59-c7e6-4435-ae4a-28d339ce6a15 |
| defi_rate STEAKUSDC@morpho-blue | 4.158 | pct_annualized | 2026-08-16 23:38 | yields.llama.fi/pools pool=931ea9be-5f4d-428e-beaf-205fc5b4e2b5 |
| defi_rate SUSDE@ethena-usde | 4.393 | pct_annualized | 2026-08-16 23:38 | yields.llama.fi/pools pool=66985a81-9c51-46ca-9977-42b4fe7bc6df |
| defi_rate USDC@aave-v3 | 3.283 | pct_annualized | 2026-08-16 23:38 | yields.llama.fi/pools pool=aa70268e-4b52-42bf-a116-608b370f9501 |
| defi_rate USDY@ondo-yield-assets | 3.55 | pct_annualized | 2026-08-16 23:38 | yields.llama.fi/pools pool=ac61ee82-2fe4-4f9b-a9cd-7fb33f598859 |
| deposit_rate 一年@boc | 0.95 | pct_annualized | 2026-08-16 23:59 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三个月@boc | 0.65 | pct_annualized | 2026-08-16 23:59 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三年@boc | 1.25 | pct_annualized | 2026-08-16 23:59 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 二年@boc | 1.05 | pct_annualized | 2026-08-16 23:59 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 五年@boc | 1.3 | pct_annualized | 2026-08-16 23:59 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 半年@boc | 0.85 | pct_annualized | 2026-08-16 23:59 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 活期@boc | 0.05 | pct_annualized | 2026-08-16 23:59 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| funding BTC@binance | 5.478 | pct_annualized | 2026-08-16 23:59 | fapi/v1/premiumIndex rate=0.00005003 per8h |
| funding BTC@okx | 7.193 | pct_annualized | 2026-08-16 23:59 | api/v5/public/funding-rate rate=0.0000656875428578 per8h |
| funding ETH@binance | 4.25 | pct_annualized | 2026-08-16 23:59 | fapi/v1/premiumIndex rate=0.00003881 per8h |
| funding ETH@okx | 8.462 | pct_annualized | 2026-08-16 23:59 | api/v5/public/funding-rate rate=0.0000772782276670 per8h |
| funding TRX@binance | 3.045 | pct_annualized | 2026-08-16 23:59 | fapi/v1/premiumIndex rate=0.00002781 per8h |
| funding TRX@okx | 5.919 | pct_annualized | 2026-08-16 23:59 | api/v5/public/funding-rate rate=0.0000540504598019 per8h |
| fx USDCNH@sina | 7.25 | price | 2026-08-16 19:52 | test:sina |
| iv BTC@deribit | 35.28 | pct | 2026-08-16 08:00 | api/v2/public/get_volatility_index_data DVOL |
| iv ETH@deribit | 47.53 | pct | 2026-08-16 08:00 | api/v2/public/get_volatility_index_data DVOL |
| reverse_repo GC001@sina | 0.865 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sh204001 |
| reverse_repo R-001@sina | 0.84 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sz131810 |
| ticker BTC@binance | 6.311e+04 | price | 2026-08-16 23:59 | fapi/v1/ticker/price |
| ticker BTC@okx | 6.311e+04 | price | 2026-08-16 23:59 | api/v5/market/tickers |
| ticker ETH@binance | 1883 | price | 2026-08-16 23:59 | fapi/v1/ticker/price |
| ticker ETH@okx | 1883 | price | 2026-08-16 23:59 | api/v5/market/tickers |
| ticker TRX@binance | 0.3318 | price | 2026-08-16 23:59 | fapi/v1/ticker/price |
| ticker TRX@okx | 0.3319 | price | 2026-08-16 23:59 | api/v5/market/tickers |

### 快照 2026-08-16 23:38（已过期 · 被 2026-08-16 23:59 快照取代）

| 事实 | 值 | 单位 | 采集时刻 | 来源 |
|------|-----|------|---------|------|
| calendar month_end@rule | 15 | days | 2026-08-16 23:38 | rule |
| calendar quarter_end@rule | 45 | days | 2026-08-16 23:38 | rule |
| calendar thursday@rule | 4 | days | 2026-08-16 23:38 | rule |
| defi_rate BUIDL@blackrock-buidl | 3.567 | pct_annualized | 2026-08-16 23:38 | yields.llama.fi/pools pool=b663ca59-c7e6-4435-ae4a-28d339ce6a15 |
| defi_rate STEAKUSDC@morpho-blue | 4.158 | pct_annualized | 2026-08-16 23:38 | yields.llama.fi/pools pool=931ea9be-5f4d-428e-beaf-205fc5b4e2b5 |
| defi_rate SUSDE@ethena-usde | 4.393 | pct_annualized | 2026-08-16 23:38 | yields.llama.fi/pools pool=66985a81-9c51-46ca-9977-42b4fe7bc6df |
| defi_rate USDC@aave-v3 | 3.283 | pct_annualized | 2026-08-16 23:38 | yields.llama.fi/pools pool=aa70268e-4b52-42bf-a116-608b370f9501 |
| defi_rate USDY@ondo-yield-assets | 3.55 | pct_annualized | 2026-08-16 23:38 | yields.llama.fi/pools pool=ac61ee82-2fe4-4f9b-a9cd-7fb33f598859 |
| deposit_rate 一年@boc | 0.95 | pct_annualized | 2026-08-16 23:38 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三个月@boc | 0.65 | pct_annualized | 2026-08-16 23:38 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三年@boc | 1.25 | pct_annualized | 2026-08-16 23:38 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 二年@boc | 1.05 | pct_annualized | 2026-08-16 23:38 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 五年@boc | 1.3 | pct_annualized | 2026-08-16 23:38 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 半年@boc | 0.85 | pct_annualized | 2026-08-16 23:38 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 活期@boc | 0.05 | pct_annualized | 2026-08-16 23:38 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| funding BTC@binance | 5.386 | pct_annualized | 2026-08-16 23:38 | fapi/v1/premiumIndex rate=0.00004919 per8h |
| funding BTC@okx | 8.194 | pct_annualized | 2026-08-16 23:38 | api/v5/public/funding-rate rate=0.0000748354361250 per8h |
| funding ETH@binance | 3.547 | pct_annualized | 2026-08-16 23:38 | fapi/v1/premiumIndex rate=0.00003239 per8h |
| funding ETH@okx | 9.573 | pct_annualized | 2026-08-16 23:38 | api/v5/public/funding-rate rate=0.0000874232895227 per8h |
| funding TRX@binance | 2.596 | pct_annualized | 2026-08-16 23:38 | fapi/v1/premiumIndex rate=0.00002371 per8h |
| funding TRX@okx | 5.154 | pct_annualized | 2026-08-16 23:38 | api/v5/public/funding-rate rate=0.0000470661729261 per8h |
| fx USDCNH@sina | 7.25 | price | 2026-08-16 19:52 | test:sina |
| iv BTC@deribit | 35.38 | pct | 2026-08-16 08:00 | api/v2/public/get_volatility_index_data DVOL |
| iv ETH@deribit | 47.56 | pct | 2026-08-16 08:00 | api/v2/public/get_volatility_index_data DVOL |
| reverse_repo GC001@sina | 0.865 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sh204001 |
| reverse_repo R-001@sina | 0.84 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sz131810 |
| ticker BTC@binance | 6.311e+04 | price | 2026-08-16 23:38 | fapi/v1/ticker/price |
| ticker BTC@okx | 6.31e+04 | price | 2026-08-16 23:38 | api/v5/market/tickers |
| ticker ETH@binance | 1884 | price | 2026-08-16 23:38 | fapi/v1/ticker/price |
| ticker ETH@okx | 1884 | price | 2026-08-16 23:38 | api/v5/market/tickers |
| ticker TRX@binance | 0.3318 | price | 2026-08-16 23:38 | fapi/v1/ticker/price |
| ticker TRX@okx | 0.3318 | price | 2026-08-16 23:38 | api/v5/market/tickers |
<!-- ARBCN-EXPORT-END -->
