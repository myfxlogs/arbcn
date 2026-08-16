// ClosePanel 平仓卡（对话 #81 业主需求：市场结构与确认下单之间的空白加平仓卡）。
// 总览右列紧凑持仓概览 + 整单平（D-019：订单全部腿一起退，绝不单腿留裸敞口）。
// 与 SimExec 持仓表同源（useSim.positions）；每订单一行——标的+生息标记合并、
// 数量、实时收益合计；平仓按钮二次点击确认防误点（practices #17，同 ConfirmPanel）。
// 确认后仍是模拟（SIMULATED），不接真实资金（§1 资金动作永远人工）。
import { useState } from "react";

import { fmtAmount } from "../format";
import type { CloseSimOrderResponse, SimPosition } from "../gen/arbcn/sim/v1/sim_pb";
import { SimTag, SimulatedBadge } from "./sim";

// ClosePanel 平仓卡（总览页）。positions/close 来自 App 层共享 useSim（与模拟执行 tab
// 同源，平仓成功后两处同刷新）。
export function ClosePanel({
  positions,
  fxAvailable,
  close,
  error,
  reload,
}: {
  positions: SimPosition[];
  fxAvailable: boolean;
  close: (id: bigint, note?: string) => Promise<CloseSimOrderResponse | null>;
  error: string;
  reload: () => void;
}) {
  const open = positions.filter((p) => p.status === "open"); // 持仓 = 当前持有（settled 已平不进）
  // 订单级合并：全部 open 腿（rowSpan 语义同 SimExec 持仓表，D-019 整单平）。
  const orders = new Map<bigint, SimPosition[]>();
  for (const p of open) {
    const legs = orders.get(p.orderId);
    if (legs) {
      legs.push(p);
    } else {
      orders.set(p.orderId, [p]);
    }
  }
  const [armId, setArmId] = useState<bigint | null>(null); // 二次确认：首次点击进入待确认态
  const [busy, setBusy] = useState(false);
  const [result, setResult] = useState<CloseSimOrderResponse | null>(null);

  const onClose = async (id: bigint) => {
    if (armId !== id) {
      setArmId(id);
      setResult(null);
      return;
    }
    setArmId(null);
    setBusy(true);
    setResult(null);
    setResult(await close(id));
    setBusy(false);
  };

  return (
    <section className="card" aria-labelledby="close-panel-title">
      <div className="sim-head">
        <h2 id="close-panel-title">平仓</h2>
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
        <div className="banner sim-result" role="status">
          <span>
            订单 {result.orderId.toString()} 已整单平仓：实现净 PnL {fmtAmount(result.realizedPnl)} USD
            {fxAvailable ? `（≈ ${fmtAmount(result.realizedRmb)} RMB）` : "（汇率不可用，标注 USD 原值）"}
            ，平 {result.closedLegs} 腿
          </span>
          <button
            type="button"
            className="banner-close"
            aria-label="关闭提示"
            title="关闭"
            onClick={() => setResult(null)}
          >
            ×
          </button>
        </div>
      ) : null}
      {orders.size === 0 ? (
        <p className="empty">暂无持仓（确认下单后这里出现平仓入口）</p>
      ) : (
        // 对话 #81：列表高度封顶卡内滚动——持仓多时不挤占下方确认下单卡。
        <ul className="close-list scroll-cap">
          {[...orders.entries()].map(([id, legs]) => {
            const qty = legs.reduce((s, p) => s + p.qty, 0);
            // 实时收益 = 已结算 PnL + 未实现浮动（口径同 SimExec 持仓表「实时收益」列）。
            const realtime = legs.reduce((s, p) => s + p.pnl + p.unrealizedPnl, 0);
            const funding = legs.some((p) => p.funding); // 含生息腿 → 标「模拟」资金费
            return (
              <li key={id.toString()}>
                <span className="pending-summary">
                  <span className="close-symbols">{legs.map((p) => p.symbol).join(" + ")}</span>
                  <span className="muted">
                    @{legs.map((p) => p.venue).join("/")}
                  </span>
                  {funding ? <SimTag /> : null}
                  <span className="num">{fmtAmount(qty)}U</span>
                  <span className={realtime >= 0 ? "close-pnl pos" : "close-pnl neg"}>
                    实时 {realtime >= 0 ? "+" : ""}
                    {fmtAmount(realtime)}
                  </span>
                </span>
                <button
                  type="button"
                  className={armId === id ? "btn-close armed" : "btn-close"}
                  onClick={() => void onClose(id)}
                  disabled={busy}
                >
                  {armId === id ? "确认平仓？" : "平仓"}
                </button>
              </li>
            );
          })}
        </ul>
      )}
      {busy ? <p className="muted">平仓处理中…</p> : null}
    </section>
  );
}
