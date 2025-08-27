package middleware

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go-slim.dev/slim"
	"go-slim.dev/l4g"
)

// Test Logger middleware logs both successful and error responses.
func TestLogger_LogBeginEnd_WithAndWithoutError(t *testing.T) {
	s := slim.New()
	// capture logs by replacing the logger instance (avoid atomic.Value type mismatch)
	var out bytes.Buffer
	s.Logger = l4g.New(&out)

	// Error handler must finalize logging on error
	s.ErrorHandler = func(c slim.Context, err error) {
		// Map HTTPError to its code; otherwise 500
		code := http.StatusInternalServerError
		var he *slim.HTTPError
		if err != nil && errors.As(err, &he) {
			code = he.Code
		}
		if !c.Response().Written() {
			c.Response().WriteHeader(code)
		}
		// end logging with error
		LogEnd(c, err)
	}

	s.Use(Logger())

	s.GET("/ok", func(c slim.Context) error {
		return c.String(http.StatusOK, "ok")
	})

	s.GET("/err", func(c slim.Context) error {
		return &slim.HTTPError{Code: 418, Message: "teapot"}
	})

	// ok request
	r := httptest.NewRequest(http.MethodGet, "http://example.com/ok", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 200 {
		t.Fatalf("/ok status = %d", w.Code)
	}
	logs := out.String()
	if !strings.Contains(logs, " 200") {
		t.Fatalf("expected log to contain 200, got: %s", logs)
	}

	// reset buffer
	out.Reset()

	// error request
	r = httptest.NewRequest(http.MethodGet, "http://example.com/err", nil)
	w = httptest.NewRecorder()
	s.ServeHTTP(w, r)
	if w.Code != 418 {
		t.Fatalf("/err status = %d", w.Code)
	}
	logs = out.String()
	if !strings.Contains(logs, " 418") {
		t.Fatalf("expected log to contain 418, got: %s", logs)
	}
}
