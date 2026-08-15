# STATE — 当前工作状态 + 交接负载

> 开工读 / 收工写。滑动窗口（AGENTS.md §4）：只存进行中 + 近期完成 + 交接负载；最旧 ✅ 移 LOG.md。

## 交接负载

- 现状: **i18n 清理完成**——页面硬编码英文全部中文化（format.ts statusText/reasonText/stateText/levelText/ruleLabel + Chip 徽标 + 铃铛规则名 + 告警流/触发器规则名 + 后端规则消息模板 activeMsg/已解除 + 机会面板标题）；新构建已 systemd 部署（SIGKILL 重启），healthz ok，源健康实测 live/stale 判定正确。M2-a 全链路完成；**下一项 = M2-b（RMB 折算 + facts.md 导出 + 台账）**。
- 方向校验: 与 AGENTS.md §1 一致 —— 不赌原则（D-019）+ 收益最大×路径最短（D-020）+ 加密三档（D-021）+ 敞口知情（D-023）+ 无密钥铁律（D-010/§13）。
- 施工表:
  | 子任务 | 状态 | 锚点 |
  |--------|------|------|
  | 脚手架 + 协议 + 门禁 | ✅ | 5bea136 / D-001~D-005 |
  | charter 正式化 ~ 不赌原则 | ✅ | D-008~D-019 |
  | 优先级反转：监控先行 | ✅ | D-020 |
  | 加密三档结构 | ✅ | D-021 |
  | 监控 v1 规格 | ✅ | D-022 |
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
  | **i18n：页面硬编码英文 → 中文（前后端一致 ruleLabel/消息模板）** | ✅ | 本次提交 |
  | M2-b：RMB 折算 + facts.md 导出 + 台账 | ⬜ | M2-a 后 |
- 阻塞/待决策:
  - 无阻塞。TRX 独立处置（业主自定，费率转正触发器已入监控规格）。
  - SMTP 授权码待办**已移除**（D-033：业主不做邮件推送，浏览器铃铛为主通道）。
  - **方向级待决策（D-034 讨论中）**：业主提议「监控 → 自动交易（人工确认）+ 模拟账号验证，尤其加密层」。触无密钥铁律（D-010/§13）与「决策监控不自动执行」形态，需 讨论→决定 后落 D#；未决前不动手。
- 下一步: M2-b 施工（RMB 折算 + facts.md 自动导出 + 台账，03-m2-spec.md §4–§6）。
- 清扫上翻: 今日生产实证入共享层——周六闭市 fx/repo 报价冻结但采集器健康、心跳用轮询时刻故元监控不误报、展示层需 freshness 区分"闭市/源死"（已入 D-033 + 03-m2-spec.md）；systemd 部署决策（mluser 非 arbcn 用户）已同步 unit 模板；仪表盘布局（告警流与机会面板同行，dialogue #32）已交付；M2-a 后端施工心得入 practices.md（time.Time 比较用 .Equal 不用 ==——PG TZ=+0800 读回带时区的坑，兼修 pgstore/dashboard_test.go 既有断言）。
