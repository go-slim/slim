package middleware

import "testing"

func TestMatchScheme(t *testing.T) {
	if !matchScheme("http://example.com", "http://*.example.com") {
		t.Fatalf("expected scheme match http")
	}
	if matchScheme("https://example.com", "http://*.example.com") {
		t.Fatalf("did not expect scheme match https vs http")
	}
}

func TestMatchSubdomain(t *testing.T) {
	// positive wildcard
	if !matchSubdomain("http://api.foo.bar.com", "http://*.bar.com") {
		t.Fatalf("expected subdomain match")
	}
	// negative: too long domain
	long := "http://" + string(make([]byte, 254))
	if matchSubdomain(long, "http://*.example.com") {
		t.Fatalf("long domain should not match")
	}
	// negative: different scheme
	if matchSubdomain("https://a.b.com", "http://*.b.com") {
		t.Fatalf("different scheme must not match")
	}
    // wildcard at leftmost should match any subdomain depth
    if !matchSubdomain("http://a.b.c", "http://*.c") {
        t.Fatalf("expected wildcard to match nested subdomains")
    }
}
