import type { FreshDot } from "./freshness";
import { FreshDot as FreshDotDot } from "./FreshDot";

export interface MatrixCell {
  value: string;
  title?: string;
  neg?: boolean;
  dot?: FreshDot;
}

// MatrixTable 行×列矩阵（funding 币×所、稳定币档位项目×币种共用）。
export function MatrixTable({
  rows,
  cols,
  cell,
  empty,
  colLabel,
}: {
  rows: string[];
  cols: string[];
  cell: (row: string, col: string) => MatrixCell | null;
  empty: string;
  // colLabel 列头显示名（key 不变，仅展示层缩写——防长协议名撑宽矩阵出横向滑块）。
  colLabel?: (col: string) => string;
}) {
  if (rows.length === 0) return <p className="empty">{empty}</p>;
  return (
    <div className="table-scroll">
      <table className="matrix">
        <thead>
          <tr>
            <th scope="col" aria-label="行头" />
            {cols.map((c) => (
              <th scope="col" key={c}>
                {colLabel ? colLabel(c) : c}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {rows.map((r) => (
            <tr key={r}>
              <th scope="row">{r}</th>
              {cols.map((c) => {
                const v = cell(r, c);
                return (
                  <td key={c} title={v?.title} className={v?.neg ? "neg" : undefined}>
                    {v ? (
                      <>
                        {v.dot ? <FreshDotDot dot={v.dot} /> : null}
                        {v.value}
                      </>
                    ) : (
                      "—"
                    )}
                  </td>
                );
              })}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
