package slim

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestNewHTTPError_DefaultMessage(t *testing.T) {
	he := NewHTTPError(http.StatusNotFound)
	if he.Code != http.StatusNotFound {
		t.Fatalf("code = %d, want %d", he.Code, http.StatusNotFound)
	}
	if he.Message != http.StatusText(http.StatusNotFound) {
		t.Fatalf("message = %v, want %q", he.Message, http.StatusText(http.StatusNotFound))
	}
	if he.Internal != nil {
		t.Fatalf("internal should be nil")
	}
}

func TestNewHTTPError_WithMessage(t *testing.T) {
	he := NewHTTPError(http.StatusBadRequest, "bad")
	if he.Message != "bad" {
		t.Fatalf("message = %v, want %q", he.Message, "bad")
	}
}

func TestHTTPError_ErrorFormatting_NoInternal(t *testing.T) {
	he := NewHTTPError(http.StatusUnauthorized, "nope")
	s := he.Error()
	if !strings.Contains(s, fmt.Sprintf("code=%d", http.StatusUnauthorized)) {
		t.Fatalf("unexpected Error output: %s", s)
	}
	if !strings.Contains(s, "message=nope") {
		t.Fatalf("unexpected Error output: %s", s)
	}
}

func TestHTTPError_ErrorFormatting_WithInternal(t *testing.T) {
	inner := errors.New("inner")
	he := NewHTTPErrorWithInternal(http.StatusInternalServerError, inner, "boom")
	s := he.Error()
	if !strings.Contains(s, fmt.Sprintf("statusCode=%d", http.StatusInternalServerError)) {
		t.Fatalf("unexpected Error output: %s", s)
	}
	if !strings.Contains(s, "message=boom") || !strings.Contains(s, "internal=inner") {
		t.Fatalf("unexpected Error output: %s", s)
	}
}

func TestHTTPError_WithInternal_Unwrap(t *testing.T) {
	inner := errors.New("deep")
	he := NewHTTPError(http.StatusTeapot).WithInternal(inner)
	if !errors.Is(he, inner) {
		t.Fatalf("errors.Is should match wrapped internal error")
	}
	if got := errors.Unwrap(he); got == nil || got.Error() != "deep" {
		t.Fatalf("Unwrap() = %v, want deep", got)
	}
}

func TestPredefinedErrors(t *testing.T) {
	// Sample a few to ensure they are initialized with proper codes
	cases := []struct{
		name string
		he   *HTTPError
		code int
	}{
		{"ErrNotFound", ErrNotFound, http.StatusNotFound},
		{"ErrUnauthorized", ErrUnauthorized, http.StatusUnauthorized},
		{"ErrBadRequest", ErrBadRequest, http.StatusBadRequest},
	}
	for _, tc := range cases {
		if tc.he.Code != tc.code {
			t.Fatalf("%s code = %d, want %d", tc.name, tc.he.Code, tc.code)
		}
		if tc.he.Message != http.StatusText(tc.code) {
			t.Fatalf("%s message = %v, want %q", tc.name, tc.he.Message, http.StatusText(tc.code))
		}
	}
}
