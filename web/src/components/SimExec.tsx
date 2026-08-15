import { Fragment, useState } from "react";

import { fmtAmount } from "../format";
import type { SimOrder, SimPosition } from "../gen/arbcn/sim/v1/sim_pb";
import { useSim } from "../hooks";

// —— 常量（C5 可检查锚点：SimExec.tsx 含固定 SIMULATED / 「模拟」渲染，grep 断言）——
const SIMULATED = "SIMULATED";
const SIMULATED_CN = "模拟";

// riskLabel 风险门禁标记 → 中文徽标文案（未知名回退原名，演进预留）。
function riskLabel(flag: string): string {
  switch (flag) {
    case "SPREAD_DRIFT":
      return "漂移";
    case "WHITELIST":
      return "未白名单";
    default:
      return flag;
  }
}

// statusText 订单状态 → 中文。
function statusText(status: string): string {
  switch (status) {
    case "suggested":
      return "待确认";
    case "confirmed":
      return "已确认";
    case "filled":
      return "已成交";
    case "rejected":
      return "已拒单";
    case "expired":
      return "已过期";
    default:
      return status;
  }
}

// kindText 套利类型 → 中文。
function kindText(kind: string): string {
  switch (kind) {
    case "funding_hedge":
      return "现货+永续对冲";
    case "carry_asset":
      return "白名单生息";
    case "repo":
      return "逆回购";
    default:
      return kind;
  }
}

// sideText 订单方向 → 中文。
function sideText(side: string): string {
  switch (side) {
    case "long":
      return "多";
    case "short":
      return "空";
    case "hedge":
      return "对冲";
    default:
      return side;
  }
}

// legSideText 持仓腿方向 → 中文。
function legSideText(side: string): string {
  switch (side) {
    case "long":
      return "现货多";
    case "short":
      return "永续空";
    default:
      return side;
  }
}

// simTag 每行固定「模拟」标注（C5 可检查锚点）。
function SimTag() {
  return <span className="sim-tag">{SIMULATED_CN}</span>;
}

// SimulatedBadge tab 顶部固定徽标（C5 可检查锚点：SIMULATED 常量固定渲染）。
function SimulatedBadge() {
  return (
    <span className="sim-badge" title="本地模拟成交，不接真实资金">
      {SIMULATED} · 不接真实资金
    </span>
  );
}

// RiskFlags 风险门禁徽标列表（拒单负样本原因明示；SPREAD_DRIFT → 「漂移」）。
function RiskFlags({ flags }: { flags: string[] }) {
  if (flags.length === 0) return <span className="muted">—</span>;
  return (
    <span className="risk-flags">
      {flags.map((f) => (
        <span key={f} className="risk-badge" title={f}>
          {riskLabel(f)}
        </span>
      ))}
    </span>
  );
}

// OrderRow 建议订单行：确认按钮仅 suggested 渲染（防误点 + 二次点击确认）。
function OrderRow({
  o,
  pending,
  onConfirm,
}: {
  o: SimOrder;
  pending: bigint | null;
  onConfirm: (id: bigint) => void;
}) {
  const canConfirm = o.status === "suggested";
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
        <RiskFlags flags={o.riskFlags} />
      </td>
      <td>{statusText(o.status)}</td>
      <td className="note-cell">{o.note || "—"}</td>
      <td>
        {canConfirm ? (
          <button
            type="button"
            className="icon"
            disabled={pending === o.id}
            onClick={() => onConfirm(o.id)}
          >
            {pending === o.id ? "再次点击确认？" : "确认成交"}
          </button>
        ) : (
          "—"
        )}
      </td>
    </tr>
  );
}

// OrderZone 建议订单列表：待确认 / 拒单负样本 / 已成交 分组。
function OrderZone({ orders, pending, onConfirm }: { orders: SimOrder[]; pending: bigint | null; onConfirm: (id: bigint) => void }) {
  const groups: { key: string; title: string; list: SimOrder[] }[] = [
    { key: "suggested", title: "待确认", list: orders.filter((o) => o.status === "suggested") },
    { key: "rejected", title: "拒单负样本", list: orders.filter((o) => o.status === "rejected") },
    { key: "filled", title: "已成交", list: orders.filter((o) => o.status === "filled" || o.status === "confirmed") },
  ];
  return (
    <section className="card" aria-labelledby="sim-orders-title">
      <h2 id="sim-orders-title">建议订单（模拟）</h2>
      {orders.length === 0 ? (
        <p className="empty">暂无建议订单</p>
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
                <th scope="col">风险标记</th>
                <th scope="col">状态</th>
                <th scope="col">备注</th>
                <th scope="col">操作</th>
              </tr>
            </thead>
            <tbody>
              {groups.map((g) => (
                <Fragment key={g.key}>
                  <tr className="sim-group">
                    <th colSpan={9} scope="colgroup">
                      {g.title}（{g.list.length}）
                    </th>
                  </tr>
                  {g.list.map((o) => (
                    <OrderRow key={o.id.toString()} o={o} pending={pending} onConfirm={onConfirm} />
                  ))}
                </Fragment>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}

// PositionZone 模拟持仓：pnl（USD 模拟）+ pnl_rmb（即期折算，汇率缺失标注 USD 原值）。
// PnL 为每 8h 结算的累计值（诚实标注），funding 生息腿明示。
function PositionZone({ positions }: { positions: SimPosition[] }) {
  const fxMissing = positions.length > 0 && positions.every((p) => p.pnlRmb === 0);
  return (
    <section className="card" aria-labelledby="sim-positions-title">
      <h2 id="sim-positions-title">模拟持仓 <SimTag /></h2>
      <p className="muted">
        PnL 为每 8h 结算的累计模拟 USD；pnl_rmb 按即期 USDCNH 折算
        {fxMissing ? "（当前汇率不可用，标注 USD 原值）" : ""}。
      </p>
      {positions.length === 0 ? (
        <p className="empty">暂无模拟持仓</p>
      ) : (
        <div className="table-scroll">
          <table className="rows">
            <thead>
              <tr>
                <th scope="col">类型</th>
                <th scope="col">标的</th>
                <th scope="col">腿方向</th>
                <th scope="col">数量</th>
                <th scope="col">PnL（USD 模拟）</th>
                <th scope="col">PnL（RMB 即期）</th>
                <th scope="col">资金费</th>
              </tr>
            </thead>
            <tbody>
              {positions.map((p) => (
                <tr key={p.id.toString()}>
                  <th scope="row">
                    {kindText(p.kind)} <SimTag />
                  </th>
                  <td>
                    {p.symbol}
                    <span className="muted">@{p.venue}</span>
                  </td>
                  <td>{legSideText(p.side)}</td>
                  <td className="num">{fmtAmount(p.qty)}</td>
                  <td className="num">{fmtAmount(p.pnl)}</td>
                  <td className={p.pnlRmb === 0 ? "muted" : "num"}>
                    {p.pnlRmb === 0 ? "USD 原值" : fmtAmount(p.pnlRmb)}
                  </td>
                  <td>{p.funding ? <span className="sim-tag">生息腿</span> : "—"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}

// ReportZone 对账报告入口：markdown 存在 → 渲染；否则显示生成周期说明。
function ReportZone({ markdown, exists, note }: { markdown: string; exists: boolean; note: string }) {
  return (
    <section className="card" aria-labelledby="sim-report-title">
      <h2 id="sim-report-title">对账报告（周频）</h2>
      {exists ? (
        <pre className="sim-report">{markdown}</pre>
      ) : (
        <p className="empty">{note || "周频报告每 7×8h tick 渲染（尚未生成）"}</p>
      )}
    </section>
  );
}

// SimExec 模拟执行面板（04-m3-spec §10.5 C4）。确认后仍是模拟（SIMULATED），
// 无任何通往真实资金的按钮/路径（§6/§8，不赌原则 D-019）。
export function SimExec() {
  const { orders, positions, report, error, confirm, reload } = useSim();
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
    <div className="sim-exec">
      <div className="sim-head">
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
      {result ? <div className="banner sim-result">{result}</div> : null}

      <OrderZone orders={orders} pending={pending} onConfirm={(id) => void onConfirm(id)} />
      <PositionZone positions={positions} />
      <ReportZone
        markdown={report?.markdown ?? ""}
        exists={report?.exists ?? false}
        note={report?.note ?? ""}
      />
      {busy ? <p className="muted">确认处理中…</p> : null}
    </div>
  );
}
