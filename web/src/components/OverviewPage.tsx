import { useKnowledge, useSim, type Snapshot } from "../hooks";
import { Alerts } from "./Alerts";
import { ConfirmPanel } from "./ConfirmPanel";
import { Insights } from "./Insights";
import { KnowledgeBoard } from "./KnowledgeBoard";
import { Opportunity } from "./Opportunity";
import { Triggers } from "./Triggers";

// OverviewPage 监控总览（D-047 P0 + D-048 第一眼原则）。
// 数据层随视图生命周期：useSim/useKnowledge 在本组件挂载/卸载（仅总览 tab 常驻，
// 不再 App 根无条件常驻）；顶部全局刷新经 refreshKey 触发 sim/knowledge 重载。
// 布局按「第一眼最需要看到什么」排序（D-048）：① 待确认下单置顶整宽（该我行动）→
// ② 机会裁决 + 告警流双栏（钱在招手还是坑 / 系统刚看到什么）→ ③ 进化建议 → ④
// 触发器（仅 active 信号面）→ ⑤ 经验库（低频参考面，命中才默认展开）。
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
  // 经验库命中即展开：进化建议出现 knowledge 类目 → 系统检测到同签名情况，
  // 把判定记录提前呈现供裁决对照（D-048 U3）。
  const hasKnowledgeMatch = snap.insights.some((i) => i.category === "knowledge");

  return (
    <>
      {error ? (
        <div className="banner" role="alert">
          加载失败：{error}
        </div>
      ) : null}
      {/* ① 该我行动：确认下单置顶整宽（有单即显，空态保底出口不消失） */}
      <ConfirmPanel orders={sim.orders} confirm={sim.confirm} error={sim.error} reload={sim.reload} />
      {/* ② 双栏：机会面板（裁决 + 折叠数据）| 告警流 */}
      <div className="row">
        <Opportunity facts={snap.facts} sourceHealth={snap.sourceHealth} cards={snap.cards} />
        <Alerts alerts={snap.alerts} ackBusy={ackBusy} onAck={ack} />
      </div>
      <Insights insights={snap.insights} />
      <Triggers states={snap.states} />
      <KnowledgeBoard
        entries={knowledge.entries}
        error={knowledge.error}
        onReload={knowledge.reload}
        defaultOpen={hasKnowledgeMatch}
      />
    </>
  );
}
