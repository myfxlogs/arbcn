import { useState, type ReactNode } from "react";

// Collapse 折叠块（D-048 U2/U3/U4）：受控 <details>。title=标题行文本；
// hint=行尾灰色提示；defaultOpen=初始展开态（数据面默认开 / 低频面默认折叠）。
// 用原生 <details> 而非自建 open state：零 JS 依赖、原生无障碍、可 grep（A 原则）。
export function Collapse({
  title,
  hint,
  defaultOpen = false,
  children,
}: {
  title: string;
  hint?: string;
  defaultOpen?: boolean;
  children: ReactNode;
}) {
  const [open, setOpen] = useState(defaultOpen);
  return (
    <details
      className="collapse"
      open={open}
      onToggle={(e) => setOpen((e.target as HTMLDetailsElement).open)}
    >
      <summary className="collapse-summary">
        <span className="collapse-chevron" aria-hidden="true">
          ▸
        </span>
        <span className="collapse-title">{title}</span>
        {hint ? <span className="collapse-hint">{hint}</span> : null}
      </summary>
      {children}
    </details>
  );
}
