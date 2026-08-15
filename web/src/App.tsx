import { useEffect, useState } from "react";

import { useSnapshot } from "./hooks";
import { fmtClock, reasonText } from "./format";
import { Bell } from "./components/Bell";
import { Chip } from "./components/Chip";
import { Opportunity } from "./components/Opportunity";
import { Triggers } from "./components/Triggers";
import { Alerts } from "./components/Alerts";

// initTheme 初始主题：本地存储 > 系统偏好。
function initTheme(): "light" | "dark" {
  const saved = localStorage.getItem("arbcn-theme");
  if (saved === "light" || saved === "dark") return saved;
  return window.matchMedia("(prefers-color-scheme: dark)").matches ? "dark" : "light";
}

export default function App() {
  const { snap, error, reload, ackAlert, ackAll } = useSnapshot();
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

      {error ? (
        <div className="banner" role="alert">
          加载失败：{error}
        </div>
      ) : null}

      {snap ? (
        <>
          {/* 双栏：机会面板 + 告警流（对话 #32 布局要求） */}
          <div className="row">
            <Opportunity facts={snap.facts} sourceHealth={snap.sourceHealth} />
            <Alerts alerts={snap.alerts} ackBusy={ackBusy} onAck={ack} />
          </div>
          <Triggers states={snap.states} />
        </>
      ) : (
        <p className="empty">连接服务中…</p>
      )}
    </main>
  );
}
