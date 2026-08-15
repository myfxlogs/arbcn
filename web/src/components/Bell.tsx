import { useEffect, useRef, useState } from "react";

import { fmtRel } from "../format";
import type { UnackedAlert } from "../gen/arbcn/dashboard/v1/dashboard_pb";
import { LevelChip } from "./Chip";

// Bell 通知中心（M2-a §1.3）：右上角铃铛 + 未读红徽标；点击下拉抽屉列出未读告警
// （level 徽标 + 规则名 + 消息 + 相对时间 + 逐条 ✓ 确认）+ 底部"全部标记已读"；
// 空态"暂无新通知"。深浅主题跟现有设计；移动端抽屉全屏（D-030 PWA 后置）。
export function Bell({
  unacked,
  ackBusy,
  onAck,
  onAckAll,
}: {
  unacked: UnackedAlert[];
  ackBusy: ReadonlySet<string>;
  onAck: (id: bigint) => void;
  onAckAll: () => Promise<number>;
}) {
  const [open, setOpen] = useState(false);
  const [ackAllBusy, setAckAllBusy] = useState(false);
  const box = useRef<HTMLDivElement>(null);

  // 点抽屉外或 Esc 关闭。
  useEffect(() => {
    if (!open) return;
    const onDown = (e: MouseEvent) => {
      if (box.current && !box.current.contains(e.target as Node)) setOpen(false);
    };
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") setOpen(false);
    };
    document.addEventListener("mousedown", onDown);
    document.addEventListener("keydown", onKey);
    return () => {
      document.removeEventListener("mousedown", onDown);
      document.removeEventListener("keydown", onKey);
    };
  }, [open]);

  const count = unacked.length;

  const handleAckAll = async () => {
    if (count === 0) return;
    setAckAllBusy(true);
    try {
      await onAckAll();
    } finally {
      setAckAllBusy(false);
    }
  };

  return (
    <div className="bell" ref={box}>
      <button
        type="button"
        className="bell-btn"
        aria-haspopup="dialog"
        aria-expanded={open}
        aria-label={count > 0 ? `未读通知 ${count} 条` : "通知中心（无未读）"}
        onClick={() => setOpen((o) => !o)}
      >
        <svg
          viewBox="0 0 24 24"
          width="18"
          height="18"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
          strokeLinecap="round"
          strokeLinejoin="round"
          aria-hidden="true"
        >
          <path d="M18 8a6 6 0 0 0-12 0c0 7-3 9-3 9h18s-3-2-3-9" />
          <path d="M13.73 21a2 2 0 0 1-3.46 0" />
        </svg>
        {count > 0 ? <span className="bell-badge">{count > 99 ? "99+" : count}</span> : null}
      </button>

      {open ? (
        <div className="bell-panel" role="dialog" aria-label="通知中心">
          <div className="bell-head">
            <span>通知中心</span>
            <span className="bell-count">{count} 条未读</span>
          </div>
          {count === 0 ? (
            <p className="empty bell-empty">暂无新通知</p>
          ) : (
            <ol className="bell-list">
              {unacked.map((a) => (
                <li key={a.id.toString()}>
                  <div className="bell-item-head">
                    <LevelChip level={a.level} />
                    <span className="bell-rule">{a.rule}</span>
                    <time className="bell-ts">{fmtRel(a.ts)}</time>
                    <button
                      type="button"
                      className="ack bell-ack"
                      disabled={ackBusy.has(a.id.toString())}
                      onClick={() => onAck(a.id)}
                    >
                      ✓ 确认
                    </button>
                  </div>
                  <p className="bell-msg">{a.message}</p>
                </li>
              ))}
            </ol>
          )}
          <div className="bell-foot">
            <button
              type="button"
              className="ack bell-ack-all"
              disabled={count === 0 || ackAllBusy}
              onClick={handleAckAll}
            >
              全部标记已读
            </button>
          </div>
        </div>
      ) : null}
    </div>
  );
}
