// 模拟执行共享展示辅助（SimExec / ConfirmPanel 共用，对话 #59 抽出）。
// SIMULATED 徽标 + 中文文案映射集中于此（C5 可检查锚点：sim.tsx 含固定 SIMULATED /
// 「模拟」渲染，TestSimExecBadgeRenderable 机械检查）。
// 任何新增套利 kind / 腿方向 / 风险标记都要同步此表（practices #18：枚举映射全量对齐）。

export const SIMULATED = "SIMULATED";
export const SIMULATED_CN = "模拟";

// SimTag 每行固定「模拟」标注（C5 可检查锚点）。
export function SimTag() {
  return <span className="sim-tag">{SIMULATED_CN}</span>;
}

// SimulatedBadge 面板顶部固定徽标（C5 可检查锚点：SIMULATED 常量固定渲染）。
export function SimulatedBadge() {
  return (
    <span className="sim-badge" title="本地模拟成交，不接真实资金">
      {SIMULATED} · 不接真实资金
    </span>
  );
}

// kindText 套利类型 → 中文。
export function kindText(kind: string): string {
  switch (kind) {
    case "funding_hedge":
      return "现货+永续对冲";
    case "carry_asset":
      return "白名单生息";
    case "repo":
      return "逆回购";
    default:
      return kind;
  }
}

// sideText 订单方向 → 中文。
export function sideText(side: string): string {
  switch (side) {
    case "long":
      return "多";
    case "short":
      return "空";
    case "hedge":
      return "对冲";
    default:
      return side;
  }
}

// legSideText 持仓腿方向 → 中文。
export function legSideText(side: string): string {
  switch (side) {
    case "long":
      return "现货多";
    case "short":
      return "永续空";
    default:
      return side;
  }
}
