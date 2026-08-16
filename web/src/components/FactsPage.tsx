import { useFactsSnapshot } from "../hooks";
import { FactsSnapshot } from "./FactsSnapshot";

// FactsPage 事实快照页（D-047 P0）：useFactsSnapshot 随本页挂载/卸载——切走即停止
// 轮询（消除跨 tab 空转：此前 App 根常驻，任何 tab 都 60s 拉事实页数据）。
export function FactsPage() {
  const factsSnap = useFactsSnapshot();
  return (
    <FactsSnapshot
      facts={factsSnap.facts}
      fxRate={factsSnap.fxRate}
      fxAvailable={factsSnap.fxAvailable}
      fxTs={factsSnap.fxTs}
      error={factsSnap.error}
      onReload={factsSnap.reload}
    />
  );
}
