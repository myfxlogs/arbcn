// OrderHistoryZone 订单历史（对话 #81 业主需求：第二门槛拦下的订单要有记录）。
// 全量 sim_orders（ts DESC，含 suggested/confirmed/filled/rejected/expired/closed
// 六态）；rejected 拒单负样本保留 note（拒单原因），供复盘门禁松紧。只读呈现，
// 拒单永不自动重试（二次门禁是硬拦截，人工确认才可能再评估）。
// 独立文件（D-059）：SimExec.tsx 曾 485 行触发 pre-commit >450 硬阻断，拆出此组件。
import { fmtAmount } from "../format";
import type { SimOrder } from "../gen/arbcn/sim/v1/sim_pb";
import { Chip, type ChipTone } from "./Chip";
import { kindText, sideText, SimTag, statusText } from "./sim";

// statusTone 订单状态 → 徽标色：拒单=红（负样本醒目）、成交/平仓=绿、
// 待确认/已确认=黄、过期=中性。practices #18 对照后端六态全集。
function statusTone(status: string): ChipTone {
  switch (status) {
    case "rejected":
      return "critical";
    case "filled":
    case "closed":
      return "good";
    case "suggested":
    case "confirmed":
      return "warn";
    default:
      return "neutral"; // expired / 未知
  }
}

export function OrderHistoryZone({ orders }: { orders: SimOrder[] }) {
  const ts = (ms: bigint): string => {
    const d = new Date(Number(ms));
    return Number.isNaN(d.getTime()) ? "—" : d.toLocaleString("zh-CN", { hour12: false });
  };
  return (
    <section className="card" aria-labelledby="sim-orders-title">
      <h2 id="sim-orders-title">
        订单历史 <SimTag />
      </h2>
      <p className="muted">
        全量订单（含门禁拒单负样本：rejected 保留拒单原因，供复盘门禁是否过严/过松）
      </p>
      {orders.length === 0 ? (
        <p className="empty">暂无订单（策略生成建议后出现在此）</p>
      ) : (
        <div className="table-scroll scroll-cap">
          <table className="rows">
            <thead>
              <tr>
                <th scope="col">时间</th>
                <th scope="col">状态</th>
                <th scope="col">类型</th>
                <th scope="col">标的</th>
                <th scope="col">方向</th>
                <th scope="col">数量</th>
                <th scope="col">预期</th>
                <th scope="col">备注</th>
              </tr>
            </thead>
            <tbody>
              {orders.map((o) => (
                <tr key={o.id.toString()}>
                  <td className="muted">{ts(o.tsMs)}</td>
                  <td>
                    <Chip tone={statusTone(o.status)}>{statusText(o.status)}</Chip>
                  </td>
                  <td>{kindText(o.kind)}</td>
                  <td>
                    {o.symbol}
                    <span className="muted">@{o.venue}</span>
                  </td>
                  <td>{sideText(o.side)}</td>
                  <td className="num">{fmtAmount(o.qty)}</td>
                  <td className="num">{o.expectedSpread.toFixed(2)}%</td>
                  <td className="note-cell">{o.note || "—"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}
