// Package config：环境变量配置（docs/design/02-monitor-architecture.md §2/§10）。
package config

import "os"

// Config 是单二进制部署所需的最小配置面。
type Config struct {
	Addr          string // HTTP 监听地址（ConnectRPC + 静态仪表盘共用单端口 :50052）
	PGDSN         string // PostgreSQL DSN（arbcn-postgres 容器，宿主端口 5434）
	MigrationsDir string // 版本化迁移 SQL 目录（启动时执行未记账文件，schema_migrations 记账）
	AlertEmail    string // 告警通道占位：SMTP 收件人（M1-f 启用；空 = 未配置）
}

// Load 从环境变量读取配置，未设置时取默认值。
func Load() Config {
	return Config{
		Addr:          getenv("ARBCN_ADDR", ":50052"),
		PGDSN:         getenv("ARBCN_PG_DSN", "postgres://arbcn:arbcn@localhost:5434/arbcn?sslmode=disable"),
		MigrationsDir: getenv("ARBCN_MIGRATIONS_DIR", "migrations"),
		AlertEmail:    os.Getenv("ARBCN_ALERT_EMAIL"),
	}
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
