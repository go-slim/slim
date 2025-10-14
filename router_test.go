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

// TestRouter_Any_WildcardMethod tests that s.Any() correctly matches all HTTP methods
// This test verifies the fix for wildcard method matching in tree.go:78
func TestRouter_Any_WildcardMethod(t *testing.T) {
	s := newSlimTest()

	// Register a route using Any() which uses "*" as the method
	s.Any("/api/test", func(c Context) error {
		return c.String(http.StatusOK, "method="+c.Request().Method)
	})

	// Test all standard HTTP methods
	methods := []string{
		http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodDelete,
		http.MethodPatch,
		http.MethodHead,
		http.MethodOptions,
	}

	for _, method := range methods {
		t.Run(method, func(t *testing.T) {
			rw := perform(t, s, method, "/api/test", nil, nil)
			if rw.Code != http.StatusOK {
				t.Errorf("method %s: expected 200, got %d", method, rw.Code)
			}
			// HEAD requests don't return body
			if method != http.MethodHead {
				expected := "method=" + method
				if body := rw.Body.String(); body != expected {
					t.Errorf("method %s: expected body %q, got %q", method, expected, body)
				}
			}
		})
	}
}

// TestRouter_Any_WithWildcard tests that s.Any() works with wildcard path routes
func TestRouter_Any_WithWildcard(t *testing.T) {
	s := newSlimTest()

	// Register wildcard route with Any()
	s.Any("/files/*", func(c Context) error {
		path := c.PathParam("*")
		return c.String(http.StatusOK, "path="+path)
	})

	testCases := []struct {
		method string
		path   string
		want   string
	}{
		{http.MethodGet, "/files/doc.txt", "path=doc.txt"},
		{http.MethodPost, "/files/data.json", "path=data.json"},
		{http.MethodPut, "/files/a/b/c", "path=a/b/c"},
		{http.MethodDelete, "/files/old.log", "path=old.log"},
	}

	for _, tc := range testCases {
		t.Run(tc.method+"_"+tc.path, func(t *testing.T) {
			rw := perform(t, s, tc.method, tc.path, nil, nil)
			if rw.Code != http.StatusOK {
				t.Errorf("expected 200, got %d", rw.Code)
			}
			if body := rw.Body.String(); body != tc.want {
				t.Errorf("expected body %q, got %q", tc.want, body)
			}
		})
	}
}
