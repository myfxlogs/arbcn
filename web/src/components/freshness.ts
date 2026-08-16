import type { SourceHealth } from "../gen/arbcn/dashboard/v1/dashboard_pb";
import { fmtInterval, fmtRel, statusText } from "../format";

// FreshDot 机会 tile 的 freshness 状态点（M2-a §2.3）。
export interface FreshDot {
  status: "live" | "stale" | "down";
  title: string;
}

// sourceForTile 机会 tile → 源健康名映射（决策层已定，M2-a §2.3）：
// kind=funding → `${venue}_funding`（binance→binance_funding）；其余 kind 固定映射；
// 映射不到 = null（不显示状态点，静默）。纯函数，便于后续测试。
export function sourceForTile(kind: string, venue: string): string | null {
  if (kind === "funding") return `${venue}_funding`;
  switch (kind) {
    case "defi_rate":
      return "defi_rates";
    case "iv":
      return "deribit_iv";
    case "reverse_repo":
      return "repo";
    case "calendar":
      return "calendar";
    default:
      return null;
  }
}

// healthMap 把 ListSourceHealth items 投影为 name→SourceHealth。
export function healthMap(items: SourceHealth[]): Map<string, SourceHealth> {
  return new Map(items.map((h) => [h.name, h]));
}

// dotFor 查源健康表生成状态点；映射不到或源未启用 = null（静默，不显示点）。
export function dotFor(kind: string, venue: string, health: Map<string, SourceHealth>): FreshDot | null {
  const name = sourceForTile(kind, venue);
  if (!name) return null;
  const h = health.get(name);
  if (!h) return null;
  return { status: h.status as FreshDot["status"], title: freshnessTooltip(h) };
}

// freshnessTooltip 悬停 tooltip："最近更新 X 前 · 源间隔 Y · 状态 Z"（§2.3）。
function freshnessTooltip(h: SourceHealth): string {
  const at = h.lastFactAt ?? h.lastPollAt;
  return `最近更新 ${fmtRel(at)} · 源间隔 ${fmtInterval(h.intervalSec)} · 状态 ${statusText(h.status)}`;
}
