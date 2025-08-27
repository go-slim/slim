package nego

import "testing"

func TestTypeIs(t *testing.T) {
	// direct match
	got, err := TypeIs("application/json; charset=UTF-8", "json", "xml")
	if err != nil || got != "json" {
		t.Fatalf("expected json, got %q err=%v", got, err)
	}
	// wildcard subtype should error
	if _, err := TypeIs("application/*", "json"); err == nil {
		t.Fatalf("expected error for wildcard subtype")
	}
	// extension resolution
	got, err = TypeIs("image/png", "png", "jpeg")
	if err != nil || got != "png" {
		t.Fatalf("expected png, got %q err=%v", got, err)
	}
	// no match
	got, err = TypeIs("text/plain", "json")
	if err != nil || got != "" {
		t.Fatalf("expected empty, got %q err=%v", got, err)
	}
}
