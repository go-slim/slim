package bench

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/gin-gonic/gin"
	"github.com/gofiber/fiber/v2"
	"github.com/labstack/echo/v4"
	"go-slim.dev/l4g"
	"go-slim.dev/slim"
)

// 说明：
// - 构建 5/10 层无副作用中间件，仅调用 next()，用于衡量中间件链开销。
// - 关闭/隐藏日志与调试，避免额外影响（Slim/Echo 通过属性，Gin 使用 Release 模式，Fiber 关闭启动信息）。

// ------------------------ Slim ------------------------
func newSlimWithMW(n int) http.Handler {
	s := slim.New()
	s.HideBanner = true
	s.Debug = false
	s.StdLogger = nil
	s.Logger = l4g.New(io.Discard)
	// 构造 n 层无操作中间件
	for i := 0; i < n; i++ {
		s.Use(func(c slim.Context, next slim.HandlerFunc) error { return next(c) })
	}
	s.GET("/mw", func(c slim.Context) error { return c.String(http.StatusOK, "ok") })
	return s
}

func benchSlimMW(b *testing.B, n int) {
	h := newSlimWithMW(n)
	req := httptest.NewRequest(http.MethodGet, "/mw", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
	}
}

func BenchmarkMiddleware5_Slim(b *testing.B)  { benchSlimMW(b, 5) }
func BenchmarkMiddleware10_Slim(b *testing.B) { benchSlimMW(b, 10) }

// ------------------------ Gin ------------------------
func newGinWithMW(n int) http.Handler {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	for i := 0; i < n; i++ {
		r.Use(func(c *gin.Context) { c.Next() })
	}
	r.GET("/mw", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	return r
}

func benchGinMW(b *testing.B, n int) {
	h := newGinWithMW(n)
	req := httptest.NewRequest(http.MethodGet, "/mw", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
	}
}

func BenchmarkMiddleware5_Gin(b *testing.B)  { benchGinMW(b, 5) }
func BenchmarkMiddleware10_Gin(b *testing.B) { benchGinMW(b, 10) }

// ------------------------ Echo ------------------------
func newEchoWithMW(n int) http.Handler {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Debug = false
	for i := 0; i < n; i++ {
		e.Use(func(next echo.HandlerFunc) echo.HandlerFunc { return func(c echo.Context) error { return next(c) } })
	}
	e.GET("/mw", func(c echo.Context) error { return c.String(http.StatusOK, "ok") })
	return e
}

func benchEchoMW(b *testing.B, n int) {
	h := newEchoWithMW(n)
	req := httptest.NewRequest(http.MethodGet, "/mw", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
	}
}

func BenchmarkMiddleware5_Echo(b *testing.B)  { benchEchoMW(b, 5) }
func BenchmarkMiddleware10_Echo(b *testing.B) { benchEchoMW(b, 10) }

// ------------------------ Fiber ------------------------
func newFiberWithMW(n int) *fiber.App {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	for i := 0; i < n; i++ {
		app.Use(func(c *fiber.Ctx) error { return c.Next() })
	}
	app.Get("/mw", func(c *fiber.Ctx) error { return c.SendString("ok") })
	return app
}

func benchFiberMW(b *testing.B, n int) {
	app := newFiberWithMW(n)
	req := httptest.NewRequest(http.MethodGet, "/mw", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = app.Test(req, -1)
	}
}

func BenchmarkMiddleware5_Fiber(b *testing.B)  { benchFiberMW(b, 5) }
func BenchmarkMiddleware10_Fiber(b *testing.B) { benchFiberMW(b, 10) }

// ------------------------ Chi ------------------------
func newChiWithMW(n int) http.Handler {
	r := chi.NewRouter()
	// 构造 n 层无操作中间件
	for i := 0; i < n; i++ {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r)
			})
		})
	}
	r.Get("/mw", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	})
	return r
}

func benchChiMW(b *testing.B, n int) {
	h := newChiWithMW(n)
	req := httptest.NewRequest(http.MethodGet, "/mw", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
	}
}

func BenchmarkMiddleware5_Chi(b *testing.B)  { benchChiMW(b, 5) }
func BenchmarkMiddleware10_Chi(b *testing.B) { benchChiMW(b, 10) }
