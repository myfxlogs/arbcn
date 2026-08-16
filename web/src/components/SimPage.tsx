import { useSim } from "../hooks";
import { SimExec } from "./SimExec";

// SimPage 模拟执行页（D-047 P0）：useSim 随本页挂载/卸载——仅在 sim tab 轮询（消除
// 跨 tab 空转）。总览页 ConfirmPanel 确认 → 切到此页挂载即重拉最新（对话 #59 曾提升
// App 层共享「确认后两处同刷新」，现由挂载即重载天然覆盖，简化回归见 D-047）。
export function SimPage() {
  const sim = useSim();
  return (
    <SimExec
      positions={sim.positions}
      accounts={sim.accounts}
      account={sim.account}
      report={sim.report}
      fxAvailable={sim.fxAvailable}
      error={sim.error}
      close={sim.close}
      reload={sim.reload}
    />
  );
}
