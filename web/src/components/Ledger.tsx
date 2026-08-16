import { create } from "@bufbuild/protobuf";
import { useState } from "react";

import { fmtAmount, fmtTs, ledgerDate, tierLabel } from "../format";
import {
  AddLedgerEntryRequestSchema,
  type TierSummary,
} from "../gen/arbcn/dashboard/v1/dashboard_pb";
import { useLedger } from "../hooks";

// TIERS 档位下拉（D-026 三档 + 持有层；空 = 未分类）。值与 store Tier* 常量一致。
const TIERS = [
  { value: "", label: "未分类" },
  { value: "protected_convexity", label: "保本凸性" },
  { value: "stable_base", label: "稳定币基档" },
  { value: "cash_management", label: "现金管理" },
  { value: "holding", label: "持有层" },
];

// nowLocal 本地时区的 datetime-local 初值（YYYY-MM-DDTHH:mm）。
function nowLocal(): string {
  const d = new Date();
  const p = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}T${p(d.getHours())}:${p(d.getMinutes())}`;
}

// emptyForm 新录入空表单。
function emptyForm() {
  return {
    date: nowLocal(),
    channel: "",
    currency: "USDT",
    amount: "",
    feeRate: "0",
    tier: "stable_base",
    note: "",
  };
}

// TierSummaryTable 档位归因汇总（GROUP BY tier 简单分组；M2-b §6）。
function TierSummaryTable({ summary }: { summary: TierSummary[] }) {
  if (summary.length === 0) {
    return <p className="empty">暂无台账流水</p>;
  }
  return (
    <div className="table-scroll">
      <table className="rows">
        <thead>
          <tr>
            <th scope="col">档位</th>
            <th scope="col">入金</th>
            <th scope="col">出金</th>
            <th scope="col">净额</th>
            <th scope="col">笔数</th>
          </tr>
        </thead>
        <tbody>
          {summary.map((s) => (
            <tr key={s.tier}>
              <th scope="row">{tierLabel(s.tier)}</th>
              <td className="ledger-in">{fmtAmount(s.inflow)}</td>
              <td className="ledger-out">{fmtAmount(s.outflow)}</td>
              <td className={s.net < 0 ? "ledger-out" : "ledger-in"}>{fmtAmount(s.net)}</td>
              <td>{s.entryCount}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

// Ledger 台账（M2-b §6）：出入金流水 + 档位归因汇总 + 手工录入表单。
// 资金动作永远人工（§1）——本视图只写人工录入的流水，无任何自动执行能力。
export function Ledger() {
  const { entries, summary, error, addEntry } = useLedger();
  const [form, setForm] = useState(emptyForm);
  const [busy, setBusy] = useState(false);
  const [formError, setFormError] = useState("");

  const set = (k: keyof ReturnType<typeof emptyForm>, v: string) =>
    setForm((f) => ({ ...f, [k]: v }));

  const submit = async (e: React.FormEvent) => {
    e.preventDefault();
    setFormError("");
    const amount = Number(form.amount);
    if (form.channel.trim() === "" || form.currency.trim() === "" || !Number.isFinite(amount)) {
      setFormError("通道、币种必填，金额须为有限数字");
      return;
    }
    const ts = ledgerDate(form.date);
    if (!ts) {
      setFormError("日期无效");
      return;
    }
    const req = create(AddLedgerEntryRequestSchema, {
      date: ts,
      channel: form.channel.trim(),
      currency: form.currency.trim(),
      amount,
      feeRate: Number(form.feeRate) || 0,
      tier: form.tier,
      note: form.note.trim(),
    });
    setBusy(true);
    const ok = await addEntry(req);
    setBusy(false);
    if (ok) {
      setForm(emptyForm());
    } else {
      setFormError("录入失败，请重试");
    }
  };

  return (
    <section className="card" aria-labelledby="ledger-title">
      <h2 id="ledger-title">出入金台账</h2>

      <h3>按档位归因</h3>
      <TierSummaryTable summary={summary} />

      <h3>手工录入</h3>
      <form className="ledger-form" onSubmit={(e) => void submit(e)}>
        <label>
          日期
          <input type="datetime-local" value={form.date} onChange={(e) => set("date", e.target.value)} />
        </label>
        <label>
          通道
          <input
            type="text"
            list="channel-options"
            value={form.channel}
            onChange={(e) => set("channel", e.target.value)}
            placeholder="binance / okx / 民营定期"
          />
          <datalist id="channel-options">
            <option value="binance" />
            <option value="okx" />
            <option value="民营定期" />
            <option value="逆回购" />
            <option value="银行" />
          </datalist>
        </label>
        <label>
          币种
          <input type="text" list="currency-options" value={form.currency} onChange={(e) => set("currency", e.target.value)} />
          <datalist id="currency-options">
            <option value="RMB" />
            <option value="USD" />
            <option value="USDT" />
            <option value="USDC" />
            <option value="BTC" />
          </datalist>
        </label>
        <label>
          金额（正=入金，负=出金）
          <input type="number" step="any" value={form.amount} onChange={(e) => set("amount", e.target.value)} />
        </label>
        <label>
          费率 %
          <input type="number" step="any" value={form.feeRate} onChange={(e) => set("feeRate", e.target.value)} />
        </label>
        <label>
          档位
          <select value={form.tier} onChange={(e) => set("tier", e.target.value)}>
            {TIERS.map((t) => (
              <option key={t.value} value={t.value}>
                {t.label}
              </option>
            ))}
          </select>
        </label>
        <label className="ledger-note">
          备注
          <input type="text" value={form.note} onChange={(e) => set("note", e.target.value)} />
        </label>
        <button type="submit" className="icon" disabled={busy}>
          {busy ? "录入中…" : "录入"}
        </button>
        {formError ? <p className="banner ledger-form-err">{formError}</p> : null}
      </form>

      {error ? (
        <div className="banner" role="alert">
          台账加载失败：{error}
        </div>
      ) : null}

      <h3>流水</h3>
      {entries.length === 0 ? (
        <p className="empty">暂无台账流水</p>
      ) : (
        <div className="table-scroll">
          <table className="rows">
            <thead>
              <tr>
                <th scope="col">日期</th>
                <th scope="col">通道</th>
                <th scope="col">币种</th>
                <th scope="col">金额</th>
                <th scope="col">费率</th>
                <th scope="col">档位</th>
                <th scope="col">备注</th>
              </tr>
            </thead>
            <tbody>
              {entries.map((e) => (
                <tr key={e.id.toString()}>
                  <th scope="row">{fmtTs(e.date)}</th>
                  <td>{e.channel}</td>
                  <td>{e.currency}</td>
                  <td className={e.amount < 0 ? "ledger-out" : "ledger-in"}>
                    {fmtAmount(e.amount)}
                  </td>
                  <td>{e.feeRate > 0 ? `${fmtAmount(e.feeRate)}%` : "—"}</td>
                  <td>{tierLabel(e.tier)}</td>
                  <td className="ledger-note-cell">{e.note || "—"}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}
