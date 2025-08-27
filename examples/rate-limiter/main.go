package main

import (
	"net/http"

	"go-slim.dev/slim"
	"go-slim.dev/slim/middleware"
)

func main() {
	s := slim.New()

	// Map slim.HTTPError to its HTTP status code so middleware like RateLimiter
	// can signal 429/403 properly.
	s.ErrorHandler = func(c slim.Context, err error) {
		if he, ok := err.(*slim.HTTPError); ok {
			http.Error(c.Response(), http.StatusText(he.Code), he.Code)
			return
		}
		http.Error(c.Response(), http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}

	// Configure in-memory rate limiter: 1 req/s, burst=1
	store := middleware.NewRateLimiterMemoryStoreWithConfig(middleware.RateLimiterMemoryStoreConfig{Rate: 1, Burst: 1})
	s.Use(middleware.RateLimiter(store))

	s.GET("/", func(c slim.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	s.Logger.Fatal(s.Start(":1326"))
}
