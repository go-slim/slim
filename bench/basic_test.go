package bench

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-chi/chi/v5"
	"github.com/gofiber/fiber/v2"
	"github.com/labstack/echo/v4"
	"go-slim.dev/slim"
)

var payload = struct {
	Message string `json:"message"`
}{Message: "pong"}

// ------------------------ Slim ------------------------
func setupSlimBasic() http.Handler {
	s := slim.New()
	// minimize overhead
	s.HideBanner = true
	s.Debug = false
	s.StdLogger = nil

	s.GET("/ping", func(c slim.Context) error { return c.String(http.StatusOK, "pong") })
	s.GET("/json", func(c slim.Context) error { return c.JSON(http.StatusOK, payload) })
	return s
}

func BenchmarkBasic_Slim(b *testing.B) {
	h := setupSlimBasic()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
	}
}

func BenchmarkJSON_Slim(b *testing.B) {
	h := setupSlimBasic()
	req := httptest.NewRequest(http.MethodGet, "/json", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
	}
}

// ------------------------ Gin ------------------------
func setupGinBasic() http.Handler {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Handle(http.MethodGet, "/ping", func(c *gin.Context) { c.String(http.StatusOK, "pong") })
	r.Handle(http.MethodGet, "/json", func(c *gin.Context) { c.JSON(http.StatusOK, payload) })
	return r
}

func BenchmarkBasic_Gin(b *testing.B) {
	h := setupGinBasic()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
	}
}

func BenchmarkJSON_Gin(b *testing.B) {
	h := setupGinBasic()
	req := httptest.NewRequest(http.MethodGet, "/json", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
	}
}

// ------------------------ Echo ------------------------
func setupEchoBasic() http.Handler {
	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.Debug = false
	e.GET("/ping", func(c echo.Context) error { return c.String(http.StatusOK, "pong") })
	e.GET("/json", func(c echo.Context) error { return c.JSON(http.StatusOK, payload) })
	return e
}

func BenchmarkBasic_Echo(b *testing.B) {
	h := setupEchoBasic()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
	}
}

func BenchmarkJSON_Echo(b *testing.B) {
	h := setupEchoBasic()
	req := httptest.NewRequest(http.MethodGet, "/json", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
	}
}

// ------------------------ Fiber ------------------------
func setupFiberBasic() *fiber.App {
	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Get("/ping", func(c *fiber.Ctx) error { return c.SendString("pong") })
	app.Get("/json", func(c *fiber.Ctx) error {
		c.Set("Content-Type", "application/json")
		b, _ := json.Marshal(payload)
		return c.Status(http.StatusOK).Send(b)
	})
	return app
}

func BenchmarkBasic_Fiber(b *testing.B) {
	app := setupFiberBasic()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = app.Test(req, -1)
	}
}

func BenchmarkJSON_Fiber(b *testing.B) {
	app := setupFiberBasic()
	req := httptest.NewRequest(http.MethodGet, "/json", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = app.Test(req, -1)
	}
}

// ------------------------ Chi ------------------------
func setupChiBasic() http.Handler {
	r := chi.NewRouter()
	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("pong"))
	})
	r.Get("/json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(payload)
	})
	return r
}

func BenchmarkBasic_Chi(b *testing.B) {
	h := setupChiBasic()
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
	}
}

func BenchmarkJSON_Chi(b *testing.B) {
	h := setupChiBasic()
	req := httptest.NewRequest(http.MethodGet, "/json", nil)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
	}
}
