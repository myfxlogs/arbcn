import { useKnowledge, useQuotes, useSim, type Snapshot } from "../hooks";
import { ClosePanel } from "./ClosePanel";
import { ConfirmPanel } from "./ConfirmPanel";
import { Insights } from "./Insights";
import { KnowledgeBoard } from "./KnowledgeBoard";
import { MarketMatrix } from "./MarketMatrix";
import { Opportunity } from "./Opportunity";
import { QuoteStrip } from "./QuoteStrip";
import { Triggers } from "./Triggers";

// OverviewPage 监控总览（D-047 P0 + D-048 + D-050 布局 + D-052 调整）。
// 数据层随视图生命周期：useSim/useKnowledge 在本组件挂载/卸载（仅总览 tab 常驻）；
// 顶部全局刷新经 refreshKey 触发 sim/knowledge 重载。
// 布局（3×2 网格 auto-flow）：左上1 数据矩阵（眼睛扫描起点）→ 右上1 确认下单
// （D-052 提上去：桌面首行、移动端第2，该我行动第一眼可见）→ 左2 机会面板（实算卡
// 裁决）→ 右2 市场结构经验库 → 左3 进化建议 → 右3 触发器。D-052 后 6 张卡高度
// 全部有界（机会面板/经验库/进化建议 用 scroll-cap 卡内滚动，不再向下无限拉伸）。
// 告警流第1性原则 = 先给状态再给内容：未读数/时间线/ack 由 header 铃铛承接，网格
// 不再塞时间线卡。
export function OverviewPage({
  snap,
  error,
  refreshKey,
}: {
  snap: Snapshot;
  error: string;
  refreshKey: number;
}) {
  // sim/knowledge 数据随本页生命周期挂载；refreshKey 递增触发重载（全局刷新联动）。
  const sim = useSim(refreshKey);
  const knowledge = useKnowledge(refreshKey);
  const { quotes } = useQuotes();

  return (
    <>
      {error ? (
        <div className="banner" role="alert">
          加载失败：{error}
        </div>
      ) : null}
      {/* D-056 Part B 实时报价条：秒级跳动，看盘第一眼数据（header 之下、网格之上） */}
      <QuoteStrip quotes={quotes} />
      {/* 网格（D-050 3×2 + D-052 提权 + D-053 右列堆叠）：行序 = 左上矩阵 /
          右1 [经验库+确认下单] 堆叠（align-self:stretch 对齐矩阵高） / 左2 机会 /
          右2 进化建议（stretch 对齐机会高） / 左3 触发器，auto-flow 天然落位 */}
      <div className="grid">
        <MarketMatrix facts={snap.facts} sourceHealth={snap.sourceHealth} />
        <div className="right-stack">
          <KnowledgeBoard
            entries={knowledge.entries}
            error={knowledge.error}
            onReload={knowledge.reload}
            defaultOpen
            review={knowledge.review}
          />
          {/* 对话 #81：市场结构与确认下单之间的空白 → 平仓卡（当前持仓 + 整单平） */}
          <ClosePanel
            positions={sim.positions}
            fxAvailable={sim.fxAvailable}
            close={sim.close}
            error={sim.error}
            reload={sim.reload}
          />
          <ConfirmPanel
            orders={sim.orders}
            confirm={sim.confirm}
            error={sim.error}
            reload={sim.reload}
          />
        </div>
        <Opportunity cards={snap.cards} />
        <Insights insights={snap.insights} />
        <Triggers states={snap.states} />
      </div>
    </>
  );
}
