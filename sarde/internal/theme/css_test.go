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
	if !strings.Contains(css, "--sarde-accent: #6366f1") {
		t.Error("expected --sarde-accent")
	}
	if !strings.Contains(css, "--sarde-bg: #ffffff") {
		t.Error("expected --sarde-bg")
	}
	if strings.Contains(css, ":root.dark") {
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
	if !strings.Contains(css, ":root.dark {") {
		t.Error("expected :root.dark block")
	}
	if !strings.Contains(css, "--sarde-bg: #0f172a") {
		t.Error("expected dark --sarde-bg")
	}
}

func TestGenerateCSS_SortedKeys(t *testing.T) {
	tokens := map[string]string{
		"text":    "#1e293b",
		"bg":      "#ffffff",
		"accent": "#6366f1",
	}
	css := GenerateCSS(tokens, nil)

	accentIdx := strings.Index(css, "--sarde-accent")
	bgIdx := strings.Index(css, "--sarde-bg")
	textIdx := strings.Index(css, "--sarde-text")

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
	if !strings.Contains(s, "--sarde-accent") {
		t.Error("expected token in style tag")
	}
}

func TestGenerateStyleTag_Empty(t *testing.T) {
	tag := GenerateStyleTag(nil, nil)
	if tag != "" {
		t.Error("expected empty for nil tokens")
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

	if !strings.Contains(css, "--sarde-accent: #0ea5e9") {
		t.Error("expected ocean accent")
	}
	if !strings.Contains(css, "--sarde-accent-hover:") {
		t.Error("expected derived accent-hover")
	}
	if !strings.Contains(css, ":root.dark {") {
		t.Error("expected dark block")
	}
}
