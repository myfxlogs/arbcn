// Package config：环境变量配置（docs/design/02-monitor-architecture.md §2/§10）。
package config

import (
	"os"
	"strings"

	"arbcn/internal/alert"
)

// Config 是单二进制部署所需的最小配置面。
type Config struct {
	Addr          string // HTTP 监听地址（ConnectRPC + 静态仪表盘共用单端口 :50052）
	PGDSN         string // PostgreSQL DSN（arbcn-postgres 容器，宿主端口 5434）
	MigrationsDir string // 版本化迁移 SQL 目录（启动时执行未记账文件，schema_migrations 记账）
	AlertEmail    string // 遗留占位：M1-f 前无告警通道（main.go 仅日志打印）；M1-h 接线改读 SMTP.To 后删除
	SMTP          alert.SMTPConfig
}

// Load 从环境变量读取配置，未设置时取默认值。
func Load() Config {
	return Config{
		Addr:          getenv("ARBCN_ADDR", ":50052"),
		PGDSN:         getenv("ARBCN_PG_DSN", "postgres://arbcn:arbcn@localhost:5434/arbcn?sslmode=disable"),
		MigrationsDir: getenv("ARBCN_MIGRATIONS_DIR", "migrations"),
		AlertEmail:    os.Getenv("ARBCN_ALERT_EMAIL"),
		SMTP: alert.SMTPConfig{
			Host: os.Getenv("ARBCN_SMTP_HOST"),
			User: os.Getenv("ARBCN_SMTP_USER"),
			Pass: os.Getenv("ARBCN_SMTP_PASS"),
			From: os.Getenv("ARBCN_SMTP_FROM"),
			To:   splitList(os.Getenv("ARBCN_SMTP_TO")),
		},
	}
}

// splitList 逗号分隔列表（SMTP 多收件人）；空白项丢弃。
func splitList(s string) []string {
	out := []string{}
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
