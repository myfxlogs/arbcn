// hooks 共享件（D-067 拆分）：POLL_MS 轮询间隔 + Snapshot/Quote 接口。
import type {
  Alert,
  Fact,
  FundingWindowPair,
  FundingWindowStats,
  HealthResponse,
  Insight,
  OpportunityCard,
  SourceHealth,
  TriggerState,
  UnackedAlert,
} from "../gen/arbcn/dashboard/v1/dashboard_pb";

// POLL_MS 快照轮询间隔（只读数据，60s 足够跟住 collector 节奏）。
export const POLL_MS = 60_000;

export interface Snapshot {
  facts: Fact[];
  states: TriggerState[];
  alerts: Alert[];
  unacked: UnackedAlert[];
  sourceHealth: SourceHealth[];
  insights: Insight[];
  cards: OpportunityCard[];
  window: FundingWindowStats; // D-064 7d 费率窗口（overall 判据主答案）
  windowPairs: FundingWindowPair[]; // D-064 逐 venue|symbol 明细
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
