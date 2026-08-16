import { fmtTs } from "../format";
import type { KnowledgeEntry } from "../gen/arbcn/dashboard/v1/dashboard_pb";
import { Chip, type ChipTone } from "./Chip";

// statusLabel 条目状态 → 中文（active=生效中 / superseded=已更新 / retracted=已撤销）。
function statusLabel(s: string): string {
  switch (s) {
    case "active":
      return "生效中";
    case "superseded":
      return "已更新";
    case "retracted":
      return "已撤销";
    default:
      return s;
  }
}

// statusTone 状态 → 徽标色（生效=绿 / 更新=黄 / 撤销=红）。
function statusTone(s: string): ChipTone {
  switch (s) {
    case "active":
      return "good";
    case "superseded":
      return "warn";
    case "retracted":
      return "critical";
    default:
      return "neutral";
  }
}

// sigLabel 受控签名键 → 中文模式名（practices #18：映射对照后端签名全集，未知名回退原名）。
function sigLabel(sig: string): string {
  switch (sig) {
    case "funding:spike_trap":
      return "资金费率尖峰陷阱";
    case "defi:single_pool_spike":
      return "单池利率尖峰";
    case "funding:cross_venue_divergence":
      return "跨所费率分歧";
    default:
      return sig;
  }
}

// KnowledgeBoard 市场结构经验库（D-046）：只读呈现已核实的模式条目（吸收=人工 D# 落盘，
// git 跟踪；系统只匹配与呈现，不自动吸收）。复核状态：validated_at 有值 = 已复核，
// 缺省 = 待复核。
export function KnowledgeBoard({
  entries,
  error,
  onReload,
}: {
  entries: KnowledgeEntry[];
  error: string;
  onReload: () => void;
}) {
  return (
    <section className="card" aria-labelledby="knowledge-title">
      <h2 id="knowledge-title">
        市场结构经验库
        <button type="button" className="facts-reload" onClick={onReload}>
          刷新
        </button>
      </h2>
      <p className="insights-sub">
        已核实模式的判定记录（人工 D# 吸收）；系统检测到同签名情况时在「进化建议」提示上回判定
      </p>
      {error ? <p className="banner" role="alert">{error}</p> : null}
      {entries.length === 0 ? (
        <p className="empty">暂无经验条目（等待吸收）</p>
      ) : (
        <ul className="insights">
          {entries.map((e) => (
            <li key={e.signature} className="insight-row">
              <div className="insight-head">
                <Chip tone={statusTone(e.status)}>{statusLabel(e.status)}</Chip>
                <span className="insight-cat">{sigLabel(e.signature)}</span>
                <span className="insight-title">{e.verdict}</span>
                <span className="insight-ts">
                  {e.validatedAt ? `复核 ${fmtTs(e.validatedAt)}` : "待复核"}
                </span>
              </div>
              {e.rationale ? <p className="insight-detail">{e.rationale}</p> : null}
              <ul className="insight-actions">
                <li>
                  出处 {e.source || "—"} · 实例 {e.venue ? `${e.venue} · ${e.symbol}` : "—"}
                </li>
                {e.validationNote ? <li>复核结论：{e.validationNote}</li> : null}
              </ul>
            </li>
          ))}
        </ul>
      )}
    </section>
  );
}
