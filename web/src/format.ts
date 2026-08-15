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
