// arbcn monitor：采集→归一→规则→告警数据管线入口（docs/design/02-monitor-architecture.md §1）。
// 铁律：只读公开 API、无密钥、资金动作永远人工（§1/§13）。
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"arbcn/internal/config"
	"arbcn/internal/httpapi"
)

func main() {
	if err := run(); err != nil {
		slog.Error("arbcn exited", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load()
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// PG 启动失败不阻断进程：/healthz 报告 degraded，元监控（M1-f）负责 critical 告警。
	pool, err := pgxpool.New(ctx, cfg.PGDSN)
	if err != nil {
		slog.Warn("pg pool init failed, /healthz reports liveness only", "err", err)
		pool = nil
	} else {
		defer pool.Close()
		if err := pool.Ping(ctx); err != nil {
			slog.Warn("postgres unreachable at boot", "err", err)
		} else {
			slog.Info("postgres connected")
		}
	}

	mux := http.NewServeMux()
	mux.Handle("/healthz", &httpapi.Healthz{DB: pool})
	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()
	slog.Info("arbcn monitor started", "addr", cfg.Addr, "alert_email", cfg.AlertEmail)

	select {
	case <-ctx.Done():
		slog.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return srv.Shutdown(shutdownCtx)
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}
