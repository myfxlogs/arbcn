import { useEffect, useState } from "react";

import { useFactsSnapshot, useSim, useSnapshot } from "./hooks";
import { fmtClock, reasonText } from "./format";
import { Alerts } from "./components/Alerts";
import { Bell } from "./components/Bell";
import { Chip } from "./components/Chip";
import { ConfirmPanel } from "./components/ConfirmPanel";
import { FactsSnapshot } from "./components/FactsSnapshot";
import { Ledger } from "./components/Ledger";
import { Opportunity } from "./components/Opportunity";
import { SimExec } from "./components/SimExec";
import { Triggers } from "./components/Triggers";

// Tab 顶部分页（M2-b：监控总览 / 事实快照 RMB 视角 / 出入金台账；M3-c：模拟执行）。
type Tab = "overview" | "facts" | "ledger" | "sim";

const TABS: { key: Tab; label: string }[] = [
  { key: "overview", label: "监控总览" },
  { key: "facts", label: "事实快照" },
  { key: "ledger", label: "出入金台账" },
  { key: "sim", label: "模拟执行" },
];

// initTheme 初始主题：本地存储 > 系统偏好。
function initTheme(): "light" | "dark" {
  const saved = localStorage.getItem("arbcn-theme");
  if (saved === "light" || saved === "dark") return saved;
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

export default function App() {
  const { snap, error, reload, ackAlert, ackAll } = useSnapshot();
  const factsSnap = useFactsSnapshot();
  // sim 数据提升到 App 层共享（对话 #59）：确认下单面板（总览页）+ 模拟执行 tab
  // 同源，ConfirmSimOrder 成功后 useSim 内部 tick 刷新两处持仓/订单。
  const sim = useSim();
  const [tab, setTab] = useState<Tab>("overview");
  const [theme, setTheme] = useState<"light" | "dark">(initTheme);
  const [ackBusy, setAckBusy] = useState<ReadonlySet<string>>(new Set());

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
    localStorage.setItem("arbcn-theme", theme);
  }, [theme]);

  const ack = async (id: bigint) => {
    const key = id.toString();
    setAckBusy((s) => new Set(s).add(key));
    try {
      await ackAlert(id);
    } finally {
      setAckBusy((s) => {
        const next = new Set(s);
        next.delete(key);
        return next;
      });
    }
  };

  const healthOk = snap?.health.status === "ok";

  return (
    <main>
      <header className="app">
        <h1>arbcn 监控仪表盘</h1>
        {snap ? (
          <Chip tone={healthOk ? "good" : "critical"}>
            {healthOk ? "正常" : `降级 · ${reasonText(snap.health.reason)}`}
          </Chip>
        ) : null}
        <span className="meta">{snap ? `更新于 ${fmtClock(snap.at)}` : "加载中…"}</span>
        {snap ? (
          <Bell unacked={snap.unacked} ackBusy={ackBusy} onAck={ack} onAckAll={ackAll} />
        ) : null}
        <button type="button" className="icon" onClick={reload}>
          刷新
        </button>
        <button
          type="button"
          className="icon"
          onClick={() => setTheme((t) => (t === "dark" ? "light" : "dark"))}
        >
          {theme === "dark" ? "浅色" : "深色"}
        </button>
      </header>

      <nav className="tabs" aria-label="视图切换">
        {TABS.map((t) => (
          <button
            key={t.key}
            type="button"
            className={t.key === tab ? "tab tab-on" : "tab"}
            onClick={() => setTab(t.key)}
          >
            {t.label}
          </button>
        ))}
      </nav>

      {tab === "overview" ? (
        <>
          {error ? (
            <div className="banner" role="alert">
              加载失败：{error}
            </div>
          ) : null}
          {snap ? (
            <>
              {/* 双栏（对话 #60 布局调整）：机会面板左列跨两行；右列 = 告警流（上）+ 确认下单（下，与机会面板同行） */}
              <div className="row">
                <Opportunity facts={snap.facts} sourceHealth={snap.sourceHealth} />
                <div className="row-col">
                  <Alerts alerts={snap.alerts} ackBusy={ackBusy} onAck={ack} />
                  <ConfirmPanel
                    orders={sim.orders}
                    confirm={sim.confirm}
                    error={sim.error}
                    reload={sim.reload}
                  />
                </div>
              </div>
              <Triggers states={snap.states} />
            </>
          ) : (
            <p className="empty">连接服务中…</p>
          )}
        </>
      ) : tab === "facts" ? (
        <FactsSnapshot
          facts={factsSnap.facts}
          fxRate={factsSnap.fxRate}
          fxAvailable={factsSnap.fxAvailable}
          fxTs={factsSnap.fxTs}
          error={factsSnap.error}
          onReload={factsSnap.reload}
        />
      ) : tab === "sim" ? (
        <SimExec
          positions={sim.positions}
          accounts={sim.accounts}
          report={sim.report}
          error={sim.error}
          reload={sim.reload}
        />
      ) : (
        <Ledger />
      )}
    </main>
  );
}
