import type { Timestamp } from "@bufbuild/protobuf/wkt";
import { useCallback, useEffect, useRef, useState } from "react";

import { dashboard, sim } from "./client";
import type {
  AddLedgerEntryRequest,
  Alert,
  Fact,
  FactRmb,
  HealthResponse,
  Insight,
  KnowledgeEntry,
  LedgerEntry,
  ListFactsResponse,
  OpportunityCard,
  SourceHealth,
  TierSummary,
  TriggerState,
  UnackedAlert,
} from "./gen/arbcn/dashboard/v1/dashboard_pb";
import type {
  CloseSimOrderResponse,
  GetSimAccountResponse,
  GetSimReportResponse,
  SimOrder,
  SimPosition,
  TestnetAccount,
} from "./gen/arbcn/sim/v1/sim_pb";

// POLL_MS 快照轮询间隔（只读数据，60s 足够跟住 collector 节奏）。
const POLL_MS = 60_000;

// useSim 模拟执行面板（M3-c C4，04-m3-spec §10.5）：建议订单 + 模拟持仓 + 对账报告
// + 测试网账户（D-040）。写路径 = ConfirmSimOrder（开仓）+ CloseSimOrder（平仓，D-055
// 整单平：订单全部腿一起退，绝不单腿留裸敞口 D-019）；成功后本地刷新各区。
// SIMULATED 徽标由 SimExec 组件固定渲染（可检查）。
// D-047 P0：本 hook 由使用它的页面组件挂载（OverviewPage/SimPage），hook 生命周期 =
// 视图生命周期；加 60s 轮询（8h 结算新单可被确认面板捕获，不再只靠手动刷新）。
// refreshKey：顶部全局刷新信号（App 递增）→ effect 重载。
export function useSim(refreshKey?: number): {
  orders: SimOrder[];
  positions: SimPosition[];
  report: GetSimReportResponse | null;
  accounts: TestnetAccount[];
  account: GetSimAccountResponse | null;
  fxAvailable: boolean;
  error: string;
  confirm: (id: bigint) => Promise<boolean>;
  close: (id: bigint, note?: string) => Promise<CloseSimOrderResponse | null>;
  reload: () => void;
} {
  const [orders, setOrders] = useState<SimOrder[]>([]);
  const [positions, setPositions] = useState<SimPosition[]>([]);
  const [report, setReport] = useState<GetSimReportResponse | null>(null);
  const [accounts, setAccounts] = useState<TestnetAccount[]>([]);
  const [account, setAccount] = useState<GetSimAccountResponse | null>(null);
  const [fxAvailable, setFxAvailable] = useState(false);
  const [error, setError] = useState("");
  const [tick, setTick] = useState(0);

  useEffect(() => {
    let alive = true;
    const load = async () => {
      try {
        const [ordersRes, positionsRes, reportRes, accountsRes, accountRes] = await Promise.all([
          sim.listSimOrders({}),
          sim.listSimPositions({}),
          sim.getSimReport({}),
          sim.getTestnetAccounts({}),
          sim.getSimAccount({}),
        ]);
        if (!alive) return;
        setOrders(ordersRes.orders);
        setPositions(positionsRes.positions);
        setReport(reportRes);
        setAccounts(accountsRes.accounts);
        setAccount(accountRes); // D-056 完整现金账本：账户净值 + 逐笔流水
        setFxAvailable(positionsRes.fxAvailable); // D-047 F4：真实汇率可用信号（区分「USD 原值」与真零 PnL）
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
  }, [tick, refreshKey]);

  const confirm = useCallback(async (id: bigint): Promise<boolean> => {
    try {
      const res = await sim.confirmSimOrder({ id });
      setTick((n) => n + 1); // 刷新各区（订单状态 + 新持仓）
      setError("");
      return res.accepted;
    } catch (e) {
      setError(String(e));
      return false;
    }
  }, []);

  // close 平仓（D-055 整单平，后端订单级原子事务：订单全部 open 腿一起退，防半仓裸敞口）。
  // 返回 CloseSimOrderResponse（含 realized_pnl 实现净 PnL，前端平仓结果横幅展示）。
  const close = useCallback(async (id: bigint, note?: string): Promise<CloseSimOrderResponse | null> => {
    try {
      const res = await sim.closeSimOrder({ id, note: note ?? "" });
      setTick((n) => n + 1); // 刷新各区（订单 closed + 腿 settled）
      setError("");
      return res;
    } catch (e) {
      setError(String(e));
      return null;
    }
  }, []);

  return {
    orders,
    positions,
    report,
    accounts,
    account,
    fxAvailable,
    error,
    confirm,
    close,
    reload: useCallback(() => setTick((n) => n + 1), []),
  };
}

export interface Snapshot {
  facts: Fact[];
  states: TriggerState[];
  alerts: Alert[];
  unacked: UnackedAlert[];
  sourceHealth: SourceHealth[];
  insights: Insight[];
  cards: OpportunityCard[];
  health: HealthResponse;
  at: Date;
}

// Quote 单标的最新价（/quote/stream SSE 负载；D-056 Part B）。
export interface Quote {
  venue: string;
  symbol: string;
  price: number;
  ts_ms: number;
}

// useQuotes 秒级实时报价（D-056 Part B 下游）：EventSource 订阅 /quote/stream，
// onmessage 解析 JSON 更新 map（key = venue|symbol）。EventSource 原生自动重连——
// 断线由浏览器自动恢复，无需手动逻辑；组件卸载时 close 释放。只做展示，不喂策略。
export function useQuotes(): { quotes: Record<string, Quote> } {
  const [quotes, setQuotes] = useState<Record<string, Quote>>({});

  useEffect(() => {
    const es = new EventSource("/quote/stream");
    es.onmessage = (ev: MessageEvent) => {
      try {
        const q = JSON.parse(ev.data) as Quote;
        if (!q.venue || !q.symbol) return;
        setQuotes((prev) => ({ ...prev, [`${q.venue}|${q.symbol}`]: q }));
      } catch {
        // 解析失败跳过（脏帧不影响整体）
      }
    };
    return () => es.close();
  }, []);

  return { quotes };
}

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

// useSnapshot 拉取全量快照（八 RPC 并行，含 M2-a 的 ListUnacked/ListSourceHealth、
// D-046 的 ListOppCards）并按 POLL_MS 轮询；ackAlert/ackAll 确认后本地更新
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
        const [facts, states, alerts, unacked, sourceHealth, insights, cards, health] = await Promise.all([
          dashboard.listLatestFacts({}),
          dashboard.listTriggerStates({}),
          dashboard.listAlerts({ limit: 200 }),
          dashboard.listUnacked({}),
          dashboard.listSourceHealth({}),
          dashboard.listInsights({}),
          dashboard.listOppCards({}),
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
