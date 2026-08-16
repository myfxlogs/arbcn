import { useEffect, useState } from "react";

import { useSnapshot } from "./hooks";
import { fmtClock, reasonText } from "./format";
import { Bell } from "./components/Bell";
import { Chip } from "./components/Chip";
import { FactsPage } from "./components/FactsPage";
import { Ledger } from "./components/Ledger";
import { OverviewPage } from "./components/OverviewPage";
import { SimPage } from "./components/SimPage";

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

// D-047 P0：数据 hook 随视图生命周期——仅 useSnapshot 留在 App 根（header 健康徽标/
// 铃铛为全局 chrome，任何 tab 都需要）；useFactsSnapshot/useSim/useKnowledge 下沉到
// 各自页面组件。refreshKey 是全局刷新信号：header 刷新 = useSnapshot.reload +
// refreshKey 递增 → 总览页 sim/knowledge 随全局刷新重载（此前「刷新只刷一半」）。
export default function App() {
  const { snap, error, reload, ackAlert, ackAll } = useSnapshot();
  const [tab, setTab] = useState<Tab>("overview");
  const [theme, setTheme] = useState<"light" | "dark">(initTheme);
  const [ackBusy, setAckBusy] = useState<ReadonlySet<string>>(new Set());
  const [refreshKey, setRefreshKey] = useState(0);

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
  // 全局刷新：header 快照 + 当前页数据（refreshKey 递增触发总览页 sim/knowledge 重载）。
  const onRefresh = () => {
    reload();
    setRefreshKey((n) => n + 1);
  };

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
        <button type="button" className="icon" onClick={onRefresh}>
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
        snap ? (
          <OverviewPage
            snap={snap}
            error={error}
            ackBusy={ackBusy}
            ack={ack}
            refreshKey={refreshKey}
          />
        ) : (
          <p className="empty">连接服务中…</p>
        )
      ) : tab === "facts" ? (
        <FactsPage />
      ) : tab === "sim" ? (
        <SimPage />
      ) : (
        <Ledger />
      )}
    </main>
  );
}
