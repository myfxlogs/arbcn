import { fmtTs, num, ruleLabel } from "../format";
import type { TriggerState } from "../gen/arbcn/dashboard/v1/dashboard_pb";
import { StateChip } from "./Chip";
import { Collapse } from "./Collapse";

// Triggers 触发器视图（D-048 U4 第一眼原则）：正在响应的 active 规则是信号面，
// 置顶呈现；全量规则表（armed/resolved 含）折叠为下钻面，标题带总数。
export function Triggers({ states }: { states: TriggerState[] }) {
  const active = states.filter((s) => s.state === "active");
  return (
    <section className="card" aria-labelledby="trig-title">
      <h2 id="trig-title">触发器</h2>
      <h3>触发中的规则</h3>
      {active.length === 0 ? (
        <p className="empty">当前无触发中的规则</p>
      ) : (
        <ul className="insights">
          {active.map((s) => (
            <li key={s.ruleName} className="insight-row">
              <div className="insight-head">
                <StateChip state={s.state} />
                <span className="insight-title">{ruleLabel(s.ruleName)}</span>
                <span className="insight-ts">
                  最近值 {s.lastValue === undefined ? "—" : num(s.lastValue)}
                </span>
              </div>
            </li>
          ))}
        </ul>
      )}
      <Collapse title={`全部规则（${states.length} 条）`} hint="含 armed / resolved">
        <div className="table-scroll">
          <table className="rows">
            <thead>
              <tr>
                <th scope="col">规则</th>
                <th scope="col">状态</th>
                <th scope="col">最近转变</th>
                <th scope="col">最近值</th>
              </tr>
            </thead>
            <tbody>
              {states.map((s) => (
                <tr key={s.ruleName}>
                  <th scope="row">{ruleLabel(s.ruleName)}</th>
                  <td>
                    <StateChip state={s.state} />
                  </td>
                  <td>{fmtTs(s.since)}</td>
                  <td>{s.lastValue === undefined ? "—" : num(s.lastValue)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </Collapse>
    </section>
  );
}
