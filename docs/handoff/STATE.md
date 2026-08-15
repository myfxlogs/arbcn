# STATE — 当前工作状态 + 交接负载

> 开工读 / 收工写。滑动窗口（AGENTS.md §4）：只存进行中 + 近期完成 + 交接负载；最旧 ✅ 移 LOG.md。

## 交接负载

- 现状: **M1/M2-a 追溯深审 + M3-a 施工复审全部闭环已部署**（D-035）；**M3 文档审计完成**（D-036：收敛口径修正 + G1–G5 全落 spec）；**M3-b 细化设计定稿 + 施工交付 + 复审修复 + S4 数据源修正部署验证**（D-037 + D-031 实证修订 + 本 commit：spec §9 S1–S5 全落地并已部署实测）。S1 规则→Signal 驱动（rule.OnActive 携带命中实体 + sim.Driver §3.1.1 映射表，删映射必红）；S2 8h 结算（(symbol,venue) 分组取真实市场 funding，跨 venue 污染必红）；S3 testnet 只读探针 + key 隔离（新包 internal/simtestnet，SIMULATED 缺标记拒绝加载，sim 包零网络由 domains_test 把关）；S4 历史收敛（exchange 历史 collector + 幂等回填 + sim_report 周频统计）——**数据源已修正并部署验证**（fapi.binance.com 满 365d + OKX funding-rate-history ~90d，见 D-031 修订；DB 实测 binance min_ts=2025-08-15）；S5 carry 白名单默认空（宁缺毋滥）。main.go 已接线：boot 历史回填（一次性幂等）+ simDriver（OnActive compose）+ 8h 结算循环 + testnet 探针随 settle tick；H1 洪水去重（8h 桶折叠 + Limit 2M）已含。**M3-c 细化设计定稿 + 施工交付**（D-038 + spec §10 C1–C5：SimService 独立 proto 域 + SPREAD_DRIFT 二次门禁 + 确认成交流原子化 + 模拟执行 UI tab + 可检查性；本 commit 全落地，见施工表）。**下一步 = M3-c 部署验收或业主提供 testnet key 启用 S3**。
- 方向校验: 与 AGENTS.md §1 一致 —— 不赌原则（D-019）+ 收益最大×路径最短（D-020）+ 加密三档（D-021）+ 敞口知情（D-023）+ 无密钥铁律（D-010/§13，M3-a 复审复核：sim 包零网络零密钥）。
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
  | 出入金通道验证（1 万小额 OTC） | ⬜ | 业主执行 |
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
- 阻塞/待决策:
  - 无阻塞。TRX 独立处置（业主自定，费率转正触发器已入监控规格）。
  - SMTP 授权码待办**已移除**（D-033：业主不做邮件推送，浏览器铃铛为主通道）。
  - R3-L3 遗留：calendar collector 的 thursday 周记语义待业主决策（低危，接受待决）。
  - **M3-a 已验收（D-035）**；**M3-b 已施工交付**（D-037 + 本 commit，S1–S5 全落地，见施工表）。
  - **testnet key 依赖业主提供**（D-037 S3 门控）：缺 key → S3 探针降级禁用（warn 不退出），不阻塞 S1/S2/S4/S5；key 到位后重启服务即启用（读 /etc/arbcn/arbcn-sim.env，SIMULATED=true 显式标记）。
  - **M3-b 交付注**：sim 已接线进 main.go（backfill / simDriver / OnActive compose / 8h 结算循环 / 探针随 tick）；结算数据源 = 真实市场公开 funding（D-037 裁决），testnet 费率不参与结算只做 key 隔离验证。sim 包保持零网络零密钥（domains_test + TestNoNetworkImports 把关）。
  - **M3-c 交付注**：SimService 独立域已接线 main.go（st 非 nil 即挂载，sim 配置缺失降级：GetSimReport 返回未启用说明，D-032 同口径）。**已知设计问题待 Claude 裁决**：repo 订单（venue=domestic, symbol=GC001）无 ticker/funding 事实 → ConfirmSimOrder 二次门禁查不到数据 → **fail-closed 恒拒单**（spec §10.3「查不到 ticker/funding → 拒」的直接推论）。当前行为符合 spec 但 repo 确认流永远走拒单路径——是否允许 repo 类订单跳过二次门禁（无市场行情可查）需决策（D# 或 dialogue 裁决）。
  - **D-031 实证修订**：data-api.binance.vision 不镜像 /fapi/*（404）；历史回填源回落 fapi.binance.com（部署机直连 200、满 365d）；OKX 历史端点为 funding-rate-history（funding-history 404），仅保留 ~90d（OKX 部分覆盖，sim_report/avg_30d 受窗口限制，已知 degrade）。
  - 待决策观察（不阻塞）：repo 信号经 SignalToOrder 时仍受 5% 门槛（SPREAD_LOW）约束——平时逆回购 2-4% 会被拒单，仅季末/年末上冲 ≥5% 时放行；与"时点逆回购"策略意图一致（宁缺毋滥），但若业主希望 repo 绕过价差门槛须走 D# 调整。
- 下一步: **M3-c 已施工交付**（04-m3-spec.md §10 C1–C5 全落地：proto/门禁/确认流/UI/可检查性；go vet + test -race 全绿）。待：①部署验收（重启 healthz ok → tab 打开 → 确认流冒烟，见 spec §10.6 部署验收清单）；②**repo 订单二次门禁设计裁决**（见交付注）；③业主 testnet key 启用 S3。
- 清扫上翻: 本次 review 教训入 practices.md #6-#10（刻度统一/NaN 门禁/状态+从属行原子/信任边界标注/时钟注入覆盖全路径）+ #11（统计效力）+ #12（数据源端点必须部署机实测）；追溯深审 + M3-a 复审结论入 decisions.md D-035；M3 文档审计结论入 D-036（收敛口径修正 + G1–G5）；M3-b 细化设计入 D-037（spec §9 + 结算数据源裁决）；S4 数据源实证修订入 D-031（data-api 前提否定 + fapi 回落 + OKX 端点修正）；M3-c 细化设计入 D-038（spec §10 C1–C5）；M3-c 施工（含 repo 二次门禁设计问题）待 dialogue 落档；对话落 dialogue.md #39/#40/#41/#42/#43/#44/#45。
