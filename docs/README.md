# arbcn 文档地图

> 新 agent 第一入口。阅读顺序 = 下表顺序。分层规则见 AGENTS.md §4。

## 阅读顺序

1. `AGENTS.md` — 契约 SSOT（角色 / 定位 / 六原则 / 分层 / 门禁 / 交接负载）
2. `docs/handoff/STATE.md` — 当前状态 + 交接负载（无损接手核心）
3. `docs/handoff/dialogue.md` — 最近对话纪要（"为什么这么做"的推理链）
4. 任务相关 T1：`decisions.md`（决策）/ `practices.md`（风格约定）

## 分层

| 层 | 文档 | 读取 |
|----|------|------|
| T0 契约 | `AGENTS.md` / `STATE.md` | 每次开工必读（≤20KB） |
| T1 知识 | `decisions.md` / `practices.md` / `dialogue.md` / 本文件 | 按需读（≤450 行） |
| T2 归档 | `LOG.md` | 永不读，git 追溯 |
