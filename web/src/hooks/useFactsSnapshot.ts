import type { Timestamp } from "@bufbuild/protobuf/wkt";
import { useCallback, useEffect, useState } from "react";

import { dashboard } from "../client";
import type { FactRmb, ListFactsResponse } from "../gen/arbcn/dashboard/v1/dashboard_pb";
import { POLL_MS } from "./shared";

// useFactsSnapshot 事实快照 + RMB 折算（M2-b §4/§5 机器可读投影，ListFacts RPC）。
// 只读轮询：ListFacts 返回覆盖 kind 折算后的 RMB 净收益视角 + fx 可用性。
export function useFactsSnapshot(): {
  facts: FactRmb[];
  fxRate: number;
  fxAvailable: boolean;
  fxTs?: Timestamp;
  error: string;
  reload: () => void;
} {
  const [snap, setSnap] = useState<ListFactsResponse | null>(null);
  const [error, setError] = useState("");
  const [tick, setTick] = useState(0);

  useEffect(() => {
    let alive = true;
    const load = async () => {
      try {
        const res = await dashboard.listFacts({});
        if (!alive) return;
        setSnap(res);
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

  return {
    facts: snap?.facts ?? [],
    fxRate: snap?.fxRate ?? 0,
    fxAvailable: snap?.fxAvailable ?? false,
    fxTs: snap?.fxTs,
    error,
    reload: useCallback(() => setTick((n) => n + 1), []),
  };
}
