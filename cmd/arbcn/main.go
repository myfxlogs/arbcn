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
	"arbcn/internal/exporter"
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
	healthz := &httpapi.Healthz{DB: pool, Migrations: migrations}
	if err != nil {
		// DSN 解析失败（pool==nil）→ 管线整体跳过；/healthz 必须报 degraded 而非谎报 ok
		//（R5#1：原实现 DB/Migrations 全 nil → 跳过全部检查返回 ok，管线静默死亡无信号）。
		healthz.BootErr = err
	}
	mux := http.NewServeMux()
	mux.Handle("/healthz", healthz)
	if st != nil {
		// 源健康信息（name/interval_sec/kind）供 ListSourceHealth 数据面（M2-a §2.2）。
		path, h := dashboard.New(st, pool, migrations, sourceInfos(enabled)).Handler()
		mux.Handle(path, h)
	}
	mux.Handle("/manual/fact", manual.NewHandler(st)) // 人工录入降级通道（store 未接线时 503）
	mux.Handle("/", web.Handler(cfg.WebDir))

	errCh := make(chan error, 8)
	if st != nil {
		if err := startPipeline(ctx, errCh, st, cfg.SMTP, cfg.FactsPath, enabled); err != nil {
			return err
		}
	}

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		// R5#6 裁定：原实现无响应体/空闲超时，慢请求（大 POST body / 慢查询）滞留时
		// SIGTERM 优雅退出 Shutdown 5s 超时 → exit(1) → systemd on-failure 误重启。
		// WriteTimeout 30s / IdleTimeout 60s 给慢请求明确上限，保优雅退出不被拖死。
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
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

// startPipeline 装配数据管线（store 可用时）：调度器 + 心跳发射方 → 规则引擎
// （关键规则触发事件接 FactsExporter）→ SMTP Alerter → FactsExporter（facts.md 快照）。
// 各组件 Run 阻塞至 ctx 取消（返回 nil）；装配错误 fail fast。
func startPipeline(ctx context.Context, errCh chan<- error, st store.Store, smtp alert.SMTPConfig, factsPath string, sources []collect.Named) error {
	hb := &alert.Heartbeat{St: st}
	for _, src := range sources {
		hb.Track(src.Name, src.Interval)
	}
	sched := &collect.Scheduler{
		Sources:   sources,
		Sink:      st.InsertFacts,
		OnSuccess: hb.Record, // 心跳契约：登记源最近成功轮询（alert.Heartbeat）
		Dedup:     true,      // M2-a §3.1：连续重复事实去重（相同 value+ts 跳过落库）
	}
	go func() { errCh <- sched.Run(ctx) }()
	go func() { errCh <- hb.Run(ctx) }()

	if n, err := rule.Seed(ctx, st); err != nil {
		return fmt.Errorf("rule seed: %w", err)
	} else if n > 0 {
		slog.Info("rules seeded", "count", n)
	}

	// FactsExporter（M2-b §5 / D-028 闭环）：定时（日）+ 规则触发事件 → 把监控
	// 最新值渲染进 facts.md。factsPath 空 = 禁用；规则引擎 OnActive 接它的
	// 非阻塞触发（关键规则激活 → 立即刷新快照）。写文件失败只 warn 不崩管线。
	factsExporter := exporter.New(st, factsPath)
	engine, err := rule.New(ctx, st, rule.Config{
		OnActive: factsExporter.OnRuleActive,
	})
	if err != nil {
		return fmt.Errorf("rule engine: %w", err)
	}
	go func() { errCh <- engine.Run(ctx) }()
	if factsPath != "" {
		go func() { errCh <- factsExporter.Run(ctx) }()
		slog.Info("facts exporter started", "path", factsPath)
	}

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

// sourceInfos 把启用源清单投影为 dashboard.SourceInfo（ListSourceHealth 数据面，
// M2-a §2.2：Name / Interval / Collector.Kind()）。
func sourceInfos(srcs []collect.Named) []dashboard.SourceInfo {
	out := make([]dashboard.SourceInfo, 0, len(srcs))
	for _, s := range srcs {
		out = append(out, dashboard.SourceInfo{
			Name:        s.Name,
			IntervalSec: int64(s.Interval.Seconds()),
			Kind:        s.Collector.Kind(),
		})
	}
	return out
}
