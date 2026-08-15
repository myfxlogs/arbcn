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

// statusText freshness 状态文案（M2-a §2.3：stale/down 区分业务含义；纯中文，无英文前缀）。
export const statusText = (status: string): string => {
  switch (status) {
    case "live":
      return "正常";
    case "stale":
      return "市场闭市/冻结";
    case "down":
      return "采集器失联";
    default:
      return status;
  }
};

// reasonText healthz 降级原因 → 中文（后端原因码为英文枚举，UI 层映射）。
export const reasonText = (reason: string): string => {
  switch (reason) {
    case "db_unreachable":
      return "数据库不可达";
    case "pending_migrations":
      return "迁移未完成";
    case "migrations_check_failed":
      return "迁移检查失败";
    default:
      return reason;
  }
};

// stateText 触发器状态机 → 中文（armed 待命 / active 触发 / resolved 已解除）。
export const stateText = (state: string): string => {
  switch (state) {
    case "armed":
      return "待命";
    case "active":
      return "触发";
    case "resolved":
      return "已解除";
    default:
      return state;
  }
};

// levelText 告警级别 → 中文（info 提示 / warn 警告 / critical 严重）。
export const levelText = (level: string): string => {
  switch (level) {
    case "info":
      return "提示";
    case "warn":
      return "警告";
    case "critical":
      return "严重";
    default:
      return level;
  }
};

// ruleLabel 规则名 → 中文标签（告警流/触发器展示；规则名本身是稳定标识符，
// 未知名回退原名，与后端 rule/ruleLabels 同语义）。
export const ruleLabel = (name: string): string => {
  switch (name) {
    case "funding_warn":
      return "资金费率预警";
    case "funding_critical":
      return "资金费率激活";
    case "trx_funding_positive":
      return "TRX 费率转正";
    case "defi_large_tier_change":
      return "金额档利率变动";
    case "ladder_trap":
      return "阶梯陷阱识别";
    case "reverse_repo_timing":
      return "逆回购时点";
    case "usdcnh_buy_line":
      return "汇率加仓线";
    case "iv_opportunity":
      return "IV 机会";
    case "nonstable_quote_change":
      return "计价币种陷阱";
    case "collector_heartbeat":
      return "采集器心跳";
    default:
      return name;
  }
};
