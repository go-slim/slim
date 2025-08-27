package slim

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDefaultErrorHandler_NotFound(t *testing.T) {
	s := New()
	s.GET("/known", func(c Context) error { return c.String(http.StatusOK, "ok") })

	r := httptest.NewRequest(http.MethodGet, "http://example.com/unknown", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("want 404, got %d", w.Code)
	}
	if body := w.Body.String(); body != "404 page not found\n" {
		t.Fatalf("unexpected body: %q", body)
	}
}

func TestDefaultErrorHandler_MethodNotAllowed(t *testing.T) {
	s := New()
	s.GET("/onlyget", func(c Context) error { return c.String(http.StatusOK, "ok") })

	r := httptest.NewRequest(http.MethodPost, "http://example.com/onlyget", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("want 405, got %d", w.Code)
	}
	if allow := w.Header().Get("Allow"); allow == "" || allow != "GET" {
		t.Fatalf("want Allow=GET, got %q", allow)
	}
	if body := w.Body.String(); body != http.StatusText(http.StatusMethodNotAllowed)+"\n" {
		t.Fatalf("unexpected body: %q", body)
	}
}

func TestDefaultErrorHandler_InternalServerError(t *testing.T) {
	s := New()
	s.GET("/boom", func(c Context) error { return errors.New("oops") })

	r := httptest.NewRequest(http.MethodGet, "http://example.com/boom", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("want 500, got %d", w.Code)
	}
	if body := w.Body.String(); body != "oops\n" {
		t.Fatalf("unexpected body: %q", body)
	}
}
