import { useCallback, useEffect, useRef, useState } from "react";

import { dashboard } from "../client";
import { emptyWindowStats } from "../windowDefaults";
import { POLL_MS } from "./shared";
import type { Snapshot } from "./shared";

// useSnapshot 拉取全量快照（九 RPC 并行，含 M2-a 的 ListUnacked/ListSourceHealth、
// D-046 的 ListOppCards、D-064 的 ListFundingWindowStats）并按 POLL_MS 轮询；
// ackAlert/ackAll 确认后本地更新
//（铃铛计数即时递减，不重拉）。
export function useSnapshot(): {
  snap: Snapshot | null;
  error: string;
  reload: () => void;
  ackAlert: (id: bigint) => Promise<void>;
  ackAll: () => Promise<number>;
} {
  const [snap, setSnap] = useState<Snapshot | null>(null);
  const [error, setError] = useState("");
  const [tick, setTick] = useState(0);
  // ackVersion：ackAlert/ackAll 成功后递增。在途 poll 落地时若版本已变（期间发生过
  // 确认），丢弃该响应——防止旧 poll 的"未确认"快照覆盖乐观更新、已确认告警复活
  //（R6#2：原实现 Promise.all 六 RPC 在途时 ack 被旧响应覆盖，最长 60s）。
  const ackVersion = useRef(0);

  useEffect(() => {
    let alive = true;
    const load = async () => {
      const v = ackVersion.current;
      try {
        const [facts, states, alerts, unacked, sourceHealth, insights, cards, window, health] = await Promise.all([
          dashboard.listLatestFacts({}),
          dashboard.listTriggerStates({}),
          dashboard.listAlerts({ limit: 200 }),
          dashboard.listUnacked({}),
          dashboard.listSourceHealth({}),
          dashboard.listInsights({}),
          dashboard.listOppCards({}),
          dashboard.listFundingWindowStats({}), // D-064 7d 费率窗口
          dashboard.health({}),
        ]);
        if (!alive) return;
        if (ackVersion.current !== v) return; // 期间确认过 → 丢弃本 poll（下轮拉服务端真实状态）
        setSnap({
          facts: facts.facts,
          states: states.states,
          alerts: alerts.alerts,
          unacked: unacked.items,
          sourceHealth: sourceHealth.items,
          insights: insights.insights,
          cards: cards.cards,
          window: window.overall ?? emptyWindowStats,
          windowPairs: window.perPair,
          health,
          at: new Date(),
        });
        setError("");
      } catch (e) {
        if (!alive) return;
        setError(String(e));
      }
    };
    void load();
    const timer = setInterval(() => void load(), POLL_MS);
    return () => {
      alive = false;
      clearInterval(timer);
    };
  }, [tick]);

  const reload = useCallback(() => setTick((n) => n + 1), []);

  const ackAlert = useCallback(async (id: bigint) => {
    try {
      await dashboard.ackAlert({ id });
      ackVersion.current += 1; // 丢弃在途 poll 的旧快照，防确认复活（R6#2）
      setSnap((prev) =>
        prev
          ? {
              ...prev,
              alerts: prev.alerts.map((a) => (a.id === id ? { ...a, acked: true } : a)),
              unacked: prev.unacked.filter((u) => u.id !== id),
            }
          : prev,
      );
      setError("");
    } catch (e) {
      setError(String(e));
    }
  }, []);

  const ackAll = useCallback(async () => {
    try {
      const res = await dashboard.ackAll({});
      ackVersion.current += 1; // 同上
      setSnap((prev) =>
        prev
          ? {
              ...prev,
              unacked: [],
              alerts: prev.alerts.map((a) => (a.acked ? a : { ...a, acked: true })),
            }
          : prev,
      );
      setError("");
      return res.ackedCount;
    } catch (e) {
      setError(String(e));
      return 0;
    }
  }, []);

  return { snap, error, reload, ackAlert, ackAll };
}
