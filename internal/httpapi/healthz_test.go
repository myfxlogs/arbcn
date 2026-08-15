package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type pingerStub struct{ err error }

func (p pingerStub) Ping(context.Context) error { return p.err }

func TestHealthzNoDB(t *testing.T) {
	h := &Healthz{}
	assertStatus(t, h, http.StatusOK)
}

// TestHealthzBootErr：[对抗测试锚点] 装配期致命错误（DSN 非法）→ 恒报 degraded
//（boot_error），不得谎报 ok（R5#1）。删掉 healthz.go 的 BootErr 分支 → 本测试必红。
func TestHealthzBootErr(t *testing.T) {
	h := &Healthz{BootErr: errors.New("invalid DSN")}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503（DSN 非法 → 管线死亡必须可见）", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "boot_error") {
		t.Fatalf("body = %q, want boot_error reason", rec.Body.String())
	}
}

func TestHealthzDBOK(t *testing.T) {
	h := &Healthz{DB: pingerStub{}}
	assertStatus(t, h, http.StatusOK)
}

func TestHealthzDBDown(t *testing.T) {
	h := &Healthz{DB: pingerStub{err: errors.New("connection refused")}}
	assertStatus(t, h, http.StatusServiceUnavailable)
}

// TestHealthzPendingMigrations：迁移未全部应用 → 503 degraded（dialogue #23）。
func TestHealthzPendingMigrations(t *testing.T) {
	h := &Healthz{Migrations: func(context.Context) ([]string, error) {
		return []string{"0003_alerts_delivered.sql"}, nil
	}}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want 503", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "pending_migrations") {
		t.Fatalf("body = %q, want pending_migrations reason", rec.Body.String())
	}
}

func TestHealthzMigrationsApplied(t *testing.T) {
	h := &Healthz{Migrations: func(context.Context) ([]string, error) { return nil, nil }}
	assertStatus(t, h, http.StatusOK)
}

func TestHealthzMigrationsCheckFailed(t *testing.T) {
	h := &Healthz{Migrations: func(context.Context) ([]string, error) {
		return nil, errors.New("schema_migrations missing")
	}}
	assertStatus(t, h, http.StatusServiceUnavailable)
}

func assertStatus(t *testing.T, h *Healthz, want int) {
	t.Helper()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if rec.Code != want {
		t.Fatalf("status = %d, want %d", rec.Code, want)
	}
}
