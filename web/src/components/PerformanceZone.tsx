import { fmtAmount } from "../format";
import type { GetPerformanceReportResponse } from "../gen/arbcn/sim/v1/sim_pb";
import { SimTag } from "./sim";

// gateMeta 判定门① 状态 → 徽标文案/样式（D-062 状态值域）。
const gateMeta: Record<string, { label: string; cls: string }> = {
  pending: { label: "PENDING · 前向验证中", cls: "gate-bg" },
  pass: { label: "PASS · 可进阶段 A", cls: "gate-ok" },
  watch: { label: "WATCH · 继续积累", cls: "gate-warn" },
  fail: { label: "FAIL · 止投信号", cls: "gate-bad" },
  env_no_window: { label: "ENV_NO_WINDOW · 环境无机会", cls: "gate-bg" },
  data_anomaly: { label: "DATA_ANOMALY · 数据异常", cls: "gate-bad" },
};

// PerformanceZone 阶段 0 判定门① 测量（D-062）：跨 30 天窗口 paper 收益 TWR/MWR +
// 判定门① 状态 + 环境条件（D-061 环境-策略分离）。诚实标注：
//   pending（窗口未满 30 天，不提前判定）/ env_no_window（零成交测环境非策略，
//   零单是正确输出，宁缺毋滥 D-019）。只测量不自动执行——进阶段 A 仍人工决策。
// D-063 可信度自检：快照覆盖/期望瓦片明示判定用数据面可信度（<90% 判定不采信）。
export function PerformanceZone({ perf }: { perf: GetPerformanceReportResponse }) {
  const meta = gateMeta[perf.status] ?? { label: perf.status, cls: "gate-bg" };
  const fmt = (v: number): string => fmtAmount(v);
  const pct = (v: number): string => `${v.toFixed(2)}%`;
  const noVal = perf.status === "pending" || perf.status === "env_no_window";
  return (
    <section className="card" aria-labelledby="sim-gate-title">
      <h2 id="sim-gate-title">
        判定门① 阶段 0 测量 <SimTag />
      </h2>
      <div className="gate-head">
        <span className={`gate-badge ${meta.cls}`}>{meta.label}</span>
        <span className="muted">
          窗口 {fmt(perf.windowDays)} 天 / 判定线 {pct(perf.gateThreshold)}（= 基线上限{" "}
          {pct(perf.baselineHigh)} + 摩擦 {pct(perf.frictionMargin)}）
        </span>
      </div>
      <p className="muted">{perf.statusNote}</p>
      <div className="account-grid">
        <div className="account-metric">
          <span className="muted">TWR 年化（时间加权）</span>
          <strong className="num">{noVal ? "—" : pct(perf.twrAnnualized)}</strong>
        </div>
        <div className="account-metric">
          <span className="muted">MWR 年化（资金加权）</span>
          <strong className="num">{noVal ? "—" : pct(perf.mwrAnnualized)}</strong>
        </div>
        <div className="account-metric">
          <span className="muted">窗口成交单</span>
          <strong className="num">{perf.orderCount}</strong>
        </div>
        <div className="account-metric">
          <span className="muted">拒单（负样本）</span>
          <strong className="num">{perf.rejectedCount}</strong>
        </div>
        <div className="account-metric">
          <span className="muted">funding 中位（年化）</span>
          <strong className="num">{perf.fundingMedian === 0 ? "—" : pct(perf.fundingMedian)}</strong>
        </div>
        <div className="account-metric">
          <span className="muted">funding max（年化）</span>
          <strong className="num">{perf.fundingMax === 0 ? "—" : pct(perf.fundingMax)}</strong>
        </div>
        <div className="account-metric">
          <span className="muted">≥15% 高费率时段</span>
          <strong className="num">{perf.highWindowEvents}</strong>
        </div>
        <div className="account-metric">
          <span className="muted">可交易面（监控内对）</span>
          <strong className="num">{perf.tradablePairs}</strong>
        </div>
        <div className="account-metric">
          <span className="muted">快照覆盖（判定可信度）</span>
          <strong className="num">
            {perf.expectedSnapshots === 0
              ? "—"
              : `${perf.snapshotCount}/${perf.expectedSnapshots} (${Math.round(perf.snapshotCoverage * 100)}%)`}
          </strong>
        </div>
      </div>
      <p className="muted">
        基线 {pct(perf.baselineLow)}–{pct(perf.baselineHigh)}（D-026 诚实基线）+ 摩擦{" "}
        {pct(perf.frictionMargin)}（D-046 普通主户双 taker）。跨 30 天窗口 TWR ≥ 判定线才进
        阶段 A 真金冒烟；测量只读，不自动执行。
      </p>
    </section>
  );
}
