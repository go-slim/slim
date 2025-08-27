package slim

import (
	"bufio"
	"bytes"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

type rwWithAll struct { // implements http.ResponseWriter + Hijacker + Flusher + Pusher
	http.ResponseWriter
}

func (r *rwWithAll) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, nil
}
func (r *rwWithAll) Flush() {}
func (r *rwWithAll) Push(target string, opts *http.PushOptions) error { return nil }

type rwBasic struct{ http.ResponseWriter }

type rwNoHijack struct{ http.ResponseWriter }

func (r *rwNoHijack) Push(target string, opts *http.PushOptions) error { return nil }

func TestResponseWriter_StatusSizeWrite(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := NewResponseWriter("GET", rec)
	if rw.Written() {
		t.Fatalf("should not be written yet")
	}
	if rw.Status() != 0 || rw.Size() != 0 {
		t.Fatalf("initial status/size wrong: %d/%d", rw.Status(), rw.Size())
	}

	// Write without explicit header sets 200 and counts bytes
	n, err := rw.Write([]byte("hello"))
	if err != nil || n != 5 {
		t.Fatalf("write returned %d, %v", n, err)
	}
	if !rw.Written() || rw.Status() != http.StatusOK || rw.Size() != 5 {
		t.Fatalf("after write status/size wrong: %d/%d", rw.Status(), rw.Size())
	}

	// WriteHeader does nothing once written
	rw.WriteHeader(http.StatusAccepted)
	if rw.Status() != http.StatusOK {
		t.Fatalf("status changed after WriteHeader post-write: %d", rw.Status())
	}
}

func TestResponseWriter_HEADDoesNotCountBody(t *testing.T) {
	rec := httptest.NewRecorder()
	rw := NewResponseWriter("HEAD", rec)
	rw.WriteHeader(http.StatusNoContent)
	rw.Write(bytes.Repeat([]byte{'a'}, 10))
	if rw.Size() != 0 {
		t.Fatalf("HEAD should not accumulate body size, got %d", rw.Size())
	}
}

func TestResponseWriter_HijackAndPushAndFlush(t *testing.T) {
	// Hijack not supported
	rec := httptest.NewRecorder()
	rw := &responseWriter{method: "GET", ResponseWriter: &rwBasic{rec}}
	if _, _, err := rw.Hijack(); err == nil {
		t.Fatalf("expected hijack error when not supported")
	}
	if err := rw.Push("/x", nil); err == nil {
		t.Fatalf("expected push error when not supported")
	}

	// Supported
	rec2 := httptest.NewRecorder()
	full := &rwWithAll{ResponseWriter: rec2}
	rw2 := &responseWriter{method: "GET", ResponseWriter: full}
	if _, _, err := rw2.Hijack(); err != nil {
		t.Fatalf("unexpected hijack err: %v", err)
	}
	if err := rw2.Push("/x", nil); err != nil {
		t.Fatalf("unexpected push err: %v", err)
	}
	rw2.Flush() // should not panic
}
