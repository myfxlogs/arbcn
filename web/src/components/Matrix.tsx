export interface MatrixCell {
  value: string;
  title?: string;
  neg?: boolean;
}

// MatrixTable 行×列矩阵（funding 币×所、稳定币档位项目×币种共用）。
export function MatrixTable({
  rows,
  cols,
  cell,
  empty,
}: {
  rows: string[];
  cols: string[];
  cell: (row: string, col: string) => MatrixCell | null;
  empty: string;
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
                {c}
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
                    {v ? v.value : "—"}
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
