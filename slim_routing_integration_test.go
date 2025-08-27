package slim

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-slim.dev/l4g"
)

// helper error handler that writes a fixed body and status
func ehWrite(body string, code int) ErrorHandlerFunc {
	return func(c Context, err error) {
		_ = c.String(code, body)
	}
}

func TestIntegration_GlobalMiddlewareErrorUsesAppErrorHandler(t *testing.T) {
	s := New()
	// silence logs
	s.StdLogger = nil
	s.Logger = l4g.New(io.Discard)

	s.ErrorHandler = ehWrite("app", http.StatusInternalServerError)

	s.Use(func(c Context, next HandlerFunc) error {
		return errors.New("boom-in-mw")
	})

	s.GET("/ok", func(c Context) error {
		return c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", rec.Code)
	}
	if got := rec.Body.String(); got != "app" {
		t.Fatalf("want body app, got %q", got)
	}
}

func TestIntegration_CollectorErrorHandlerPreferred(t *testing.T) {
	s := New()
	s.StdLogger = nil
	s.Logger = l4g.New(io.Discard)

	s.Route("/api", func(sub RouteCollector) {
		sub.UseErrorHandler(ehWrite("collector", http.StatusInternalServerError))
		sub.GET("/x", func(c Context) error { return errors.New("x") })
	})

	req := httptest.NewRequest(http.MethodGet, "/api/x", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", rec.Code)
	}
	if got := rec.Body.String(); got != "collector" {
		t.Fatalf("want body collector, got %q", got)
	}
}

func TestIntegration_RouterErrorHandlerWhenNoCollectorHandler(t *testing.T) {
	s := New()
	s.StdLogger = nil
	s.Logger = l4g.New(io.Discard)

	// set router-level error handler
	if r, ok := s.Router().(*routerImpl); ok {
		r.UseErrorHandler(ehWrite("router", http.StatusInternalServerError))
	} else {
		t.Fatal("default router is not routerImpl")
	}

	s.GET("/x", func(c Context) error { return errors.New("x") })

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", rec.Code)
	}
	if got := rec.Body.String(); got != "router" {
		t.Fatalf("want body router, got %q", got)
	}
}

func TestIntegration_DefaultErrorHandler500(t *testing.T) {
	s := New()
	s.StdLogger = nil
	s.Logger = l4g.New(io.Discard)

	// remove custom error handlers to use DefaultErrorHandler
	s.ErrorHandler = DefaultErrorHandler

	s.GET("/err", func(c Context) error { return errors.New("oops") })

	req := httptest.NewRequest(http.MethodGet, "/err", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", rec.Code)
	}
	if got := rec.Body.String(); got != "oops\n" { // http.Error appends newline
		t.Fatalf("want body 'oops\\n', got %q", got)
	}
}

func TestIntegration_MethodNotAllowed405(t *testing.T) {
	s := New()
	s.StdLogger = nil
	s.Logger = l4g.New(io.Discard)

	s.POST("/m", func(c Context) error { return c.String(http.StatusOK, "posted") })

	req := httptest.NewRequest(http.MethodGet, "/m", nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("want 405, got %d", rec.Code)
	}
	if got := rec.Header().Get("Allow"); got == "" || !strings.Contains(got, http.MethodPost) || strings.Contains(got, http.MethodGet) {
		t.Fatalf("want Allow header contains POST and not GET, got %q", got)
	}
	if got := rec.Body.String(); got != http.StatusText(http.StatusMethodNotAllowed)+"\n" {
		t.Fatalf("unexpected body: %q", got)
	}
}
