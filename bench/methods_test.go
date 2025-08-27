package bench

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofiber/fiber/v2"
	"github.com/go-chi/chi/v5"
	"github.com/labstack/echo/v4"
	"go-slim.dev/slim"
)

// 说明：
// - 显式注册 HEAD 与 OPTIONS，避免不同框架对隐式规则的分歧，保证对比公平
// - 所有框架均返回 200，无响应体
// - Slim/Gin/Echo 使用 httptest + ServeHTTP；Fiber 使用 app.Test

// ------------------------ Slim ------------------------
func setupSlimMethods() http.Handler {
	s := slim.New()
	s.HideBanner = true
	s.Debug = false
	s.HEAD("/head", func(c slim.Context) error { return c.NoContent(http.StatusOK) })
	s.OPTIONS("/opt", func(c slim.Context) error { return c.NoContent(http.StatusOK) })
	return s
}

func BenchmarkHEAD_Explicit_Slim(b *testing.B) {
	h := setupSlimMethods()
	req := httptest.NewRequest(http.MethodHead, "/head", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
	}
}

func BenchmarkOPTIONS_Explicit_Slim(b *testing.B) {
	h := setupSlimMethods()
	req := httptest.NewRequest(http.MethodOptions, "/opt", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
	}
}

// ------------------------ Gin ------------------------
func setupGinMethods() http.Handler {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.HEAD("/head", func(c *gin.Context) { c.Status(http.StatusOK) })
	r.OPTIONS("/opt", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func BenchmarkHEAD_Explicit_Gin(b *testing.B) {
	h := setupGinMethods()
	req := httptest.NewRequest(http.MethodHead, "/head", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
	}
}

func BenchmarkOPTIONS_Explicit_Gin(b *testing.B) {
	h := setupGinMethods()
	req := httptest.NewRequest(http.MethodOptions, "/opt", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
	}
}

// ------------------------ Echo ------------------------
func setupEchoMethods() http.Handler {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Debug = false
	e.HEAD("/head", func(c echo.Context) error { return c.NoContent(http.StatusOK) })
	e.OPTIONS("/opt", func(c echo.Context) error { return c.NoContent(http.StatusOK) })
	return e
}

func BenchmarkHEAD_Explicit_Echo(b *testing.B) {
	h := setupEchoMethods()
	req := httptest.NewRequest(http.MethodHead, "/head", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
	}
}

func BenchmarkOPTIONS_Explicit_Echo(b *testing.B) {
	h := setupEchoMethods()
	req := httptest.NewRequest(http.MethodOptions, "/opt", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
	}
}

// ------------------------ Fiber ------------------------
func setupFiberMethods() *fiber.App {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Head("/head", func(c *fiber.Ctx) error { return c.SendStatus(http.StatusOK) })
	app.Options("/opt", func(c *fiber.Ctx) error { return c.SendStatus(http.StatusOK) })
	return app
}

func BenchmarkHEAD_Explicit_Fiber(b *testing.B) {
	app := setupFiberMethods()
	req := httptest.NewRequest(http.MethodHead, "/head", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = app.Test(req, -1)
	}
}

func BenchmarkOPTIONS_Explicit_Fiber(b *testing.B) {
	app := setupFiberMethods()
	req := httptest.NewRequest(http.MethodOptions, "/opt", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = app.Test(req, -1)
	}
}

// ------------------------ Chi ------------------------
func setupChiMethods() http.Handler {
	r := chi.NewRouter()
	r.Method(http.MethodHead, "/head", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	r.Method(http.MethodOptions, "/opt", http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	return r
}

func BenchmarkHEAD_Explicit_Chi(b *testing.B) {
	h := setupChiMethods()
	req := httptest.NewRequest(http.MethodHead, "/head", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
	}
}

func BenchmarkOPTIONS_Explicit_Chi(b *testing.B) {
	h := setupChiMethods()
	req := httptest.NewRequest(http.MethodOptions, "/opt", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
	}
}
