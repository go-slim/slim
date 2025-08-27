package slim

import (
	"strings"
	"testing"
)

func TestRecovery_Default_WithStack(t *testing.T) {
	mw := Recovery()
	err := mw(nil, func(c Context) error {
		panic("boom")
	})
	if err == nil {
		t.Fatalf("expected error from panic recovery")
	}
	s := err.Error()
	if !strings.Contains(s, "boom") {
		t.Fatalf("recovered error should contain panic message, got: %s", s)
	}
	if !strings.Contains(s, "PANIC RECOVER") {
		t.Fatalf("expected stack prefix in error, got: %s", s)
	}
}

func TestRecovery_DisablePrintStack(t *testing.T) {
	mw := RecoveryWithConfig(RecoveryConfig{DisablePrintStack: true})
	err := mw(nil, func(c Context) error {
		panic("no-stack")
	})
	if err == nil {
		t.Fatalf("expected error from panic recovery")
	}
	s := err.Error()
	if !strings.Contains(s, "no-stack") {
		t.Fatalf("expected panic message, got: %s", s)
	}
	if strings.Contains(s, "PANIC RECOVER") {
		t.Fatalf("did not expect stack prefix when DisablePrintStack is true, got: %s", s)
	}
}
