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
### 快照 2026-08-16 13:24（现行）

| 事实 | 值 | 单位 | 采集时刻 | 来源 |
|------|-----|------|---------|------|
| calendar month_end@rule | 15 | days | 2026-08-16 13:19 | rule |
| calendar quarter_end@rule | 45 | days | 2026-08-16 13:19 | rule |
| calendar thursday@rule | 4 | days | 2026-08-16 13:19 | rule |
| defi_rate BUIDL@blackrock-buidl | 3.567 | pct_annualized | 2026-08-16 13:19 | yields.llama.fi/pools pool=b663ca59-c7e6-4435-ae4a-28d339ce6a15 |
| defi_rate STEAKUSDC@morpho-blue | 4.158 | pct_annualized | 2026-08-16 13:19 | yields.llama.fi/pools pool=931ea9be-5f4d-428e-beaf-205fc5b4e2b5 |
| defi_rate SUSDE@ethena-usde | 4.346 | pct_annualized | 2026-08-16 13:19 | yields.llama.fi/pools pool=66985a81-9c51-46ca-9977-42b4fe7bc6df |
| defi_rate USDC@aave-v3 | 12.57 | pct_annualized | 2026-08-16 13:19 | yields.llama.fi/pools pool=aa70268e-4b52-42bf-a116-608b370f9501 |
| defi_rate USDY@ondo-yield-assets | 3.55 | pct_annualized | 2026-08-16 13:19 | yields.llama.fi/pools pool=ac61ee82-2fe4-4f9b-a9cd-7fb33f598859 |
| deposit_rate 一年@boc | 0.95 | pct_annualized | 2026-08-16 13:19 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三个月@boc | 0.65 | pct_annualized | 2026-08-16 13:19 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三年@boc | 1.25 | pct_annualized | 2026-08-16 13:19 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 二年@boc | 1.05 | pct_annualized | 2026-08-16 13:19 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 五年@boc | 1.3 | pct_annualized | 2026-08-16 13:19 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 半年@boc | 0.85 | pct_annualized | 2026-08-16 13:19 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 活期@boc | 0.05 | pct_annualized | 2026-08-16 13:19 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| funding BTC@binance | -0.3121 | pct_annualized | 2026-08-16 13:24 | fapi/v1/premiumIndex rate=-0.00000285 per8h |
| funding BTC@okx | 5.27 | pct_annualized | 2026-08-16 13:19 | api/v5/public/funding-rate rate=0.0000481259016938 per8h |
| funding ETH@binance | -1.685 | pct_annualized | 2026-08-16 13:24 | fapi/v1/premiumIndex rate=-0.00001539 per8h |
| funding ETH@okx | 5.267 | pct_annualized | 2026-08-16 13:19 | api/v5/public/funding-rate rate=0.0000481018008539 per8h |
| funding TRX@binance | -1.684 | pct_annualized | 2026-08-16 13:24 | fapi/v1/premiumIndex rate=-0.00001538 per8h |
| funding TRX@okx | -8.315 | pct_annualized | 2026-08-16 13:19 | api/v5/public/funding-rate rate=-0.0000759325119035 per8h |
| fx USDCNH@sina | 6.744 | price | 2026-08-15 04:59 | hq.sinajs.cn/list=fx_susdcnh |
| iv BTC@deribit | 34.98 | pct | 2026-08-16 08:00 | api/v2/public/get_volatility_index_data DVOL |
| iv ETH@deribit | 47.57 | pct | 2026-08-16 08:00 | api/v2/public/get_volatility_index_data DVOL |
| reverse_repo GC001@sina | 0.865 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sh204001 |
| reverse_repo R-001@sina | 0.84 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sz131810 |
| ticker BTC@binance | 6.304e+04 | price | 2026-08-16 13:24 | fapi/v1/ticker/price |
| ticker BTC@okx | 6.304e+04 | price | 2026-08-16 13:24 | api/v5/market/tickers |
| ticker ETH@binance | 1880 | price | 2026-08-16 13:24 | fapi/v1/ticker/price |
| ticker ETH@okx | 1880 | price | 2026-08-16 13:24 | api/v5/market/tickers |
| ticker TRX@binance | 0.331 | price | 2026-08-16 13:24 | fapi/v1/ticker/price |
| ticker TRX@okx | 0.3309 | price | 2026-08-16 13:24 | api/v5/market/tickers |

### 快照 2026-08-16 13:19（已过期 · 被 2026-08-16 13:24 快照取代）

| 事实 | 值 | 单位 | 采集时刻 | 来源 |
|------|-----|------|---------|------|
| calendar month_end@rule | 15 | days | 2026-08-16 13:19 | rule |
| calendar quarter_end@rule | 45 | days | 2026-08-16 13:19 | rule |
| calendar thursday@rule | 4 | days | 2026-08-16 13:19 | rule |
| defi_rate BUIDL@blackrock-buidl | 3.567 | pct_annualized | 2026-08-16 13:13 | yields.llama.fi/pools pool=b663ca59-c7e6-4435-ae4a-28d339ce6a15 |
| defi_rate STEAKUSDC@morpho-blue | 4.158 | pct_annualized | 2026-08-16 13:13 | yields.llama.fi/pools pool=931ea9be-5f4d-428e-beaf-205fc5b4e2b5 |
| defi_rate SUSDE@ethena-usde | 4.346 | pct_annualized | 2026-08-16 13:13 | yields.llama.fi/pools pool=66985a81-9c51-46ca-9977-42b4fe7bc6df |
| defi_rate USDC@aave-v3 | 12.57 | pct_annualized | 2026-08-16 13:13 | yields.llama.fi/pools pool=aa70268e-4b52-42bf-a116-608b370f9501 |
| defi_rate USDY@ondo-yield-assets | 3.55 | pct_annualized | 2026-08-16 13:13 | yields.llama.fi/pools pool=ac61ee82-2fe4-4f9b-a9cd-7fb33f598859 |
| deposit_rate 一年@boc | 0.95 | pct_annualized | 2026-08-16 13:19 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三个月@boc | 0.65 | pct_annualized | 2026-08-16 13:19 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三年@boc | 1.25 | pct_annualized | 2026-08-16 13:19 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 二年@boc | 1.05 | pct_annualized | 2026-08-16 13:19 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 五年@boc | 1.3 | pct_annualized | 2026-08-16 13:19 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 半年@boc | 0.85 | pct_annualized | 2026-08-16 13:19 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 活期@boc | 0.05 | pct_annualized | 2026-08-16 13:19 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| funding BTC@binance | -0.2584 | pct_annualized | 2026-08-16 13:19 | fapi/v1/premiumIndex rate=-0.00000236 per8h |
| funding BTC@okx | 5.27 | pct_annualized | 2026-08-16 13:19 | api/v5/public/funding-rate rate=0.0000481259016938 per8h |
| funding ETH@binance | -1.577 | pct_annualized | 2026-08-16 13:19 | fapi/v1/premiumIndex rate=-0.00001440 per8h |
| funding ETH@okx | 5.267 | pct_annualized | 2026-08-16 13:19 | api/v5/public/funding-rate rate=0.0000481018008539 per8h |
| funding TRX@binance | -1.754 | pct_annualized | 2026-08-16 13:19 | fapi/v1/premiumIndex rate=-0.00001602 per8h |
| funding TRX@okx | -8.315 | pct_annualized | 2026-08-16 13:19 | api/v5/public/funding-rate rate=-0.0000759325119035 per8h |
| fx USDCNH@sina | 6.744 | price | 2026-08-15 04:59 | hq.sinajs.cn/list=fx_susdcnh |
| iv BTC@deribit | 34.98 | pct | 2026-08-16 08:00 | api/v2/public/get_volatility_index_data DVOL |
| iv ETH@deribit | 47.57 | pct | 2026-08-16 08:00 | api/v2/public/get_volatility_index_data DVOL |
| reverse_repo GC001@sina | 0.865 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sh204001 |
| reverse_repo R-001@sina | 0.84 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sz131810 |
| ticker BTC@binance | 6.304e+04 | price | 2026-08-16 13:19 | fapi/v1/ticker/price |
| ticker BTC@okx | 6.304e+04 | price | 2026-08-16 13:19 | api/v5/market/tickers |
| ticker ETH@binance | 1879 | price | 2026-08-16 13:19 | fapi/v1/ticker/price |
| ticker ETH@okx | 1880 | price | 2026-08-16 13:19 | api/v5/market/tickers |
| ticker TRX@binance | 0.331 | price | 2026-08-16 13:19 | fapi/v1/ticker/price |
| ticker TRX@okx | 0.3309 | price | 2026-08-16 13:19 | api/v5/market/tickers |

### 快照 2026-08-16 13:13（已过期 · 被 2026-08-16 13:19 快照取代）

| 事实 | 值 | 单位 | 采集时刻 | 来源 |
|------|-----|------|---------|------|
| calendar month_end@rule | 15 | days | 2026-08-16 13:13 | rule |
| calendar quarter_end@rule | 45 | days | 2026-08-16 13:13 | rule |
| calendar thursday@rule | 4 | days | 2026-08-16 13:13 | rule |
| defi_rate BUIDL@blackrock-buidl | 3.567 | pct_annualized | 2026-08-16 13:13 | yields.llama.fi/pools pool=b663ca59-c7e6-4435-ae4a-28d339ce6a15 |
| defi_rate STEAKUSDC@morpho-blue | 4.158 | pct_annualized | 2026-08-16 13:13 | yields.llama.fi/pools pool=931ea9be-5f4d-428e-beaf-205fc5b4e2b5 |
| defi_rate SUSDE@ethena-usde | 4.346 | pct_annualized | 2026-08-16 13:13 | yields.llama.fi/pools pool=66985a81-9c51-46ca-9977-42b4fe7bc6df |
| defi_rate USDC@aave-v3 | 12.57 | pct_annualized | 2026-08-16 13:13 | yields.llama.fi/pools pool=aa70268e-4b52-42bf-a116-608b370f9501 |
| defi_rate USDY@ondo-yield-assets | 3.55 | pct_annualized | 2026-08-16 13:13 | yields.llama.fi/pools pool=ac61ee82-2fe4-4f9b-a9cd-7fb33f598859 |
| deposit_rate 一年@boc | 0.95 | pct_annualized | 2026-08-16 13:13 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三个月@boc | 0.65 | pct_annualized | 2026-08-16 13:13 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三年@boc | 1.25 | pct_annualized | 2026-08-16 13:13 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 二年@boc | 1.05 | pct_annualized | 2026-08-16 13:13 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 五年@boc | 1.3 | pct_annualized | 2026-08-16 13:13 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 半年@boc | 0.85 | pct_annualized | 2026-08-16 13:13 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 活期@boc | 0.05 | pct_annualized | 2026-08-16 13:13 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| funding BTC@binance | -0.3405 | pct_annualized | 2026-08-16 13:13 | fapi/v1/premiumIndex rate=-0.00000311 per8h |
| funding BTC@okx | 5.384 | pct_annualized | 2026-08-16 13:13 | api/v5/public/funding-rate rate=0.0000491646990597 per8h |
| funding ETH@binance | -1.589 | pct_annualized | 2026-08-16 13:13 | fapi/v1/premiumIndex rate=-0.00001451 per8h |
| funding ETH@okx | 5.221 | pct_annualized | 2026-08-16 13:13 | api/v5/public/funding-rate rate=0.0000476774805509 per8h |
| funding TRX@binance | -1.924 | pct_annualized | 2026-08-16 13:13 | fapi/v1/premiumIndex rate=-0.00001757 per8h |
| funding TRX@okx | -8.136 | pct_annualized | 2026-08-16 13:13 | api/v5/public/funding-rate rate=-0.0000743000411198 per8h |
| fx USDCNH@sina | 6.744 | price | 2026-08-15 04:59 | hq.sinajs.cn/list=fx_susdcnh |
| iv BTC@deribit | 34.98 | pct | 2026-08-16 08:00 | api/v2/public/get_volatility_index_data DVOL |
| iv ETH@deribit | 47.57 | pct | 2026-08-16 08:00 | api/v2/public/get_volatility_index_data DVOL |
| reverse_repo GC001@sina | 0.865 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sh204001 |
| reverse_repo R-001@sina | 0.84 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sz131810 |
| ticker BTC@binance | 6.307e+04 | price | 2026-08-16 13:13 | fapi/v1/ticker/price |
| ticker BTC@okx | 6.306e+04 | price | 2026-08-16 13:13 | api/v5/market/tickers |
| ticker ETH@binance | 1881 | price | 2026-08-16 13:13 | fapi/v1/ticker/price |
| ticker ETH@okx | 1881 | price | 2026-08-16 13:13 | api/v5/market/tickers |
| ticker TRX@binance | 0.3312 | price | 2026-08-16 13:13 | fapi/v1/ticker/price |
| ticker TRX@okx | 0.3311 | price | 2026-08-16 13:13 | api/v5/market/tickers |

### 快照 2026-08-16 13:13（已过期 · 被 2026-08-16 13:13 快照取代）

| 事实 | 值 | 单位 | 采集时刻 | 来源 |
|------|-----|------|---------|------|
| calendar month_end@rule | 15 | days | 2026-08-16 13:08 | rule |
| calendar quarter_end@rule | 45 | days | 2026-08-16 13:08 | rule |
| calendar thursday@rule | 4 | days | 2026-08-16 13:08 | rule |
| defi_rate BUIDL@blackrock-buidl | 3.567 | pct_annualized | 2026-08-16 13:08 | yields.llama.fi/pools pool=b663ca59-c7e6-4435-ae4a-28d339ce6a15 |
| defi_rate STEAKUSDC@morpho-blue | 4.158 | pct_annualized | 2026-08-16 13:08 | yields.llama.fi/pools pool=931ea9be-5f4d-428e-beaf-205fc5b4e2b5 |
| defi_rate SUSDE@ethena-usde | 4.346 | pct_annualized | 2026-08-16 13:08 | yields.llama.fi/pools pool=66985a81-9c51-46ca-9977-42b4fe7bc6df |
| defi_rate USDC@aave-v3 | 12.57 | pct_annualized | 2026-08-16 13:08 | yields.llama.fi/pools pool=aa70268e-4b52-42bf-a116-608b370f9501 |
| defi_rate USDY@ondo-yield-assets | 3.55 | pct_annualized | 2026-08-16 13:08 | yields.llama.fi/pools pool=ac61ee82-2fe4-4f9b-a9cd-7fb33f598859 |
| deposit_rate 一年@boc | 0.95 | pct_annualized | 2026-08-16 13:08 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三个月@boc | 0.65 | pct_annualized | 2026-08-16 13:08 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三年@boc | 1.25 | pct_annualized | 2026-08-16 13:08 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 二年@boc | 1.05 | pct_annualized | 2026-08-16 13:08 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 五年@boc | 1.3 | pct_annualized | 2026-08-16 13:08 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 半年@boc | 0.85 | pct_annualized | 2026-08-16 13:08 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 活期@boc | 0.05 | pct_annualized | 2026-08-16 13:08 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| funding BTC@binance | -0.3646 | pct_annualized | 2026-08-16 13:08 | fapi/v1/premiumIndex rate=-0.00000333 per8h |
| funding BTC@okx | 5.384 | pct_annualized | 2026-08-16 13:13 | api/v5/public/funding-rate rate=0.0000491646990597 per8h |
| funding ETH@binance | -1.683 | pct_annualized | 2026-08-16 13:08 | fapi/v1/premiumIndex rate=-0.00001537 per8h |
| funding ETH@okx | 5.221 | pct_annualized | 2026-08-16 13:13 | api/v5/public/funding-rate rate=0.0000476774805509 per8h |
| funding TRX@binance | -2.264 | pct_annualized | 2026-08-16 13:08 | fapi/v1/premiumIndex rate=-0.00002068 per8h |
| funding TRX@okx | -8.136 | pct_annualized | 2026-08-16 13:13 | api/v5/public/funding-rate rate=-0.0000743000411198 per8h |
| fx USDCNH@sina | 6.744 | price | 2026-08-15 04:59 | hq.sinajs.cn/list=fx_susdcnh |
| iv BTC@deribit | 34.98 | pct | 2026-08-16 08:00 | api/v2/public/get_volatility_index_data DVOL |
| iv ETH@deribit | 47.57 | pct | 2026-08-16 08:00 | api/v2/public/get_volatility_index_data DVOL |
| reverse_repo GC001@sina | 0.865 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sh204001 |
| reverse_repo R-001@sina | 0.84 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sz131810 |
| ticker BTC@binance | 6.307e+04 | price | 2026-08-16 13:13 | fapi/v1/ticker/price |
| ticker BTC@okx | 6.306e+04 | price | 2026-08-16 13:12 | api/v5/market/tickers |
| ticker ETH@binance | 1881 | price | 2026-08-16 13:13 | fapi/v1/ticker/price |
| ticker ETH@okx | 1881 | price | 2026-08-16 13:12 | api/v5/market/tickers |
| ticker TRX@binance | 0.3312 | price | 2026-08-16 13:13 | fapi/v1/ticker/price |
| ticker TRX@okx | 0.3311 | price | 2026-08-16 13:12 | api/v5/market/tickers |

### 快照 2026-08-16 13:08（已过期 · 被 2026-08-16 13:13 快照取代）

| 事实 | 值 | 单位 | 采集时刻 | 来源 |
|------|-----|------|---------|------|
| calendar month_end@rule | 15 | days | 2026-08-16 13:08 | rule |
| calendar quarter_end@rule | 45 | days | 2026-08-16 13:08 | rule |
| calendar thursday@rule | 4 | days | 2026-08-16 13:08 | rule |
| defi_rate BUIDL@blackrock-buidl | 3.567 | pct_annualized | 2026-08-16 13:08 | yields.llama.fi/pools pool=b663ca59-c7e6-4435-ae4a-28d339ce6a15 |
| defi_rate STEAKUSDC@morpho-blue | 4.158 | pct_annualized | 2026-08-16 13:08 | yields.llama.fi/pools pool=931ea9be-5f4d-428e-beaf-205fc5b4e2b5 |
| defi_rate SUSDE@ethena-usde | 4.346 | pct_annualized | 2026-08-16 13:08 | yields.llama.fi/pools pool=66985a81-9c51-46ca-9977-42b4fe7bc6df |
| defi_rate USDC@aave-v3 | 12.57 | pct_annualized | 2026-08-16 13:08 | yields.llama.fi/pools pool=aa70268e-4b52-42bf-a116-608b370f9501 |
| defi_rate USDY@ondo-yield-assets | 3.55 | pct_annualized | 2026-08-16 13:08 | yields.llama.fi/pools pool=ac61ee82-2fe4-4f9b-a9cd-7fb33f598859 |
| deposit_rate 一年@boc | 0.95 | pct_annualized | 2026-08-16 13:08 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三个月@boc | 0.65 | pct_annualized | 2026-08-16 13:08 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三年@boc | 1.25 | pct_annualized | 2026-08-16 13:08 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 二年@boc | 1.05 | pct_annualized | 2026-08-16 13:08 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 五年@boc | 1.3 | pct_annualized | 2026-08-16 13:08 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 半年@boc | 0.85 | pct_annualized | 2026-08-16 13:08 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 活期@boc | 0.05 | pct_annualized | 2026-08-16 13:08 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| funding BTC@binance | -0.3646 | pct_annualized | 2026-08-16 13:08 | fapi/v1/premiumIndex rate=-0.00000333 per8h |
| funding BTC@okx | 5.493 | pct_annualized | 2026-08-16 13:08 | api/v5/public/funding-rate rate=0.0000501658109785 per8h |
| funding ETH@binance | -1.683 | pct_annualized | 2026-08-16 13:08 | fapi/v1/premiumIndex rate=-0.00001537 per8h |
| funding ETH@okx | 5.145 | pct_annualized | 2026-08-16 13:08 | api/v5/public/funding-rate rate=0.0000469883114315 per8h |
| funding TRX@binance | -2.264 | pct_annualized | 2026-08-16 13:08 | fapi/v1/premiumIndex rate=-0.00002068 per8h |
| funding TRX@okx | -8.19 | pct_annualized | 2026-08-16 13:08 | api/v5/public/funding-rate rate=-0.0000747974453030 per8h |
| fx USDCNH@sina | 6.744 | price | 2026-08-15 04:59 | hq.sinajs.cn/list=fx_susdcnh |
| iv BTC@deribit | 34.98 | pct | 2026-08-16 08:00 | api/v2/public/get_volatility_index_data DVOL |
| iv ETH@deribit | 47.57 | pct | 2026-08-16 08:00 | api/v2/public/get_volatility_index_data DVOL |
| reverse_repo GC001@sina | 0.865 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sh204001 |
| reverse_repo R-001@sina | 0.84 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sz131810 |
| ticker BTC@binance | 6.306e+04 | price | 2026-08-16 13:08 | fapi/v1/ticker/price |
| ticker BTC@okx | 6.306e+04 | price | 2026-08-16 13:08 | api/v5/market/tickers |
| ticker ETH@binance | 1881 | price | 2026-08-16 13:08 | fapi/v1/ticker/price |
| ticker ETH@okx | 1881 | price | 2026-08-16 13:08 | api/v5/market/tickers |
| ticker TRX@binance | 0.3312 | price | 2026-08-16 13:08 | fapi/v1/ticker/price |
| ticker TRX@okx | 0.3312 | price | 2026-08-16 13:08 | api/v5/market/tickers |

### 快照 2026-08-16 13:02（已过期 · 被 2026-08-16 13:08 快照取代）

| 事实 | 值 | 单位 | 采集时刻 | 来源 |
|------|-----|------|---------|------|
| calendar month_end@rule | 15 | days | 2026-08-16 13:01 | rule |
| calendar quarter_end@rule | 45 | days | 2026-08-16 13:01 | rule |
| calendar thursday@rule | 4 | days | 2026-08-16 13:01 | rule |
| defi_rate BUIDL@blackrock-buidl | 3.567 | pct_annualized | 2026-08-16 12:35 | yields.llama.fi/pools pool=b663ca59-c7e6-4435-ae4a-28d339ce6a15 |
| defi_rate STEAKUSDC@morpho-blue | 4.158 | pct_annualized | 2026-08-16 12:35 | yields.llama.fi/pools pool=931ea9be-5f4d-428e-beaf-205fc5b4e2b5 |
| defi_rate SUSDE@ethena-usde | 4.346 | pct_annualized | 2026-08-16 12:35 | yields.llama.fi/pools pool=66985a81-9c51-46ca-9977-42b4fe7bc6df |
| defi_rate USDC@aave-v3 | 12.57 | pct_annualized | 2026-08-16 12:35 | yields.llama.fi/pools pool=aa70268e-4b52-42bf-a116-608b370f9501 |
| defi_rate USDY@ondo-yield-assets | 3.55 | pct_annualized | 2026-08-16 12:35 | yields.llama.fi/pools pool=ac61ee82-2fe4-4f9b-a9cd-7fb33f598859 |
| deposit_rate 一年@boc | 0.95 | pct_annualized | 2026-08-16 13:01 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三个月@boc | 0.65 | pct_annualized | 2026-08-16 13:01 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三年@boc | 1.25 | pct_annualized | 2026-08-16 13:01 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 二年@boc | 1.05 | pct_annualized | 2026-08-16 13:01 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 五年@boc | 1.3 | pct_annualized | 2026-08-16 13:01 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 半年@boc | 0.85 | pct_annualized | 2026-08-16 13:01 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 活期@boc | 0.05 | pct_annualized | 2026-08-16 13:01 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| funding BTC@binance | -0.2902 | pct_annualized | 2026-08-16 13:01 | fapi/v1/premiumIndex rate=-0.00000265 per8h |
| funding BTC@okx | 5.644 | pct_annualized | 2026-08-16 13:01 | api/v5/public/funding-rate rate=0.0000515425811700 per8h |
| funding ETH@binance | -1.718 | pct_annualized | 2026-08-16 13:01 | fapi/v1/premiumIndex rate=-0.00001569 per8h |
| funding ETH@okx | 5.137 | pct_annualized | 2026-08-16 13:01 | api/v5/public/funding-rate rate=0.0000469124727054 per8h |
| funding TRX@binance | -2.751 | pct_annualized | 2026-08-16 13:01 | fapi/v1/premiumIndex rate=-0.00002512 per8h |
| funding TRX@okx | -8.612 | pct_annualized | 2026-08-16 13:01 | api/v5/public/funding-rate rate=-0.0000786494587001 per8h |
| fx USDCNH@sina | 6.744 | price | 2026-08-15 04:59 | hq.sinajs.cn/list=fx_susdcnh |
| iv BTC@deribit | 34.98 | pct | 2026-08-16 08:00 | api/v2/public/get_volatility_index_data DVOL |
| iv ETH@deribit | 47.57 | pct | 2026-08-16 08:00 | api/v2/public/get_volatility_index_data DVOL |
| reverse_repo GC001@sina | 0.865 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sh204001 |
| reverse_repo R-001@sina | 0.84 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sz131810 |
| ticker BTC@binance | 6.304e+04 | price | 2026-08-16 13:01 | fapi/v1/ticker/price |
| ticker BTC@okx | 6.304e+04 | price | 2026-08-16 13:01 | api/v5/market/tickers |
| ticker ETH@binance | 1880 | price | 2026-08-16 13:01 | fapi/v1/ticker/price |
| ticker ETH@okx | 1881 | price | 2026-08-16 13:01 | api/v5/market/tickers |
| ticker TRX@binance | 0.3309 | price | 2026-08-16 13:01 | fapi/v1/ticker/price |
| ticker TRX@okx | 0.3308 | price | 2026-08-16 13:01 | api/v5/market/tickers |

### 快照 2026-08-16 12:55（已过期 · 被 2026-08-16 13:02 快照取代）

| 事实 | 值 | 单位 | 采集时刻 | 来源 |
|------|-----|------|---------|------|
| calendar month_end@rule | 15 | days | 2026-08-16 12:35 | rule |
| calendar quarter_end@rule | 45 | days | 2026-08-16 12:35 | rule |
| calendar thursday@rule | 4 | days | 2026-08-16 12:35 | rule |
| defi_rate BUIDL@blackrock-buidl | 3.567 | pct_annualized | 2026-08-16 12:35 | yields.llama.fi/pools pool=b663ca59-c7e6-4435-ae4a-28d339ce6a15 |
| defi_rate STEAKUSDC@morpho-blue | 4.158 | pct_annualized | 2026-08-16 12:35 | yields.llama.fi/pools pool=931ea9be-5f4d-428e-beaf-205fc5b4e2b5 |
| defi_rate SUSDE@ethena-usde | 4.346 | pct_annualized | 2026-08-16 12:35 | yields.llama.fi/pools pool=66985a81-9c51-46ca-9977-42b4fe7bc6df |
| defi_rate USDC@aave-v3 | 12.57 | pct_annualized | 2026-08-16 12:35 | yields.llama.fi/pools pool=aa70268e-4b52-42bf-a116-608b370f9501 |
| defi_rate USDY@ondo-yield-assets | 3.55 | pct_annualized | 2026-08-16 12:35 | yields.llama.fi/pools pool=ac61ee82-2fe4-4f9b-a9cd-7fb33f598859 |
| deposit_rate 一年@boc | 0.95 | pct_annualized | 2026-08-16 12:35 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三个月@boc | 0.65 | pct_annualized | 2026-08-16 12:35 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三年@boc | 1.25 | pct_annualized | 2026-08-16 12:35 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 二年@boc | 1.05 | pct_annualized | 2026-08-16 12:35 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 五年@boc | 1.3 | pct_annualized | 2026-08-16 12:35 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 半年@boc | 0.85 | pct_annualized | 2026-08-16 12:35 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 活期@boc | 0.05 | pct_annualized | 2026-08-16 12:35 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| funding BTC@binance | -0.2267 | pct_annualized | 2026-08-16 12:50 | fapi/v1/premiumIndex rate=-0.00000207 per8h |
| funding BTC@okx | 6.013 | pct_annualized | 2026-08-16 12:50 | api/v5/public/funding-rate rate=0.0000549128208321 per8h |
| funding ETH@binance | -1.786 | pct_annualized | 2026-08-16 12:50 | fapi/v1/premiumIndex rate=-0.00001631 per8h |
| funding ETH@okx | 5.18 | pct_annualized | 2026-08-16 12:50 | api/v5/public/funding-rate rate=0.0000473024129661 per8h |
| funding TRX@binance | -2.984 | pct_annualized | 2026-08-16 12:50 | fapi/v1/premiumIndex rate=-0.00002725 per8h |
| funding TRX@okx | -9.05 | pct_annualized | 2026-08-16 12:50 | api/v5/public/funding-rate rate=-0.0000826444064585 per8h |
| fx USDCNH@sina | 6.744 | price | 2026-08-15 04:59 | hq.sinajs.cn/list=fx_susdcnh |
| iv BTC@deribit | 34.98 | pct | 2026-08-16 08:00 | api/v2/public/get_volatility_index_data DVOL |
| iv ETH@deribit | 47.57 | pct | 2026-08-16 08:00 | api/v2/public/get_volatility_index_data DVOL |
| reverse_repo GC001@sina | 0.865 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sh204001 |
| reverse_repo R-001@sina | 0.84 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sz131810 |
| ticker BTC@binance | 6.309e+04 | price | 2026-08-16 12:54 | fapi/v1/ticker/price |
| ticker BTC@okx | 6.309e+04 | price | 2026-08-16 12:54 | api/v5/market/tickers |
| ticker ETH@binance | 1881 | price | 2026-08-16 12:54 | fapi/v1/ticker/price |
| ticker ETH@okx | 1881 | price | 2026-08-16 12:54 | api/v5/market/tickers |
| ticker TRX@binance | 0.3309 | price | 2026-08-16 12:54 | fapi/v1/ticker/price |
| ticker TRX@okx | 0.3308 | price | 2026-08-16 12:54 | api/v5/market/tickers |

### 快照 2026-08-16 12:35（已过期 · 被 2026-08-16 12:55 快照取代）

| 事实 | 值 | 单位 | 采集时刻 | 来源 |
|------|-----|------|---------|------|
| calendar month_end@rule | 15 | days | 2026-08-16 12:35 | rule |
| calendar quarter_end@rule | 45 | days | 2026-08-16 12:35 | rule |
| calendar thursday@rule | 4 | days | 2026-08-16 12:35 | rule |
| defi_rate BUIDL@blackrock-buidl | 3.567 | pct_annualized | 2026-08-16 12:17 | yields.llama.fi/pools pool=b663ca59-c7e6-4435-ae4a-28d339ce6a15 |
| defi_rate STEAKUSDC@morpho-blue | 4.158 | pct_annualized | 2026-08-16 12:17 | yields.llama.fi/pools pool=931ea9be-5f4d-428e-beaf-205fc5b4e2b5 |
| defi_rate SUSDE@ethena-usde | 4.344 | pct_annualized | 2026-08-16 12:17 | yields.llama.fi/pools pool=66985a81-9c51-46ca-9977-42b4fe7bc6df |
| defi_rate USDC@aave-v3 | 12.57 | pct_annualized | 2026-08-16 12:17 | yields.llama.fi/pools pool=aa70268e-4b52-42bf-a116-608b370f9501 |
| defi_rate USDY@ondo-yield-assets | 3.55 | pct_annualized | 2026-08-16 12:17 | yields.llama.fi/pools pool=ac61ee82-2fe4-4f9b-a9cd-7fb33f598859 |
| deposit_rate 一年@boc | 0.95 | pct_annualized | 2026-08-16 12:35 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三个月@boc | 0.65 | pct_annualized | 2026-08-16 12:35 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三年@boc | 1.25 | pct_annualized | 2026-08-16 12:35 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 二年@boc | 1.05 | pct_annualized | 2026-08-16 12:35 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 五年@boc | 1.3 | pct_annualized | 2026-08-16 12:35 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 半年@boc | 0.85 | pct_annualized | 2026-08-16 12:35 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 活期@boc | 0.05 | pct_annualized | 2026-08-16 12:35 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| funding BTC@binance | -0.1577 | pct_annualized | 2026-08-16 12:35 | fapi/v1/premiumIndex rate=-0.00000144 per8h |
| funding BTC@okx | 6.324 | pct_annualized | 2026-08-16 12:35 | api/v5/public/funding-rate rate=0.0000577530287087 per8h |
| funding ETH@binance | -1.378 | pct_annualized | 2026-08-16 12:35 | fapi/v1/premiumIndex rate=-0.00001258 per8h |
| funding ETH@okx | 5.177 | pct_annualized | 2026-08-16 12:35 | api/v5/public/funding-rate rate=0.0000472784703932 per8h |
| funding TRX@binance | -2.872 | pct_annualized | 2026-08-16 12:35 | fapi/v1/premiumIndex rate=-0.00002623 per8h |
| funding TRX@okx | -9.629 | pct_annualized | 2026-08-16 12:35 | api/v5/public/funding-rate rate=-0.0000879389323551 per8h |
| fx USDCNH@sina | 6.744 | price | 2026-08-15 04:59 | hq.sinajs.cn/list=fx_susdcnh |
| iv BTC@deribit | 34.98 | pct | 2026-08-16 08:00 | api/v2/public/get_volatility_index_data DVOL |
| iv ETH@deribit | 47.57 | pct | 2026-08-16 08:00 | api/v2/public/get_volatility_index_data DVOL |
| reverse_repo GC001@sina | 0.865 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sh204001 |
| reverse_repo R-001@sina | 0.84 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sz131810 |
| ticker BTC@binance | 6.309e+04 | price | 2026-08-16 12:35 | fapi/v1/ticker/price |
| ticker BTC@okx | 6.309e+04 | price | 2026-08-16 12:35 | api/v5/market/tickers |
| ticker ETH@binance | 1882 | price | 2026-08-16 12:35 | fapi/v1/ticker/price |
| ticker ETH@okx | 1882 | price | 2026-08-16 12:35 | api/v5/market/tickers |
| ticker TRX@binance | 0.3309 | price | 2026-08-16 12:35 | fapi/v1/ticker/price |
| ticker TRX@okx | 0.3309 | price | 2026-08-16 12:35 | api/v5/market/tickers |

### 快照 2026-08-16 12:35（已过期 · 被 2026-08-16 12:35 快照取代）

| 事实 | 值 | 单位 | 采集时刻 | 来源 |
|------|-----|------|---------|------|
| calendar month_end@rule | 15 | days | 2026-08-16 12:35 | rule |
| calendar quarter_end@rule | 45 | days | 2026-08-16 12:35 | rule |
| calendar thursday@rule | 4 | days | 2026-08-16 12:35 | rule |
| defi_rate BUIDL@blackrock-buidl | 3.567 | pct_annualized | 2026-08-16 12:17 | yields.llama.fi/pools pool=b663ca59-c7e6-4435-ae4a-28d339ce6a15 |
| defi_rate STEAKUSDC@morpho-blue | 4.158 | pct_annualized | 2026-08-16 12:17 | yields.llama.fi/pools pool=931ea9be-5f4d-428e-beaf-205fc5b4e2b5 |
| defi_rate SUSDE@ethena-usde | 4.344 | pct_annualized | 2026-08-16 12:17 | yields.llama.fi/pools pool=66985a81-9c51-46ca-9977-42b4fe7bc6df |
| defi_rate USDC@aave-v3 | 12.57 | pct_annualized | 2026-08-16 12:17 | yields.llama.fi/pools pool=aa70268e-4b52-42bf-a116-608b370f9501 |
| defi_rate USDY@ondo-yield-assets | 3.55 | pct_annualized | 2026-08-16 12:17 | yields.llama.fi/pools pool=ac61ee82-2fe4-4f9b-a9cd-7fb33f598859 |
| deposit_rate 一年@boc | 0.95 | pct_annualized | 2026-08-16 12:35 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三个月@boc | 0.65 | pct_annualized | 2026-08-16 12:35 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三年@boc | 1.25 | pct_annualized | 2026-08-16 12:35 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 二年@boc | 1.05 | pct_annualized | 2026-08-16 12:35 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 五年@boc | 1.3 | pct_annualized | 2026-08-16 12:35 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 半年@boc | 0.85 | pct_annualized | 2026-08-16 12:35 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 活期@boc | 0.05 | pct_annualized | 2026-08-16 12:35 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| funding BTC@binance | -0.1577 | pct_annualized | 2026-08-16 12:35 | fapi/v1/premiumIndex rate=-0.00000144 per8h |
| funding BTC@okx | 6.324 | pct_annualized | 2026-08-16 12:35 | api/v5/public/funding-rate rate=0.0000577530287087 per8h |
| funding ETH@binance | -1.378 | pct_annualized | 2026-08-16 12:35 | fapi/v1/premiumIndex rate=-0.00001258 per8h |
| funding ETH@okx | 5.177 | pct_annualized | 2026-08-16 12:35 | api/v5/public/funding-rate rate=0.0000472784703932 per8h |
| funding TRX@binance | -2.872 | pct_annualized | 2026-08-16 12:35 | fapi/v1/premiumIndex rate=-0.00002623 per8h |
| funding TRX@okx | -9.629 | pct_annualized | 2026-08-16 12:35 | api/v5/public/funding-rate rate=-0.0000879389323551 per8h |
| fx USDCNH@sina | 6.744 | price | 2026-08-15 04:59 | hq.sinajs.cn/list=fx_susdcnh |
| iv BTC@deribit | 34.98 | pct | 2026-08-16 08:00 | api/v2/public/get_volatility_index_data DVOL |
| iv ETH@deribit | 47.57 | pct | 2026-08-16 08:00 | api/v2/public/get_volatility_index_data DVOL |
| reverse_repo GC001@sina | 0.865 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sh204001 |
| reverse_repo R-001@sina | 0.84 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sz131810 |
| ticker BTC@binance | 6.309e+04 | price | 2026-08-16 12:35 | fapi/v1/ticker/price |
| ticker BTC@okx | 6.309e+04 | price | 2026-08-16 12:35 | api/v5/market/tickers |
| ticker ETH@binance | 1882 | price | 2026-08-16 12:35 | fapi/v1/ticker/price |
| ticker ETH@okx | 1882 | price | 2026-08-16 12:35 | api/v5/market/tickers |
| ticker TRX@binance | 0.3309 | price | 2026-08-16 12:35 | fapi/v1/ticker/price |
| ticker TRX@okx | 0.3309 | price | 2026-08-16 12:35 | api/v5/market/tickers |

### 快照 2026-08-16 12:17（已过期 · 被 2026-08-16 12:35 快照取代）

| 事实 | 值 | 单位 | 采集时刻 | 来源 |
|------|-----|------|---------|------|
| calendar month_end@rule | 15 | days | 2026-08-16 12:17 | rule |
| calendar quarter_end@rule | 45 | days | 2026-08-16 12:17 | rule |
| calendar thursday@rule | 4 | days | 2026-08-16 12:17 | rule |
| defi_rate BUIDL@blackrock-buidl | 3.567 | pct_annualized | 2026-08-16 12:16 | yields.llama.fi/pools pool=b663ca59-c7e6-4435-ae4a-28d339ce6a15 |
| defi_rate STEAKUSDC@morpho-blue | 4.158 | pct_annualized | 2026-08-16 12:16 | yields.llama.fi/pools pool=931ea9be-5f4d-428e-beaf-205fc5b4e2b5 |
| defi_rate SUSDE@ethena-usde | 4.344 | pct_annualized | 2026-08-16 12:16 | yields.llama.fi/pools pool=66985a81-9c51-46ca-9977-42b4fe7bc6df |
| defi_rate USDC@aave-v3 | 12.57 | pct_annualized | 2026-08-16 12:16 | yields.llama.fi/pools pool=aa70268e-4b52-42bf-a116-608b370f9501 |
| defi_rate USDY@ondo-yield-assets | 3.55 | pct_annualized | 2026-08-16 12:16 | yields.llama.fi/pools pool=ac61ee82-2fe4-4f9b-a9cd-7fb33f598859 |
| deposit_rate 一年@boc | 0.95 | pct_annualized | 2026-08-16 12:17 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三个月@boc | 0.65 | pct_annualized | 2026-08-16 12:17 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三年@boc | 1.25 | pct_annualized | 2026-08-16 12:17 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 二年@boc | 1.05 | pct_annualized | 2026-08-16 12:17 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 五年@boc | 1.3 | pct_annualized | 2026-08-16 12:17 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 半年@boc | 0.85 | pct_annualized | 2026-08-16 12:17 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 活期@boc | 0.05 | pct_annualized | 2026-08-16 12:17 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| funding BTC@binance | -0.023 | pct_annualized | 2026-08-16 12:17 | fapi/v1/premiumIndex rate=-0.00000021 per8h |
| funding BTC@okx | 6.825 | pct_annualized | 2026-08-16 12:17 | api/v5/public/funding-rate rate=0.0000623297846024 per8h |
| funding ETH@binance | -1.395 | pct_annualized | 2026-08-16 12:17 | fapi/v1/premiumIndex rate=-0.00001274 per8h |
| funding ETH@okx | 5.091 | pct_annualized | 2026-08-16 12:17 | api/v5/public/funding-rate rate=0.0000464929956024 per8h |
| funding TRX@binance | -3.878 | pct_annualized | 2026-08-16 12:17 | fapi/v1/premiumIndex rate=-0.00003542 per8h |
| funding TRX@okx | -10.87 | pct_annualized | 2026-08-16 12:17 | api/v5/public/funding-rate rate=-0.0000992819434757 per8h |
| fx USDCNH@sina | 6.744 | price | 2026-08-15 04:59 | hq.sinajs.cn/list=fx_susdcnh |
| iv BTC@deribit | 34.96 | pct | 2026-08-16 08:00 | api/v2/public/get_volatility_index_data DVOL |
| iv ETH@deribit | 47.57 | pct | 2026-08-16 08:00 | api/v2/public/get_volatility_index_data DVOL |
| reverse_repo GC001@sina | 0.865 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sh204001 |
| reverse_repo R-001@sina | 0.84 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sz131810 |
| ticker BTC@binance | 6.31e+04 | price | 2026-08-16 12:16 | fapi/v1/ticker/price |
| ticker BTC@okx | 6.31e+04 | price | 2026-08-16 12:17 | api/v5/market/tickers |
| ticker ETH@binance | 1882 | price | 2026-08-16 12:16 | fapi/v1/ticker/price |
| ticker ETH@okx | 1882 | price | 2026-08-16 12:17 | api/v5/market/tickers |
| ticker TRX@binance | 0.3309 | price | 2026-08-16 12:16 | fapi/v1/ticker/price |
| ticker TRX@okx | 0.3308 | price | 2026-08-16 12:17 | api/v5/market/tickers |

### 快照 2026-08-16 12:09（已过期 · 被 2026-08-16 12:17 快照取代）

| 事实 | 值 | 单位 | 采集时刻 | 来源 |
|------|-----|------|---------|------|
| calendar month_end@rule | 15 | days | 2026-08-16 00:09 | rule |
| calendar quarter_end@rule | 45 | days | 2026-08-16 00:09 | rule |
| calendar thursday@rule | 4 | days | 2026-08-16 00:09 | rule |
| defi_rate BUIDL@blackrock-buidl | 3.567 | pct_annualized | 2026-08-16 11:42 | yields.llama.fi/pools pool=b663ca59-c7e6-4435-ae4a-28d339ce6a15 |
| defi_rate STEAKUSDC@morpho-blue | 4.158 | pct_annualized | 2026-08-16 11:42 | yields.llama.fi/pools pool=931ea9be-5f4d-428e-beaf-205fc5b4e2b5 |
| defi_rate SUSDE@ethena-usde | 4.344 | pct_annualized | 2026-08-16 11:42 | yields.llama.fi/pools pool=66985a81-9c51-46ca-9977-42b4fe7bc6df |
| defi_rate USDC@aave-v3 | 12.57 | pct_annualized | 2026-08-16 11:42 | yields.llama.fi/pools pool=aa70268e-4b52-42bf-a116-608b370f9501 |
| defi_rate USDY@ondo-yield-assets | 3.55 | pct_annualized | 2026-08-16 11:42 | yields.llama.fi/pools pool=ac61ee82-2fe4-4f9b-a9cd-7fb33f598859 |
| deposit_rate 一年@boc | 0.95 | pct_annualized | 2026-08-16 11:15 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三个月@boc | 0.65 | pct_annualized | 2026-08-16 11:15 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三年@boc | 1.25 | pct_annualized | 2026-08-16 11:15 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 二年@boc | 1.05 | pct_annualized | 2026-08-16 11:15 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 五年@boc | 1.3 | pct_annualized | 2026-08-16 11:15 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 半年@boc | 0.85 | pct_annualized | 2026-08-16 11:15 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 活期@boc | 0.05 | pct_annualized | 2026-08-16 11:15 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| funding BTC@binance | 0.15 | pct_annualized | 2026-08-16 12:07 | fapi/v1/premiumIndex rate=0.00000137 per8h |
| funding BTC@okx | 7.039 | pct_annualized | 2026-08-16 12:07 | api/v5/public/funding-rate rate=0.0000642846507513 per8h |
| funding ETH@binance | -1.144 | pct_annualized | 2026-08-16 12:07 | fapi/v1/premiumIndex rate=-0.00001045 per8h |
| funding ETH@okx | 5.144 | pct_annualized | 2026-08-16 12:07 | api/v5/public/funding-rate rate=0.0000469810345820 per8h |
| funding TRX@binance | -3.904 | pct_annualized | 2026-08-16 12:07 | fapi/v1/premiumIndex rate=-0.00003565 per8h |
| funding TRX@okx | -11.2 | pct_annualized | 2026-08-16 12:07 | api/v5/public/funding-rate rate=-0.0001022767047599 per8h |
| fx USDCNH@sina | 6.744 | price | 2026-08-15 04:59 | hq.sinajs.cn/list=fx_susdcnh |
| iv BTC@deribit | 34.96 | pct | 2026-08-16 08:00 | api/v2/public/get_volatility_index_data DVOL |
| iv ETH@deribit | 47.57 | pct | 2026-08-16 08:00 | api/v2/public/get_volatility_index_data DVOL |
| reverse_repo GC001@sina | 0.865 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sh204001 |
| reverse_repo R-001@sina | 0.84 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sz131810 |
| ticker BTC@binance | 6.31e+04 | price | 2026-08-16 12:09 | fapi/v1/ticker/price |
| ticker BTC@okx | 6.31e+04 | price | 2026-08-16 12:06 | api/v5/market/tickers |
| ticker ETH@binance | 1883 | price | 2026-08-16 12:09 | fapi/v1/ticker/price |
| ticker ETH@okx | 1883 | price | 2026-08-16 12:06 | api/v5/market/tickers |
| ticker TRX@binance | 0.3309 | price | 2026-08-16 12:09 | fapi/v1/ticker/price |
| ticker TRX@okx | 0.3308 | price | 2026-08-16 12:06 | api/v5/market/tickers |

### 快照 2026-08-16 08:39（已过期 · 被 2026-08-16 12:09 快照取代）

| 事实 | 值 | 单位 | 采集时刻 | 来源 |
|------|-----|------|---------|------|
| calendar month_end@rule | 15 | days | 2026-08-16 00:09 | rule |
| calendar quarter_end@rule | 45 | days | 2026-08-16 00:09 | rule |
| calendar thursday@rule | 4 | days | 2026-08-16 00:09 | rule |
| defi_rate BUIDL@blackrock-buidl | 3.567 | pct_annualized | 2026-08-16 08:37 | yields.llama.fi/pools pool=b663ca59-c7e6-4435-ae4a-28d339ce6a15 |
| defi_rate STEAKUSDC@morpho-blue | 4.16 | pct_annualized | 2026-08-16 08:37 | yields.llama.fi/pools pool=931ea9be-5f4d-428e-beaf-205fc5b4e2b5 |
| defi_rate SUSDE@ethena-usde | 4.345 | pct_annualized | 2026-08-16 08:37 | yields.llama.fi/pools pool=66985a81-9c51-46ca-9977-42b4fe7bc6df |
| defi_rate USDC@aave-v3 | 12.57 | pct_annualized | 2026-08-16 08:37 | yields.llama.fi/pools pool=aa70268e-4b52-42bf-a116-608b370f9501 |
| defi_rate USDY@ondo-yield-assets | 3.55 | pct_annualized | 2026-08-16 08:37 | yields.llama.fi/pools pool=ac61ee82-2fe4-4f9b-a9cd-7fb33f598859 |
| deposit_rate 一年@boc | 0.95 | pct_annualized | 2026-08-16 08:12 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三个月@boc | 0.65 | pct_annualized | 2026-08-16 08:12 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三年@boc | 1.25 | pct_annualized | 2026-08-16 08:12 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 二年@boc | 1.05 | pct_annualized | 2026-08-16 08:12 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 五年@boc | 1.3 | pct_annualized | 2026-08-16 08:12 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 半年@boc | 0.85 | pct_annualized | 2026-08-16 08:12 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 活期@boc | 0.05 | pct_annualized | 2026-08-16 08:12 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| funding BTC@binance | 0.853 | pct_annualized | 2026-08-16 08:35 | fapi/v1/premiumIndex rate=0.00000779 per8h |
| funding BTC@okx | 6.778 | pct_annualized | 2026-08-16 08:35 | api/v5/public/funding-rate rate=0.0000618979555143 per8h |
| funding ETH@binance | -1.413 | pct_annualized | 2026-08-16 08:35 | fapi/v1/premiumIndex rate=-0.00001290 per8h |
| funding ETH@okx | 3.433 | pct_annualized | 2026-08-16 08:35 | api/v5/public/funding-rate rate=0.0000313486128818 per8h |
| funding TRX@binance | -2.943 | pct_annualized | 2026-08-16 08:35 | fapi/v1/premiumIndex rate=-0.00002688 per8h |
| funding TRX@okx | -3.227 | pct_annualized | 2026-08-16 08:35 | api/v5/public/funding-rate rate=-0.0000294701495439 per8h |
| fx USDCNH@sina | 6.744 | price | 2026-08-15 04:59 | hq.sinajs.cn/list=fx_susdcnh |
| iv BTC@deribit | 34.98 | pct | 2026-08-16 08:00 | api/v2/public/get_volatility_index_data DVOL |
| iv ETH@deribit | 47.57 | pct | 2026-08-16 08:00 | api/v2/public/get_volatility_index_data DVOL |
| reverse_repo GC001@sina | 0.865 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sh204001 |
| reverse_repo R-001@sina | 0.84 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sz131810 |
| ticker BTC@binance | 6.304e+04 | price | 2026-08-16 08:39 | fapi/v1/ticker/price |
| ticker BTC@okx | 6.305e+04 | price | 2026-08-16 08:38 | api/v5/market/tickers |
| ticker ETH@binance | 1882 | price | 2026-08-16 08:39 | fapi/v1/ticker/price |
| ticker ETH@okx | 1882 | price | 2026-08-16 08:38 | api/v5/market/tickers |
| ticker TRX@binance | 0.3309 | price | 2026-08-16 08:39 | fapi/v1/ticker/price |
| ticker TRX@okx | 0.3308 | price | 2026-08-16 08:38 | api/v5/market/tickers |

### 快照 2026-08-16 05:32（已过期 · 被 2026-08-16 08:39 快照取代）

| 事实 | 值 | 单位 | 采集时刻 | 来源 |
|------|-----|------|---------|------|
| calendar month_end@rule | 15 | days | 2026-08-16 00:09 | rule |
| calendar quarter_end@rule | 45 | days | 2026-08-16 00:09 | rule |
| calendar thursday@rule | 4 | days | 2026-08-16 00:09 | rule |
| defi_rate BUIDL@blackrock-buidl | 3.567 | pct_annualized | 2026-08-16 05:04 | yields.llama.fi/pools pool=b663ca59-c7e6-4435-ae4a-28d339ce6a15 |
| defi_rate STEAKUSDC@morpho-blue | 4.163 | pct_annualized | 2026-08-16 05:04 | yields.llama.fi/pools pool=931ea9be-5f4d-428e-beaf-205fc5b4e2b5 |
| defi_rate SUSDE@ethena-usde | 4.345 | pct_annualized | 2026-08-16 05:04 | yields.llama.fi/pools pool=66985a81-9c51-46ca-9977-42b4fe7bc6df |
| defi_rate USDC@aave-v3 | 3.271 | pct_annualized | 2026-08-16 05:04 | yields.llama.fi/pools pool=aa70268e-4b52-42bf-a116-608b370f9501 |
| defi_rate USDY@ondo-yield-assets | 3.55 | pct_annualized | 2026-08-16 05:04 | yields.llama.fi/pools pool=ac61ee82-2fe4-4f9b-a9cd-7fb33f598859 |
| deposit_rate 一年@boc | 0.95 | pct_annualized | 2026-08-16 05:17 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三个月@boc | 0.65 | pct_annualized | 2026-08-16 05:17 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三年@boc | 1.25 | pct_annualized | 2026-08-16 05:17 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 二年@boc | 1.05 | pct_annualized | 2026-08-16 05:17 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 五年@boc | 1.3 | pct_annualized | 2026-08-16 05:17 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 半年@boc | 0.85 | pct_annualized | 2026-08-16 05:17 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 活期@boc | 0.05 | pct_annualized | 2026-08-16 05:17 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| funding BTC@binance | 2.44 | pct_annualized | 2026-08-16 05:27 | fapi/v1/premiumIndex rate=0.00002228 per8h |
| funding BTC@okx | 7.194 | pct_annualized | 2026-08-16 05:29 | api/v5/public/funding-rate rate=0.0000657010215581 per8h |
| funding ETH@binance | 4.467 | pct_annualized | 2026-08-16 05:27 | fapi/v1/premiumIndex rate=0.00004079 per8h |
| funding ETH@okx | 5.303 | pct_annualized | 2026-08-16 05:29 | api/v5/public/funding-rate rate=0.0000484256942081 per8h |
| funding TRX@binance | 2.261 | pct_annualized | 2026-08-16 05:27 | fapi/v1/premiumIndex rate=0.00002065 per8h |
| funding TRX@okx | -0.4259 | pct_annualized | 2026-08-16 05:29 | api/v5/public/funding-rate rate=-0.0000038898820750 per8h |
| fx USDCNH@sina | 6.744 | price | 2026-08-15 04:59 | hq.sinajs.cn/list=fx_susdcnh |
| iv BTC@deribit | 34.82 | pct | 2026-08-15 08:00 | api/v2/public/get_volatility_index_data DVOL |
| iv ETH@deribit | 47.11 | pct | 2026-08-15 08:00 | api/v2/public/get_volatility_index_data DVOL |
| reverse_repo GC001@sina | 0.865 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sh204001 |
| reverse_repo R-001@sina | 0.84 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sz131810 |
| ticker BTC@binance | 6.312e+04 | price | 2026-08-16 05:29 | fapi/v1/ticker/price |
| ticker BTC@okx | 6.311e+04 | price | 2026-08-16 05:31 | api/v5/market/tickers |
| ticker ETH@binance | 1884 | price | 2026-08-16 05:29 | fapi/v1/ticker/price |
| ticker ETH@okx | 1884 | price | 2026-08-16 05:31 | api/v5/market/tickers |
| ticker TRX@binance | 0.3315 | price | 2026-08-16 05:29 | fapi/v1/ticker/price |
| ticker TRX@okx | 0.3314 | price | 2026-08-16 05:31 | api/v5/market/tickers |

### 快照 2026-08-16 04:43（已过期 · 被 2026-08-16 05:32 快照取代）

| 事实 | 值 | 单位 | 采集时刻 | 来源 |
|------|-----|------|---------|------|
| calendar month_end@rule | 15 | days | 2026-08-16 00:09 | rule |
| calendar quarter_end@rule | 45 | days | 2026-08-16 00:09 | rule |
| calendar thursday@rule | 4 | days | 2026-08-16 00:09 | rule |
| defi_rate BUIDL@blackrock-buidl | 3.567 | pct_annualized | 2026-08-16 04:36 | yields.llama.fi/pools pool=b663ca59-c7e6-4435-ae4a-28d339ce6a15 |
| defi_rate STEAKUSDC@morpho-blue | 4.163 | pct_annualized | 2026-08-16 04:36 | yields.llama.fi/pools pool=931ea9be-5f4d-428e-beaf-205fc5b4e2b5 |
| defi_rate SUSDE@ethena-usde | 4.345 | pct_annualized | 2026-08-16 04:36 | yields.llama.fi/pools pool=66985a81-9c51-46ca-9977-42b4fe7bc6df |
| defi_rate USDC@aave-v3 | 3.271 | pct_annualized | 2026-08-16 04:36 | yields.llama.fi/pools pool=aa70268e-4b52-42bf-a116-608b370f9501 |
| defi_rate USDY@ondo-yield-assets | 3.55 | pct_annualized | 2026-08-16 04:36 | yields.llama.fi/pools pool=ac61ee82-2fe4-4f9b-a9cd-7fb33f598859 |
| deposit_rate 一年@boc | 0.95 | pct_annualized | 2026-08-16 04:13 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三个月@boc | 0.65 | pct_annualized | 2026-08-16 04:13 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三年@boc | 1.25 | pct_annualized | 2026-08-16 04:13 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 二年@boc | 1.05 | pct_annualized | 2026-08-16 04:13 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 五年@boc | 1.3 | pct_annualized | 2026-08-16 04:13 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 半年@boc | 0.85 | pct_annualized | 2026-08-16 04:13 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 活期@boc | 0.05 | pct_annualized | 2026-08-16 04:13 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| funding BTC@binance | 2.459 | pct_annualized | 2026-08-16 04:37 | fapi/v1/premiumIndex rate=0.00002246 per8h |
| funding BTC@okx | 6.971 | pct_annualized | 2026-08-16 04:40 | api/v5/public/funding-rate rate=0.0000636658035020 per8h |
| funding ETH@binance | 5.084 | pct_annualized | 2026-08-16 04:37 | fapi/v1/premiumIndex rate=0.00004643 per8h |
| funding ETH@okx | 4.603 | pct_annualized | 2026-08-16 04:40 | api/v5/public/funding-rate rate=0.0000420380901801 per8h |
| funding TRX@binance | 6.432 | pct_annualized | 2026-08-16 04:37 | fapi/v1/premiumIndex rate=0.00005874 per8h |
| funding TRX@okx | 1.038 | pct_annualized | 2026-08-16 04:40 | api/v5/public/funding-rate rate=0.0000094827647804 per8h |
| fx USDCNH@sina | 6.744 | price | 2026-08-15 04:59 | hq.sinajs.cn/list=fx_susdcnh |
| iv BTC@deribit | 34.82 | pct | 2026-08-15 08:00 | api/v2/public/get_volatility_index_data DVOL |
| iv ETH@deribit | 47.13 | pct | 2026-08-15 08:00 | api/v2/public/get_volatility_index_data DVOL |
| reverse_repo GC001@sina | 0.865 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sh204001 |
| reverse_repo R-001@sina | 0.84 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sz131810 |
| ticker BTC@binance | 6.305e+04 | price | 2026-08-16 04:40 | fapi/v1/ticker/price |
| ticker BTC@okx | 6.305e+04 | price | 2026-08-16 04:42 | api/v5/market/tickers |
| ticker ETH@binance | 1882 | price | 2026-08-16 04:40 | fapi/v1/ticker/price |
| ticker ETH@okx | 1882 | price | 2026-08-16 04:42 | api/v5/market/tickers |
| ticker TRX@binance | 0.3314 | price | 2026-08-16 04:40 | fapi/v1/ticker/price |
| ticker TRX@okx | 0.3313 | price | 2026-08-16 04:42 | api/v5/market/tickers |

### 快照 2026-08-16 00:35（已过期 · 被 2026-08-16 04:43 快照取代）

| 事实 | 值 | 单位 | 采集时刻 | 来源 |
|------|-----|------|---------|------|
| calendar month_end@rule | 15 | days | 2026-08-16 00:09 | rule |
| calendar quarter_end@rule | 45 | days | 2026-08-16 00:09 | rule |
| calendar thursday@rule | 4 | days | 2026-08-16 00:09 | rule |
| defi_rate BUIDL@blackrock-buidl | 3.567 | pct_annualized | 2026-08-16 00:24 | yields.llama.fi/pools pool=b663ca59-c7e6-4435-ae4a-28d339ce6a15 |
| defi_rate STEAKUSDC@morpho-blue | 4.163 | pct_annualized | 2026-08-16 00:24 | yields.llama.fi/pools pool=931ea9be-5f4d-428e-beaf-205fc5b4e2b5 |
| defi_rate SUSDE@ethena-usde | 4.34 | pct_annualized | 2026-08-16 00:24 | yields.llama.fi/pools pool=66985a81-9c51-46ca-9977-42b4fe7bc6df |
| defi_rate USDC@aave-v3 | 3.271 | pct_annualized | 2026-08-16 00:24 | yields.llama.fi/pools pool=aa70268e-4b52-42bf-a116-608b370f9501 |
| defi_rate USDY@ondo-yield-assets | 3.55 | pct_annualized | 2026-08-16 00:24 | yields.llama.fi/pools pool=ac61ee82-2fe4-4f9b-a9cd-7fb33f598859 |
| deposit_rate 一年@boc | 0.95 | pct_annualized | 2026-08-16 00:09 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三个月@boc | 0.65 | pct_annualized | 2026-08-16 00:09 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三年@boc | 1.25 | pct_annualized | 2026-08-16 00:09 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 二年@boc | 1.05 | pct_annualized | 2026-08-16 00:09 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 五年@boc | 1.3 | pct_annualized | 2026-08-16 00:09 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 半年@boc | 0.85 | pct_annualized | 2026-08-16 00:09 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 活期@boc | 0.05 | pct_annualized | 2026-08-16 00:09 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| funding BTC@binance | 5.452 | pct_annualized | 2026-08-16 00:28 | fapi/v1/premiumIndex rate=0.00004979 per8h |
| funding BTC@okx | 6.819 | pct_annualized | 2026-08-16 00:33 | api/v5/public/funding-rate rate=0.0000622777422565 per8h |
| funding ETH@binance | 6.482 | pct_annualized | 2026-08-16 00:28 | fapi/v1/premiumIndex rate=0.00005920 per8h |
| funding ETH@okx | 3.81 | pct_annualized | 2026-08-16 00:33 | api/v5/public/funding-rate rate=0.0000347980699077 per8h |
| funding TRX@binance | 3.963 | pct_annualized | 2026-08-16 00:28 | fapi/v1/premiumIndex rate=0.00003619 per8h |
| funding TRX@okx | -2.783 | pct_annualized | 2026-08-16 00:33 | api/v5/public/funding-rate rate=-0.0000254154167287 per8h |
| fx USDCNH@sina | 6.744 | price | 2026-08-15 04:59 | hq.sinajs.cn/list=fx_susdcnh |
| iv BTC@deribit | 34.79 | pct | 2026-08-15 08:00 | api/v2/public/get_volatility_index_data DVOL |
| iv ETH@deribit | 47.2 | pct | 2026-08-15 08:00 | api/v2/public/get_volatility_index_data DVOL |
| reverse_repo GC001@sina | 0.865 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sh204001 |
| reverse_repo R-001@sina | 0.84 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sz131810 |
| ticker BTC@binance | 6.309e+04 | price | 2026-08-16 00:32 | fapi/v1/ticker/price |
| ticker BTC@okx | 6.306e+04 | price | 2026-08-16 00:34 | api/v5/market/tickers |
| ticker ETH@binance | 1885 | price | 2026-08-16 00:32 | fapi/v1/ticker/price |
| ticker ETH@okx | 1884 | price | 2026-08-16 00:34 | api/v5/market/tickers |
| ticker TRX@binance | 0.332 | price | 2026-08-16 00:32 | fapi/v1/ticker/price |
| ticker TRX@okx | 0.332 | price | 2026-08-16 00:34 | api/v5/market/tickers |

### 快照 2026-08-16 00:09（已过期 · 被 2026-08-16 00:35 快照取代）

| 事实 | 值 | 单位 | 采集时刻 | 来源 |
|------|-----|------|---------|------|
| calendar month_end@rule | 15 | days | 2026-08-16 00:09 | rule |
| calendar quarter_end@rule | 45 | days | 2026-08-16 00:09 | rule |
| calendar thursday@rule | 4 | days | 2026-08-16 00:09 | rule |
| defi_rate BUIDL@blackrock-buidl | 3.567 | pct_annualized | 2026-08-15 23:55 | yields.llama.fi/pools pool=b663ca59-c7e6-4435-ae4a-28d339ce6a15 |
| defi_rate STEAKUSDC@morpho-blue | 4.163 | pct_annualized | 2026-08-15 23:55 | yields.llama.fi/pools pool=931ea9be-5f4d-428e-beaf-205fc5b4e2b5 |
| defi_rate SUSDE@ethena-usde | 4.437 | pct_annualized | 2026-08-15 23:55 | yields.llama.fi/pools pool=66985a81-9c51-46ca-9977-42b4fe7bc6df |
| defi_rate USDC@aave-v3 | 3.266 | pct_annualized | 2026-08-15 23:55 | yields.llama.fi/pools pool=aa70268e-4b52-42bf-a116-608b370f9501 |
| defi_rate USDY@ondo-yield-assets | 3.55 | pct_annualized | 2026-08-15 23:55 | yields.llama.fi/pools pool=ac61ee82-2fe4-4f9b-a9cd-7fb33f598859 |
| deposit_rate 一年@boc | 0.95 | pct_annualized | 2026-08-16 00:09 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三个月@boc | 0.65 | pct_annualized | 2026-08-16 00:09 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三年@boc | 1.25 | pct_annualized | 2026-08-16 00:09 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 二年@boc | 1.05 | pct_annualized | 2026-08-16 00:09 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 五年@boc | 1.3 | pct_annualized | 2026-08-16 00:09 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 半年@boc | 0.85 | pct_annualized | 2026-08-16 00:09 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 活期@boc | 0.05 | pct_annualized | 2026-08-16 00:09 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| funding BTC@binance | 5.76 | pct_annualized | 2026-08-16 00:05 | fapi/v1/premiumIndex rate=0.00005260 per8h |
| funding BTC@okx | 6.943 | pct_annualized | 2026-08-16 00:09 | api/v5/public/funding-rate rate=0.0000634090464492 per8h |
| funding ETH@binance | 6.535 | pct_annualized | 2026-08-16 00:05 | fapi/v1/premiumIndex rate=0.00005968 per8h |
| funding ETH@okx | 4.006 | pct_annualized | 2026-08-16 00:09 | api/v5/public/funding-rate rate=0.0000365868801159 per8h |
| funding TRX@binance | 1.734 | pct_annualized | 2026-08-16 00:05 | fapi/v1/premiumIndex rate=0.00001584 per8h |
| funding TRX@okx | -3.785 | pct_annualized | 2026-08-16 00:09 | api/v5/public/funding-rate rate=-0.0000345648450309 per8h |
| fx USDCNH@sina | 6.744 | price | 2026-08-15 04:59 | hq.sinajs.cn/list=fx_susdcnh |
| iv BTC@deribit | 34.93 | pct | 2026-08-15 08:00 | api/v2/public/get_volatility_index_data DVOL |
| iv ETH@deribit | 47.13 | pct | 2026-08-15 08:00 | api/v2/public/get_volatility_index_data DVOL |
| reverse_repo GC001@sina | 0.865 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sh204001 |
| reverse_repo R-001@sina | 0.84 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sz131810 |
| ticker BTC@binance | 6.304e+04 | price | 2026-08-16 00:08 | fapi/v1/ticker/price |
| ticker BTC@okx | 6.305e+04 | price | 2026-08-16 00:09 | api/v5/market/tickers |
| ticker ETH@binance | 1883 | price | 2026-08-16 00:08 | fapi/v1/ticker/price |
| ticker ETH@okx | 1884 | price | 2026-08-16 00:09 | api/v5/market/tickers |
| ticker TRX@binance | 0.3315 | price | 2026-08-16 00:08 | fapi/v1/ticker/price |
| ticker TRX@okx | 0.3315 | price | 2026-08-16 00:09 | api/v5/market/tickers |

### 快照 2026-08-16 00:05（已过期 · 被 2026-08-16 00:09 快照取代）

| 事实 | 值 | 单位 | 采集时刻 | 来源 |
|------|-----|------|---------|------|
| calendar month_end@rule | 15 | days | 2026-08-16 00:05 | rule |
| calendar quarter_end@rule | 45 | days | 2026-08-16 00:05 | rule |
| calendar thursday@rule | 4 | days | 2026-08-16 00:05 | rule |
| defi_rate BUIDL@blackrock-buidl | 3.567 | pct_annualized | 2026-08-15 23:55 | yields.llama.fi/pools pool=b663ca59-c7e6-4435-ae4a-28d339ce6a15 |
| defi_rate STEAKUSDC@morpho-blue | 4.163 | pct_annualized | 2026-08-15 23:55 | yields.llama.fi/pools pool=931ea9be-5f4d-428e-beaf-205fc5b4e2b5 |
| defi_rate SUSDE@ethena-usde | 4.437 | pct_annualized | 2026-08-15 23:55 | yields.llama.fi/pools pool=66985a81-9c51-46ca-9977-42b4fe7bc6df |
| defi_rate USDC@aave-v3 | 3.266 | pct_annualized | 2026-08-15 23:55 | yields.llama.fi/pools pool=aa70268e-4b52-42bf-a116-608b370f9501 |
| defi_rate USDY@ondo-yield-assets | 3.55 | pct_annualized | 2026-08-15 23:55 | yields.llama.fi/pools pool=ac61ee82-2fe4-4f9b-a9cd-7fb33f598859 |
| deposit_rate 一年@boc | 0.95 | pct_annualized | 2026-08-16 00:05 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三个月@boc | 0.65 | pct_annualized | 2026-08-16 00:05 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三年@boc | 1.25 | pct_annualized | 2026-08-16 00:05 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 二年@boc | 1.05 | pct_annualized | 2026-08-16 00:05 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 五年@boc | 1.3 | pct_annualized | 2026-08-16 00:05 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 半年@boc | 0.85 | pct_annualized | 2026-08-16 00:05 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 活期@boc | 0.05 | pct_annualized | 2026-08-16 00:05 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| funding BTC@binance | 5.76 | pct_annualized | 2026-08-16 00:05 | fapi/v1/premiumIndex rate=0.00005260 per8h |
| funding BTC@okx | 6.983 | pct_annualized | 2026-08-16 00:05 | api/v5/public/funding-rate rate=0.0000637721529245 per8h |
| funding ETH@binance | 6.535 | pct_annualized | 2026-08-16 00:05 | fapi/v1/premiumIndex rate=0.00005968 per8h |
| funding ETH@okx | 4.062 | pct_annualized | 2026-08-16 00:05 | api/v5/public/funding-rate rate=0.0000370932381066 per8h |
| funding TRX@binance | 1.734 | pct_annualized | 2026-08-16 00:05 | fapi/v1/premiumIndex rate=0.00001584 per8h |
| funding TRX@okx | -4.019 | pct_annualized | 2026-08-16 00:05 | api/v5/public/funding-rate rate=-0.0000366997750445 per8h |
| fx USDCNH@sina | 6.744 | price | 2026-08-15 04:59 | hq.sinajs.cn/list=fx_susdcnh |
| iv BTC@deribit | 34.98 | pct | 2026-08-15 08:00 | api/v2/public/get_volatility_index_data DVOL |
| iv ETH@deribit | 47.24 | pct | 2026-08-15 08:00 | api/v2/public/get_volatility_index_data DVOL |
| reverse_repo GC001@sina | 0.865 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sh204001 |
| reverse_repo R-001@sina | 0.84 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sz131810 |
| ticker BTC@binance | 6.306e+04 | price | 2026-08-16 00:05 | fapi/v1/ticker/price |
| ticker BTC@okx | 6.306e+04 | price | 2026-08-16 00:05 | api/v5/market/tickers |
| ticker ETH@binance | 1884 | price | 2026-08-16 00:05 | fapi/v1/ticker/price |
| ticker ETH@okx | 1884 | price | 2026-08-16 00:05 | api/v5/market/tickers |
| ticker TRX@binance | 0.3314 | price | 2026-08-16 00:05 | fapi/v1/ticker/price |
| ticker TRX@okx | 0.3314 | price | 2026-08-16 00:05 | api/v5/market/tickers |

### 快照 2026-08-15 23:55（已过期 · 被 2026-08-16 00:05 快照取代）

| 事实 | 值 | 单位 | 采集时刻 | 来源 |
|------|-----|------|---------|------|
| calendar month_end@rule | 16 | days | 2026-08-15 23:55 | rule |
| calendar quarter_end@rule | 46 | days | 2026-08-15 23:55 | rule |
| calendar thursday@rule | 5 | days | 2026-08-15 23:55 | rule |
| defi_rate BUIDL@blackrock-buidl | 3.567 | pct_annualized | 2026-08-15 23:55 | yields.llama.fi/pools pool=b663ca59-c7e6-4435-ae4a-28d339ce6a15 |
| defi_rate STEAKUSDC@morpho-blue | 4.163 | pct_annualized | 2026-08-15 23:55 | yields.llama.fi/pools pool=931ea9be-5f4d-428e-beaf-205fc5b4e2b5 |
| defi_rate SUSDE@ethena-usde | 4.437 | pct_annualized | 2026-08-15 23:55 | yields.llama.fi/pools pool=66985a81-9c51-46ca-9977-42b4fe7bc6df |
| defi_rate USDC@aave-v3 | 3.266 | pct_annualized | 2026-08-15 23:55 | yields.llama.fi/pools pool=aa70268e-4b52-42bf-a116-608b370f9501 |
| defi_rate USDY@ondo-yield-assets | 3.55 | pct_annualized | 2026-08-15 23:55 | yields.llama.fi/pools pool=ac61ee82-2fe4-4f9b-a9cd-7fb33f598859 |
| deposit_rate 一年@boc | 0.95 | pct_annualized | 2026-08-15 23:55 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三个月@boc | 0.65 | pct_annualized | 2026-08-15 23:55 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三年@boc | 1.25 | pct_annualized | 2026-08-15 23:55 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 二年@boc | 1.05 | pct_annualized | 2026-08-15 23:55 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 五年@boc | 1.3 | pct_annualized | 2026-08-15 23:55 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 半年@boc | 0.85 | pct_annualized | 2026-08-15 23:55 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 活期@boc | 0.05 | pct_annualized | 2026-08-15 23:55 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| funding BTC@binance | 5.937 | pct_annualized | 2026-08-15 23:55 | fapi/v1/premiumIndex rate=0.00005422 per8h |
| funding BTC@okx | 7.225 | pct_annualized | 2026-08-15 23:55 | api/v5/public/funding-rate rate=0.0000659776772162 per8h |
| funding ETH@binance | 6.612 | pct_annualized | 2026-08-15 23:55 | fapi/v1/premiumIndex rate=0.00006038 per8h |
| funding ETH@okx | 4.311 | pct_annualized | 2026-08-15 23:55 | api/v5/public/funding-rate rate=0.0000393722613317 per8h |
| funding TRX@binance | 1.307 | pct_annualized | 2026-08-15 23:55 | fapi/v1/premiumIndex rate=0.00001194 per8h |
| funding TRX@okx | -4.349 | pct_annualized | 2026-08-15 23:55 | api/v5/public/funding-rate rate=-0.0000397162574327 per8h |
| fx USDCNH@sina | 6.744 | price | 2026-08-15 04:59 | hq.sinajs.cn/list=fx_susdcnh |
| iv BTC@deribit | 34.84 | pct | 2026-08-15 08:00 | api/v2/public/get_volatility_index_data DVOL |
| iv ETH@deribit | 47.13 | pct | 2026-08-15 08:00 | api/v2/public/get_volatility_index_data DVOL |
| reverse_repo GC001@sina | 0.865 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sh204001 |
| reverse_repo R-001@sina | 0.84 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sz131810 |
| ticker BTC@binance | 6.307e+04 | price | 2026-08-15 23:55 | fapi/v1/ticker/price |
| ticker BTC@okx | 6.306e+04 | price | 2026-08-15 23:55 | api/v5/market/tickers |
| ticker ETH@binance | 1884 | price | 2026-08-15 23:55 | fapi/v1/ticker/price |
| ticker ETH@okx | 1884 | price | 2026-08-15 23:55 | api/v5/market/tickers |
| ticker TRX@binance | 0.3311 | price | 2026-08-15 23:55 | fapi/v1/ticker/price |
| ticker TRX@okx | 0.3311 | price | 2026-08-15 23:55 | api/v5/market/tickers |

### 快照 2026-08-15 23:50（已过期 · 被 2026-08-15 23:55 快照取代）

| 事实 | 值 | 单位 | 采集时刻 | 来源 |
|------|-----|------|---------|------|
| calendar month_end@rule | 16 | days | 2026-08-15 23:50 | rule |
| calendar quarter_end@rule | 46 | days | 2026-08-15 23:50 | rule |
| calendar thursday@rule | 5 | days | 2026-08-15 23:50 | rule |
| defi_rate BUIDL@blackrock-buidl | 3.567 | pct_annualized | 2026-08-15 23:32 | yields.llama.fi/pools pool=b663ca59-c7e6-4435-ae4a-28d339ce6a15 |
| defi_rate STEAKUSDC@morpho-blue | 4.163 | pct_annualized | 2026-08-15 23:32 | yields.llama.fi/pools pool=931ea9be-5f4d-428e-beaf-205fc5b4e2b5 |
| defi_rate SUSDE@ethena-usde | 4.437 | pct_annualized | 2026-08-15 23:32 | yields.llama.fi/pools pool=66985a81-9c51-46ca-9977-42b4fe7bc6df |
| defi_rate USDC@aave-v3 | 3.266 | pct_annualized | 2026-08-15 23:32 | yields.llama.fi/pools pool=aa70268e-4b52-42bf-a116-608b370f9501 |
| defi_rate USDY@ondo-yield-assets | 3.55 | pct_annualized | 2026-08-15 23:32 | yields.llama.fi/pools pool=ac61ee82-2fe4-4f9b-a9cd-7fb33f598859 |
| deposit_rate 一年@boc | 0.95 | pct_annualized | 2026-08-15 23:50 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三个月@boc | 0.65 | pct_annualized | 2026-08-15 23:50 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三年@boc | 1.25 | pct_annualized | 2026-08-15 23:50 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 二年@boc | 1.05 | pct_annualized | 2026-08-15 23:50 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 五年@boc | 1.3 | pct_annualized | 2026-08-15 23:50 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 半年@boc | 0.85 | pct_annualized | 2026-08-15 23:50 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 活期@boc | 0.05 | pct_annualized | 2026-08-15 23:50 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| funding BTC@binance | 6.052 | pct_annualized | 2026-08-15 23:50 | fapi/v1/premiumIndex rate=0.00005527 per8h |
| funding BTC@okx | 7.407 | pct_annualized | 2026-08-15 23:50 | api/v5/public/funding-rate rate=0.0000676396333111 per8h |
| funding ETH@binance | 6.665 | pct_annualized | 2026-08-15 23:50 | fapi/v1/premiumIndex rate=0.00006087 per8h |
| funding ETH@okx | 4.407 | pct_annualized | 2026-08-15 23:50 | api/v5/public/funding-rate rate=0.0000402429399674 per8h |
| funding TRX@binance | 1.156 | pct_annualized | 2026-08-15 23:50 | fapi/v1/premiumIndex rate=0.00001056 per8h |
| funding TRX@okx | -4.357 | pct_annualized | 2026-08-15 23:50 | api/v5/public/funding-rate rate=-0.0000397927920973 per8h |
| fx USDCNH@sina | 6.744 | price | 2026-08-15 04:59 | hq.sinajs.cn/list=fx_susdcnh |
| iv BTC@deribit | 34.78 | pct | 2026-08-15 08:00 | api/v2/public/get_volatility_index_data DVOL |
| iv ETH@deribit | 47.07 | pct | 2026-08-15 08:00 | api/v2/public/get_volatility_index_data DVOL |
| reverse_repo GC001@sina | 0.865 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sh204001 |
| reverse_repo R-001@sina | 0.84 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sz131810 |
| ticker BTC@binance | 6.308e+04 | price | 2026-08-15 23:49 | fapi/v1/ticker/price |
| ticker BTC@okx | 6.308e+04 | price | 2026-08-15 23:50 | api/v5/market/tickers |
| ticker ETH@binance | 1885 | price | 2026-08-15 23:50 | fapi/v1/ticker/price |
| ticker ETH@okx | 1885 | price | 2026-08-15 23:50 | api/v5/market/tickers |
| ticker TRX@binance | 0.3311 | price | 2026-08-15 23:49 | fapi/v1/ticker/price |
| ticker TRX@okx | 0.3311 | price | 2026-08-15 23:50 | api/v5/market/tickers |

### 快照 2026-08-15 23:49（已过期 · 被 2026-08-15 23:50 快照取代）

| 事实 | 值 | 单位 | 采集时刻 | 来源 |
|------|-----|------|---------|------|
| calendar month_end@rule | 16 | days | 2026-08-15 23:49 | rule |
| calendar quarter_end@rule | 46 | days | 2026-08-15 23:49 | rule |
| calendar thursday@rule | 5 | days | 2026-08-15 23:49 | rule |
| defi_rate BUIDL@blackrock-buidl | 3.567 | pct_annualized | 2026-08-15 23:32 | yields.llama.fi/pools pool=b663ca59-c7e6-4435-ae4a-28d339ce6a15 |
| defi_rate STEAKUSDC@morpho-blue | 4.163 | pct_annualized | 2026-08-15 23:32 | yields.llama.fi/pools pool=931ea9be-5f4d-428e-beaf-205fc5b4e2b5 |
| defi_rate SUSDE@ethena-usde | 4.437 | pct_annualized | 2026-08-15 23:32 | yields.llama.fi/pools pool=66985a81-9c51-46ca-9977-42b4fe7bc6df |
| defi_rate USDC@aave-v3 | 3.266 | pct_annualized | 2026-08-15 23:32 | yields.llama.fi/pools pool=aa70268e-4b52-42bf-a116-608b370f9501 |
| defi_rate USDY@ondo-yield-assets | 3.55 | pct_annualized | 2026-08-15 23:32 | yields.llama.fi/pools pool=ac61ee82-2fe4-4f9b-a9cd-7fb33f598859 |
| deposit_rate 一年@boc | 0.95 | pct_annualized | 2026-08-15 23:49 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三个月@boc | 0.65 | pct_annualized | 2026-08-15 23:49 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三年@boc | 1.25 | pct_annualized | 2026-08-15 23:49 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 二年@boc | 1.05 | pct_annualized | 2026-08-15 23:49 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 五年@boc | 1.3 | pct_annualized | 2026-08-15 23:49 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 半年@boc | 0.85 | pct_annualized | 2026-08-15 23:49 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 活期@boc | 0.05 | pct_annualized | 2026-08-15 23:49 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| funding BTC@binance | 6.087 | pct_annualized | 2026-08-15 23:49 | fapi/v1/premiumIndex rate=0.00005559 per8h |
| funding BTC@okx | 7.452 | pct_annualized | 2026-08-15 23:49 | api/v5/public/funding-rate rate=0.0000680527477784 per8h |
| funding ETH@binance | 6.688 | pct_annualized | 2026-08-15 23:49 | fapi/v1/premiumIndex rate=0.00006108 per8h |
| funding ETH@okx | 4.42 | pct_annualized | 2026-08-15 23:49 | api/v5/public/funding-rate rate=0.0000403681955428 per8h |
| funding TRX@binance | 1.184 | pct_annualized | 2026-08-15 23:49 | fapi/v1/premiumIndex rate=0.00001081 per8h |
| funding TRX@okx | -4.319 | pct_annualized | 2026-08-15 23:49 | api/v5/public/funding-rate rate=-0.0000394427714828 per8h |
| fx USDCNH@sina | 6.744 | price | 2026-08-15 04:59 | hq.sinajs.cn/list=fx_susdcnh |
| iv BTC@deribit | 35 | pct | 2026-08-15 08:00 | api/v2/public/get_volatility_index_data DVOL |
| iv ETH@deribit | 47.02 | pct | 2026-08-15 08:00 | api/v2/public/get_volatility_index_data DVOL |
| reverse_repo GC001@sina | 0.865 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sh204001 |
| reverse_repo R-001@sina | 0.84 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sz131810 |
| ticker BTC@binance | 6.308e+04 | price | 2026-08-15 23:48 | fapi/v1/ticker/price |
| ticker BTC@okx | 6.308e+04 | price | 2026-08-15 23:49 | api/v5/market/tickers |
| ticker ETH@binance | 1885 | price | 2026-08-15 23:48 | fapi/v1/ticker/price |
| ticker ETH@okx | 1884 | price | 2026-08-15 23:49 | api/v5/market/tickers |
| ticker TRX@binance | 0.3311 | price | 2026-08-15 23:48 | fapi/v1/ticker/price |
| ticker TRX@okx | 0.3312 | price | 2026-08-15 23:49 | api/v5/market/tickers |

### 快照 2026-08-15 23:36（已过期 · 被 2026-08-15 23:49 快照取代）

| 事实 | 值 | 单位 | 采集时刻 | 来源 |
|------|-----|------|---------|------|
| calendar month_end@rule | 16 | days | 2026-08-15 23:36 | rule |
| calendar quarter_end@rule | 46 | days | 2026-08-15 23:36 | rule |
| calendar thursday@rule | 5 | days | 2026-08-15 23:36 | rule |
| defi_rate BUIDL@blackrock-buidl | 3.567 | pct_annualized | 2026-08-15 23:32 | yields.llama.fi/pools pool=b663ca59-c7e6-4435-ae4a-28d339ce6a15 |
| defi_rate STEAKUSDC@morpho-blue | 4.163 | pct_annualized | 2026-08-15 23:32 | yields.llama.fi/pools pool=931ea9be-5f4d-428e-beaf-205fc5b4e2b5 |
| defi_rate SUSDE@ethena-usde | 4.437 | pct_annualized | 2026-08-15 23:32 | yields.llama.fi/pools pool=66985a81-9c51-46ca-9977-42b4fe7bc6df |
| defi_rate USDC@aave-v3 | 3.266 | pct_annualized | 2026-08-15 23:32 | yields.llama.fi/pools pool=aa70268e-4b52-42bf-a116-608b370f9501 |
| defi_rate USDY@ondo-yield-assets | 3.55 | pct_annualized | 2026-08-15 23:32 | yields.llama.fi/pools pool=ac61ee82-2fe4-4f9b-a9cd-7fb33f598859 |
| deposit_rate 一年@boc | 0.95 | pct_annualized | 2026-08-15 23:36 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三个月@boc | 0.65 | pct_annualized | 2026-08-15 23:36 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三年@boc | 1.25 | pct_annualized | 2026-08-15 23:36 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 二年@boc | 1.05 | pct_annualized | 2026-08-15 23:36 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 五年@boc | 1.3 | pct_annualized | 2026-08-15 23:36 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 半年@boc | 0.85 | pct_annualized | 2026-08-15 23:36 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 活期@boc | 0.05 | pct_annualized | 2026-08-15 23:36 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| funding BTC@binance | 6.151 | pct_annualized | 2026-08-15 23:36 | fapi/v1/premiumIndex rate=0.00005617 per8h |
| funding BTC@okx | 7.821 | pct_annualized | 2026-08-15 23:36 | api/v5/public/funding-rate rate=0.0000714266637479 per8h |
| funding ETH@binance | 6.598 | pct_annualized | 2026-08-15 23:36 | fapi/v1/premiumIndex rate=0.00006026 per8h |
| funding ETH@okx | 4.619 | pct_annualized | 2026-08-15 23:36 | api/v5/public/funding-rate rate=0.0000421825865202 per8h |
| funding TRX@binance | 0.945 | pct_annualized | 2026-08-15 23:36 | fapi/v1/premiumIndex rate=0.00000863 per8h |
| funding TRX@okx | -4.552 | pct_annualized | 2026-08-15 23:36 | api/v5/public/funding-rate rate=-0.0000415679435944 per8h |
| fx USDCNH@sina | 6.744 | price | 2026-08-15 04:59 | hq.sinajs.cn/list=fx_susdcnh |
| iv BTC@deribit | 35.02 | pct | 2026-08-15 08:00 | api/v2/public/get_volatility_index_data DVOL |
| iv ETH@deribit | 47 | pct | 2026-08-15 08:00 | api/v2/public/get_volatility_index_data DVOL |
| reverse_repo GC001@sina | 0.865 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sh204001 |
| reverse_repo R-001@sina | 0.84 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sz131810 |
| ticker BTC@binance | 6.309e+04 | price | 2026-08-15 23:36 | fapi/v1/ticker/price |
| ticker BTC@okx | 6.31e+04 | price | 2026-08-15 23:36 | api/v5/market/tickers |
| ticker ETH@binance | 1884 | price | 2026-08-15 23:36 | fapi/v1/ticker/price |
| ticker ETH@okx | 1884 | price | 2026-08-15 23:36 | api/v5/market/tickers |
| ticker TRX@binance | 0.3313 | price | 2026-08-15 23:36 | fapi/v1/ticker/price |
| ticker TRX@okx | 0.3313 | price | 2026-08-15 23:36 | api/v5/market/tickers |

### 快照 2026-08-15 22:45（已过期 · 被 2026-08-15 23:36 快照取代）

| 事实 | 值 | 单位 | 采集时刻 | 来源 |
|------|-----|------|---------|------|
| calendar month_end@rule | 16 | days | 2026-08-15 22:45 | rule |
| calendar quarter_end@rule | 46 | days | 2026-08-15 22:45 | rule |
| calendar thursday@rule | 5 | days | 2026-08-15 22:45 | rule |
| defi_rate BUIDL@blackrock-buidl | 3.567 | pct_annualized | 2026-08-15 22:38 | yields.llama.fi/pools pool=b663ca59-c7e6-4435-ae4a-28d339ce6a15 |
| defi_rate STEAKUSDC@morpho-blue | 4.163 | pct_annualized | 2026-08-15 22:38 | yields.llama.fi/pools pool=931ea9be-5f4d-428e-beaf-205fc5b4e2b5 |
| defi_rate SUSDE@ethena-usde | 4.34 | pct_annualized | 2026-08-15 22:38 | yields.llama.fi/pools pool=66985a81-9c51-46ca-9977-42b4fe7bc6df |
| defi_rate USDC@aave-v3 | 3.264 | pct_annualized | 2026-08-15 22:38 | yields.llama.fi/pools pool=aa70268e-4b52-42bf-a116-608b370f9501 |
| defi_rate USDY@ondo-yield-assets | 3.55 | pct_annualized | 2026-08-15 22:38 | yields.llama.fi/pools pool=ac61ee82-2fe4-4f9b-a9cd-7fb33f598859 |
| deposit_rate 一年@boc | 0.95 | pct_annualized | 2026-08-15 22:45 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三个月@boc | 0.65 | pct_annualized | 2026-08-15 22:45 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三年@boc | 1.25 | pct_annualized | 2026-08-15 22:45 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 二年@boc | 1.05 | pct_annualized | 2026-08-15 22:45 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 五年@boc | 1.3 | pct_annualized | 2026-08-15 22:45 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 半年@boc | 0.85 | pct_annualized | 2026-08-15 22:45 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 活期@boc | 0.05 | pct_annualized | 2026-08-15 22:45 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| funding BTC@binance | 6.464 | pct_annualized | 2026-08-15 22:45 | fapi/v1/premiumIndex rate=0.00005903 per8h |
| funding BTC@okx | 7.931 | pct_annualized | 2026-08-15 22:45 | api/v5/public/funding-rate rate=0.0000724334240291 per8h |
| funding ETH@binance | 6.086 | pct_annualized | 2026-08-15 22:45 | fapi/v1/premiumIndex rate=0.00005558 per8h |
| funding ETH@okx | 4.961 | pct_annualized | 2026-08-15 22:45 | api/v5/public/funding-rate rate=0.0000453054535389 per8h |
| funding TRX@binance | -1.801 | pct_annualized | 2026-08-15 22:45 | fapi/v1/premiumIndex rate=-0.00001645 per8h |
| funding TRX@okx | -6.8 | pct_annualized | 2026-08-15 22:45 | api/v5/public/funding-rate rate=-0.0000620993878983 per8h |
| fx USDCNH@sina | 6.744 | price | 2026-08-15 04:59 | hq.sinajs.cn/list=fx_susdcnh |
| iv BTC@deribit | 34.79 | pct | 2026-08-15 08:00 | api/v2/public/get_volatility_index_data DVOL |
| iv ETH@deribit | 47.11 | pct | 2026-08-15 08:00 | api/v2/public/get_volatility_index_data DVOL |
| reverse_repo GC001@sina | 0.865 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sh204001 |
| reverse_repo R-001@sina | 0.84 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sz131810 |
| ticker BTC@binance | 6.301e+04 | price | 2026-08-15 22:45 | fapi/v1/ticker/price |
| ticker BTC@okx | 6.301e+04 | price | 2026-08-15 22:45 | api/v5/market/tickers |
| ticker ETH@binance | 1882 | price | 2026-08-15 22:45 | fapi/v1/ticker/price |
| ticker ETH@okx | 1882 | price | 2026-08-15 22:45 | api/v5/market/tickers |
| ticker TRX@binance | 0.331 | price | 2026-08-15 22:45 | fapi/v1/ticker/price |
| ticker TRX@okx | 0.3309 | price | 2026-08-15 22:45 | api/v5/market/tickers |

### 快照 2026-08-15 22:38（已过期 · 被 2026-08-15 22:45 快照取代）

| 事实 | 值 | 单位 | 采集时刻 | 来源 |
|------|-----|------|---------|------|
| calendar month_end@rule | 16 | days | 2026-08-15 22:38 | rule |
| calendar quarter_end@rule | 46 | days | 2026-08-15 22:38 | rule |
| calendar thursday@rule | 5 | days | 2026-08-15 22:38 | rule |
| defi_rate BUIDL@blackrock-buidl | 3.567 | pct_annualized | 2026-08-15 22:08 | yields.llama.fi/pools pool=b663ca59-c7e6-4435-ae4a-28d339ce6a15 |
| defi_rate STEAKUSDC@morpho-blue | 4.163 | pct_annualized | 2026-08-15 22:08 | yields.llama.fi/pools pool=931ea9be-5f4d-428e-beaf-205fc5b4e2b5 |
| defi_rate SUSDE@ethena-usde | 4.437 | pct_annualized | 2026-08-15 22:08 | yields.llama.fi/pools pool=66985a81-9c51-46ca-9977-42b4fe7bc6df |
| defi_rate USDC@aave-v3 | 3.266 | pct_annualized | 2026-08-15 22:08 | yields.llama.fi/pools pool=aa70268e-4b52-42bf-a116-608b370f9501 |
| defi_rate USDY@ondo-yield-assets | 3.55 | pct_annualized | 2026-08-15 22:08 | yields.llama.fi/pools pool=ac61ee82-2fe4-4f9b-a9cd-7fb33f598859 |
| deposit_rate 一年@boc | 0.95 | pct_annualized | 2026-08-15 22:38 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三个月@boc | 0.65 | pct_annualized | 2026-08-15 22:38 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三年@boc | 1.25 | pct_annualized | 2026-08-15 22:38 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 二年@boc | 1.05 | pct_annualized | 2026-08-15 22:38 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 五年@boc | 1.3 | pct_annualized | 2026-08-15 22:38 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 半年@boc | 0.85 | pct_annualized | 2026-08-15 22:38 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 活期@boc | 0.05 | pct_annualized | 2026-08-15 22:38 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| funding BTC@binance | 6.557 | pct_annualized | 2026-08-15 22:38 | fapi/v1/premiumIndex rate=0.00005988 per8h |
| funding BTC@okx | 8.142 | pct_annualized | 2026-08-15 22:17 | api/v5/public/funding-rate rate=0.0000743532727699 per8h |
| funding ETH@binance | 5.758 | pct_annualized | 2026-08-15 22:38 | fapi/v1/premiumIndex rate=0.00005258 per8h |
| funding ETH@okx | 5.208 | pct_annualized | 2026-08-15 22:17 | api/v5/public/funding-rate rate=0.0000475598915720 per8h |
| funding TRX@binance | -2.178 | pct_annualized | 2026-08-15 22:38 | fapi/v1/premiumIndex rate=-0.00001989 per8h |
| funding TRX@okx | -8.097 | pct_annualized | 2026-08-15 22:17 | api/v5/public/funding-rate rate=-0.0000739477893184 per8h |
| fx USDCNH@sina | 6.744 | price | 2026-08-15 04:59 | hq.sinajs.cn/list=fx_susdcnh |
| iv BTC@deribit | 34.79 | pct | 2026-08-15 08:00 | api/v2/public/get_volatility_index_data DVOL |
| iv ETH@deribit | 47.13 | pct | 2026-08-15 08:00 | api/v2/public/get_volatility_index_data DVOL |
| reverse_repo GC001@sina | 0.865 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sh204001 |
| reverse_repo R-001@sina | 0.84 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sz131810 |
| ticker BTC@binance | 6.3e+04 | price | 2026-08-15 22:38 | fapi/v1/ticker/price |
| ticker BTC@okx | 6.3e+04 | price | 2026-08-15 22:38 | api/v5/market/tickers |
| ticker ETH@binance | 1881 | price | 2026-08-15 22:38 | fapi/v1/ticker/price |
| ticker ETH@okx | 1881 | price | 2026-08-15 22:38 | api/v5/market/tickers |
| ticker TRX@binance | 0.3309 | price | 2026-08-15 22:38 | fapi/v1/ticker/price |
| ticker TRX@okx | 0.331 | price | 2026-08-15 22:38 | api/v5/market/tickers |

### 快照 2026-08-15 22:16（已过期 · 被 2026-08-15 22:38 快照取代）

| 事实 | 值 | 单位 | 采集时刻 | 来源 |
|------|-----|------|---------|------|
| calendar month_end@rule | 16 | days | 2026-08-15 21:25 | rule |
| calendar quarter_end@rule | 46 | days | 2026-08-15 21:25 | rule |
| calendar thursday@rule | 5 | days | 2026-08-15 21:25 | rule |
| defi_rate BUIDL@blackrock-buidl | 3.567 | pct_annualized | 2026-08-15 22:08 | yields.llama.fi/pools pool=b663ca59-c7e6-4435-ae4a-28d339ce6a15 |
| defi_rate STEAKUSDC@morpho-blue | 4.163 | pct_annualized | 2026-08-15 22:08 | yields.llama.fi/pools pool=931ea9be-5f4d-428e-beaf-205fc5b4e2b5 |
| defi_rate SUSDE@ethena-usde | 4.437 | pct_annualized | 2026-08-15 22:08 | yields.llama.fi/pools pool=66985a81-9c51-46ca-9977-42b4fe7bc6df |
| defi_rate USDC@aave-v3 | 3.266 | pct_annualized | 2026-08-15 22:08 | yields.llama.fi/pools pool=aa70268e-4b52-42bf-a116-608b370f9501 |
| defi_rate USDY@ondo-yield-assets | 3.55 | pct_annualized | 2026-08-15 22:08 | yields.llama.fi/pools pool=ac61ee82-2fe4-4f9b-a9cd-7fb33f598859 |
| deposit_rate 一年@boc | 0.95 | pct_annualized | 2026-08-15 21:25 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三个月@boc | 0.65 | pct_annualized | 2026-08-15 21:25 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三年@boc | 1.25 | pct_annualized | 2026-08-15 21:25 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 二年@boc | 1.05 | pct_annualized | 2026-08-15 21:25 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 五年@boc | 1.3 | pct_annualized | 2026-08-15 21:25 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 半年@boc | 0.85 | pct_annualized | 2026-08-15 21:25 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 活期@boc | 0.05 | pct_annualized | 2026-08-15 21:25 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| funding BTC@binance | 7.018 | pct_annualized | 2026-08-15 22:15 | fapi/v1/premiumIndex rate=0.00006409 per8h |
| funding BTC@okx | 8.284 | pct_annualized | 2026-08-15 22:12 | api/v5/public/funding-rate rate=0.0000756563529595 per8h |
| funding ETH@binance | 5.521 | pct_annualized | 2026-08-15 22:15 | fapi/v1/premiumIndex rate=0.00005042 per8h |
| funding ETH@okx | 5.223 | pct_annualized | 2026-08-15 22:12 | api/v5/public/funding-rate rate=0.0000476982492466 per8h |
| funding TRX@binance | -2.191 | pct_annualized | 2026-08-15 22:15 | fapi/v1/premiumIndex rate=-0.00002001 per8h |
| funding TRX@okx | -8.162 | pct_annualized | 2026-08-15 22:12 | api/v5/public/funding-rate rate=-0.0000745367542677 per8h |
| fx USDCNH@sina | 6.744 | price | 2026-08-15 04:59 | hq.sinajs.cn/list=fx_susdcnh |
| iv BTC@deribit | 34.84 | pct | 2026-08-15 08:00 | api/v2/public/get_volatility_index_data DVOL |
| iv ETH@deribit | 47.07 | pct | 2026-08-15 08:00 | api/v2/public/get_volatility_index_data DVOL |
| reverse_repo GC001@sina | 0.865 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sh204001 |
| reverse_repo R-001@sina | 0.84 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sz131810 |
| ticker BTC@binance | 6.302e+04 | price | 2026-08-15 22:16 | fapi/v1/ticker/price |
| ticker BTC@okx | 6.301e+04 | price | 2026-08-15 22:13 | api/v5/market/tickers |
| ticker ETH@binance | 1882 | price | 2026-08-15 22:15 | fapi/v1/ticker/price |
| ticker ETH@okx | 1882 | price | 2026-08-15 22:13 | api/v5/market/tickers |
| ticker TRX@binance | 0.3309 | price | 2026-08-15 22:16 | fapi/v1/ticker/price |
| ticker TRX@okx | 0.331 | price | 2026-08-15 22:13 | api/v5/market/tickers |

### 快照 2026-08-15 21:25（已过期 · 被 2026-08-15 22:16 快照取代）

| 事实 | 值 | 单位 | 采集时刻 | 来源 |
|------|-----|------|---------|------|
| calendar month_end@rule | 16 | days | 2026-08-15 21:25 | rule |
| calendar quarter_end@rule | 46 | days | 2026-08-15 21:25 | rule |
| calendar thursday@rule | 5 | days | 2026-08-15 21:25 | rule |
| defi_rate BUIDL@blackrock-buidl | 3.567 | pct_annualized | 2026-08-15 21:17 | yields.llama.fi/pools pool=b663ca59-c7e6-4435-ae4a-28d339ce6a15 |
| defi_rate STEAKUSDC@morpho-blue | 4.163 | pct_annualized | 2026-08-15 21:17 | yields.llama.fi/pools pool=931ea9be-5f4d-428e-beaf-205fc5b4e2b5 |
| defi_rate SUSDE@ethena-usde | 4.321 | pct_annualized | 2026-08-15 21:17 | yields.llama.fi/pools pool=66985a81-9c51-46ca-9977-42b4fe7bc6df |
| defi_rate USDC@aave-v3 | 3.281 | pct_annualized | 2026-08-15 21:17 | yields.llama.fi/pools pool=aa70268e-4b52-42bf-a116-608b370f9501 |
| defi_rate USDY@ondo-yield-assets | 3.55 | pct_annualized | 2026-08-15 21:17 | yields.llama.fi/pools pool=ac61ee82-2fe4-4f9b-a9cd-7fb33f598859 |
| deposit_rate 一年@boc | 0.95 | pct_annualized | 2026-08-15 20:47 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三个月@boc | 0.65 | pct_annualized | 2026-08-15 20:47 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三年@boc | 1.25 | pct_annualized | 2026-08-15 20:47 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 二年@boc | 1.05 | pct_annualized | 2026-08-15 20:47 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 五年@boc | 1.3 | pct_annualized | 2026-08-15 20:47 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 半年@boc | 0.85 | pct_annualized | 2026-08-15 20:47 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 活期@boc | 0.05 | pct_annualized | 2026-08-15 20:47 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| funding BTC@binance | 7.563 | pct_annualized | 2026-08-15 21:21 | fapi/v1/premiumIndex rate=0.00006907 per8h |
| funding BTC@okx | 8.504 | pct_annualized | 2026-08-15 21:24 | api/v5/public/funding-rate rate=0.0000776591264970 per8h |
| funding ETH@binance | 5.593 | pct_annualized | 2026-08-15 21:21 | fapi/v1/premiumIndex rate=0.00005108 per8h |
| funding ETH@okx | 5.728 | pct_annualized | 2026-08-15 21:24 | api/v5/public/funding-rate rate=0.0000523111366741 per8h |
| funding TRX@binance | -4.841 | pct_annualized | 2026-08-15 21:21 | fapi/v1/premiumIndex rate=-0.00004421 per8h |
| funding TRX@okx | -8.599 | pct_annualized | 2026-08-15 21:24 | api/v5/public/funding-rate rate=-0.0000785320873625 per8h |
| fx USDCNH@sina | 6.744 | price | 2026-08-15 04:59 | hq.sinajs.cn/list=fx_susdcnh |
| iv BTC@deribit | 34.97 | pct | 2026-08-15 08:00 | api/v2/public/get_volatility_index_data DVOL |
| iv ETH@deribit | 47.11 | pct | 2026-08-15 08:00 | api/v2/public/get_volatility_index_data DVOL |
| reverse_repo GC001@sina | 0.865 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sh204001 |
| reverse_repo R-001@sina | 0.84 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sz131810 |
| ticker BTC@binance | 6.301e+04 | price | 2026-08-15 21:24 | fapi/v1/ticker/price |
| ticker BTC@okx | 6.302e+04 | price | 2026-08-15 21:24 | api/v5/market/tickers |
| ticker ETH@binance | 1882 | price | 2026-08-15 21:24 | fapi/v1/ticker/price |
| ticker ETH@okx | 1882 | price | 2026-08-15 21:24 | api/v5/market/tickers |
| ticker TRX@binance | 0.3311 | price | 2026-08-15 21:24 | fapi/v1/ticker/price |
| ticker TRX@okx | 0.3311 | price | 2026-08-15 21:24 | api/v5/market/tickers |

### 快照 2026-08-15 20:47（已过期 · 被 2026-08-15 21:25 快照取代）

| 事实 | 值 | 单位 | 采集时刻 | 来源 |
|------|-----|------|---------|------|
| calendar month_end@rule | 16 | days | 2026-08-15 20:47 | rule |
| calendar quarter_end@rule | 46 | days | 2026-08-15 20:47 | rule |
| calendar thursday@rule | 5 | days | 2026-08-15 20:47 | rule |
| defi_rate BUIDL@blackrock-buidl | 3.567 | pct_annualized | 2026-08-15 20:44 | yields.llama.fi/pools pool=b663ca59-c7e6-4435-ae4a-28d339ce6a15 |
| defi_rate STEAKUSDC@morpho-blue | 4.163 | pct_annualized | 2026-08-15 20:44 | yields.llama.fi/pools pool=931ea9be-5f4d-428e-beaf-205fc5b4e2b5 |
| defi_rate SUSDE@ethena-usde | 4.321 | pct_annualized | 2026-08-15 20:44 | yields.llama.fi/pools pool=66985a81-9c51-46ca-9977-42b4fe7bc6df |
| defi_rate USDC@aave-v3 | 3.281 | pct_annualized | 2026-08-15 20:44 | yields.llama.fi/pools pool=aa70268e-4b52-42bf-a116-608b370f9501 |
| defi_rate USDY@ondo-yield-assets | 3.55 | pct_annualized | 2026-08-15 20:44 | yields.llama.fi/pools pool=ac61ee82-2fe4-4f9b-a9cd-7fb33f598859 |
| deposit_rate 一年@boc | 0.95 | pct_annualized | 2026-08-15 20:44 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三个月@boc | 0.65 | pct_annualized | 2026-08-15 20:44 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三年@boc | 1.25 | pct_annualized | 2026-08-15 20:44 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 二年@boc | 1.05 | pct_annualized | 2026-08-15 20:44 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 五年@boc | 1.3 | pct_annualized | 2026-08-15 20:44 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 半年@boc | 0.85 | pct_annualized | 2026-08-15 20:44 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 活期@boc | 0.05 | pct_annualized | 2026-08-15 20:44 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| funding BTC@binance | 7.55 | pct_annualized | 2026-08-15 20:44 | fapi/v1/premiumIndex rate=0.00006895 per8h |
| funding BTC@okx | 8.062 | pct_annualized | 2026-08-15 20:44 | api/v5/public/funding-rate rate=0.0000736233133554 per8h |
| funding ETH@binance | 5.212 | pct_annualized | 2026-08-15 20:44 | fapi/v1/premiumIndex rate=0.00004760 per8h |
| funding ETH@okx | 5.616 | pct_annualized | 2026-08-15 20:44 | api/v5/public/funding-rate rate=0.0000512872528642 per8h |
| funding TRX@binance | -5.895 | pct_annualized | 2026-08-15 20:44 | fapi/v1/premiumIndex rate=-0.00005384 per8h |
| funding TRX@okx | -7.1 | pct_annualized | 2026-08-15 20:44 | api/v5/public/funding-rate rate=-0.0000648392022295 per8h |
| fx USDCNH@sina | 6.744 | price | 2026-08-15 04:59 | hq.sinajs.cn/list=fx_susdcnh |
| iv BTC@deribit | 34.93 | pct | 2026-08-15 08:00 | api/v2/public/get_volatility_index_data DVOL |
| iv ETH@deribit | 47.13 | pct | 2026-08-15 08:00 | api/v2/public/get_volatility_index_data DVOL |
| reverse_repo GC001@sina | 0.865 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sh204001 |
| reverse_repo R-001@sina | 0.84 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sz131810 |
| ticker BTC@binance | 6.292e+04 | price | 2026-08-15 20:46 | fapi/v1/ticker/price |
| ticker BTC@okx | 6.292e+04 | price | 2026-08-15 20:46 | api/v5/market/tickers |
| ticker ETH@binance | 1877 | price | 2026-08-15 20:46 | fapi/v1/ticker/price |
| ticker ETH@okx | 1877 | price | 2026-08-15 20:46 | api/v5/market/tickers |
| ticker TRX@binance | 0.3309 | price | 2026-08-15 20:46 | fapi/v1/ticker/price |
| ticker TRX@okx | 0.3308 | price | 2026-08-15 20:46 | api/v5/market/tickers |

## 监控快照（arbcn 自动导出 · M2-b §5 / D-028 闭环）

> 机器生成：监控最新值渲染进事实库；新快照到来 → 旧快照标「已过期」不删除。
> 机器可读投影：DashboardService.ListFacts（web 前端事实快照视图）。

### 快照 2026-08-15 20:44（已过期 · 被 2026-08-15 20:47 快照取代）

| 事实 | 值 | 单位 | 采集时刻 | 来源 |
|------|-----|------|---------|------|
| calendar month_end@rule | 16 | days | 2026-08-15 20:44 | rule |
| calendar quarter_end@rule | 46 | days | 2026-08-15 20:44 | rule |
| calendar thursday@rule | 5 | days | 2026-08-15 20:44 | rule |
| defi_rate BUIDL@blackrock-buidl | 3.567 | pct_annualized | 2026-08-15 20:37 | yields.llama.fi/pools pool=b663ca59-c7e6-4435-ae4a-28d339ce6a15 |
| defi_rate STEAKUSDC@morpho-blue | 4.163 | pct_annualized | 2026-08-15 20:37 | yields.llama.fi/pools pool=931ea9be-5f4d-428e-beaf-205fc5b4e2b5 |
| defi_rate SUSDE@ethena-usde | 4.321 | pct_annualized | 2026-08-15 20:37 | yields.llama.fi/pools pool=66985a81-9c51-46ca-9977-42b4fe7bc6df |
| defi_rate USDC@aave-v3 | 3.281 | pct_annualized | 2026-08-15 20:37 | yields.llama.fi/pools pool=aa70268e-4b52-42bf-a116-608b370f9501 |
| defi_rate USDY@ondo-yield-assets | 3.55 | pct_annualized | 2026-08-15 20:37 | yields.llama.fi/pools pool=ac61ee82-2fe4-4f9b-a9cd-7fb33f598859 |
| deposit_rate 一年@boc | 0.95 | pct_annualized | 2026-08-15 20:07 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三个月@boc | 0.65 | pct_annualized | 2026-08-15 20:07 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 三年@boc | 1.25 | pct_annualized | 2026-08-15 20:07 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 二年@boc | 1.05 | pct_annualized | 2026-08-15 20:07 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 五年@boc | 1.3 | pct_annualized | 2026-08-15 20:07 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 半年@boc | 0.85 | pct_annualized | 2026-08-15 20:07 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| deposit_rate 活期@boc | 0.05 | pct_annualized | 2026-08-15 20:07 | boc/fimarkets/lilv/fd31 表2025-05-20 |
| funding BTC@binance | 6.905 | pct_annualized | 2026-08-15 19:59 | fapi/v1/premiumIndex rate=0.00006306 per8h |
| funding BTC@okx | 8.018 | pct_annualized | 2026-08-15 20:42 | api/v5/public/funding-rate rate=0.0000732240515131 per8h |
| funding ETH@binance | 4.665 | pct_annualized | 2026-08-15 19:59 | fapi/v1/premiumIndex rate=0.00004260 per8h |
| funding ETH@okx | 5.561 | pct_annualized | 2026-08-15 20:42 | api/v5/public/funding-rate rate=0.0000507823537719 per8h |
| funding TRX@binance | -4.888 | pct_annualized | 2026-08-15 19:59 | fapi/v1/premiumIndex rate=-0.00004464 per8h |
| funding TRX@okx | -6.969 | pct_annualized | 2026-08-15 20:42 | api/v5/public/funding-rate rate=-0.0000636474585435 per8h |
| fx USDCNH@sina | 6.744 | price | 2026-08-15 04:59 | hq.sinajs.cn/list=fx_susdcnh |
| iv BTC@deribit | 34.93 | pct | 2026-08-15 08:00 | api/v2/public/get_volatility_index_data DVOL |
| iv ETH@deribit | 47.07 | pct | 2026-08-15 08:00 | api/v2/public/get_volatility_index_data DVOL |
| reverse_repo GC001@sina | 0.865 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sh204001 |
| reverse_repo R-001@sina | 0.84 | pct_annualized | 2026-08-14 15:30 | hq.sinajs.cn/list=sz131810 |
| ticker BTC@binance | 6.3e+04 | price | 2026-08-15 19:59 | fapi/v1/ticker/price |
| ticker BTC@okx | 6.293e+04 | price | 2026-08-15 20:43 | api/v5/market/tickers |
| ticker ETH@binance | 1879 | price | 2026-08-15 19:59 | fapi/v1/ticker/price |
| ticker ETH@okx | 1878 | price | 2026-08-15 20:43 | api/v5/market/tickers |
| ticker TRX@binance | 0.3309 | price | 2026-08-15 19:59 | fapi/v1/ticker/price |
| ticker TRX@okx | 0.3309 | price | 2026-08-15 20:43 | api/v5/market/tickers |
<!-- ARBCN-EXPORT-END -->
