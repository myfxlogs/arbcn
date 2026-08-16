import { fmtClock } from "../format";
import type { Quote } from "../hooks";

// venueLabel 报价源 → 中文（QuoteStrip 展示）。
function venueLabel(venue: string): string {
  switch (venue) {
    case "binance":
      return "Binance";
    case "okx":
      return "OKX";
    default:
      return venue;
  }
}

// QuoteStrip 顶部实时报价条（D-056 Part B）：秒级跳动，展示 BTC/ETH/TRX @ binance/okx
// 最新价。只做看盘展示——不喂策略、不落库（facts ticker 1min 仍是 equity/positions 真相
// 源，诚实标注）。数据 = useQuotes（SSE /quote/stream，EventSource 原生自动重连）。
export function QuoteStrip({ quotes }: { quotes: Record<string, Quote> }) {
  // 按 venue 分组展示（binance 在前，okx 在后）；同一标的两所并列方便比价。
  const venues = ["binance", "okx"];
  const latestTs = Object.values(quotes).reduce<number>(
    (m, q) => (q.ts_ms > m ? q.ts_ms : m),
    0,
  );

  const items: { venue: string; symbol: string; q?: Quote }[] = [];
  for (const v of venues) {
    // 符号集 = 任意 venue 下出现的 symbol 并集，按名称排序（稳定）。
    const symbols = [...new Set(Object.values(quotes).map((q) => q.symbol))].sort();
    for (const s of symbols) {
      items.push({ venue: v, symbol: s, q: quotes[`${v}|${s}`] });
    }
  }

  return (
    <div className="quote-strip" aria-label="实时报价">
      {items.length === 0 ? (
        <span className="muted">实时报价加载中…</span>
      ) : (
        <>
          {items.map((it) => (
            <span key={`${it.venue}|${it.symbol}`} className="quote-tile">
              <span className="muted">{it.symbol}</span>
              <span className="quote-venue">{venueLabel(it.venue)}</span>
              <strong className={it.q ? "num" : "muted"}>
                {it.q ? it.q.price.toLocaleString("zh-CN", { maximumFractionDigits: 2 }) : "—"}
              </strong>
            </span>
          ))}
          {latestTs > 0 ? (
            <span className="quote-ts muted">
              {fmtClock(new Date(latestTs))}
            </span>
          ) : null}
        </>
      )}
    </div>
  );
}
