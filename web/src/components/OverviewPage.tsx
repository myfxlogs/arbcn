import { useKnowledge, useSim, type Snapshot } from "../hooks";
import { ConfirmPanel } from "./ConfirmPanel";
import { Insights } from "./Insights";
import { KnowledgeBoard } from "./KnowledgeBoard";
import { MarketMatrix } from "./MarketMatrix";
import { Opportunity } from "./Opportunity";
import { Triggers } from "./Triggers";

// OverviewPage 监控总览（D-047 P0 + D-048 + D-050 布局）。
// 数据层随视图生命周期：useSim/useKnowledge 在本组件挂载/卸载（仅总览 tab 常驻）；
// 顶部全局刷新经 refreshKey 触发 sim/knowledge 重载。
// 布局（D-050 业主指定 3×2 网格，对话 #68）：左上1 数据矩阵（眼睛扫描起点）→
// 右上1 市场结构经验库 → 左2 机会面板（实算卡裁决）→ 右2 确认下单（鼠标顺手高度，
// 操作热区）→ 左3 进化建议 → 右3 触发器。告警流第1性原则 = 先给状态再给内容：
// 未读数/时间线/ack 由 header 铃铛承接（状态第一眼可见、处置一键可达），网格不再
// 塞时间线卡。
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

  return (
    <>
      {error ? (
        <div className="banner" role="alert">
          加载失败：{error}
        </div>
      ) : null}
      {/* 3×2 网格：行序 = 左上/右上 / 左中/右中 / 左下/右下，auto-flow 天然落位 */}
      <div className="grid">
        <MarketMatrix facts={snap.facts} sourceHealth={snap.sourceHealth} />
        <KnowledgeBoard
          entries={knowledge.entries}
          error={knowledge.error}
          onReload={knowledge.reload}
          defaultOpen
        />
        <Opportunity cards={snap.cards} />
        <ConfirmPanel orders={sim.orders} confirm={sim.confirm} error={sim.error} reload={sim.reload} />
        <Insights insights={snap.insights} />
        <Triggers states={snap.states} />
      </div>
    </>
  );
}
