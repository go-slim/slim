package slim

import "testing"

func TestVersionConstant(t *testing.T) {
	if Version == "" {
		t.Fatalf("Version must not be empty")
	}
}
