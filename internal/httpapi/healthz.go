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

// PendingMigrations 返回尚未应用的迁移文件（空 = 全部应用）。
// 由 pgstore.PendingMigrations 适配；nil = 未启用该检查。
type PendingMigrations func(ctx context.Context) ([]string, error)

// Healthz 报告进程存活、数据库可达性与迁移应用状态（§10 元监控入口）。
// 未应用迁移 = degraded（dialogue #23 裁决：PG 恢复后进程补迁移依赖重启）。
type Healthz struct {
	DB         Pinger
	Migrations PendingMigrations
	// BootErr 记录装配期致命错误（如 DSN 非法 → 数据管线整体跳过）；非 nil 时
	// /healthz 恒报 degraded（boot_error），让"进程存活但管线死亡"可见
	// （R5#1 裁定：原实现 DB/Migrations 全 nil 时跳过全部检查返回 ok，静默死亡无信号）。
	BootErr error
}

func (h *Healthz) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.BootErr != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "degraded", "reason": "boot_error"})
		return
	}
	if h.DB != nil {
		if err := h.DB.Ping(r.Context()); err != nil {
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "degraded", "reason": "db_unreachable"})
			return
		}
	}
	if h.Migrations != nil {
		pending, err := h.Migrations(r.Context())
		switch {
		case err != nil:
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "degraded", "reason": "migrations_check_failed"})
			return
		case len(pending) > 0:
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{"status": "degraded", "reason": "pending_migrations"})
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
