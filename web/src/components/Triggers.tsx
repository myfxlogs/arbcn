import { fmtTs, num } from "../format";
import type { TriggerState } from "../gen/arbcn/dashboard/v1/dashboard_pb";
import { StateChip } from "./Chip";

// Triggers 触发器视图：各规则状态机 + 最近转变。
export function Triggers({ states }: { states: TriggerState[] }) {
  return (
    <section className="card" aria-labelledby="trig-title">
      <h2 id="trig-title">触发器</h2>
      {states.length === 0 ? (
        <p className="empty">暂无规则</p>
      ) : (
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
                  <th scope="row">{s.ruleName}</th>
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
      )}
    </section>
  );
}
