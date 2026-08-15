// arbcn monitor：采集→归一→规则→告警数据管线入口（docs/design/02-monitor-architecture.md §1）。
// M1-h 总装：config → PG store（迁移先行）→ collector 注册表 + Scheduler
// （ARBCN_COLLECT_SOURCES 决定启停）→ 心跳发射方（挂 Scheduler.OnSuccess）→ 规则引擎
// （Seed 默认规则 + Run）→ SMTP Alerter（SMTP.Configured() 门控，dialogue #27）→
// DashboardService + 人工录入 + go:embed 仪表盘，单端口 :50052。
// 铁律：只读公开 API、无密钥、资金动作永远人工（§1/§13）。
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"arbcn/internal/alert"
	"arbcn/internal/collect"
	"arbcn/internal/collect/calendar"
	"arbcn/internal/collect/defirate"
	"arbcn/internal/collect/domestic"
	"arbcn/internal/collect/exchange"
	"arbcn/internal/collect/fx"
	"arbcn/internal/collect/manual"
	"arbcn/internal/collect/optionsiv"
	"arbcn/internal/config"
	"arbcn/internal/dashboard"
	"arbcn/internal/httpapi"
	"arbcn/internal/rule"
	"arbcn/internal/store"
	"arbcn/internal/store/pgstore"
	"arbcn/web"
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

	// PG 启动失败不阻断进程：/healthz 报告 degraded（dialogue #22）。两级区分：
	// 池创建失败（DSN 非法）= 管线整体跳过（healthz 只报存活）；
	// Ping 失败（PG 暂不可达）= 管线照常启动，Sink 失败走调度器退避，PG 恢复后自愈。
	pool, err := pgxpool.New(ctx, cfg.PGDSN)
	if err != nil {
		slog.Warn("pg pool init failed, data pipeline skipped", "err", err)
	} else {
		defer pool.Close()
		if err := pool.Ping(ctx); err != nil {
			slog.Warn("postgres unreachable at boot, pipeline retries via backoff", "err", err)
		} else {
			slog.Info("postgres connected")
			// 版本化迁移（schema_migrations 记账）。PG 可达但迁移失败 = schema 契约
			// 不成立，fail fast 交给 systemd Restart=on-failure 重试；未应用迁移
			// 的 degraded 状态由 /healthz 的 pending_migrations 检查覆盖（dialogue #23）。
			if n, err := pgstore.Migrate(ctx, pool, cfg.MigrationsDir); err != nil {
				return fmt.Errorf("migrate: %w", err)
			} else if n > 0 {
				slog.Info("migrations applied", "count", n)
			}
		}
	}

	var st store.Store
	if pool != nil {
		st = pgstore.New(pool)
	}

	enabled, err := collect.LoadSources(os.Getenv("ARBCN_COLLECT_SOURCES"), allSources())
	if err != nil {
		return err
	}

	// 单端口 :50052：/healthz + ConnectRPC + 人工录入 + 嵌入式仪表盘（"/" 兜底）。
	migrations := pendingMigrations(pool, cfg.MigrationsDir)
	mux := http.NewServeMux()
	mux.Handle("/healthz", &httpapi.Healthz{DB: pool, Migrations: migrations})
	if st != nil {
		path, h := dashboard.New(st, pool, migrations).Handler()
		mux.Handle(path, h)
	}
	mux.Handle("/manual/fact", manual.NewHandler(st)) // 人工录入降级通道（store 未接线时 503）
	mux.Handle("/", web.Handler(cfg.WebDir))

	errCh := make(chan error, 8)
	if st != nil {
		if err := startPipeline(ctx, errCh, st, cfg.SMTP, enabled); err != nil {
			return err
		}
	}

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}
	go func() { errCh <- srv.ListenAndServe() }()
	slog.Info("arbcn monitor started", "addr", cfg.Addr,
		"sources", sourceNames(enabled), "smtp_configured", cfg.SMTP.Configured())

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

// startPipeline 装配数据管线（store 可用时）：调度器 + 心跳发射方 → 规则引擎 →
// SMTP Alerter。各组件 Run 阻塞至 ctx 取消（返回 nil）；装配错误 fail fast。
func startPipeline(ctx context.Context, errCh chan<- error, st store.Store, smtp alert.SMTPConfig, sources []collect.Named) error {
	hb := &alert.Heartbeat{St: st}
	for _, src := range sources {
		hb.Track(src.Name, src.Interval)
	}
	sched := &collect.Scheduler{
		Sources:   sources,
		Sink:      st.InsertFacts,
		OnSuccess: hb.Record, // 心跳契约：登记源最近成功轮询（alert.Heartbeat）
	}
	go func() { errCh <- sched.Run(ctx) }()
	go func() { errCh <- hb.Run(ctx) }()

	if n, err := rule.Seed(ctx, st); err != nil {
		return fmt.Errorf("rule seed: %w", err)
	} else if n > 0 {
		slog.Info("rules seeded", "count", n)
	}
	engine, err := rule.New(ctx, st, rule.Config{})
	if err != nil {
		return fmt.Errorf("rule engine: %w", err)
	}
	go func() { errCh <- engine.Run(ctx) }()

	// SMTP 接线（dialogue #27 门控 + D-032 修订）：未配置或配置非法 → warn +
	// 降级禁用（告警留在 alerts 表排队，进程不退出）；合法 → 启动消费循环。
	(&alert.Alerter{St: st, SMTP: smtp}).Start(ctx, errCh)
	return nil
}

// allSources 汇总全部数据源默认清单（name + 默认间隔）；ARBCN_COLLECT_SOURCES
// 决定启用与间隔覆盖（collect.LoadSources）。
func allSources() []collect.Named {
	var out []collect.Named
	out = append(out, exchange.All(exchange.FromEnv(os.Getenv))...)
	out = append(out, defirate.All(defirate.FromEnv(os.Getenv))...)
	out = append(out, domestic.All(domestic.FromEnv(os.Getenv))...)
	out = append(out, fx.All(fx.FromEnv(os.Getenv))...)
	out = append(out, calendar.All(calendar.FromEnv(os.Getenv))...)
	out = append(out, optionsiv.All(optionsiv.FromEnv(os.Getenv))...)
	return out
}

// pendingMigrations 把 pgstore.PendingMigrations 适配为 httpapi.PendingMigrations
// （/healthz 与 DashboardService.Health 同源复用）；pool nil = 不启用检查。
func pendingMigrations(pool *pgxpool.Pool, dir string) httpapi.PendingMigrations {
	if pool == nil {
		return nil
	}
	return func(ctx context.Context) ([]string, error) {
		return pgstore.PendingMigrations(ctx, pool, dir)
	}
}

// sourceNames 供启动日志列出启用源。
func sourceNames(srcs []collect.Named) []string {
	names := make([]string, 0, len(srcs))
	for _, s := range srcs {
		names = append(names, s.Name)
	}
	return names
}
