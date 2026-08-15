// Package web：仪表盘静态资源托管（M1-h 接线，docs/design/02-monitor-architecture.md §2）。
// 本目录同时是 vite 前端工程（src/、package.json）；本文件是唯一 Go 入口
// （import 路径 arbcn/web）。生产路径：go:embed dist（npm run build 产物）
// 嵌入单二进制，与 ConnectRPC 同端口；dev 路径：ARBCN_WEB_DIR 非空时改由
// http.FileServer 直接服务该目录（免嵌入调试）。
// 构建顺序约定：部署前必须先 npm run build（web/dist），go build 才嵌入真实构建物；
// dist/.gitkeep 保证未构建的树仍可编译（嵌入空目录，请求 404 而非编译失败）。
package web

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed all:dist
var dist embed.FS

// Handler 返回静态文件处理器：webDir 非空 = 服务该目录（dev 模式）；
// 空 = 服务嵌入的 dist 构建物（生产路径）。
func Handler(webDir string) http.Handler {
	if webDir != "" {
		return http.FileServer(http.Dir(webDir))
	}
	sub, err := fs.Sub(dist, "dist")
	if err != nil {
		panic("web: embedded dist unavailable: " + err.Error()) // go:embed 保证存在，不可达
	}
	return http.FileServer(http.FS(sub))
}
