import { defineConfig } from "vite";
import react from "@vitejs/plugin-react";

// dev 代理：ConnectRPC 转发到本机 :50052；生产由 M1-h go:embed 同源托管，无需代理。
export default defineConfig({
  plugins: [react()],
  server: {
    proxy: {
      "/arbcn.dashboard.v1.DashboardService": "http://localhost:50052",
    },
  },
});
