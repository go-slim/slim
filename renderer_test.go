package slim

import (
	"errors"
	"io"
	"strings"
	"testing"
)

type fakeTmpl struct{
	lastName string
	lastData any
	shouldErr bool
}

func (f *fakeTmpl) ExecuteTemplate(wr io.Writer, name string, data any) error {
	f.lastName = name
	f.lastData = data
	if f.shouldErr {
		return errors.New("tmpl err")
	}
	_, _ = wr.Write([]byte("ok"))
	return nil
}

func TestTemplateRenderer_Render_Success(t *testing.T) {
	ft := &fakeTmpl{}
	r := &TemplateRenderer{Template: ft}
	var b strings.Builder
	if err := r.Render(nil, &b, "hello", 123); err != nil {
		t.Fatalf("Render error: %v", err)
	}
	if ft.lastName != "hello" || ft.lastData.(int) != 123 {
		t.Fatalf("template called with wrong params: %v %v", ft.lastName, ft.lastData)
	}
	if b.String() != "ok" {
		t.Fatalf("unexpected render output: %q", b.String())
	}
}

func TestTemplateRenderer_Render_Error(t *testing.T) {
	ft := &fakeTmpl{shouldErr: true}
	r := &TemplateRenderer{Template: ft}
	var b strings.Builder
	if err := r.Render(nil, &b, "bad", nil); err == nil {
		t.Fatalf("expected error from ExecuteTemplate")
	}
}
