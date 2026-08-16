import { useState } from "react";
import { fmtAmount } from "../format";
import type {
  CashFlow,
  CloseSimOrderResponse,
  GetSimAccountResponse,
  GetSimReportResponse,
  SimPosition,
  TestnetAccount,
} from "../gen/arbcn/sim/v1/sim_pb";
import { kindText, legSideText, SimTag, SimulatedBadge } from "./sim";

// flowKindText 现金流类型 → 中文（D-056 逐笔流水表）。
function flowKindText(kind: string): string {
  switch (kind) {
    case "capital_in":
      return "入金";
    case "open":
      return "开仓";
    case "funding":
      return "资金费";
    case "close":
      return "平仓";
    default:
      return kind;
  }
}

// PositionZone 模拟持仓（对话 #57 需求 3 实时数值 + D-055 平仓）：
//   - pnl（USD 模拟）= 每 8h 已结算累计；pnl_rmb = 即期折算（汇率缺失标注 USD 原值）。
//   - 开仓价 = ref_price；当前价 = ticker 实时（查不到标 —）。
//   - 预期年化 = 当前 funding 年化%（仅生息腿；现货腿/查不到标 —）。
//   - 浮动收益 = 未实现浮动（当前价 − 开仓价）× 数量 × 方向；当前价缺失标 —（不编造）。
//   - 实时收益 = 已结算 pnl + 未实现浮动（funding_hedge 两腿对冲浮动≈0，主要体现资金费）。
//   - 平仓按钮 = 订单级（D-019 整单平：rowSpan 跨该订单全部腿，绝不单腿留裸敞口）。
//     二次点击确认（同 ConfirmPanel 模式）；成功后横幅显示实现净 PnL。
function PositionZone({
  positions,
  fxAvailable,
  close,
}: {
  positions: SimPosition[];
  fxAvailable: boolean;
  close: (id: bigint, note?: string) => Promise<CloseSimOrderResponse | null>;
}) {
  const open = positions.filter((p) => p.status === "open"); // 持仓 = 当前持有（settled 已平，不进持仓表）
  // 平仓按钮跨订单全部腿（整单平）：rowSpan = 该订单 open 腿数。
  const legsPerOrder = new Map<bigint, number>();
  for (const p of open) {
    legsPerOrder.set(p.orderId, (legsPerOrder.get(p.orderId) ?? 0) + 1);
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
    const res = await close(id);
    setBusy(false);
    setResult(res);
  };

  const renderedOrder = new Set<bigint>();
  return (
    <section className="card" aria-labelledby="sim-positions-title">
      <h2 id="sim-positions-title">模拟持仓 <SimTag /></h2>
      <p className="muted">
        浮动收益 = 未实现（当前价 − 开仓价）；实时收益 = 已结算 PnL + 浮动；pnl_rmb 按即期 USDCNH 折算
        {fxAvailable ? "" : "（当前汇率不可用，标注 USD 原值）"}。
      </p>
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
      {open.length === 0 ? (
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
                <th scope="col">开仓价</th>
                <th scope="col">当前价</th>
                <th scope="col">预期年化</th>
                <th scope="col">已结算 PnL</th>
                <th scope="col">浮动收益</th>
                <th scope="col">实时收益（USD）</th>
                <th scope="col">PnL（RMB 即期）</th>
                <th scope="col">资金费</th>
                <th scope="col">操作</th>
              </tr>
            </thead>
            <tbody>
              {open.map((p) => {
                const first = !renderedOrder.has(p.orderId);
                renderedOrder.add(p.orderId);
                return (
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
                    <td className="num">{fmtAmount(p.refPrice)}</td>
                    <td className={p.curPrice === 0 ? "muted" : "num"}>
                      {p.curPrice === 0 ? "—" : fmtAmount(p.curPrice)}
                    </td>
                    <td className={p.expectedAnn > 0 ? "num" : "muted"}>
                      {p.expectedAnn > 0 ? `${p.expectedAnn.toFixed(2)}%` : "—"}
                    </td>
                    <td className="num">{fmtAmount(p.pnl)}</td>
                    <td className={p.curPrice === 0 ? "muted" : "num"}>
                      {p.curPrice === 0 ? "—" : fmtAmount(p.unrealizedPnl)}
                    </td>
                    <td className="num">{fmtAmount(p.pnl + p.unrealizedPnl)}</td>
                    <td className={fxAvailable ? "num" : "muted"}>
                      {fxAvailable ? fmtAmount(p.pnlRmb) : "USD 原值"}
                    </td>
                    <td>{p.funding ? <span className="sim-tag">生息腿</span> : "—"}</td>
                    <td>
                      {first ? (
                        <button
                          type="button"
                          className={armId === p.orderId ? "btn-close armed" : "btn-close"}
                          onClick={() => void onClose(p.orderId)}
                          disabled={busy}
                        >
                          {armId === p.orderId ? "确认平仓？" : "平仓"}
                        </button>
                      ) : null}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
      {busy ? <p className="muted">平仓处理中…</p> : null}
    </section>
  );
}

// AccountZone 模拟账户「按真实账户对待」卡（D-056 完整现金账本）：
// 五数对账（初始本金 / 现金余额 / 持仓市值 / 已实现 / 未实现）+ 净值（USD + RMB 即期）
// + 逐笔现金流流水。净值双恒等式供交叉校验：
//   equity = cash + market_value = capital + realized + unrealized
// 前端同时展示两式口径，防止单边账目偏差被隐藏（P3 可检查性）。账户区置顶
// PositionZone 之前——账户是模拟盘「第一眼」数据（本金是否还在、现金流是否对得上）。
function AccountZone({
  account,
  fxAvailable,
}: {
  account: GetSimAccountResponse;
  fxAvailable: boolean;
}) {
  const fmt = (v: number): string => fmtAmount(v);
  const flow = (f: CashFlow): string => {
    const d = new Date(Number(f.tsMs));
    return Number.isNaN(d.getTime()) ? "—" : d.toLocaleString("zh-CN", { hour12: false });
  };
  return (
    <section className="card" aria-labelledby="sim-account-title">
      <h2 id="sim-account-title">
        模拟账户 <SimTag />
      </h2>
      <p className="muted">
        净值对账双恒等式：cash + 持仓市值 = capital + 已实现 + 未实现（D-056；互相校验，
        单边账目偏差立即暴露）
      </p>
      <div className="account-grid">
        <div className="account-metric">
          <span className="muted">初始本金</span>
          <strong className="num">{fmt(account.capital)}</strong>
        </div>
        <div className="account-metric">
          <span className="muted">现金余额</span>
          <strong className="num">{fmt(account.cash)}</strong>
        </div>
        <div className="account-metric">
          <span className="muted">持仓市值</span>
          <strong className="num">{fmt(account.marketValue)}</strong>
        </div>
        <div className="account-metric">
          <span className="muted">已实现 PnL</span>
          <strong className="num">{fmt(account.realizedPnl)}</strong>
        </div>
        <div className="account-metric">
          <span className="muted">未实现浮动</span>
          <strong className="num">{fmt(account.unrealizedPnl)}</strong>
        </div>
        <div className="account-metric account-metric-total">
          <span className="muted">净值（USD）</span>
          <strong className="num">{fmt(account.equity)}</strong>
        </div>
        <div className="account-metric">
          <span className="muted">净值（RMB 即期）</span>
          <strong className="num">
            {fxAvailable ? fmt(account.equityRmb) : "—"}
          </strong>
        </div>
      </div>
      <h3>逐笔现金流</h3>
      {account.flows.length === 0 ? (
        <p className="empty">暂无流水（首次入金后开始记账）</p>
      ) : (
        <div className="table-scroll">
          <table className="rows">
            <thead>
              <tr>
                <th scope="col">时间</th>
                <th scope="col">类型</th>
                <th scope="col">金额（USD）</th>
                <th scope="col">订单</th>
                <th scope="col">备注</th>
              </tr>
            </thead>
            <tbody>
              {account.flows.map((f) => (
                <tr key={f.id.toString()}>
                  <td>{flow(f)}</td>
                  <td>{flowKindText(f.kind)}</td>
                  <td className={f.amount >= 0 ? "num flow-pos" : "num flow-neg"}>
                    {f.amount >= 0 ? "+" : ""}
                    {fmt(f.amount)}
                  </td>
                  <td className={f.orderId === 0n ? "muted" : "num"}>
                    {f.orderId === 0n ? "—" : `#${f.orderId}`}
                  </td>
                  <td className="note-cell">{f.note || "—"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}

// AccountCard 单账户快照卡（D-040 测试网账户区）。equity_usd 口径按 source 明示
// （诚实标注）：binance = 稳定币合计近似（非全量净值，非稳定币无行情折算标 —）；
// okx = totalEq（交易所精确折算）。
function AccountCard({ a }: { a: TestnetAccount }) {
  const isBinance = a.source === "sim_testnet_binance";
  const name = isBinance ? "Binance Testnet" : "OKX Demo";
  const eqLabel = isBinance
    ? "权益 USD（稳定币合计近似 · 非全量净值）"
    : "权益 USD（totalEq 精确）";
  const updated = new Date(Number(a.updatedAtMs));
  return (
    <div className="testnet-account">
      <h3>
        {name} <SimTag />
        {a.accountAlias ? <span className="muted">· {a.accountAlias}</span> : null}
      </h3>
      <p className="account-equity">
        <span className="muted">{eqLabel}：</span>
        <strong className="num">{fmtAmount(a.equityUsd)}</strong>
      </p>
      <p className="muted">
        更新：{Number.isNaN(updated.getTime()) ? "—" : updated.toLocaleString()}
      </p>
      {a.details.length === 0 ? (
        <p className="empty">暂无余额明细</p>
      ) : (
        <table className="rows">
          <thead>
            <tr>
              <th scope="col">资产</th>
              <th scope="col">余额</th>
              <th scope="col">USD 折算</th>
            </tr>
          </thead>
          <tbody>
            {a.details.map((d) => (
              <tr key={d.asset}>
                <th scope="row">{d.asset}</th>
                <td className="num">{d.balance || "—"}</td>
                <td className={d.equityUsd === 0 ? "muted" : "num"}>
                  {d.equityUsd === 0 ? "—" : fmtAmount(d.equityUsd)}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}
    </div>
  );
}

// TestnetAccountZone 测试网账户区（D-040）：两路 testnet 模拟资金 + 账户信息。
// 无数据 = 探针未启用 / 首次余额查询未完成（诚实空态，不编造）。
function TestnetAccountZone({ accounts }: { accounts: TestnetAccount[] }) {
  return (
    <section className="card" aria-labelledby="sim-accounts-title">
      <h2 id="sim-accounts-title">
        测试网账户 <SimTag />
      </h2>
      {accounts.length === 0 ? (
        <p className="empty">暂无测试网账户数据（探针未启用或首次余额查询未完成，8h tick 自动刷新）</p>
      ) : (
        <div className="testnet-accounts">
          {accounts.map((a) => (
            <AccountCard key={a.source} a={a} />
          ))}
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

// SimExec 模拟执行面板（04-m3-spec §10.5 C4）：模拟持仓 + 测试网账户 + 对账报告。
// 确认下单已移监控总览页 ConfirmPanel（对话 #59）；数据经 App 层共享 useSim 传入
// （与 ConfirmPanel 同源，确认后两处同刷新）。SIMULATED 徽标渲染自共享 sim.tsx。
// 无任何通往真实资金的按钮/路径（§6/§8，不赌原则 D-019）。
export function SimExec({
  positions,
  accounts,
  account,
  report,
  fxAvailable,
  error,
  close,
  reload,
}: {
  positions: SimPosition[];
  accounts: TestnetAccount[];
  account: GetSimAccountResponse | null;
  report: GetSimReportResponse | null;
  fxAvailable: boolean;
  error: string;
  close: (id: bigint, note?: string) => Promise<CloseSimOrderResponse | null>;
  reload: () => void;
}) {
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
      {account ? <AccountZone account={account} fxAvailable={fxAvailable} /> : null}
      <PositionZone positions={positions} fxAvailable={fxAvailable} close={close} />
      <TestnetAccountZone accounts={accounts} />
      <ReportZone
        markdown={report?.markdown ?? ""}
        exists={report?.exists ?? false}
        note={report?.note ?? ""}
      />
    </div>
  );
}
