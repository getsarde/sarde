package theme

import (
	"strings"
	"testing"
)

func TestGenerateCSS_LightOnly(t *testing.T) {
	css := GenerateCSS(map[string]string{"accent": "#6366f1", "bg": "#ffffff"}, nil)

	if !strings.Contains(css, ":root {") {
		t.Error("expected :root block")
	}
	if !strings.Contains(css, "--sd-accent: #6366f1") {
		t.Error("expected --sd-accent")
	}
	if !strings.Contains(css, "--sd-bg: #ffffff") {
		t.Error("expected --sd-bg")
	}
	if strings.Contains(css, `[data-theme="dark"]`) {
		t.Error("should not have dark block with nil dark tokens")
	}
}

func TestGenerateCSS_LightAndDark(t *testing.T) {
	light := map[string]string{"bg": "#ffffff"}
	dark := map[string]string{"bg": "#0f172a"}
	css := GenerateCSS(light, dark)

	if !strings.Contains(css, ":root {") {
		t.Error("expected :root block")
	}
	if !strings.Contains(css, `:root[data-theme="dark"] {`) {
		t.Error(`expected :root[data-theme="dark"] block`)
	}
	if !strings.Contains(css, "--sd-bg: #0f172a") {
		t.Error("expected dark --sd-bg")
	}
}

func TestGenerateCSS_SortedKeys(t *testing.T) {
	tokens := map[string]string{
		"text":    "#1e293b",
		"bg":      "#ffffff",
		"accent": "#6366f1",
	}
	css := GenerateCSS(tokens, nil)

	accentIdx := strings.Index(css, "--sd-accent")
	bgIdx := strings.Index(css, "--sd-bg")
	textIdx := strings.Index(css, "--sd-text")

	if accentIdx > bgIdx || bgIdx > textIdx {
		t.Error("keys should be sorted alphabetically")
	}
}

func TestGenerateCSS_Empty(t *testing.T) {
	css := GenerateCSS(nil, nil)
	if css != "" {
		t.Errorf("expected empty, got %q", css)
	}
}

func TestGenerateStyleTag(t *testing.T) {
	tag := GenerateStyleTag(map[string]string{"accent": "#6366f1"}, nil)

	s := string(tag)
	if !strings.HasPrefix(s, "<style") {
		t.Error("expected <style> prefix")
	}
	if !strings.HasSuffix(s, "</style>") {
		t.Error("expected </style> suffix")
	}
	if !strings.Contains(s, "--sd-accent") {
		t.Error("expected token in style tag")
	}
}

func TestGenerateStyleTag_Empty(t *testing.T) {
	tag := GenerateStyleTag(nil, nil)
	if tag != "" {
		t.Error("expected empty for nil tokens")
	}
}

func TestGenerateLightDarkCSS_DualKeys(t *testing.T) {
	light := map[string]string{"bg": "#ffffff", "text": "#1e293b"}
	dark := map[string]string{"bg": "#0f172a", "text": "#e2e8f0"}
	css := GenerateLightDarkCSS(light, dark)

	if !strings.Contains(css, "light-dark(#ffffff, #0f172a)") {
		t.Error("expected light-dark() for bg")
	}
	if !strings.Contains(css, "light-dark(#1e293b, #e2e8f0)") {
		t.Error("expected light-dark() for text")
	}
}

func TestGenerateLightDarkCSS_LightOnlyKeys(t *testing.T) {
	light := map[string]string{"bg": "#ffffff", "radius": "0.5rem"}
	dark := map[string]string{"bg": "#0f172a"}
	css := GenerateLightDarkCSS(light, dark)

	if !strings.Contains(css, "--sd-radius: 0.5rem;") {
		t.Error("expected plain value for light-only key")
	}
	if strings.Contains(css, "light-dark(0.5rem") {
		t.Error("light-only key should not be wrapped in light-dark()")
	}
}

func TestGenerateLightDarkCSS_ColorScheme(t *testing.T) {
	light := map[string]string{"bg": "#ffffff"}
	dark := map[string]string{"bg": "#0f172a"}
	css := GenerateLightDarkCSS(light, dark)

	if !strings.Contains(css, "color-scheme: light dark;") {
		t.Error("expected color-scheme: light dark on :root")
	}
	if !strings.Contains(css, ":root[data-theme=\"dark\"] {\n  color-scheme: dark;\n}") {
		t.Error(`expected color-scheme: dark on :root[data-theme="dark"]`)
	}
}

func TestGenerateLightDarkCSS_SupportsWrapper(t *testing.T) {
	light := map[string]string{"bg": "#ffffff"}
	css := GenerateLightDarkCSS(light, nil)

	if !strings.Contains(css, "@supports (color: light-dark(red, blue))") {
		t.Error("expected @supports wrapper")
	}
}

func TestGenerateLightDarkCSS_Empty(t *testing.T) {
	css := GenerateLightDarkCSS(nil, nil)
	if css != "" {
		t.Errorf("expected empty, got %q", css)
	}
}

func TestGenerateStyleTag_IncludesBothPaths(t *testing.T) {
	light := map[string]string{"bg": "#ffffff"}
	dark := map[string]string{"bg": "#0f172a"}
	tag := string(GenerateStyleTag(light, dark))

	if !strings.Contains(tag, ":root {") {
		t.Error("expected legacy :root block")
	}
	if !strings.Contains(tag, `:root[data-theme="dark"] {`) {
		t.Error(`expected legacy :root[data-theme="dark"] block`)
	}
	if !strings.Contains(tag, "@supports (color: light-dark(red, blue))") {
		t.Error("expected @supports block with light-dark()")
	}
}

func TestGenerateCSS_FullPipeline(t *testing.T) {
	// Simulate the full pipeline: defaults → theme → preset → derive → CSS.
	theme := &Theme{
		Tokens: map[string]string{"accent": "#3b82f6"},
		Presets: map[string]Preset{
			"ocean": {
				Tokens:     map[string]string{"accent": "#0ea5e9"},
				DarkTokens: map[string]string{"accent": "#38bdf8"},
			},
		},
	}

	light := ResolveTokens(DefaultTokens(), theme, "ocean", nil)
	dark := ResolveDarkTokens(DefaultDarkTokens(), theme, "ocean", nil)
	DeriveTokens(light)

	css := GenerateCSS(light, dark)

	if !strings.Contains(css, "--sd-accent: #0ea5e9") {
		t.Error("expected ocean accent")
	}
	if !strings.Contains(css, "--sd-accent-hover:") {
		t.Error("expected derived accent-hover")
	}
	if !strings.Contains(css, `:root[data-theme="dark"] {`) {
		t.Error("expected dark block")
	}
}
