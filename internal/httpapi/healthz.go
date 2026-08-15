// Package httpapi：HTTP 端点。M1-a 只有 /healthz；ConnectRPC 与嵌入式仪表盘后续接入同端口。
package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
)

// Pinger 由 pgxpool.Pool 满足；DB 为 nil 时 /healthz 只报告进程存活。
type Pinger interface {
	Ping(ctx context.Context) error
}

// Healthz 报告进程存活与数据库可达性（§10 元监控入口）。
type Healthz struct {
	DB Pinger
}

func (h *Healthz) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.DB != nil {
		if err := h.DB.Ping(r.Context()); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "degraded"})
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}
