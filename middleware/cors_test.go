package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-slim.dev/slim"
	"go-slim.dev/slim/nego"
)

func newAppWith(mws ...slim.MiddlewareFunc) *slim.Slim {
	s := slim.New()
	if len(mws) > 0 {
		s.Use(mws...)
	}
	s.GET("/", func(c slim.Context) error { return c.String(http.StatusOK, "ok") })
	return s
}

func performReq(t *testing.T, s *slim.Slim, method, target string, hdr map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, target, nil)
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	rw := httptest.NewRecorder()
	s.ServeHTTP(rw, req)
	return rw
}

func TestCORS_SimpleRequest_AllowedOrigin(t *testing.T) {
	s := newAppWith(CORSWithConfig(CORSConfig{AllowOrigins: []string{"http://example.com"}}))
	rw := performReq(t, s, http.MethodGet, "http://host/", map[string]string{
		nego.HeaderOrigin: "http://example.com",
	})
	if rw.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rw.Code)
	}
	if got := rw.Header().Get(nego.HeaderAccessControlAllowOrigin); got != "http://example.com" {
		t.Fatalf("missing allow-origin, got %q", got)
	}
}

func TestCORS_Preflight_DefaultAllowHeadersMirror(t *testing.T) {
	s := newAppWith(CORSWithConfig(CORSConfig{AllowOrigins: []string{"http://example.com"}}))
	rw := performReq(t, s, http.MethodOptions, "http://host/", map[string]string{
		nego.HeaderOrigin:                        "http://example.com",
		nego.HeaderAccessControlRequestMethod:    "GET",
		nego.HeaderAccessControlRequestHeaders:   "X-Custom, X-Other",
	})
	if rw.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rw.Code)
	}
	if got := rw.Header().Get(nego.HeaderAccessControlAllowHeaders); got != "X-Custom, X-Other" {
		t.Fatalf("expected mirror headers, got %q", got)
	}
}

func TestCORS_DisallowedOrigin_NoHeaders(t *testing.T) {
	s := newAppWith(CORSWithConfig(CORSConfig{AllowOrigins: []string{"http://foo.com"}}))
	rw := performReq(t, s, http.MethodGet, "http://host/", map[string]string{
		nego.HeaderOrigin: "http://bar.com",
	})
	if rw.Header().Get(nego.HeaderAccessControlAllowOrigin) != "" {
		t.Fatalf("should not set allow-origin for disallowed origin")
	}
}

func TestCORS_WildcardWithCredentials_ReflectOrigin(t *testing.T) {
	s := newAppWith(CORSWithConfig(CORSConfig{AllowOrigins: []string{"*"}, AllowCredentials: true}))
	rw := performReq(t, s, http.MethodGet, "http://host/", map[string]string{
		nego.HeaderOrigin: "http://site.com",
	})
	if got := rw.Header().Get(nego.HeaderAccessControlAllowOrigin); got != "http://site.com" {
		t.Fatalf("expected reflect origin, got %q", got)
	}
	if rw.Header().Get(nego.HeaderAccessControlAllowCredentials) != "true" {
		t.Fatalf("expected credentials true")
	}
}

func TestCORS_AllowOriginFunc_Error(t *testing.T) {
	s := newAppWith(CORSWithConfig(CORSConfig{AllowOriginFunc: func(origin string) (bool, error) {
		return false, fmt.Errorf("boom")
	}}))
	rw := performReq(t, s, http.MethodGet, "http://host/", map[string]string{
		nego.HeaderOrigin: "http://x.com",
	})
	// error from AllowOriginFunc should bubble up to ErrorHandler -> 500
	if rw.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rw.Code)
	}
}
