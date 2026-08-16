import { pct } from "../format";
import type { OpportunityCard } from "../gen/arbcn/dashboard/v1/dashboard_pb";
import { Chip, type ChipTone } from "./Chip";
import { kindText } from "./sim";
import { StatTile } from "./StatTile";

// ratingLabel 实算卡三档判定 → 中文徽标（grab=可抓 / breakeven=打平观望 / trap=坑）。
function ratingLabel(r: string): string {
  switch (r) {
    case "grab":
      return "可抓";
    case "breakeven":
      return "打平/观望";
    case "trap":
      return "坑";
    default:
      return r;
  }
}

// ratingTone 判定 → 徽标色（可抓=绿 / 观望=黄 / 坑=红；色永不带字面语义，配文字徽标）。
function ratingTone(r: string): ChipTone {
  switch (r) {
    case "grab":
      return "good";
    case "breakeven":
      return "warn";
    case "trap":
      return "critical";
    default:
      return "neutral";
  }
}

// oppValue 数值 → 展示串（NaN = 样本不足/不适用 → 「—」，不编造 0）。
function oppValue(v: number, suffix: (n: number) => string): string {
  return Number.isNaN(v) ? "—" : suffix(v);
}

// Opportunity 机会面板（D-050 拆分：数据矩阵已独立成 MarketMatrix 左上1，本卡只剩
// 实算卡裁决 = 更短）。「钱在招手还是坑」的裁决面，第一眼在数据监控之后读。
export function Opportunity({ cards }: { cards: OpportunityCard[] }) {
  return (
    <section className="card" aria-labelledby="opp-title">
      <h2 id="opp-title">机会面板</h2>

      {/* D-046 机会实算卡：确定性算账（投运后无需 Claude 在场）。只读证据表面——
          卡只说「这笔账划不划算」，执行门禁仍由规则引擎把关。
          D-052：卡多时高度封顶（scroll-cap）卡内滚动，不向下无限拉伸。 */}
      <h3>
        机会实算卡（确定性算账 · 扣摩擦净收益 · 共 {cards.length} 张）
      </h3>
      {cards.length === 0 ? (
        <p className="empty">暂无实算卡（数据不足不产卡）</p>
      ) : (
        <ul className="opp-cards scroll-cap">
          {cards.map((c) => (
            <li key={`${c.kind}:${c.venue}:${c.symbol}`} className="opp-card">
              <div className="opp-card-head">
                <Chip tone={ratingTone(c.rating)}>{ratingLabel(c.rating)}</Chip>
                <span className="opp-card-kind">{kindText(c.kind)}</span>
                <span className="opp-card-who">
                  {c.venue} · {c.symbol}
                </span>
                <span className="opp-card-friction">摩擦 {oppValue(c.frictionPct, (n) => pct(n, 2))}</span>
              </div>
              <div className="stats opp-card-stats">
                <StatTile label="瞬时" value={oppValue(c.inst, (n) => pct(n))} />
                <StatTile label="30日均值" value={oppValue(c.avg30d, (n) => pct(n))} />
                <StatTile label="保本天数" value={oppValue(c.breakEvenDays, (n) => `${n.toFixed(0)} 天`)} />
                <StatTile label="净年化" value={oppValue(c.netAnnualized, (n) => pct(n))} />
              </div>
              <p className="opp-card-narrative">{c.narrative}</p>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
