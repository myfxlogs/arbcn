# 共享实践记忆

> 高频打回模式 + 代码风格约定。施工 agent 写代码前读，避免被打回。
> 超 ~450 行时最旧内容机械移 LOG.md（留索引行）。

## 默认风格（起步）

- 输出纪律：汇报给结论 + 依据，不堆文件转储。
- 先调研后动手：写前读相关代码、查 git 历史、理解现状。
- Cross-scope：一个 task 只改一个语义范围，跨范围拆成多个 task。
- 不因困难妥协最优解：遇到阻碍回根因，禁止快捷方式（回退代替重构 / 沉默代替修复）。
- 精度 / 并发 / 规模等硬约束见 AGENTS.md §7 门禁 + 后续 constraints.md。

## 已知打回模式（持续追加）

1. **优化/审计提案必须先数据核实再落盘**：2026-08-15 自审计中，北交所打新（需冻结 500 万+）与 QDII-LOF 溢价（限购 10–100 元/日 + 退市新规）两项"更优解"提案被当场证伪。凭印象提"更好方案"而不核实当前市场结构 = 打回。

2. **外部 AI/agent 方案必须逐条核实再采纳**：2026-08-15 千问方案 7 项主张中 4 项有错（工行利率 0.8% 实为 2.8%、两得宝 R5/C5 门槛遗漏、降息前提与加息预期矛盾、压力测试年息套 6 个月夸大 2×）。核实后吸收其 1 个真知（华夏全对冲互认基金）并落 D-018。外部方案的定位 = 候选输入，不是决策；产品准入、风险画像、宏观前提、数学三样必须验。

3. **time.Time 比较一律用 .Equal，不用 `==`/`!=`**：2026-08-15 M2-a 修掉 pgstore/dashboard_test.go 既有坑——arbcn-postgres 服务器 TZ=+0800，pgx v5 读回 timestamptz 带本地时区，`alerts[1].Ts != ts1`（ts1 为 UTC）恒真误报。`.Equal` 只比瞬间（时区无关），`==` 还比 location 指针。同理：写断言前先看读回路径是否带时区。

4. **展示层值格式化必须按 unit 感知，禁止对 kind 一刀切 pct()**：2026-08-15 M2-b 复审（负责人 F2）——FactsSnapshot 对所有 kind 统一 pct()，导致 fx（unit=price，6.7443）显示 "674.43%"、calendar（unit=days，16）显示 "16.00%"。事实值语义由 unit 承载（pct_annualized/pct/days/price/ratio），格式化按 unit 分支；新增事实类型时先看 unit 再决定显示。

5. **快照/投影类 RPC 要排除内部遥测事实**：2026-08-15 M2-b 复审（负责人 F1）——ListFacts 快照投影把 heartbeat（内部遥测，非市场事实）一起返回，与 exporter skipKinds 排除不一致，污染"事实快照"视图。exporter（写 facts.md）和 dashboard 投影（机器可读面）必须同口径排除 heartbeat；实现两处时互相引用锚点，别各写各的。

6. **结算/折算类数值先统一刻度（点数 vs 分数）**：2026-08-15 追溯深审（R6#1）+ M3-a 复审（H1）——rmb 折算与 funding 结算同一根因栽两次。事实层 `pct_annualized` = 百分点点数（6.0 = 6%）；凡是要**乘名义（货币单位）**的，必须先 ÷100 转分数费率。H1 具体：`Per8hRate(10.95)` 原返回 0.01 实为 0.0001，`SettleFundingPnl(0.01, 10000)` = 100 实为 1，模拟 PnL 虚高 100 倍；`RMBDayEnd` 同缺 ÷100。**对抗测试锚点必须锚"正确数值"**——H1 的旧锚点锚了错误值 100，正是靠人工复核才发现，锚点自身的正确性要独立验证。

7. **数值门禁必须防 NaN/±Inf 绕过**：2026-08-15 M3-a 复审（M3）——Go 里 `NaN < x` / `NaN > x` 恒 false，NaN 输入会静默穿过全部 `<`/`>` 门禁，且 NaN 落库污染聚合（SUM→NaN → DAILY_OVER 永久失效）。门禁函数开头对每个数值输入 `IsNaN/IsInf` 检查 → 拒单；配对抗测试（NaN 输入断言拒单）。这条对任何"阈值/超限"逻辑通用，不限于模拟盘。

8. **"置状态 + 建从属行"必须原子，否则不可自愈**：2026-08-15 M3-a 复审（M1）——ConfirmAndFill 先置订单 filled 再逐条插腿，第二腿失败留下"filled 但缺腿"半对冲（违反不赌 D-019），且订单已是 filled、重试被状态守卫拒绝 = 死状态。解法：store 层单事务方法（`FillSimOrder`：UPDATE status 带 `WHERE status='confirmed'` 的 RowsAffected 守卫 + INSERT 全腿），业务代码只调一个原子入口。副作用：状态守卫顺带消掉并发双插。

9. **"非白名单/信任边界"的布尔标记不算可验证门禁**：2026-08-15 M3-a 复审（M2）——`Signal.CarryWhite bool` 由调用方置位，纯函数只查布尔 = 信任边界，不是门禁；数据源/驱动误置即形同虚设。此类信任边界要在 spec 明示，且要在外部数据源接入前落显式配置（如 M3-b 白名单 symbol 集合）。

10. **测试注入时钟要覆盖整个 RPC 路径，grep 掉所有 time.Now()**：2026-08-15 追溯深审（R4#7）——sourceHealth 判定函数接受注入 now，但 ListSourceHealth 入口自己 `time.Now()`，测试注入的固定 now 与运行时 time.Now() 漂移毫秒级 → 边界断言 flake。注入 `svc.Now` 后要 grep 该路径所有 `time.Now()`，入口层最容易漏。

11. **统计性结论只能来自有统计效力的数据**：2026-08-15 M3 文档审计（D-036）——spec 把"funding 套利是否真收敛/收敛速度/残差"列为 M3 主要交付物，但验证载体是 testnet 前向周级小样本（样本量不足 + testnet 费率偏差污染），统计问题用了没统计效力的工具；且 spec 同段"❌ 回测引擎/历史数据回放"与"收敛结论是主要交付物"自相矛盾。裁定：**前向模拟只验证机制（结算管线/行为观察），统计结论由历史数据出**（回填 funding/价差历史 + 收敛分析）。验证目标与方法不匹配 = 交付物承诺它给不了的证据；规格里"不做 X"与"交付物依赖 X"必须互检。

12. **数据源端点假设必须部署机实测，不能信文档/记忆**：2026-08-15 M3-b S4 部署验证——D-031 假定 data-api.binance.vision 镜像 fapi fundingRate，实测 404（该域只镜像现货 /api/v3，/fapi/* 全 404）；OKX 记成 funding-history，实测业务 404，正确端点是 funding-rate-history；数据源 451 封锁也是间歇性（M1-c 通/M1-d 451/M3-b 部署机直连通）。**每次换新数据源/新端点：部署机上用 curl 实测真实响应 + 深度**（OKX funding 历史仅 ~90d，写 365d 窗口必须知道是部分覆盖）再写进代码，写死"data 可用 + 深度够"假设 = 给部署埋雷。与 D-028 先核实再采纳同源，区别在它是针对"端点存在性/数据深度"这类运行期才暴露的假设。**同类还含请求协议格式**：2026-08-15 S3 探针把 Binance 的 Unix 毫秒 timestamp 惯例照搬到 OKX——OKX 要求 `OK-ACCESS-TIMESTAMP` 为 **ISO 8601 UTC**（Unix 毫秒 → 50102 "Timestamp request expired"，实测 3 轮才暴露），且 "一种 API 的签名/格式惯例可移植到另一种 API" 是隐蔽假设（probe_test 已加 ISO 格式对抗锚点）。

13. **门禁/折算的数据面必须按实体类型分派，不能硬编码单一实体的数据源**：2026-08-15 M3-c 复审（D-039）——SPREAD_DRIFT 二次门禁数据面按 spec §10.3 硬编码 `LatestFacts(ticker)` + `LatestFacts(funding)` 双查（这是 funding_hedge 语义），导致 repo（无 ticker，面值锚 100）、carry（无 funding，稳定币生息）两类订单**确认恒拒**——门禁不是"从严"，是功能残缺。修法：按 kind 分派**权威数据源**（repo→reverse_repo 利率、carry→defi_rate 年化、funding_hedge→ticker/funding），fail-closed 语义保持（每类权威源查不到仍拒）。模式：任何"阈值/门禁/折算"若服务多类实体，先枚举**每类实体的数据源**再写数据面；"一类实体的数据面天然通用" = 隐蔽的假设坑。与 #11 同源（验证目标与方法匹配），区别在 #11 是统计工具，本条是数据面映射。

14. **跨数据源同名数值字段口径不同，必须按 source/来源标注口径，不假设语义一致**：2026-08-16 D-040（SimExec 测试网账户区）——`equity_usd` 同名，OKX 给 `totalEq`（交易所精确折算），binance `fapi/v2/balance` 不给 USD 值，只能做稳定币（USDT/USDC/BUSD/FDUSD）合计**近似**（非稳定币无行情折算）。把两路都当"账户权益"展示 = 误导（binance 会低估）。落地：store 类型注释标口径 + 前端按 source 明示「精确 / 近似·非全量净值」+ 无折算资产标 —。任何聚合展示只要跨数据源，先问"这两个同名数字的语义/精确度是否相同"再决定是否并排。

15. **SQL 动态 WHERE 拼接，每个子句必须加括号**：2026-08-16 D-042 演练单拒单根因——`where := []string{"$1 = '' OR kind = $1", "$2 = '' OR venue = $2", "$3 = '' OR symbol = $3"}` 用 `AND` join 但各子句缺括号 → 求值 `$1='' OR kind=$1 AND $2='' OR ...`，`AND` 优先级高于 `OR` → 多参数组合退化为「只生效符号条件」：`LatestFacts(ticker, okx, BTC)` 返回 funding+iv+ticker 五行、首行 funding@binance 负值 → `fundingHedgeSignal` 取 `fs[0]` 拿负值当 ticker 价 → UNHEDGED 拒单。修法：每子句自括 `($1='' OR kind=$1)`。教训：**任何 `strings.Join(cond, " AND ")` 的动态 WHERE，条件内一旦含 OR 必须子句自括**；且测试必须覆盖多参数组合（仅空参/单 kind 测不出）。对抗锚点：TestLatestFactsFilters（删括号必红）。

16. **/proc/PID/exe 与磁盘文件对比必须用 `stat -L`（跟随符号链接）**：2026-08-16 D-042 排查曾误判「运行旧二进制」——`stat -c '%i' /proc/PID/exe`（不带 `-L`）返回的是 procfs 为魔法符号链接分配的**伪 inode**，与磁盘文件 inode 必然不同，被误读为"运行进程加载了已替换的旧二进制"，一度把调查引向反汇编/二进制比对错误方向。真相：`/proc/PID/exe` 是符号链接，跟到目标才是文件真身——**对比运行二进制与磁盘须 `stat -L`，或 `readlink /proc/PID/exe` 看是否 `(deleted)`**。教训：procfs 魔法文件（exe/maps/fd）默认 stat 语义特殊，涉及"运行的是什么文件"一律先跟符号链接再下结论。
