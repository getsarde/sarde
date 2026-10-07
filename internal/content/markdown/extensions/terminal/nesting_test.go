package terminal

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
)

func TestClose_CaseInsensitiveNamedCloser(t *testing.T) {
	gm := goldmark.New(goldmark.WithExtensions(&Extension{}))
	var buf bytes.Buffer
	if err := gm.Convert([]byte(":::Terminal\n$ ls\n:::/Terminal\nafter\n"), &buf); err != nil {
		t.Fatal(err)
	}
	out := buf.String()
	if strings.Contains(out, ":::/Terminal") {
		t.Errorf("the closer must not render as text:\n%s", out)
	}
	if !strings.Contains(out, "sarde-terminal") {
		t.Fatalf("terminal block missing:\n%s", out)
	}
	if i, j := strings.LastIndex(out, "</div>"), strings.Index(out, "<p>after</p>"); j < i {
		t.Errorf(":::/Terminal must close a block opened as :::Terminal:\n%s", out)
	}
}
