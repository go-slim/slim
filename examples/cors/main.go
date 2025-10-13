package main

import (
	"log"
	"net/http"

	"go-slim.dev/slim"
	"go-slim.dev/slim/middleware"
)

func main() {
	s := slim.New()
	// Allow specific origins and credentials
	s.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins:     []string{"http://localhost:3000", "https://example.com"},
		AllowMethods:     []string{http.MethodGet, http.MethodPost, http.MethodOptions},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		ExposeHeaders:    []string{"X-Request-Id"},
		AllowCredentials: true,
		MaxAge:           600,
	}))

	s.GET("/hello", func(c slim.Context) error {
		return c.String(http.StatusOK, "CORS OK")
	})

	log.Fatal(s.Start(":1325"))
}
