import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";

import { DashboardService } from "./gen/arbcn/dashboard/v1/dashboard_pb";
import { SimService } from "./gen/arbcn/sim/v1/sim_pb";

// 相对路径：dev 经 vite 代理到 :50052；生产由 M1-h go:embed 同源托管。
export const dashboard = createClient(
  DashboardService,
  createConnectTransport({ baseUrl: "/" }),
);

// sim 独立域（arbcn.sim.v1，D-038 ①）：模拟执行面板——确认后仍是模拟（SIMULATED），
// 无任何通往真实资金的按钮/路径（不赌原则 D-019）。
export const sim = createClient(
  SimService,
  createConnectTransport({ baseUrl: "/" }),
);
