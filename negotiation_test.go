package slim

import (
	"net/http"
	"testing"
)

func Test_parseMediaRange(t *testing.T) {
	// valid
	p, ts, err := parseMediaRange("text/html; q=0.8; level=1")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(p) == 0 || ts[0] != "text" || ts[1] != "html" {
		t.Fatalf("unexpected parse: %v %v", p, ts)
	}
	// invalid format
	_, _, err = parseMediaRange("application/json/extra")
	if err == nil {
		t.Fatalf("expected error for invalid media range")
	}
	// empty type -> becomes wildcard, but acceptable
	_, ts, err = parseMediaRange("/json")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if ts[0] != "*" || ts[1] != "json" {
		t.Fatalf("unexpected types: %v", ts)
	}
}

func TestAcceptSlice_SortAndNegotiate(t *testing.T) {
	s := newSlice("text/*;q=0.9, text/plain, text/plain;format=flowed, */*;q=0.1", onAcceptParsed)
	if len(s) == 0 {
		t.Fatalf("expected non-empty slice")
	}
	// should negotiate most specific first
	ct, i, err := s.Negotiate("text/plain", "application/json")
	if err != nil || i != 0 || ct != "text/plain" {
		t.Fatalf("unexpected negotiate: ct=%s i=%d err=%v", ct, i, err)
	}
	// wildcard
	ct, i, err = s.Negotiate("*/*")
	if err != nil || i != 0 || ct != "*/*" {
		t.Fatalf("unexpected wildcard negotiate: ct=%s i=%d err=%v", ct, i, err)
	}
	if !s.Accepts("text/plain") || !s.Accepts("image/png") {
		t.Fatalf("accepts logic failed")
	}
}

func TestNegotiator_AcceptsAndType(t *testing.T) {
	n := NewNegotiator(10, nil)
	req, _ := http.NewRequest(http.MethodGet, "http://example.com/", nil)
	req.Header.Set(HeaderAccept, "application/json, text/*;q=0.5")
	if got := n.Accepts(req.Header.Get(HeaderAccept), MIMETextPlain, MIMEApplicationJSON); got != MIMEApplicationJSON {
		t.Fatalf("Accepts got %s", got)
	}
	if typ := n.Type(req, "json", "xml", "text"); typ != "json" {
		t.Fatalf("Type got %s", typ)
	}
}
