package slim

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVHost_ExactMatchAndWildcardFallback(t *testing.T) {
	s := New()

	rDefault := s.Router()
	rDefault.GET("/", func(c Context) error { return c.String(http.StatusOK, "default") })

	rExact := s.Host("app.example.com")
	rExact.GET("/", func(c Context) error { return c.String(http.StatusOK, "exact") })

	rWildcard := s.Host("*.example.com")
	rWildcard.GET("/", func(c Context) error { return c.String(http.StatusOK, "wildcard") })

	// 1) exact host matches specific router
	{
		r := httptest.NewRequest(http.MethodGet, "http://app.example.com/", nil)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code != 200 || w.Body.String() != "exact" {
			t.Fatalf("want exact, got code=%d body=%q", w.Code, w.Body.String())
		}
	}

	// 2) other subdomain should hit wildcard
	{
		r := httptest.NewRequest(http.MethodGet, "http://foo.example.com/", nil)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code != 200 || w.Body.String() != "wildcard" {
			t.Fatalf("want wildcard, got code=%d body=%q", w.Code, w.Body.String())
		}
	}

	// 3) non-domain (localhost) falls back to default router
	{
		r := httptest.NewRequest(http.MethodGet, "http://localhost/", nil)
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code != 200 || w.Body.String() != "default" {
			t.Fatalf("want default, got code=%d body=%q", w.Code, w.Body.String())
		}
	}
}

func TestVHost_XForwardedHostAndForwardedHeader(t *testing.T) {
	s := New()
	s.Router().GET("/", func(c Context) error { return c.String(http.StatusOK, "default") })
	s.Host("api.example.com").GET("/", func(c Context) error { return c.String(http.StatusOK, "api") })

	// Prefer X-Forwarded-Host
	{
		r := httptest.NewRequest(http.MethodGet, "http://ignored/", nil)
		r.Header.Set("X-Forwarded-Host", "api.example.com")
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code != 200 || w.Body.String() != "api" {
			t.Fatalf("want api, got code=%d body=%q", w.Code, w.Body.String())
		}
	}

	// If X-Forwarded-Host empty, use Forwarded: host=...
	{
		r := httptest.NewRequest(http.MethodGet, "http://ignored/", nil)
		r.Header.Set("Forwarded", "for=1.1.1.1; host=api.example.com")
		w := httptest.NewRecorder()
		s.ServeHTTP(w, r)
		if w.Code != 200 || w.Body.String() != "api" {
			t.Fatalf("want api via Forwarded, got code=%d body=%q", w.Code, w.Body.String())
		}
	}
}
