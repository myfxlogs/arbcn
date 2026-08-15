import { timestampDate } from "@bufbuild/protobuf/wkt";
import type { Timestamp } from "@bufbuild/protobuf/wkt";

// pct 年化/百分比格式化（value 已按 % 口径存储）。
export const pct = (v: number, digits = 2): string => `${v.toFixed(digits)}%`;

// days 日历倒计时（Fact.Value = 距事件天数，当日 = 0）。
export const days = (v: number): string => `${Math.round(v)} 天`;

// num 触发器 last_value 通用显示（口径随规则而异，保留 4 位有效数字）。
export const num = (v: number): string => v.toPrecision(4);

// fmtTs 精确到分钟的时间戳；缺省 = "—"。
export const fmtTs = (t?: Timestamp): string =>
  t
    ? timestampDate(t).toLocaleString("zh-CN", {
        month: "2-digit",
        day: "2-digit",
        hour: "2-digit",
        minute: "2-digit",
        hour12: false,
      })
    : "—";

// fmtClock 仅时分秒（顶部"更新于"）。
export const fmtClock = (d: Date): string =>
  d.toLocaleTimeString("zh-CN", { hour12: false });

// fmtRel 相对时间（铃铛条目时间 / freshness tooltip"最近更新 X 前"）。
export const fmtRel = (t?: Timestamp): string => {
  if (!t) return "—";
  const diff = Math.max(0, Date.now() - timestampDate(t).getTime());
  const m = Math.floor(diff / 60_000);
  if (m < 1) return "刚刚";
  if (m < 60) return `${m} 分钟前`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h} 小时前`;
  return `${Math.floor(h / 24)} 天前`;
};

// fmtInterval 源轮询间隔（秒 → 中文）。
export const fmtInterval = (sec: bigint): string => {
  const s = Number(sec);
  if (s % 3600 === 0 && s > 0) return `${s / 3600} 小时`;
  if (s % 60 === 0 && s > 0) return `${s / 60} 分钟`;
  return `${s} 秒`;
};

// statusText freshness 状态文案（M2-a §2.3：stale/down 区分业务含义）。
export const statusText = (status: string): string => {
  switch (status) {
    case "live":
      return "live";
    case "stale":
      return "stale · 市场闭市/冻结";
    case "down":
      return "down · 采集器失联";
    default:
      return status;
  }
};
