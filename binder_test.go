package slim

import (
	"reflect"
	"testing"
)

type myBU string

func (m *myBU) UnmarshalParam(param string) error {
	*m = myBU("u:" + param)
	return nil
}

type inner struct {
	X int `form:"x"`
}

type outer struct {
	A string   `form:"a"`
	B []int    `form:"b"`
	C inner    // no tag: should recurse into inner
	D myBU     `form:"d"`
	E *myBU    `form:"e"`
	M map[string]string // map binding path: not used in struct binding
}

type Anon struct {
	Y int `query:"y"`
}

type withAnon struct {
	Anon `query:"ignored"`
}

func TestBindData_FormStructBasics(t *testing.T) {
	var dst outer
	data := map[string][]string{
		"a": {"hello"},
		"b": {"1", "2", "3"},
		"x": {"42"},
		"d": {"val"},
		"e": {"ptr"},
	}
	if err := bindData(&dst, data, "form"); err != nil {
		t.Fatalf("bindData error: %v", err)
	}
	if dst.A != "hello" { t.Fatalf("A=%q", dst.A) }
	if !reflect.DeepEqual(dst.B, []int{1,2,3}) { t.Fatalf("B=%v", dst.B) }
	if dst.C.X != 42 { t.Fatalf("C.X=%d", dst.C.X) }
	if string(dst.D) != "u:val" { t.Fatalf("D=%q", dst.D) }
	if dst.E == nil || string(*dst.E) != "u:ptr" { t.Fatalf("E=%v", dst.E) }
}

func TestBindData_CaseInsensitiveKeyMatch(t *testing.T) {
	type S struct{ A int `query:"id"` }
	var dst S
	data := map[string][]string{"ID": {"7"}}
	if err := bindData(&dst, data, "query"); err != nil { t.Fatal(err) }
	if dst.A != 7 { t.Fatalf("A=%d", dst.A) }
}

func TestBindData_MapDestination(t *testing.T) {
	m := map[string]string{}
	data := map[string][]string{"k": {"v"}}
	if err := bindData(&m, data, "query"); err != nil { t.Fatal(err) }
	if m["k"] != "v" { t.Fatalf("map value not set") }
}

func TestBindData_AnonymousWithTagError(t *testing.T) {
	var dst withAnon
	data := map[string][]string{"y": {"1"}}
	if err := bindData(&dst, data, "query"); err == nil {
		t.Fatalf("expected error for anonymous struct with tag")
	}
}

func TestBindData_NonStructWithFormTagErrors(t *testing.T) {
	var x int
	data := map[string][]string{"x": {"1"}}
	if err := bindData(&x, data, "form"); err == nil {
		t.Fatalf("expected error when binding non-struct for form")
	}
}
