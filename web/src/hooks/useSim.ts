import { useCallback, useEffect, useState } from "react";

import { sim } from "../client";
import type {
  CloseSimOrderResponse,
  GetPerformanceReportResponse, // D-062 判定门① 测量
  GetSimAccountResponse,
  GetSimReportResponse,
  SimOrder,
  SimPosition,
  TestnetAccount,
} from "../gen/arbcn/sim/v1/sim_pb";
import { POLL_MS } from "./shared";

// useSim 模拟执行面板（M3-c C4，04-m3-spec §10.5）：建议订单 + 模拟持仓 + 对账报告
// + 测试网账户（D-040）。写路径 = ConfirmSimOrder（开仓）+ CloseSimOrder（平仓，D-055
// 整单平：订单全部腿一起退，绝不单腿留裸敞口 D-019）；成功后本地刷新各区。
// SIMULATED 徽标由 SimExec 组件固定渲染（可检查）。
// D-047 P0：本 hook 由使用它的页面组件挂载（OverviewPage/SimPage），hook 生命周期 =
// 视图生命周期；加 60s 轮询（8h 结算新单可被确认面板捕获，不再只靠手动刷新）。
// refreshKey：顶部全局刷新信号（App 递增）→ effect 重载。
export function useSim(refreshKey?: number): {
  orders: SimOrder[];
  positions: SimPosition[];
  report: GetSimReportResponse | null;
  accounts: TestnetAccount[];
  account: GetSimAccountResponse | null;
  performance: GetPerformanceReportResponse | null; // D-062 判定门① 测量
  fxAvailable: boolean;
  error: string;
  confirm: (id: bigint) => Promise<boolean>;
  close: (id: bigint, note?: string) => Promise<CloseSimOrderResponse | null>;
  reload: () => void;
} {
  const [orders, setOrders] = useState<SimOrder[]>([]);
  const [positions, setPositions] = useState<SimPosition[]>([]);
  const [report, setReport] = useState<GetSimReportResponse | null>(null);
  const [accounts, setAccounts] = useState<TestnetAccount[]>([]);
  const [account, setAccount] = useState<GetSimAccountResponse | null>(null);
  const [performance, setPerformance] = useState<GetPerformanceReportResponse | null>(null);
  const [fxAvailable, setFxAvailable] = useState(false);
  const [error, setError] = useState("");
  const [tick, setTick] = useState(0);

  useEffect(() => {
    let alive = true;
    const load = async () => {
      try {
        const [ordersRes, positionsRes, reportRes, accountsRes, accountRes, perfRes] = await Promise.all([
          sim.listSimOrders({}),
          sim.listSimPositions({}),
          sim.getSimReport({}),
          sim.getTestnetAccounts({}),
          sim.getSimAccount({}),
          sim.getPerformanceReport({}), // D-062 判定门① 跨窗口测量（只读）
        ]);
        if (!alive) return;
        setOrders(ordersRes.orders);
        setPositions(positionsRes.positions);
        setReport(reportRes);
        setAccounts(accountsRes.accounts);
        setAccount(accountRes); // D-056 完整现金账本：账户净值 + 逐笔流水
        setPerformance(perfRes);
        setFxAvailable(positionsRes.fxAvailable); // D-047 F4：真实汇率可用信号（区分「USD 原值」与真零 PnL）
        setError("");
      } catch (e) {
        if (!alive) return;
        setError(String(e));
      }
    };
    void load();
    const timer = setInterval(() => void load(), POLL_MS);
    return () => {
      alive = false;
      clearInterval(timer);
    };
  }, [tick, refreshKey]);

  const confirm = useCallback(async (id: bigint): Promise<boolean> => {
    try {
      const res = await sim.confirmSimOrder({ id });
      setTick((n) => n + 1); // 刷新各区（订单状态 + 新持仓）
      setError("");
      return res.accepted;
    } catch (e) {
      setError(String(e));
      return false;
    }
  }, []);

  // close 平仓（D-055 整单平，后端订单级原子事务：订单全部 open 腿一起退，防半仓裸敞口）。
  // 返回 CloseSimOrderResponse（含 realized_pnl 实现净 PnL，前端平仓结果横幅展示）。
  const close = useCallback(async (id: bigint, note?: string): Promise<CloseSimOrderResponse | null> => {
    try {
      const res = await sim.closeSimOrder({ id, note: note ?? "" });
      setTick((n) => n + 1); // 刷新各区（订单 closed + 腿 settled）
      setError("");
      return res;
    } catch (e) {
      setError(String(e));
      return null;
    }
  }, []);

  return {
    orders,
    positions,
    report,
    accounts,
    account,
    performance,
    fxAvailable,
    error,
    confirm,
    close,
    reload: useCallback(() => setTick((n) => n + 1), []),
  };
}
