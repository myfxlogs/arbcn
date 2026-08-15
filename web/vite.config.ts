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
  build: {
    // 不整目录清空：dist/.gitkeep（git 跟踪，保证未构建树仍可 go build，
    // 见 embed.go）须跨构建存活。旧哈希产物滞留无害（dist 内容 gitignored）。
    emptyOutDir: false,
  },
});
