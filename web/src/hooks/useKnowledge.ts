import { useCallback, useEffect, useState } from "react";

import { dashboard } from "../client";
import type { KnowledgeEntry } from "../gen/arbcn/dashboard/v1/dashboard_pb";

// useKnowledge 市场结构经验库（D-046）：浏览已核实模式条目。只读、低频（仅在 D#
// 吸收/复核部署时变化），故挂载加载 + 手动刷新，不加入 60s 轮询（省一路 RPC）。
// refreshKey：顶部全局刷新信号（D-047 P0，总览页经验库随全局刷新联动）。
export function useKnowledge(refreshKey?: number): {
  entries: KnowledgeEntry[];
  error: string;
  reload: () => void;
  review: (signature: string, status: string, verdict: string, note: string) => Promise<void>;
} {
  const [entries, setEntries] = useState<KnowledgeEntry[]>([]);
  const [error, setError] = useState("");
  const [tick, setTick] = useState(0);

  useEffect(() => {
    let alive = true;
    const load = async () => {
      try {
        const res = await dashboard.listKnowledgeEntries({});
        if (!alive) return;
        setEntries(res.entries);
        setError("");
      } catch (e) {
        if (!alive) return;
        setError(String(e));
      }
    };
    void load();
    return () => {
      alive = false;
    };
  }, [tick, refreshKey]);

  // review 人工复核（D-054）：决策层评估后写 validated_at + status + verdict + note
  // （人工在环，系统永不自动复核；status 为生命周期三态之一，服务端白名单校验）。
  // 成功后重拉列表呈现新状态。
  const review = useCallback(async (signature: string, status: string, verdict: string, note: string) => {
    try {
      await dashboard.reviewKnowledgeEntry({ signature, status, verdict, validationNote: note });
      setError("");
    } catch (e) {
      setError(String(e));
      throw e;
    }
    setTick((n) => n + 1);
  }, []);

  return {
    entries,
    error,
    reload: useCallback(() => setTick((n) => n + 1), []),
    review,
  };
}
