import { fmtAmount } from "../format";
import type {
  GetSimReportResponse,
  SimPosition,
  TestnetAccount,
} from "../gen/arbcn/sim/v1/sim_pb";
import { kindText, legSideText, SimTag, SimulatedBadge } from "./sim";

// PositionZone 模拟持仓（对话 #57 需求 3 实时数值）：
//   - pnl（USD 模拟）= 每 8h 已结算累计；pnl_rmb = 即期折算（汇率缺失标注 USD 原值）。
//   - 开仓价 = ref_price；当前价 = ticker 实时（查不到标 —）。
//   - 预期年化 = 当前 funding 年化%（仅生息腿；现货腿/查不到标 —）。
//   - 实时收益 = 已结算 pnl + 未实现浮动（funding_hedge 两腿对冲浮动≈0，主要体现资金费）。
// funding 生息腿明示。
function PositionZone({ positions, fxAvailable }: { positions: SimPosition[]; fxAvailable: boolean }) {
  return (
    <section className="card" aria-labelledby="sim-positions-title">
      <h2 id="sim-positions-title">模拟持仓 <SimTag /></h2>
      <p className="muted">
        实时收益 = 已结算 PnL + 未实现浮动（当前价 − 开仓价）；pnl_rmb 按即期 USDCNH 折算
        {fxAvailable ? "" : "（当前汇率不可用，标注 USD 原值）"}。
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
                <th scope="col">开仓价</th>
                <th scope="col">当前价</th>
                <th scope="col">预期年化</th>
                <th scope="col">已结算 PnL</th>
                <th scope="col">实时收益（USD）</th>
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
                  <td className="num">{fmtAmount(p.refPrice)}</td>
                  <td className={p.curPrice === 0 ? "muted" : "num"}>
                    {p.curPrice === 0 ? "—" : fmtAmount(p.curPrice)}
                  </td>
                  <td className={p.expectedAnn > 0 ? "num" : "muted"}>
                    {p.expectedAnn > 0 ? `${p.expectedAnn.toFixed(2)}%` : "—"}
                  </td>
                  <td className="num">{fmtAmount(p.pnl)}</td>
                  <td className="num">{fmtAmount(p.pnl + p.unrealizedPnl)}</td>
                  <td className={fxAvailable ? "num" : "muted"}>
                    {fxAvailable ? fmtAmount(p.pnlRmb) : "USD 原值"}
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
  report,
  fxAvailable,
  error,
  reload,
}: {
  positions: SimPosition[];
  accounts: TestnetAccount[];
  report: GetSimReportResponse | null;
  fxAvailable: boolean;
  error: string;
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
      <PositionZone positions={positions} fxAvailable={fxAvailable} />
      <TestnetAccountZone accounts={accounts} />
      <ReportZone
        markdown={report?.markdown ?? ""}
        exists={report?.exists ?? false}
        note={report?.note ?? ""}
      />
    </div>
  );
}
