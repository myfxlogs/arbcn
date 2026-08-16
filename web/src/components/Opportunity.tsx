import { days, fmtTs, pct } from "../format";
import type { Fact, OpportunityCard, SourceHealth } from "../gen/arbcn/dashboard/v1/dashboard_pb";
import { Chip, type ChipTone } from "./Chip";
import { Collapse } from "./Collapse";
import { dotFor, healthMap } from "./freshness";
import { MatrixTable, type MatrixCell } from "./Matrix";
import { kindText } from "./sim";
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

// venueLabel 事实 venue → 矩阵列头显示名（practices #18：映射对照后端 venue 全集，
// 未知名回退原名）。defi_rate 协议 ID 过长（aave-v3 等 14~16 字符）会撑宽矩阵
// 出横向滑块，缩为协议短名；funding 的 binance/okx 本就短，走 default 回退。
function venueLabel(v: string): string {
  switch (v) {
    case "aave-v3":
      return "Aave";
    case "blackrock-buidl":
      return "BlackRock";
    case "ethena-usde":
      return "Ethena";
    case "morpho-blue":
      return "Morpho";
    case "ondo-yield-assets":
      return "Ondo";
    default:
      return v;
  }
}

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

// Opportunity 机会面板（D-048 U2 第一眼原则）：实算卡裁决置前（「钱在招手还是坑」先于
// 数据下钻）；funding 矩阵默认展开（主数据恒可扫），defi/IV/逆回购折叠为低频下钻面。
export function Opportunity({
  facts,
  sourceHealth,
  cards,
}: {
  facts: Fact[];
  sourceHealth: SourceHealth[];
  cards: OpportunityCard[];
}) {
  const health = healthMap(sourceHealth);
  const funding = buildMatrix(facts.filter((f) => f.kind === KIND_FUNDING), KIND_FUNDING, health);
  const defi = buildMatrix(facts.filter((f) => f.kind === KIND_DEFI), KIND_DEFI, health);
  const iv = facts.filter((f) => f.kind === KIND_IV);
  const repo = facts.filter((f) => f.kind === KIND_REPO);
  const cal = facts.filter((f) => f.kind === KIND_CALENDAR).sort((a, b) => a.value - b.value);

  return (
    <section className="card" aria-labelledby="opp-title">
      <h2 id="opp-title">机会面板</h2>

      {/* D-046 机会实算卡：确定性算账（投运后无需 Claude 在场）。只读证据表面——
          卡只说「这笔账划不划算」，执行门禁仍由规则引擎把关。D-048 U2：裁决置前，
          第一眼先见「钱在招手还是坑」，数据矩阵折叠在其下。 */}
      <h3>机会实算卡（确定性算账 · 扣摩擦净收益）</h3>
      {cards.length === 0 ? (
        <p className="empty">暂无实算卡（数据不足不产卡）</p>
      ) : (
        <ul className="opp-cards">
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

      {/* 数据下钻（D-048 U2）：funding 主矩阵默认展开，其余折叠——首屏给裁决，下钻留给需要时。 */}
      <Collapse title="资金费率矩阵（币 × 所 · 年化 %）" defaultOpen>
        <MatrixTable rows={funding.rows} cols={funding.cols} cell={funding.get} empty="暂无资金费率数据" />
      </Collapse>

      <Collapse title="稳定币金额档利率（项目 × 币种 · 年化 %）">
        <MatrixTable rows={defi.rows} cols={defi.cols} cell={defi.get} colLabel={venueLabel} empty="暂无 DeFi 利率数据" />
      </Collapse>

      <Collapse title="IV 隐含波动率">
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
      </Collapse>

      <Collapse title="逆回购 + 下个时点倒计时">
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
      </Collapse>
    </section>
  );
}
