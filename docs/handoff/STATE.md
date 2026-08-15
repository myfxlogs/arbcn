# STATE — 当前工作状态 + 交接负载

> 开工读 / 收工写。滑动窗口（AGENTS.md §4）：只存进行中 + 近期完成 + 交接负载；最旧 ✅ 移 LOG.md。

## 交接负载

- 现状: **M1/M2-a 追溯深审 + M3-a 施工复审全部闭环已部署**（D-035）。追溯深审（R1-R6 六路 review + 决策层逐条验证）修复 14 项（高危 5/中危 6/低危 3），M3-a 复审修复 H1 结算 PnL 100 倍放大（点数÷100）+ M1 成交原子化（FillSimOrder 单事务）+ M3 NaN 门禁加固 + L1/L2/L3；M3-a 达验收线，SIGKILL 部署验证通过（PID 2328862，healthz ok，migration 0005 applied）。**下一步 = M3-b 细化设计（testnet 只读收敛，D-034 条款）**。
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
  | M3：模拟执行验证（b testnet 只读收敛 → c 一键确认 UI） | ⬜ | D-034 · 下一项 |
- 阻塞/待决策:
  - 无阻塞。TRX 独立处置（业主自定，费率转正触发器已入监控规格）。
  - SMTP 授权码待办**已移除**（D-033：业主不做邮件推送，浏览器铃铛为主通道）。
  - R3-L3 遗留：calendar collector 的 thursday 周记语义待业主决策（低危，接受待决）。
  - **M3-a 已验收（D-035）**；M3-b 细化设计待排（testnet key 豁免条款 D-034：隔离+SIMULATED 标记+只读，真金零密钥不变）。
  - **M3-a 交付形态注**：sim 是零侵入纯库（订单生成器 + 模拟盘回填逻辑，全量对抗测试；migration 0005 已 applied），**尚未接线进服务**（main.go/dashboard 无 sim import）——运行驱动（调度/CLI/RPC）属 M3-b/c 集成时接。
- 下一步: **M3-b 细化设计**（04-m3-spec.md §2 模拟对账：testnet 只读收敛 + 白名单显式配置落地，M3-a 复审 M2 接受项）。
- 清扫上翻: 本次 review 教训入 practices.md #6-#10（刻度统一/NaN 门禁/状态+从属行原子/信任边界标注/时钟注入覆盖全路径）；追溯深审 + M3-a 复审结论入 decisions.md D-035；对话落 dialogue.md #39/#40。
