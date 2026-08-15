package web

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// TestHandlerDevDir：ARBCN_WEB_DIR 覆盖分支——服务本地目录而非嵌入 dist。
func TestHandlerDevDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<h1>dev</h1>"), 0o644); err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	Handler(dir).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusOK || rec.Body.String() != "<h1>dev</h1>" {
		t.Fatalf("dev dir serve = %d %q, want 200 <h1>dev</h1>", rec.Code, rec.Body.String())
	}
}

// TestHandlerEmbed：嵌入分支——dist 缺 index.html（未构建）时 404 而非 500。
func TestHandlerEmbed(t *testing.T) {
	h := Handler("")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if _, err := dist.ReadFile("dist/index.html"); err == nil {
		// dist 已构建：嵌入分支应正常服务首页。
		if rec.Code != http.StatusOK {
			t.Fatalf("embedded index.html serve = %d, want 200", rec.Code)
		}
		return
	}
	if rec.Code != http.StatusNotFound {
		t.Fatalf("unbuilt dist serve = %d, want 404", rec.Code)
	}
}
