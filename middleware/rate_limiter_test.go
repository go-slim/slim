package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go-slim.dev/slim"
)

func TestRateLimiterMemoryStore_AllowAndCleanup(t *testing.T) {
	store := NewRateLimiterMemoryStoreWithConfig(RateLimiterMemoryStoreConfig{
		Rate:      1,              // 1 rps
		Burst:     1,              // burst 1
		ExpiresIn: 10 * time.Millisecond,
	})
	// make time controllable
	now := time.Now()
	store.timeNow = func() time.Time { return now }

	ok, err := store.Allow("a")
	if err != nil || !ok {
		t.Fatalf("first allow should pass, ok=%v err=%v", ok, err)
	}
	ok, _ = store.Allow("a")
	if ok {
		t.Fatalf("second immediate allow should be denied due to rate limit")
	}
	// advance time to expire visitor and cleanup
	now = now.Add(20 * time.Millisecond)
	store.cleanupStaleVisitors()
	if len(store.visitors) != 0 {
		t.Fatalf("expected visitors cleaned up, got %d", len(store.visitors))
	}
}

func TestRateLimiter_Middleware_DeniesOnSecondRequest(t *testing.T) {
	s := slim.New()
	store := NewRateLimiterMemoryStoreWithConfig(RateLimiterMemoryStoreConfig{Rate: 1, Burst: 1})
	// Override error handler to use HTTPError code
	s.ErrorHandler = func(c slim.Context, err error) {
		if he, ok := err.(*slim.HTTPError); ok {
			http.Error(c.Response(), http.StatusText(he.Code), he.Code)
			return
		}
		http.Error(c.Response(), http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
	s.Use(RateLimiter(store))
	s.GET("/", func(c slim.Context) error { return c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest(http.MethodGet, "http://host/", nil)
	rw := httptest.NewRecorder()
	s.ServeHTTP(rw, req)
	if rw.Code != http.StatusOK {
		t.Fatalf("first request should pass, got %d", rw.Code)
	}
	// second request immediately should be 429
	rw = httptest.NewRecorder()
	s.ServeHTTP(rw, req)
	if rw.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 on second request, got %d", rw.Code)
	}
}
