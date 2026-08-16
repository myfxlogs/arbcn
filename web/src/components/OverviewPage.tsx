import { useKnowledge, useSim, type Snapshot } from "../hooks";
import { Alerts } from "./Alerts";
import { ConfirmPanel } from "./ConfirmPanel";
import { Insights } from "./Insights";
import { KnowledgeBoard } from "./KnowledgeBoard";
import { Opportunity } from "./Opportunity";
import { Triggers } from "./Triggers";

// OverviewPage 监控总览（D-047 P0）：数据层随视图生命周期——useSim/useKnowledge
// 在本组件挂载/卸载（仅在总览 tab 常驻，不再 App 根无条件常驻）；顶部全局刷新
// 经 refreshKey 触发 sim/knowledge 重载（与 useSnapshot 同步）。snap 来自 App 层
// useSnapshot（header 全局健康徽标/铃铛同源，属全局 chrome 合理常驻）。
export function OverviewPage({
  snap,
  error,
  ackBusy,
  ack,
  refreshKey,
}: {
  snap: Snapshot;
  error: string;
  ackBusy: ReadonlySet<string>;
  ack: (id: bigint) => void;
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
      {/* 双栏（对话 #60 布局调整）：机会面板左列跨两行；右列 = 告警流（上）+ 确认下单（下，与机会面板同行） */}
      <div className="row">
        <Opportunity facts={snap.facts} sourceHealth={snap.sourceHealth} cards={snap.cards} />
        <div className="row-col">
          <Alerts alerts={snap.alerts} ackBusy={ackBusy} onAck={ack} />
          <ConfirmPanel
            orders={sim.orders}
            confirm={sim.confirm}
            error={sim.error}
            reload={sim.reload}
          />
        </div>
      </div>
      <Triggers states={snap.states} />
      <Insights insights={snap.insights} />
      <KnowledgeBoard
        entries={knowledge.entries}
        error={knowledge.error}
        onReload={knowledge.reload}
      />
    </>
  );
}
