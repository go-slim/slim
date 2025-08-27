package slim

import (
	"errors"
	"strings"
	"testing"
)

func TestCompose_Order(t *testing.T) {
	var steps []string
	m1 := func(c Context, next HandlerFunc) error {
		steps = append(steps, "m1-in")
		if err := next(c); err != nil { return err }
		steps = append(steps, "m1-out")
		return nil
	}
	m2 := func(c Context, next HandlerFunc) error {
		steps = append(steps, "m2-in")
		if err := next(c); err != nil { return err }
		steps = append(steps, "m2-out")
		return nil
	}
	mw := Compose(m1, m2)
	if mw == nil { t.Fatalf("Compose returned nil") }
	err := mw(nil, func(c Context) error {
		steps = append(steps, "end")
		return nil
	})
	if err != nil { t.Fatalf("unexpected err: %v", err) }
	expected := []string{"m1-in","m2-in","end","m2-out","m1-out"}
	if strings.Join(steps, ",") != strings.Join(expected, ",") {
		t.Fatalf("order = %v, want %v", steps, expected)
	}
}

func TestCompose_NextCalledMultipleTimes(t *testing.T) {
	bad := func(c Context, next HandlerFunc) error {
		if err := next(c); err != nil { return err }
		return next(c)
	}
	noop := func(c Context, next HandlerFunc) error { return next(c) }
	mw := Compose(bad, noop)
	err := mw(nil, func(c Context) error { return nil })
	if err == nil { t.Fatalf("expected error when next called multiple times") }
	if !strings.Contains(err.Error(), "multiple times") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCompose_EdgeCases(t *testing.T) {
	if got := Compose(); got != nil {
		t.Fatalf("Compose() expected nil, got %v", got)
	}
	// single middleware returns same function
	m := func(c Context, next HandlerFunc) error { return errors.New("x") }
	if Compose(m) == nil {
		t.Fatalf("Compose(single) should not be nil")
	}
}
