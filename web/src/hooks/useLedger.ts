import { useCallback, useEffect, useState } from "react";

import { dashboard } from "../client";
import type {
  AddLedgerEntryRequest,
  LedgerEntry,
  TierSummary,
} from "../gen/arbcn/dashboard/v1/dashboard_pb";

// useLedger 台账（M2-b §6）：流水 + 档位归因汇总；addEntry 手工录入后本地刷新。
export function useLedger(): {
  entries: LedgerEntry[];
  summary: TierSummary[];
  error: string;
  addEntry: (req: AddLedgerEntryRequest) => Promise<boolean>;
  reload: () => void;
} {
  const [entries, setEntries] = useState<LedgerEntry[]>([]);
  const [summary, setSummary] = useState<TierSummary[]>([]);
  const [error, setError] = useState("");
  const [tick, setTick] = useState(0);

  useEffect(() => {
    let alive = true;
    const load = async () => {
      try {
        const [entriesRes, summaryRes] = await Promise.all([
          dashboard.listLedgerEntries({ limit: 200 }),
          dashboard.ledgerSummary({}),
        ]);
        if (!alive) return;
        setEntries(entriesRes.entries);
        setSummary(summaryRes.items);
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
  }, [tick]);

  const addEntry = useCallback(async (req: AddLedgerEntryRequest): Promise<boolean> => {
    try {
      await dashboard.addLedgerEntry(req);
      setTick((n) => n + 1);
      setError("");
      return true;
    } catch (e) {
      setError(String(e));
      return false;
    }
  }, []);

  return {
    entries,
    summary,
    error,
    addEntry,
    reload: useCallback(() => setTick((n) => n + 1), []),
  };
}
