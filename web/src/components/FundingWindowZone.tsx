import { pct } from "../format";
import type { FundingWindowPair, FundingWindowStats } from "../gen/arbcn/dashboard/v1/dashboard_pb";
import { Chip, type ChipTone } from "./Chip";
import { StatTile } from "./StatTile";

// windowLabel 窗口判据类 → 中文徽标（与后端 ClassifyFundingWindow 同值域，D-064）。
function windowLabel(c: string): string {
  switch (c) {
    case "high":
      return "高费率档";
    case "tradable":
      return "可交易窗口";
    case "watch":
      return "临界窗口";
    case "not":
      return "非窗口";
    default:
      return c;
  }
}

// windowTone 类 → 徽标色（可交易=绿 / 临界=黄 / 非窗口=灰；色不带字面语义，配文字）。
function windowTone(c: string): ChipTone {
  switch (c) {
    case "high":
    case "tradable":
      return "good";
    case "watch":
      return "warn";
    default:
      return "neutral";
  }
}

// pctShare 正费率占比 0-1 → 百分比串（0 = "0%" 正常展示，不遮空）。
const pctShare = (v: number): string => `${(v * 100).toFixed(0)}%`;

// FundingWindowZone 7d 费率窗口卡（D-064）：滚动 7d funding 环境 →「当前是否处于
// 可交易窗口」判据。机会实算卡同域只读证据面——判据回答「basis/carry 环境行不行」，
// 机会卡回答「这笔账划不划算」，两者互补不重复。只读展示，不碰任何执行门禁。
export function FundingWindowZone({
  window,
  pairs,
}: {
  window: FundingWindowStats;
  pairs: FundingWindowPair[];
}) {
  return (
    <section className="card" aria-labelledby="window-title">
      <h2 id="window-title">费率窗口</h2>
      <h3>7d 判据（滚动 {window.count > 0 ? `${window.count} 份` : "窗口无数据"} funding 读数）</h3>

      <div className="window-head">
        <Chip tone={windowTone(window.class)}>{windowLabel(window.class)}</Chip>
        <span className="window-note">{window.note}</span>
      </div>

      <div className="stats">
        <StatTile label="均值年化" value={window.count > 0 ? pct(window.mean) : "—"} />
        <StatTile label="最低" value={window.count > 0 ? pct(window.min) : "—"} />
        <StatTile label="最高" value={window.count > 0 ? pct(window.max) : "—"} />
        <StatTile label="正费率占比" value={window.count > 0 ? pctShare(window.positiveShare) : "—"} />
        <StatTile label="样本" value={String(window.count)} />
      </div>

      {pairs.length > 0 ? (
        <>
          <h3>逐 venue·symbol（均值降序前 50）</h3>
          <div className="table-scroll scroll-cap">
            <table className="rows">
              <thead>
                <tr>
                  <th scope="col">标的</th>
                  <th scope="col">判据</th>
                  <th scope="col">均值年化</th>
                  <th scope="col">正占比</th>
                  <th scope="col">样本</th>
                </tr>
              </thead>
              <tbody>
                {pairs.map((p) => (
                  <tr key={`${p.venue}|${p.symbol}`}>
                    <td>
                      {p.venue} · {p.symbol}
                    </td>
                    <td>
                      <Chip tone={windowTone(p.stats?.class ?? "not")}>
                        {windowLabel(p.stats?.class ?? "not")}
                      </Chip>
                    </td>
                    <td>{p.stats && p.stats.count > 0 ? pct(p.stats.mean) : "—"}</td>
                    <td>{p.stats && p.stats.count > 0 ? pctShare(p.stats.positiveShare) : "—"}</td>
                    <td>{p.stats?.count ?? 0}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        </>
      ) : null}
    </section>
  );
}
