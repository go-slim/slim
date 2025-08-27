package slim

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-slim.dev/l4g"
	"strconv"
	"strings"
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
func BenchmarkRouter_MultiMethodsSamePath(b *testing.B) {
	benchServe(b, func(s *Slim) {
		s.GET("/mm", func(c Context) error { return c.NoContent(http.StatusOK) })
		s.POST("/mm", func(c Context) error { return c.NoContent(http.StatusOK) })
		s.PUT("/mm", func(c Context) error { return c.NoContent(http.StatusOK) })
		s.DELETE("/mm", func(c Context) error { return c.NoContent(http.StatusOK) })
	}, http.MethodGet, "/mm", http.StatusOK)
}

func BenchmarkRouter_LongQueryString(b *testing.B) {
	// Use a long query string to test parsing/lookup overhead
	q := strings.Repeat("a", 2048)
	benchServe(b, func(s *Slim) {
		s.GET("/q", func(c Context) error { _ = c.QueryParam("x"); return c.NoContent(http.StatusOK) })
	}, http.MethodGet, "/q?x="+q, http.StatusOK)
}

func BenchmarkRouter_LargeHeaders(b *testing.B) {
	s := New()
	s.StdLogger = nil
	s.Logger = l4g.New(io.Discard)
	s.GET("/h", func(c Context) error { return c.NoContent(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/h", nil)
	// Add many/large headers once (request reused)
	for i := 0; i < 50; i++ {
		req.Header.Set("X-K-"+strconv.Itoa(i), strings.Repeat("v", 64))
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rr := httptest.NewRecorder()
		s.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			b.Fatalf("unexpected status: %d", rr.Code)
		}
	}
}

func BenchmarkJSON_Serialize_Small(b *testing.B) {
	benchServe(b, func(s *Slim) {
		type resp struct {
			OK  bool   `json:"ok"`
			Msg string `json:"msg"`
		}
		s.GET("/json-small", func(c Context) error { return c.JSON(http.StatusOK, resp{OK: true, Msg: "hi"}) })
	}, http.MethodGet, "/json-small", http.StatusOK)
}

func BenchmarkJSON_Serialize_Large(b *testing.B) {
	benchServe(b, func(s *Slim) {
		type item struct {
			V string `json:"v"`
		}
		big := make([]item, 0, 1000)
		for i := 0; i < 1000; i++ {
			big = append(big, item{V: strings.Repeat("x", 40)})
		}
		s.GET("/json-large", func(c Context) error { return c.JSON(http.StatusOK, big) })
	}, http.MethodGet, "/json-large", http.StatusOK)
}

func BenchmarkBind_JSON_Small(b *testing.B) {
	s := New()
	s.StdLogger = nil
	s.Logger = l4g.New(io.Discard)
	type reqBody struct {
		Name string `json:"name"`
	}
	s.POST("/bind", func(c Context) error {
		var rb reqBody
		if err := c.Bind(&rb); err != nil {
			return err
		}
		return c.NoContent(http.StatusOK)
	})

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		body := bytes.NewReader([]byte(`{"name":"slim"}`))
		req := httptest.NewRequest(http.MethodPost, "/bind", body)
		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()
		s.ServeHTTP(rr, req)
		if rr.Code != http.StatusOK {
			b.Fatalf("unexpected status: %d", rr.Code)
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

// Deep path matching and multi-parameter routes
func BenchmarkRouter_DeepPath(b *testing.B) {
	benchServe(b, func(s *Slim) {
		s.GET("/a/b/c/d/e/f/g/h/i/j", func(c Context) error { return c.NoContent(http.StatusOK) })
	}, http.MethodGet, "/a/b/c/d/e/f/g/h/i/j", http.StatusOK)
}

func BenchmarkRouter_MultiParams(b *testing.B) {
	benchServe(b, func(s *Slim) {
		s.GET("/users/:uid/books/:bid/chapters/:cid", func(c Context) error {
			_ = c.PathParam("uid")
			_ = c.PathParam("bid")
			_ = c.PathParam("cid")
			return c.NoContent(http.StatusOK)
		})
	}, http.MethodGet, "/users/1/books/2/chapters/3", http.StatusOK)
}

// Large route sets
func registerManyRoutes(s *Slim, n int) {
	for i := 0; i < n; i++ {
		p := "/r/" + strconv.Itoa(i)
		s.GET(p, func(c Context) error { return c.NoContent(http.StatusOK) })
	}
}

func BenchmarkRouter_LargeRouteSet_1k(b *testing.B) {
	benchServe(b, func(s *Slim) {
		registerManyRoutes(s, 1000)
	}, http.MethodGet, "/r/999", http.StatusOK)
}

func BenchmarkRouter_LargeRouteSet_10k(b *testing.B) {
	benchServe(b, func(s *Slim) {
		registerManyRoutes(s, 10000)
	}, http.MethodGet, "/r/9999", http.StatusOK)
}

// Parallel benchmarks
func BenchmarkRouter_Parallel_Simple(b *testing.B) {
	s := New()
	s.StdLogger = nil
	s.Logger = l4g.New(io.Discard)
	s.GET("/p", func(c Context) error { return c.NoContent(http.StatusOK) })

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/p", nil)
			s.ServeHTTP(rr, req)
			if rr.Code != http.StatusOK {
				b.Fatalf("unexpected status: %d", rr.Code)
			}
		}
	})
}

func BenchmarkRouter_Parallel_Param(b *testing.B) {
	s := New()
	s.StdLogger = nil
	s.Logger = l4g.New(io.Discard)
	s.GET("/u/:id", func(c Context) error { _ = c.PathParam("id"); return c.NoContent(http.StatusOK) })

	b.ReportAllocs()
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			rr := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/u/42", nil)
			s.ServeHTTP(rr, req)
			if rr.Code != http.StatusOK {
				b.Fatalf("unexpected status: %d", rr.Code)
			}
		}
	})
}
