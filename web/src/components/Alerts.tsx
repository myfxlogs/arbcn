import { fmtTs, ruleLabel } from "../format";
import type { Alert } from "../gen/arbcn/dashboard/v1/dashboard_pb";
import { Chip, LevelChip } from "./Chip";

// Alerts 告警流：时间线（服务端已降序）+ 单条确认。
export function Alerts({
  alerts,
  ackBusy,
  onAck,
}: {
  alerts: Alert[];
  ackBusy: ReadonlySet<string>;
  onAck: (id: bigint) => void;
}) {
  return (
    <section className="card" aria-labelledby="alerts-title">
      <h2 id="alerts-title">告警流</h2>
      {alerts.length === 0 ? (
        <p className="empty">暂无告警</p>
      ) : (
        <ol className="timeline">
          {alerts.map((a) => (
            <li key={a.id.toString()}>
              <div className="timeline-head">
                <LevelChip level={a.level} />
                <span className="timeline-rule">{ruleLabel(a.ruleName)}</span>
                <time className="timeline-ts">{fmtTs(a.ts)}</time>
                {a.acked ? (
                  <Chip tone="good">已确认</Chip>
                ) : (
                  <button
                    type="button"
                    className="ack"
                    disabled={ackBusy.has(a.id.toString())}
                    onClick={() => onAck(a.id)}
                  >
                    确认
                  </button>
                )}
              </div>
              <p className="timeline-msg">{a.message}</p>
            </li>
          ))}
        </ol>
      )}
    </section>
  );
}
