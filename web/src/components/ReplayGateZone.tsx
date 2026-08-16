import { useEffect, useState } from "react";

import { sim } from "../client";
import type { GetReplayStateResponse, ReplayStateKind } from "../gen/arbcn/sim/v1/sim_pb";

// replayMeta 回放判据 → 徽标文案/样式（D-065 值域；与判定门① PerformanceZone 同色调）。
const replayMeta: Record<string, { label: string; cls: string }> = {
  falsified: { label: "FALSIFIED · 结构证伪拒单", cls: "gate-bad" },
  watch: { label: "WATCH · 净不抵基档拒单", cls: "gate-warn" },
  pass: { label: "PASS · 证伪未发生放行", cls: "gate-ok" },
  no_window: { label: "NO_WINDOW · 环境无窗口放行", cls: "gate-bg" },
};

// kindLabel 策略 kind → 展示名（D-065 每策略自有高费率档）。
const kindLabel: Record<string, string> = {
  funding_hedge: "funding_hedge · 资金费率对冲",
  carry_asset: "carry_asset · 生息资产（sUSDe 类）",
  repo: "repo · 时点逆回购",
};

// ReplayGateZone 回放证伪门禁证据面（D-065 修订：每个策略强制自动，做成门禁；P4 可检查
// 性——门禁休眠 no_window 也可见，不靠拒单负样本才暴露）。自包含只读卡：挂载拉取 +
// 60s 轮询（判据随历史增长变化，低频即可）；不进 hooks.ts useSim（防 hooks.ts 顶穿 450
// 行 check-lines 硬线，practices #40）。falsified/watch → 订单管线 REPLAY_VETO 拒单
// （本卡只展示判据，不触发任何写路径）。
export function ReplayGateZone() {
  const [state, setState] = useState<GetReplayStateResponse | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    let alive = true;
    const load = async () => {
      try {
        const res = await sim.getReplayState({});
        if (!alive) return;
        setState(res);
        setError("");
      } catch (e) {
        if (!alive) return;
        setError(String(e));
      }
    };
    void load();
    const timer = setInterval(() => void load(), 60_000);
    return () => {
      alive = false;
      clearInterval(timer);
    };
  }, []);

  const pct = (v: number): string => `${v.toFixed(2)}%`;
  return (
    <section className="card" aria-labelledby="sim-replay-title">
      <h2 id="sim-replay-title">回放证伪门禁 · 强制自动</h2>
      <p className="muted">
        每个策略在自有高费率档下回放历史（{state?.historyDays ?? 365} 天）：{" "}
        <strong>falsified / watch</strong> → 订单管线 REPLAY_VETO 拒单；<strong>pass</strong>{" "}
        → 放行（证伪未发生，非收益预测）；<strong>no_window</strong> → 放行（D-061 ② 环境
        无窗口，门禁休眠 = 正确输出）。
      </p>
      {error ? (
        <div className="banner" role="alert">
          回放门禁加载失败：{error}
        </div>
      ) : null}
      {(state?.kinds ?? []).map((k: ReplayStateKind) => {
        const meta = replayMeta[k.verdict] ?? { label: k.verdict, cls: "gate-bg" };
        return (
          <div key={k.kind} className="card-inner">
            <div className="gate-head">
              <span className={`gate-badge ${meta.cls}`}>{meta.label}</span>
              <span className="muted">
                {kindLabel[k.kind] ?? k.kind} · 高费率档 ≥{pct(k.tierPct)} · 摩擦{" "}
                {k.frictionPct === 0 ? "0" : pct(k.frictionPct)}
              </span>
            </div>
            <p className="muted">{k.note}</p>
            <div className="account-grid">
              <div className="account-metric">
                <span className="muted">历史高费率窗口</span>
                <strong className="num">{k.windowCount}</strong>
              </div>
              <div className="account-metric">
                <span className="muted">回放样本 / ≥档读数</span>
                <strong className="num">
                  {k.totalSamples} / {k.highSamples}
                </strong>
              </div>
              <div className="account-metric">
                <span className="muted">均值净年化</span>
                <strong className="num">{k.windowCount === 0 ? "—" : pct(k.meanNetAnn)}</strong>
              </div>
              <div className="account-metric">
                <span className="muted">窗口净年化区间</span>
                <strong className="num">
                  {k.windowCount === 0 ? "—" : `${pct(k.worstNetAnn)} ~ ${pct(k.bestNetAnn)}`}
                </strong>
              </div>
            </div>
          </div>
        );
      })}
      <p className="muted">
        <strong>证伪不证真</strong>（practices #38）：回放只答「门禁机制在历史高费率窗口能否
        扣摩擦净正」，不答「未来能赚多少」；pass 非收益预测。
      </p>
    </section>
  );
}
