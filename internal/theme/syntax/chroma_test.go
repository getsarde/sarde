package syntax

import (
	"strings"
	"testing"
)

func TestGenerateChromaCSS_ValidThemes(t *testing.T) {
	css, err := GenerateChromaCSS("github", "dracula")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, want := range []string{
		".chroma",
		"background-color:",
		"@layer sarde.components",
	} {
		if !strings.Contains(css, want) {
			t.Errorf("expected CSS to contain %q", want)
		}
	}
}

func TestGenerateChromaCSS_DarkScoping(t *testing.T) {
	css, err := GenerateChromaCSS("github", "dracula")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(css, `[data-theme="dark"] {`) {
		t.Error(`expected dark theme CSS to be wrapped in [data-theme="dark"] { }`)
	}
}

func TestGenerateChromaCSS_InvalidLightTheme(t *testing.T) {
	_, err := GenerateChromaCSS("nonexistent", "dracula")
	if err == nil {
		t.Fatal("expected error for invalid light theme")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("error should mention the invalid theme name: %v", err)
	}
	if !strings.Contains(err.Error(), "light theme") {
		t.Errorf("error should indicate it's the light theme: %v", err)
	}
}

func TestGenerateChromaCSS_InvalidDarkTheme(t *testing.T) {
	_, err := GenerateChromaCSS("github", "nonexistent")
	if err == nil {
		t.Fatal("expected error for invalid dark theme")
	}
	if !strings.Contains(err.Error(), "dark theme") {
		t.Errorf("error should indicate it's the dark theme: %v", err)
	}
}

