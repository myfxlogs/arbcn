// ConfirmPanel 确认下单面板（对话 #59）：监控总览页只显示「待确认」订单 + 人工确认按钮
// （资金动作永远人工，§1）。确认后仍是模拟（SIMULATED），不接真实资金（D-019 不赌原则）。
// 拒单负样本 / 已成交历史在「模拟执行」tab 查看。
// 对话 #74 业主删减：每单**一行摘要**（对冲对合成一行），删 类型/方向/数量 独立列——
// funding_hedge 单 side 恒为「对冲」与类型列重复，行内 SimTag 与标题 SimulatedBadge 重复，
// 均冗余删除（锚点 TestSimExecBadgeRenderable 仅要求 SimulatedBadge 引用，已保留）。
// 二次点击确认防误点（practices #17）：disabled 绑 busy，不绑 pending。
import { useState } from "react";

import { fmtAmount } from "../format";
import type { SimOrder } from "../gen/arbcn/sim/v1/sim_pb";
import { sideText, SimulatedBadge } from "./sim";

// ConfirmRow 单条待确认订单单行摘要（对话 #74）：
//   BTC @okx · 对冲 20,000U · 预期 6.61%           [确认成交]
// side 用色标小标签，数量千分位 + U，预期年化是决策核心保留。
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
    <li>
      <span className="pending-summary">
        {o.symbol}
        <span className="muted">@{o.venue}</span>
        <span className="dir-tag">{sideText(o.side)}</span>
        <span className="num">{fmtAmount(o.qty)}U</span>
        <span className="spread">预期 {o.expectedSpread.toFixed(2)}%</span>
      </span>
      <button
        type="button"
        className="icon"
        disabled={busy}
        onClick={() => onConfirm(o.id)}
      >
        {pending === o.id ? "再次点击确认？" : "确认成交"}
      </button>
    </li>
  );
}

// ConfirmPanel 确认下单面板（总览页）。orders 来自 App 层共享 useSim
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
        <h2 id="confirm-panel-title">确认下单</h2>
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
        <ul className="pending-list">
          {pendingList.map((o) => (
            <ConfirmRow key={o.id.toString()} o={o} pending={pending} busy={busy} onConfirm={onConfirm} />
          ))}
        </ul>
      )}
      {busy ? <p className="muted">确认处理中…</p> : null}
    </section>
  );
}
