package bench

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/gofiber/fiber/v2"
	"github.com/labstack/echo/v4"
	"go-slim.dev/slim"
)

// 说明：
// - 各框架注册相同的参数路由（/users/:id），返回路径参数 id
// - 关闭/隐藏日志与调试（在 basic_test.go 中已对 Slim/Gin/Echo 做了最小化配置）
// - Fiber 依旧通过 app.Test 执行 *http.Request，保持可比性

// ------------------------ Slim ------------------------
func setupSlimParams() http.Handler {
	s := slim.New()
	s.HideBanner = true
	s.Debug = false
	s.GET("/users/:id", func(c slim.Context) error {
		return c.String(http.StatusOK, c.PathParam("id"))
	})
	return s
}

func BenchmarkParams_Slim(b *testing.B) {
	h := setupSlimParams()
	req := httptest.NewRequest(http.MethodGet, "/users/12345", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
	}
}

// ------------------------ Gin ------------------------
func setupGinParams() http.Handler {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.GET("/users/:id", func(c *gin.Context) { c.String(http.StatusOK, c.Param("id")) })
	return r
}

func BenchmarkParams_Gin(b *testing.B) {
	h := setupGinParams()
	req := httptest.NewRequest(http.MethodGet, "/users/12345", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
	}
}

// ------------------------ Echo ------------------------
func setupEchoParams() http.Handler {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Debug = false
	e.GET("/users/:id", func(c echo.Context) error { return c.String(http.StatusOK, c.Param("id")) })
	return e
}

func BenchmarkParams_Echo(b *testing.B) {
	h := setupEchoParams()
	req := httptest.NewRequest(http.MethodGet, "/users/12345", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
	}
}

// ------------------------ Fiber ------------------------
func setupFiberParams() *fiber.App {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/users/:id", func(c *fiber.Ctx) error { return c.SendString(c.Params("id")) })
	return app
}

func BenchmarkParams_Fiber(b *testing.B) {
	app := setupFiberParams()
	req := httptest.NewRequest(http.MethodGet, "/users/12345", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = app.Test(req, -1)
	}
}
