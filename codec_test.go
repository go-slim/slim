package slim

import (
	"bytes"
	"strings"
	"testing"
)

type jObj struct {
	A int `json:"a"`
}

func TestJSONCodec_Encode_NoIndent(t *testing.T) {
	s := JSONCodec{}
	var b bytes.Buffer
	if err := s.Encode(&b, jObj{A: 1}, ""); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	if out != "{\"a\":1}\n" { // json.Encoder adds trailing newline
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestJSONCodec_Encode_Indent(t *testing.T) {
	s := JSONCodec{}
	var b bytes.Buffer
	if err := s.Encode(&b, map[string]int{"a": 1}, "  "); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	if !strings.Contains(out, "\n  \"a\": 1\n") {
		t.Fatalf("expected indented JSON, got: %q", out)
	}
}

func TestJSONCodec_Decode(t *testing.T) {
	s := JSONCodec{}
	var v jObj
	if err := s.Decode(strings.NewReader("{\"a\":7}"), &v); err != nil {
		t.Fatal(err)
	}
	if v.A != 7 {
		t.Fatalf("A=%d", v.A)
	}
}

type xObj struct {
	A int `xml:"a"`
}

func TestXMCodec_Encode_NoIndent(t *testing.T) {
	s := XMLCodec{}
	var b bytes.Buffer
	if err := s.Encode(&b, xObj{A: 1}, ""); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	// xml.Encoder omits declaration by default; single-line encode
	if !strings.Contains(out, "<a>1</a>") {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestXMLCodec_Encode_Indent(t *testing.T) {
	s := XMLCodec{}
	var b bytes.Buffer
	if err := s.Encode(&b, xObj{A: 2}, "  "); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	if !strings.Contains(out, "\n  <a>2</a>\n") {
		t.Fatalf("expected indented xml, got: %q", out)
	}
}

func TestXMLCodec_Decode(t *testing.T) {
	s := XMLCodec{}
	var v xObj
	if err := s.Decode(strings.NewReader("<xObj><a>7</a></xObj>"), &v); err != nil {
		t.Fatal(err)
	}
	if v.A != 7 {
		t.Fatalf("A=%d", v.A)
	}
}
