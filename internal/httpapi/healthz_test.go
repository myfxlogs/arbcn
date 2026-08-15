package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type pingerStub struct{ err error }

func (p pingerStub) Ping(context.Context) error { return p.err }

func TestHealthzNoDB(t *testing.T) {
	h := &Healthz{}
	assertStatus(t, h, http.StatusOK)
}

func TestHealthzDBOK(t *testing.T) {
	h := &Healthz{DB: pingerStub{}}
	assertStatus(t, h, http.StatusOK)
}

func TestHealthzDBDown(t *testing.T) {
	h := &Healthz{DB: pingerStub{err: errors.New("connection refused")}}
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
