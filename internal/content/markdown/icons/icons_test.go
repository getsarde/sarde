package icons

import (
	"strings"
	"testing"
)

func TestRenderDecorativeByDefault(t *testing.T) {
	got := Render("rocket", "sarde-icon", nil)
	for _, want := range []string{
		`<svg`,
		`class="sarde-icon"`,
		`viewBox="0 0 24 24"`,
		`aria-hidden="true"`,
		`focusable="false"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("Render(rocket) missing %q in %q", want, got)
		}
	}
	if strings.Contains(got, `role="img"`) {
		t.Errorf("decorative icon should not have role=img: %q", got)
	}
}

func TestRenderRoleImgWhenLabeled(t *testing.T) {
	got := Render("home", "", map[string]string{"aria-label": "Home"})
	if !strings.Contains(got, `aria-label="Home"`) || !strings.Contains(got, `role="img"`) {
		t.Errorf("labeled icon = %q, want aria-label + role=img", got)
	}
	if strings.Contains(got, "aria-hidden") {
		t.Errorf("labeled icon should not be aria-hidden: %q", got)
	}
}

func TestRenderTitlePromotesAndPrepends(t *testing.T) {
	got := Render("arrow-up", "", map[string]string{"title": "Go up"})
	if !strings.Contains(got, `<title>Go up</title>`) {
		t.Errorf("missing <title>: %q", got)
	}
	if !strings.Contains(got, `role="img"`) || strings.Contains(got, "aria-hidden") {
		t.Errorf("title should promote to role=img and drop aria-hidden: %q", got)
	}
}

func TestRenderTransforms(t *testing.T) {
	tests := []struct {
		name  string
		attrs map[string]string
		want  string
	}{
		{"rotate", map[string]string{"rotate": "90"}, `style="transform: rotate(90deg)"`},
		{"flip-h", map[string]string{"flip": "horizontal"}, `style="transform: scaleX(-1)"`},
		{"flip-both", map[string]string{"flip": "both"}, `style="transform: scaleX(-1) scaleY(-1)"`},
		{"rotate-flip", map[string]string{"rotate": "90", "flip": "vertical"}, `style="transform: rotate(90deg) scaleY(-1)"`},
	}
	for _, tt := range tests {
		got := Render("arrow-up", "", tt.attrs)
		if !strings.Contains(got, tt.want) {
			t.Errorf("%s: Render missing %q in %q", tt.name, tt.want, got)
		}
	}
}

func TestRenderOpacityPassThrough(t *testing.T) {
	got := Render("rocket", "", map[string]string{"opacity": "0.5"})
	if !strings.Contains(got, `opacity="0.5"`) {
		t.Errorf("missing opacity attr: %q", got)
	}
}

func TestRenderClassConcat(t *testing.T) {
	got := Render("rocket", "base", map[string]string{"class": "extra"})
	if !strings.Contains(got, `class="base extra"`) {
		t.Errorf("class concat = %q, want class=\"base extra\"", got)
	}
}

func TestRenderFallback(t *testing.T) {
	unknown := Render("definitely-not-a-real-icon-zzz", "c", map[string]string{"title": "x"})
	fallback := Render("circle-help", "c", map[string]string{"title": "x"})
	if unknown == "" {
		t.Fatal("fallback returned empty")
	}
	if unknown != fallback {
		t.Errorf("unknown icon should fall back to circle-help:\n got = %q\nwant = %q", unknown, fallback)
	}
}

func TestRenderEscapesAttrValues(t *testing.T) {
	got := Render("rocket", "", map[string]string{"data-x": `a"b<c`})
	if !strings.Contains(got, `data-x="a&#34;b&lt;c"`) {
		t.Errorf("attr value not escaped: %q", got)
	}
}

func TestRenderDropsInvalidAttrKey(t *testing.T) {
	got := Render("rocket", "", map[string]string{"bad key": "v"})
	if strings.Contains(got, "bad") {
		t.Errorf("invalid attr key should be dropped: %q", got)
	}
}

func TestGetWithClassUnchanged(t *testing.T) {
	// Block-extension path: no fallback, width/height hardcoded to 16, decorative aria-hidden.
	got := GetWithClass("rocket", "sarde-aside-icon")
	if !strings.Contains(got, `class="sarde-aside-icon"`) {
		t.Errorf("GetWithClass missing class: %q", got)
	}
	if !strings.Contains(got, `width="16"`) || !strings.Contains(got, `height="16"`) {
		t.Errorf("GetWithClass should hardcode width/height to 16: %q", got)
	}
	if !strings.Contains(got, `aria-hidden="true"`) || !strings.Contains(got, `focusable="false"`) {
		t.Errorf("GetWithClass should emit aria-hidden and focusable: %q", got)
	}
	if GetWithClass("definitely-not-real-zzz", "") != "" {
		t.Error("GetWithClass should return empty for an unknown icon (no fallback)")
	}
}
