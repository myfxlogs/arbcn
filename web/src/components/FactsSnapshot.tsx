import type { Timestamp } from "@bufbuild/protobuf/wkt";

import { factValue, fmtRel, fmtTs, fxText, unitText } from "../format";
import type { FactRmb } from "../gen/arbcn/dashboard/v1/dashboard_pb";
import { Chip } from "./Chip";

// COVERED_KINDS：展示层做 RMB 折算的 kind（与后端 rmb.CoveredKinds 同语义；
// 非覆盖 kind 原样显示，不折算）。
const COVERED_KINDS = new Set(["funding", "defi_rate", "deposit_rate"]);

// kindLabel 事实 kind → 中文（快照表可读性）。
function kindLabel(kind: string): string {
  switch (kind) {
    case "funding":
      return "资金费率";
    case "defi_rate":
      return "DeFi 利率";
    case "deposit_rate":
      return "定存利率";
    case "reverse_repo":
      return "逆回购";
    case "fx":
      return "汇率";
    case "iv":
      return "IV";
    case "calendar":
      return "时点";
    default:
      return kind;
  }
}

// FactsSnapshot 事实快照 + RMB 折算（M2-b §4/§5 机器可读投影）。
// 覆盖 kind（funding/defi_rate/deposit_rate）× 当日 USDCNH → RMB 净收益视角；
// 汇率缺失 → USD 原值 + 「汇率不可用」标记（不崩、不静默错值）。
export function FactsSnapshot({
  facts,
  fxRate,
  fxAvailable,
  fxTs,
  error,
  onReload,
}: {
  facts: FactRmb[];
  fxRate: number;
  fxAvailable: boolean;
  fxTs?: Timestamp;
  error: string;
  onReload: () => void;
}) {
  // 稳定排序：kind → symbol → venue。
  const rows = [...facts].sort((a, b) =>
    a.kind === b.kind ? (a.symbol === b.symbol ? a.venue.localeCompare(b.venue) : a.symbol.localeCompare(b.symbol)) : a.kind.localeCompare(b.kind),
  );

  return (
    <section className="card" aria-labelledby="facts-title">
      <h2 id="facts-title">
        事实快照（RMB 视角）
        <button type="button" className="icon facts-reload" onClick={onReload}>
          刷新
        </button>
      </h2>

      <div className="facts-fx">
        {fxAvailable ? (
          <>
            <Chip tone="good">汇率可用</Chip>
            <span>
              USD/CNH <strong>{fxRate.toFixed(4)}</strong> · 报价 {fxTs ? fmtRel(fxTs) : "—"}
            </span>
          </>
        ) : (
          <Chip tone="critical">汇率不可用</Chip>
        )}
        <span className="facts-hint">
          覆盖 kind（资金费率/DeFi 利率/定存利率）USD 计价 × USDCNH → RMB 净收益视角（原始值不污染）
        </span>
      </div>

      {error ? (
        <div className="banner" role="alert">
          加载失败：{error}
        </div>
      ) : null}

      {rows.length === 0 ? (
        <p className="empty">暂无事实数据</p>
      ) : (
        <div className="table-scroll">
          <table className="rows">
            <thead>
              <tr>
                <th scope="col">事实</th>
                <th scope="col">标的</th>
                <th scope="col">交易所</th>
                <th scope="col">单位</th>
                <th scope="col">USD 原值</th>
                <th scope="col">RMB 视角</th>
                <th scope="col">采集时刻</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((f) => {
                const covered = COVERED_KINDS.has(f.kind);
                return (
                  <tr key={`${f.kind}:${f.symbol}:${f.venue}`}>
                    <th scope="row">{kindLabel(f.kind)}</th>
                    <td>{f.symbol}</td>
                    <td>{f.venue}</td>
                    <td>{unitText(f.unit)}</td>
                    <td>{factValue(f.value, f.unit)}</td>
                    <td>
                      {covered ? (
                        fxAvailable ? (
                          <>
                            <strong>{factValue(f.rmbValue, f.unit)}</strong>
                            <Chip tone="good">已折算</Chip>
                          </>
                        ) : (
                          <>
                            <span>{factValue(f.rmbValue, f.unit)}</span>
                            <Chip tone="warn">{fxText(false)}</Chip>
                          </>
                        )
                      ) : (
                        <span>{factValue(f.rmbValue, f.unit)}</span>
                      )}
                    </td>
                    <td className="facts-ts">{fmtTs(f.ts)}</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </section>
  );
}
