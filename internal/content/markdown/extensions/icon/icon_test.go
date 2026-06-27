package icon

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

func TestIconBasic(t *testing.T) {
	out := render(t, ":icon[rocket]")
	if !strings.Contains(out, "<svg") || !strings.Contains(out, `class="sarde-icon-inline"`) {
		t.Errorf(":icon[rocket] = %q", out)
	}
	if !strings.Contains(out, `aria-hidden="true"`) {
		t.Errorf("expected decorative icon: %q", out)
	}
}

func TestIconClassAttrConcat(t *testing.T) {
	out := render(t, `:icon[github class="sarde-icon-lg"]`)
	if !strings.Contains(out, `class="sarde-icon-inline sarde-icon-lg"`) {
		t.Errorf("class concat = %q", out)
	}
}

func TestIconQuotingForms(t *testing.T) {
	out := render(t, `:icon[arrow-up rotate="90" flip='horizontal' opacity=0.5]`)
	if !strings.Contains(out, `style="transform: rotate(90deg) scaleX(-1)"`) {
		t.Errorf("quoting forms transform = %q", out)
	}
	if !strings.Contains(out, `opacity="0.5"`) {
		t.Errorf("bare value opacity = %q", out)
	}
}

func TestIconAriaLabelRoleImg(t *testing.T) {
	out := render(t, `:icon[home aria-label="Home"]`)
	if !strings.Contains(out, `role="img"`) || !strings.Contains(out, `aria-label="Home"`) {
		t.Errorf("aria-label = %q", out)
	}
}

func TestIconUnknownFallsBack(t *testing.T) {
	out := render(t, ":icon[definitely-not-real-zzz]")
	if !strings.Contains(out, "<svg") {
		t.Errorf("unknown icon should fall back to an svg: %q", out)
	}
}

func TestIconMalformedRendersLiterally(t *testing.T) {
	out := render(t, ":icon[home")
	if strings.Contains(out, "<svg") {
		t.Errorf("unterminated token should not render an svg: %q", out)
	}
	if !strings.Contains(out, ":icon[home") {
		t.Errorf("unterminated token should render literally: %q", out)
	}
}

func TestIconEmptyRendersLiterally(t *testing.T) {
	out := render(t, ":icon[]")
	if strings.Contains(out, "<svg") {
		t.Errorf("empty token should not render an svg: %q", out)
	}
}
