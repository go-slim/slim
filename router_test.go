package slim

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newSlimTest() *Slim {
	s := New()
	s.HideBanner = true
	s.HidePort = true
	s.Debug = true
	return s
}

func perform(t *testing.T, s *Slim, method, target string, body io.Reader, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, body)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	return rec
}

func TestRouter_TrailingSlash_Tolerant(t *testing.T) {
	s := newSlimTest()
	r := NewRouter(RouterConfig{RoutingTrailingSlash: true})
	if x, ok := r.(*routerImpl); ok {
		x.slim = s
	}
	s.router = r
	s.GET("/ts/", func(c Context) error { return c.String(http.StatusOK, "ok") })

	rw := perform(t, s, http.MethodGet, "/ts", nil, nil)
	if rw.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rw.Code)
	}
}

func TestRouter_TrailingSlash_Strict_405(t *testing.T) {
	s := newSlimTest()
	s.GET("/strict/", func(c Context) error { return c.NoContent(http.StatusOK) })
	rw := perform(t, s, http.MethodGet, "/strict", nil, nil)
	if rw.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", rw.Code)
	}
	if allow := rw.Header().Get("Allow"); allow != http.MethodGet {
		t.Fatalf("expected Allow=GET, got %q", allow)
	}
}

func TestRouteCollector_File_ServesAbsoluteFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "robots.txt")
	if err := os.WriteFile(file, []byte("User-agent: *\nDisallow:"), 0o644); err != nil {
		t.Fatal(err)
	}
	abs, err := filepath.Abs(file)
	if err != nil { t.Fatal(err) }

	s := newSlimTest()
	s.File("/robots.txt", abs)
	rw := perform(t, s, http.MethodGet, "/robots.txt", nil, nil)
	if rw.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rw.Code)
	}
	if !strings.Contains(rw.Body.String(), "User-agent: *") {
		t.Fatalf("unexpected body: %q", rw.Body.String())
	}

	// HEAD must be 405 as only GET route is registered
	rw = perform(t, s, http.MethodHead, "/robots.txt", nil, nil)
	if rw.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for HEAD, got %d", rw.Code)
	}
}

func TestStatic_ServesIndexAndEscapedNames(t *testing.T) {
	dir := t.TempDir()
	pub := filepath.Join(dir, "pub")
	if err := os.MkdirAll(pub, 0o755); err != nil { t.Fatal(err) }
	if err := os.WriteFile(filepath.Join(pub, "index.html"), []byte("index-ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pub, "hello world.txt"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	s := newSlimTest()
	// Mount Static middleware at the app level
	s.Use(Static(dir))

	// directory with trailing slash serves index.html
	rw := perform(t, s, http.MethodGet, "/pub/", nil, nil)
	if rw.Code != http.StatusOK || !strings.Contains(rw.Body.String(), "index-ok") {
		t.Fatalf("expected 200 with index-ok, got code=%d body=%q", rw.Code, rw.Body.String())
	}

	// url-escaped filename
	rw = perform(t, s, http.MethodGet, "/pub/hello%20world.txt", nil, nil)
	if rw.Code != http.StatusOK || rw.Body.String() != "hi" {
		t.Fatalf("expected 200 with hi, got code=%d body=%q", rw.Code, rw.Body.String())
	}

	// missing file
	rw = perform(t, s, http.MethodGet, "/pub/missing.txt", nil, nil)
	if rw.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rw.Code)
	}
}

func TestDefaultHandlers_404_500(t *testing.T) {
	s := newSlimTest()
	// 404
	rw := perform(t, s, http.MethodGet, "/nope", nil, nil)
	if rw.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rw.Code)
	}
	// 500 from returned error
	s.GET("/boom", func(c Context) error { return io.EOF })
	rw = perform(t, s, http.MethodGet, "/boom", nil, nil)
	if rw.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rw.Code)
	}
}
