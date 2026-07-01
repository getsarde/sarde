package kbd

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
)

func render(t *testing.T, src string) string {
	t.Helper()
	md := goldmark.New(goldmark.WithExtensions(&Extension{}))
	var buf bytes.Buffer
	if err := md.Convert([]byte(src), &buf); err != nil {
		t.Fatalf("convert %q: %v", src, err)
	}
	return buf.String()
}

func TestKbdSingleKey(t *testing.T) {
	out := render(t, "::kbd[Ctrl]")
	if !strings.Contains(out, `<kbd class="sarde-kbd">Ctrl</kbd>`) {
		t.Errorf("single key = %q", out)
	}
}

func TestKbdCombo(t *testing.T) {
	out := render(t, "::kbd[Ctrl+S]")
	if !strings.Contains(out, `sarde-kbd-group`) {
		t.Errorf("expected kbd-group: %q", out)
	}
	if strings.Count(out, `<kbd class="sarde-kbd">`) != 2 {
		t.Errorf("expected 2 kbd elements: %q", out)
	}
	if !strings.Contains(out, `sarde-kbd-separator`) {
		t.Errorf("expected separator: %q", out)
	}
}

func TestKbdSizeSm(t *testing.T) {
	out := render(t, `::kbd[Ctrl](size="sm")`)
	if !strings.Contains(out, `class="sarde-kbd sarde-kbd-sm"`) {
		t.Errorf("size sm = %q", out)
	}
}

func TestKbdSizeLg(t *testing.T) {
	out := render(t, `::kbd[Ctrl](size="lg")`)
	if !strings.Contains(out, `class="sarde-kbd sarde-kbd-lg"`) {
		t.Errorf("size lg = %q", out)
	}
}

func TestKbdWide(t *testing.T) {
	out := render(t, "::kbd[Space](wide)")
	if !strings.Contains(out, `sarde-kbd sarde-kbd-wide`) {
		t.Errorf("wide = %q", out)
	}
}

func TestKbdSizeLgWide(t *testing.T) {
	out := render(t, `::kbd[Enter](size="lg" wide)`)
	if !strings.Contains(out, `sarde-kbd-lg`) {
		t.Errorf("expected lg: %q", out)
	}
	if !strings.Contains(out, `sarde-kbd-wide`) {
		t.Errorf("expected wide: %q", out)
	}
}

func TestKbdComboWithSize(t *testing.T) {
	out := render(t, `::kbd[Ctrl+S](size="sm")`)
	count := strings.Count(out, `sarde-kbd sarde-kbd-sm`)
	if count != 2 {
		t.Errorf("expected 2 kbd-sm elements, got %d: %q", count, out)
	}
}

func TestKbdInvalidSize(t *testing.T) {
	out := render(t, `::kbd[X](size="xl")`)
	if !strings.Contains(out, `class="sarde-kbd"`) {
		t.Errorf("expected plain sarde-kbd: %q", out)
	}
	if strings.Contains(out, `sarde-kbd-xl`) {
		t.Errorf("should not contain invalid size class: %q", out)
	}
}

func TestKbdNoAttrBlock(t *testing.T) {
	out := render(t, "::kbd[Alt+F4]")
	if !strings.Contains(out, `class="sarde-kbd"`) {
		t.Errorf("expected plain sarde-kbd: %q", out)
	}
	if strings.Contains(out, `sarde-kbd-sm`) || strings.Contains(out, `sarde-kbd-lg`) || strings.Contains(out, `sarde-kbd-wide`) {
		t.Errorf("should not have modifier classes: %q", out)
	}
}

func TestKbdHTMLEscape(t *testing.T) {
	out := render(t, "::kbd[<b>]")
	if strings.Contains(out, "<b>") && !strings.Contains(out, "&lt;b&gt;") {
		t.Errorf("expected escaped HTML: %q", out)
	}
}
