package serde

import (
	"bytes"
	"strings"
	"testing"
)

type jObj struct{ A int `json:"a"` }

func TestJSONSerializer_Serialize_NoIndent(t *testing.T) {
	s := JSONSerializer{}
	var b bytes.Buffer
	if err := s.Serialize(&b, jObj{A: 1}, ""); err != nil { t.Fatal(err) }
	out := b.String()
	if out != "{\"a\":1}\n" { // json.Encoder adds trailing newline
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestJSONSerializer_Serialize_Indent(t *testing.T) {
	s := JSONSerializer{}
	var b bytes.Buffer
	if err := s.Serialize(&b, map[string]int{"a":1}, "  "); err != nil { t.Fatal(err) }
	out := b.String()
	if !strings.Contains(out, "\n  \"a\": 1\n") {
		t.Fatalf("expected indented JSON, got: %q", out)
	}
}

func TestJSONSerializer_Deserialize(t *testing.T) {
	s := JSONSerializer{}
	var v jObj
	if err := s.Deserialize(strings.NewReader("{\"a\":7}"), &v); err != nil { t.Fatal(err) }
	if v.A != 7 { t.Fatalf("A=%d", v.A) }
}
