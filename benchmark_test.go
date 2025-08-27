package slim

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-slim.dev/l4g"
	"strconv"
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
		s.GET("/users/:id", func(c Context) error { return c.String(http.StatusOK, c.PathParam("id")) })
	}, http.MethodGet, "/users/12345", http.StatusOK)
}

func BenchmarkRouter_Wildcard(b *testing.B) {
	benchServe(b, func(s *Slim) {
		s.GET("/static/*filepath", func(c Context) error { return c.NoContent(http.StatusOK) })
	}, http.MethodGet, "/static/css/app.css", http.StatusOK)
}

func BenchmarkRouter_NotFound(b *testing.B) {
	benchServe(b, func(s *Slim) {
		// no routes registered
	}, http.MethodGet, "/nope", http.StatusNotFound)
}

func BenchmarkRouter_MethodNotAllowed(b *testing.B) {
	benchServe(b, func(s *Slim) {
		s.POST("/same", func(c Context) error { return c.NoContent(http.StatusOK) })
	}, http.MethodGet, "/same", http.StatusMethodNotAllowed)
}

func BenchmarkRouter_MultiCollectors(b *testing.B) {
	benchServe(b, func(s *Slim) {
		s.Router().Route("/api", func(gr RouteCollector) {
			gr.GET("/ping", func(c Context) error { return c.NoContent(http.StatusOK) })
		})
		s.Router().Route("/admin", func(gr RouteCollector) {
			gr.GET("/health", func(c Context) error { return c.NoContent(http.StatusOK) })
		})
	}, http.MethodGet, "/api/ping", http.StatusOK)
}

func BenchmarkVHost_Router(b *testing.B) {
	benchServe(b, func(s *Slim) {
		s.GET("/", func(c Context) error { return c.NoContent(http.StatusOK) }) // default
		s.Host("api.example.com").GET("/ping", func(c Context) error { return c.NoContent(http.StatusOK) })
	}, http.MethodGet, "http://api.example.com/ping", http.StatusOK)
}

// makeNMiddlewares returns n middlewares chained in order.
func makeNMiddlewares(n int) []MiddlewareFunc {
	mws := make([]MiddlewareFunc, 0, n)
	for i := 0; i < n; i++ {
		mws = append(mws, func(c Context, next HandlerFunc) error { return next(c) })
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

func BenchmarkResponse_BodySize_Small(b *testing.B) {
	benchServe(b, func(s *Slim) {
		payload := []byte("hello")
		s.GET("/small", func(c Context) error { return c.Blob(http.StatusOK, "text/plain", payload) })
	}, http.MethodGet, "/small", http.StatusOK)
}

func BenchmarkResponse_BodySize_Large(b *testing.B) {
	benchServe(b, func(s *Slim) {
		payload := bytes.Repeat([]byte("x"), 1<<20) // 1MB
		s.GET("/large", func(c Context) error { return c.Blob(http.StatusOK, "application/octet-stream", payload) })
	}, http.MethodGet, "/large", http.StatusOK)
}
