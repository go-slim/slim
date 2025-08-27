package slim

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	return p
}

func TestStatic_ServeFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "hello.txt", "hi")

	s := New()
	s.Use(Static(dir))

	r := httptest.NewRequest(http.MethodGet, "http://example.com/hello.txt", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	if body := w.Body.String(); body != "hi" {
		t.Fatalf("body=%q", body)
	}
}

func TestStatic_ServeDirectoryIndex(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "dir/index.html", "index-ok")

	s := New()
	s.Use(Static(dir))

	r := httptest.NewRequest(http.MethodGet, "http://example.com/dir/", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	if body := w.Body.String(); body != "index-ok" {
		t.Fatalf("body=%q", body)
	}
}

func TestStatic_HTML5_FallbackToIndexOn404FromNext(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "index.html", "spa-index")

	s := New()
	cfg := StaticConfig{Root: dir, HTML5: true}
	s.Use(cfg.ToMiddleware())
	// Next returns *HTTPError{404} so Static will serve index.html
	s.Use(func(c Context, next HandlerFunc) error { return &HTTPError{Code: http.StatusNotFound, Message: "not found"} })

	r := httptest.NewRequest(http.MethodGet, "http://example.com/does-not-exist", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	if body := w.Body.String(); body != "spa-index" {
		t.Fatalf("body=%q", body)
	}
}

func TestIsIgnorableOpenFileError(t *testing.T) {
	if !isIgnorableOpenFileError(os.ErrNotExist) {
		t.Fatalf("expected true for os.ErrNotExist")
	}
}
