import { useState } from "react";
import { fmtTs } from "../format";
import type { KnowledgeEntry } from "../gen/arbcn/dashboard/v1/dashboard_pb";
import { Chip, type ChipTone } from "./Chip";
import { Collapse } from "./Collapse";

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

// 复核表单行（D-054）：决策层人工在环——选择生命周期状态 + 可选改判定文本（留空 =
// 保留原判定）+ 必填复核结论（留痕）。服务端白名单校验状态三态；note 必填。
function ReviewForm({
  entry,
  onSubmit,
  onCancel,
}: {
  entry: KnowledgeEntry;
  onSubmit: (status: string, verdict: string, note: string) => Promise<void>;
  onCancel: () => void;
}) {
  const [status, setStatus] = useState(entry.status || "active");
  const [verdict, setVerdict] = useState(entry.verdict || "");
  // D-059：note 自动预填当前核验证据（「当前命中/未命中 + 关键数值」），复核人确认或
  // 修改后提交 = 人工在环裁决；证据缺失/不可用 → 留空让复核人自填，不代写结论。
  const [note, setNote] = useState(
    entry.currentEvidence?.startsWith("自动核验：") ? entry.currentEvidence : "",
  );
  const [submitting, setSubmitting] = useState(false);
  const [err, setErr] = useState("");

  return (
    <form
      className="review-form"
      onSubmit={(e) => {
        e.preventDefault();
        if (!note.trim()) {
          setErr("复核结论必填（决策留痕）");
          return;
        }
        setSubmitting(true);
        onSubmit(status, verdict, note).catch((e2) => {
          setErr(String(e2));
          setSubmitting(false);
        });
      }}
    >
      <label>
        生命周期
        <select
          value={status}
          onChange={(e) => setStatus(e.target.value)}
          disabled={submitting}
        >
          <option value="active">生效中</option>
          <option value="superseded">已更新</option>
          <option value="retracted">已撤销</option>
        </select>
      </label>
      <label>
        判定文本
        <input
          type="text"
          value={verdict}
          onChange={(e) => setVerdict(e.target.value)}
          placeholder="留空 = 保留原判定"
          disabled={submitting}
        />
      </label>
      <label className="review-note">
        复核结论
        <input
          type="text"
          value={note}
          onChange={(e) => setNote(e.target.value)}
          placeholder="必填：本次复核的依据/结论"
          disabled={submitting}
        />
      </label>
      <div className="review-actions">
        <button type="button" className="facts-reload" onClick={onCancel} disabled={submitting}>
          取消
        </button>
        <button type="submit" className="review-submit" disabled={submitting}>
          {submitting ? "提交中…" : "确认复核"}
        </button>
      </div>
      {err ? (
        <p className="review-err" role="alert">
          {err}
        </p>
      ) : null}
    </form>
  );
}

// KnowledgeBoard 市场结构经验库（D-046）：呈现已核实的模式条目（吸收=人工 D# 落盘，
// git 跟踪；系统只匹配与呈现，不自动吸收）。复核状态：validated_at 有值 = 已复核，
// 缺省 = 待复核。复核 = 人工在环（D-054），按钮展开内联表单，仅写判定记录不改规则。
// D-048 U3 第一眼原则：低频参考面默认折叠，仅系统检测到同签名命中（knowledge_match →
// 进化建议提示）时由上层以 defaultOpen 展开呈现裁决对照。
export function KnowledgeBoard({
  entries,
  error,
  onReload,
  defaultOpen = false,
  review,
}: {
  entries: KnowledgeEntry[];
  error: string;
  onReload: () => void;
  defaultOpen?: boolean;
  review: (signature: string, status: string, verdict: string, note: string) => Promise<void>;
}) {
  // editing：正在复核的签名（同时只开一个表单）；完成后置 null。
  const [editing, setEditing] = useState<string | null>(null);

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
        <Collapse title={`已核实模式（${entries.length} 条）`} defaultOpen={defaultOpen}>
          {/* D-052：条目多时高度封顶卡内滚动，不无限延伸 */}
          <ul className="insights scroll-cap">
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
                {/* D-059 复核自动证据：系统用当前数据重跑探测器 → 当前命中/未命中 +
                    关键数值，供人工复核裁决（只读；判定仍由下方表单人工提交） */}
                {e.currentEvidence ? (
                  <p className="evidence-note">
                    <span className="evidence-label">当前核验</span>
                    {e.currentEvidence}
                  </p>
                ) : null}
                <ul className="insight-actions">
                  <li>
                    出处 {e.source || "—"} · 实例 {e.venue ? `${e.venue} · ${e.symbol}` : "—"}
                  </li>
                  {e.validationNote ? <li>复核结论：{e.validationNote}</li> : null}
                </ul>
                <div className="review-bar">
                  {editing === e.signature ? (
                    <ReviewForm
                      entry={e}
                      onSubmit={(status, verdict, note) =>
                        review(e.signature, status, verdict, note).then(() => setEditing(null))
                      }
                      onCancel={() => setEditing(null)}
                    />
                  ) : (
                    <button
                      type="button"
                      className="review-open"
                      onClick={() => setEditing(e.signature)}
                    >
                      {e.validatedAt ? "再次复核" : "复核"}
                    </button>
                  )}
                </div>
              </li>
            ))}
          </ul>
        </Collapse>
      )}
    </section>
  );
}
