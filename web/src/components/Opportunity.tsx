import { days, fmtTs, pct } from "../format";
import type { Fact, SourceHealth } from "../gen/arbcn/dashboard/v1/dashboard_pb";
import { dotFor, healthMap } from "./freshness";
import { MatrixTable, type MatrixCell } from "./Matrix";
import { StatTile } from "./StatTile";

const KIND_FUNDING = "funding";
const KIND_DEFI = "defi_rate";
const KIND_IV = "iv";
const KIND_REPO = "reverse_repo";
const KIND_CALENDAR = "calendar";

// uniq 保序去重。
function uniq(xs: string[]): string[] {
  return [...new Set(xs)];
}

// buildMatrix 按 rows=Symbol / cols=Venue 组装矩阵（每格取该键最新事实；
// 单元格附该列 venue 对应的源 freshness 点，映射不到则静默）。
function buildMatrix(
  facts: Fact[],
  kind: string,
  health: Map<string, SourceHealth>,
): {
  rows: string[];
  cols: string[];
  get: (row: string, col: string) => MatrixCell | null;
} {
  const rows = uniq(facts.map((f) => f.symbol)).sort();
  const cols = uniq(facts.map((f) => f.venue)).sort();
  const byKey = new Map<string, Fact>();
  for (const f of facts) byKey.set(`${f.symbol}/${f.venue}`, f);
  return {
    rows,
    cols,
    get: (row, col) => {
      const f = byKey.get(`${row}/${col}`);
      if (!f) return null;
      return {
        value: pct(f.value),
        title: `更新 ${fmtTs(f.ts)}`,
        neg: f.value < 0,
        dot: dotFor(kind, col, health) ?? undefined,
      };
    },
  };
}

// calLabel 日历事件中文名（规则事件），人工事件直接用符号。
function calLabel(symbol: string): string {
  switch (symbol) {
    case "month_end":
      return "月末";
    case "quarter_end":
      return "季末";
    case "thursday":
      return "周四";
    default:
      return symbol;
  }
}

// Opportunity 机会面板：funding 矩阵 + 稳定币利率表 + IV + 逆回购/时点倒计时。
export function Opportunity({ facts, sourceHealth }: { facts: Fact[]; sourceHealth: SourceHealth[] }) {
  const health = healthMap(sourceHealth);
  const funding = buildMatrix(facts.filter((f) => f.kind === KIND_FUNDING), KIND_FUNDING, health);
  const defi = buildMatrix(facts.filter((f) => f.kind === KIND_DEFI), KIND_DEFI, health);
  const iv = facts.filter((f) => f.kind === KIND_IV);
  const repo = facts.filter((f) => f.kind === KIND_REPO);
  const cal = facts.filter((f) => f.kind === KIND_CALENDAR).sort((a, b) => a.value - b.value);

  return (
    <section className="card" aria-labelledby="opp-title">
      <h2 id="opp-title">机会面板</h2>

      <h3>funding 矩阵（币 × 所 · 年化 %）</h3>
      <MatrixTable rows={funding.rows} cols={funding.cols} cell={funding.get} empty="暂无 funding 数据" />

      <h3>稳定币金额档利率（项目 × 币种 · 年化 %）</h3>
      <MatrixTable rows={defi.rows} cols={defi.cols} cell={defi.get} empty="暂无 DeFi 利率数据" />

      <h3>IV 隐含波动率</h3>
      <div className="stats">
        {iv.length === 0 ? (
          <p className="empty">暂无 IV 数据</p>
        ) : (
          iv.map((f) => (
            <StatTile
              key={`${f.venue}:${f.symbol}`}
              label={`${f.symbol} · ${f.venue}`}
              value={pct(f.value, 1)}
              sub={`更新 ${fmtTs(f.ts)}`}
              title={f.src}
              dot={dotFor(KIND_IV, f.venue, health) ?? undefined}
            />
          ))
        )}
      </div>

      <h3>逆回购 + 下个时点倒计时</h3>
      <div className="stats">
        {repo.length === 0 ? (
          <p className="empty">暂无逆回购数据</p>
        ) : (
          repo.map((f) => (
            <StatTile
              key={f.symbol}
              label={`逆回购 ${f.symbol}`}
              value={pct(f.value)}
              sub={`更新 ${fmtTs(f.ts)}`}
              title={f.src}
              dot={dotFor(KIND_REPO, f.venue, health) ?? undefined}
            />
          ))
        )}
        {cal.map((f) => (
          <StatTile
            key={`${f.venue}:${f.symbol}`}
            label={`时点 ${calLabel(f.symbol)}`}
            value={days(f.value)}
            sub={`来源 ${f.venue}`}
            title={f.src}
            dot={dotFor(KIND_CALENDAR, f.venue, health) ?? undefined}
          />
        ))}
      </div>
    </section>
  );
}
