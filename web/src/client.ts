import { createClient } from "@connectrpc/connect";
import { createConnectTransport } from "@connectrpc/connect-web";

import { DashboardService } from "./gen/arbcn/dashboard/v1/dashboard_pb";

// 相对路径：dev 经 vite 代理到 :50052；生产由 M1-h go:embed 同源托管。
export const dashboard = createClient(
  DashboardService,
  createConnectTransport({ baseUrl: "/" }),
);
