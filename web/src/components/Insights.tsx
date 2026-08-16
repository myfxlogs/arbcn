import { fmtTs } from "../format";
import type { Insight } from "../gen/arbcn/dashboard/v1/dashboard_pb";
import { LevelChip } from "./Chip";

// catLabel 洞察类别 → 中文（practices #18：映射对照后端 category 全集，未知名回退原名）。
function catLabel(c: string): string {
  switch (c) {
    case "risk":
      return "风险";
    case "anomaly":
      return "异常";
    case "opportunity":
      return "机会";
    case "data":
      return "数据";
    case "knowledge":
      return "经验";
    default:
      return c;
  }
}

// Insights 进化建议卡（D-044 L0）：系统依据数据自动标记的「待核实证据候选」，
// 供决策层参考——只读证据表面，永不自动执行（action 一律指向 D# 人工决策）。
export function Insights({ insights }: { insights: Insight[] }) {
  return (
    <section className="card" aria-labelledby="insights-title">
      <h2 id="insights-title">进化建议</h2>
      <p className="insights-sub">系统依据数据自动标记的待核实项，供决策层参考，不自动执行</p>
      {insights.length === 0 ? (
        <p className="empty">暂无建议（系统运行正常 / 数据不足）</p>
      ) : (
        <ul className="insights">
          {insights.map((it) => (
            <li key={it.id} className="insight-row">
              <div className="insight-head">
                <LevelChip level={it.severity} />
                <span className="insight-cat">{catLabel(it.category)}</span>
                <span className="insight-title">{it.title}</span>
                <time className="insight-ts">{fmtTs(it.at)}</time>
              </div>
              <p className="insight-detail">{it.detail}</p>
              {it.actions.length > 0 ? (
                <ul className="insight-actions">
                  {it.actions.map((a, i) => (
                    <li key={i}>{a}</li>
                  ))}
                </ul>
              ) : null}
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
