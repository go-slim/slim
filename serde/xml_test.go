package serde

import (
	"bytes"
	"strings"
	"testing"
)

type xObj struct{ A int `xml:"a"` }

func TestXMLSerializer_Serialize_NoIndent(t *testing.T) {
	s := XMLSerializer{}
	var b bytes.Buffer
	if err := s.Serialize(&b, xObj{A: 1}, ""); err != nil { t.Fatal(err) }
	out := b.String()
	// xml.Encoder omits declaration by default; single-line encode
	if !strings.Contains(out, "<a>1</a>") {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestXMLSerializer_Serialize_Indent(t *testing.T) {
	s := XMLSerializer{}
	var b bytes.Buffer
	if err := s.Serialize(&b, xObj{A: 2}, "  "); err != nil { t.Fatal(err) }
	out := b.String()
	if !strings.Contains(out, "\n  <a>2</a>\n") {
		t.Fatalf("expected indented xml, got: %q", out)
	}
}

func TestXMLSerializer_Deserialize(t *testing.T) {
	s := XMLSerializer{}
	var v xObj
	if err := s.Deserialize(strings.NewReader("<xObj><a>7</a></xObj>"), &v); err != nil { t.Fatal(err) }
	if v.A != 7 { t.Fatalf("A=%d", v.A) }
}
