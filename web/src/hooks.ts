import { useCallback, useEffect, useState } from "react";

import { dashboard } from "./client";
import type { Alert, Fact, HealthResponse, TriggerState } from "./gen/arbcn/dashboard/v1/dashboard_pb";

// POLL_MS 快照轮询间隔（只读数据，60s 足够跟住 collector 节奏）。
const POLL_MS = 60_000;

export interface Snapshot {
  facts: Fact[];
  states: TriggerState[];
  alerts: Alert[];
  health: HealthResponse;
  at: Date;
}

// useSnapshot 拉取全量快照（四 RPC 并行）并按 POLL_MS 轮询；ackAlert 确认后本地更新。
export function useSnapshot(): {
  snap: Snapshot | null;
  error: string;
  reload: () => void;
  ackAlert: (id: bigint) => Promise<void>;
} {
  const [snap, setSnap] = useState<Snapshot | null>(null);
  const [error, setError] = useState("");
  const [tick, setTick] = useState(0);

  useEffect(() => {
    let alive = true;
    const load = async () => {
      try {
        const [facts, states, alerts, health] = await Promise.all([
          dashboard.listLatestFacts({}),
          dashboard.listTriggerStates({}),
          dashboard.listAlerts({ limit: 200 }),
          dashboard.health({}),
        ]);
        if (!alive) return;
        setSnap({
          facts: facts.facts,
          states: states.states,
          alerts: alerts.alerts,
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
      setSnap((prev) =>
        prev
          ? { ...prev, alerts: prev.alerts.map((a) => (a.id === id ? { ...a, acked: true } : a)) }
          : prev,
      );
      setError("");
    } catch (e) {
      setError(String(e));
    }
  }, []);

  return { snap, error, reload, ackAlert };
}
