package slim

import (
	"io"
	"fmt"
	"strconv"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-slim.dev/l4g"
)

// benchServe runs b.N requests against s after setup, asserting expected status.
func benchServe(b *testing.B, setup func(s *Slim), method, path string, want int) {
	b.Helper()

	s := New()
	// Silence logs to avoid benchmark noise
	s.StdLogger = nil
	s.Logger = l4g.New(io.Discard)

	setup(s)

	req := httptest.NewRequest(method, path, nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		s.ServeHTTP(rr, req)
		if rr.Code != want {
			b.Fatalf("unexpected status: got=%d want=%d", rr.Code, want)
		}
	}
}

func BenchmarkRouter_Simple(b *testing.B) {
	benchServe(b, func(s *Slim) {
		s.GET("/hello", func(c Context) error { return c.String(http.StatusOK, "ok") })
	}, http.MethodGet, "/hello", http.StatusOK)
}

func BenchmarkRouter_Param(b *testing.B) {
	benchServe(b, func(s *Slim) {
		s.GET("/users/:id", func(c Context) error { return c.String(http.StatusOK, c.Param("id")) })
	}, http.MethodGet, "/users/12345", http.StatusOK)
}

func BenchmarkRouter_Wildcard(b *testing.B) {
	benchServe(b, func(s *Slim) {
		s.GET("/static/*filepath", func(c Context) error { return c.NoContent(http.StatusOK) })
	}, http.MethodGet, "/static/css/app.css", http.StatusOK)
}

// makeNMiddlewares returns n middlewares chained in order.
func makeNMiddlewares(n int) []MiddlewareFunc {
	mws := make([]MiddlewareFunc, 0, n)
	for i := 0; i < n; i++ {
		mws = append(mws, func(next HandlerFunc) HandlerFunc {
			return func(c Context) error { return next(c) }
		})
	}
	return mws
}

func BenchmarkMiddleware_ChainDepth(b *testing.B) {
	for _, depth := range []int{0, 1, 5, 10} {
		b.Run("depth="+strconv.Itoa(depth), func(b *testing.B) {
			d := depth // capture
			benchServe(b, func(s *Slim) {
				// apply middlewares then a simple handler
				mws := makeNMiddlewares(d)
				if len(mws) > 0 {
					s.Use(mws...)
				}
				s.GET("/mw", func(c Context) error { return c.NoContent(http.StatusOK) })
			}, http.MethodGet, "/mw", http.StatusOK)
		})
	}
}
