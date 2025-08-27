package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"go-slim.dev/slim"
	"go-slim.dev/l4g"
)

func TestMiddlewareRecovery_PanicWrites500AndPrintsStack(t *testing.T) {
	s := slim.New()
	var logBuf bytes.Buffer
	s.Logger = l4g.New(&logBuf)

	// ensure error maps to 500 and logger ends
	s.ErrorHandler = func(c slim.Context, err error) {
		if !c.Response().Written() {
			c.Response().WriteHeader(http.StatusInternalServerError)
		}
		LogEnd(c, err)
	}

	// capture pretty stack output
	var stackBuf bytes.Buffer
	orig := recovererErrorWriter
	recovererErrorWriter = &stackBuf
	defer func() { recovererErrorWriter = orig }()

	s.Use(Logger())
	s.Use(Recovery())

	s.GET("/panic", func(c slim.Context) error {
		panic("boom")
	})

	r := httptest.NewRequest(http.MethodGet, "http://example.com/panic", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
	// pretty stack should be printed
	if stackBuf.Len() == 0 {
		t.Fatalf("expected pretty stack output, got empty")
	}
	if !bytes.Contains(stackBuf.Bytes(), []byte("panic:")) {
		t.Fatalf("expected panic header in stack output, got: %s", stackBuf.String())
	}
}

func TestMiddlewareRecovery_DisablePrintStack_NoStackOutput(t *testing.T) {
	s := slim.New()
	// capture pretty stack output
	var stackBuf bytes.Buffer
	orig := recovererErrorWriter
	recovererErrorWriter = &stackBuf
	defer func() { recovererErrorWriter = orig }()

	s.ErrorHandler = func(c slim.Context, err error) {
		if !c.Response().Written() {
			c.Response().WriteHeader(http.StatusInternalServerError)
		}
	}

	s.Use(RecoveryWithConfig(RecoveryConfig{DisablePrintStack: true}))

	s.GET("/panic", func(c slim.Context) error { panic("silent") })

	r := httptest.NewRequest(http.MethodGet, "http://example.com/panic", nil)
	w := httptest.NewRecorder()
	s.ServeHTTP(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
	if stackBuf.Len() != 0 {
		t.Fatalf("expected no stack output when disabled, got: %s", stackBuf.String())
	}
}
