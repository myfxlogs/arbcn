// ConfirmPanel 确认下单面板（对话 #59）：监控总览页「告警流」卡片下方，只显示
// 「待确认」订单 + 人工确认按钮（资金动作永远人工，§1）。确认后仍是模拟（SIMULATED），
// 不接真实资金（D-019 不赌原则）。拒单负样本 / 已成交历史在「模拟执行」tab 查看。
// 二次点击确认防误点（practices #17）：disabled 绑 busy，不绑 pending。
import { useState } from "react";

import { fmtAmount } from "../format";
import type { SimOrder } from "../gen/arbcn/sim/v1/sim_pb";
import { kindText, sideText, SimTag, SimulatedBadge } from "./sim";

// ConfirmRow 单条待确认订单：确认按钮仅 suggested 渲染（suggested 订单 riskFlags 恒空，
// 有标记即 rejected，故简化不显示风险/状态/备注列——业主「简化一些」决策）。
function ConfirmRow({
  o,
  pending,
  busy,
  onConfirm,
}: {
  o: SimOrder;
  pending: bigint | null;
  busy: boolean;
  onConfirm: (id: bigint) => void;
}) {
  return (
    <tr>
      <th scope="row">
        {kindText(o.kind)} <SimTag />
      </th>
      <td>
        {o.symbol}
        <span className="muted">@{o.venue}</span>
      </td>
      <td>{sideText(o.side)}</td>
      <td className="num">{fmtAmount(o.qty)}</td>
      <td className="num">{o.expectedSpread.toFixed(2)}%</td>
      <td>
        <button
          type="button"
          className="icon"
          disabled={busy}
          onClick={() => onConfirm(o.id)}
        >
          {pending === o.id ? "再次点击确认？" : "确认成交"}
        </button>
      </td>
    </tr>
  );
}

// ConfirmPanel 确认下单面板（总览页告警流下方）。orders 来自 App 层共享 useSim
// （与模拟执行 tab 同源，确认成功后两处同刷新）。
export function ConfirmPanel({
  orders,
  confirm,
  error,
  reload,
}: {
  orders: SimOrder[];
  confirm: (id: bigint) => Promise<boolean>;
  error: string;
  reload: () => void;
}) {
  const pendingList = orders.filter((o) => o.status === "suggested");
  const [pending, setPending] = useState<bigint | null>(null);
  const [busy, setBusy] = useState(false);
  const [result, setResult] = useState("");

  // 二次点击确认：首次点击进入待确认态，再次点击真正确认（防误点）。
  const onConfirm = async (id: bigint) => {
    if (pending !== id) {
      setPending(id);
      setResult("");
      return;
    }
    setPending(null);
    setBusy(true);
    setResult("");
    const accepted = await confirm(id);
    setBusy(false);
    setResult(accepted ? `订单 ${id} 已本地模拟成交` : `订单 ${id} 拒单（二次门禁未过，已存负样本）`);
  };

  return (
    <section className="card" aria-labelledby="confirm-panel-title">
      <div className="sim-head">
        <h2 id="confirm-panel-title">确认下单 <SimTag /></h2>
        <SimulatedBadge />
        <button type="button" className="icon" onClick={reload}>
          刷新
        </button>
      </div>
      {error ? (
        <div className="banner" role="alert">
          模拟执行加载失败：{error}
        </div>
      ) : null}
      {result ? (
        <div className="banner sim-result">
          <span>{result}</span>
          <button
            type="button"
            className="banner-close"
            aria-label="关闭提示"
            title="关闭"
            onClick={() => setResult("")}
          >
            ×
          </button>
        </div>
      ) : null}
      {pendingList.length === 0 ? (
        <p className="empty">暂无待确认订单（funding 窗口档低水位时常态）</p>
      ) : (
        <div className="table-scroll">
          <table className="rows">
            <thead>
              <tr>
                <th scope="col">类型</th>
                <th scope="col">标的</th>
                <th scope="col">方向</th>
                <th scope="col">数量</th>
                <th scope="col">预期年化</th>
                <th scope="col">操作</th>
              </tr>
            </thead>
            <tbody>
              {pendingList.map((o) => (
                <ConfirmRow key={o.id.toString()} o={o} pending={pending} busy={busy} onConfirm={onConfirm} />
              ))}
            </tbody>
          </table>
        </div>
      )}
      {busy ? <p className="muted">确认处理中…</p> : null}
    </section>
  );
}
