package theme

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

var cssVarRe = regexp.MustCompile(`--sd-([\w-]+)\s*:`)

func TestKnownTokens_MatchesCSS(t *testing.T) {
	data, err := os.ReadFile("../../embedded/theme/css/tokens.css")
	if err != nil {
		t.Fatalf("reading tokens.css: %v", err)
	}
	known := KnownTokens()
	matches := cssVarRe.FindAllStringSubmatch(string(data), -1)
	var missing []string
	for _, m := range matches {
		name := m[1]
		if !known[name] {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		t.Errorf("tokens.css defines tokens not in KnownTokens(): %s", strings.Join(missing, ", "))
	}
}

func TestValidateOverrides_ValidKeys(t *testing.T) {
	known := KnownTokens()
	overrides := map[string]string{
		"accent":     "#e11d48",
		"bg-sidebar": "#f1f5f9",
		"font-sans":  "'Geist', system-ui",
	}
	if err := ValidateOverrides("theme.overrides", overrides, known); err != nil {
		t.Errorf("unexpected error for valid keys: %v", err)
	}
}

func TestValidateOverrides_UnknownKey(t *testing.T) {
	known := KnownTokens()
	overrides := map[string]string{
		"bg-sidbar": "#f1f5f9",
	}
	err := ValidateOverrides("theme.overrides", overrides, known)
	if err == nil {
		t.Fatal("expected error for unknown key")
	}
	if !strings.Contains(err.Error(), "bg-sidbar") {
		t.Errorf("error should mention the unknown key: %v", err)
	}
	if !strings.Contains(err.Error(), "bg-sidebar") {
		t.Errorf("error should suggest bg-sidebar: %v", err)
	}
}

func TestValidateOverrides_NilMap(t *testing.T) {
	known := KnownTokens()
	if err := ValidateOverrides("theme.overrides", nil, known); err != nil {
		t.Errorf("unexpected error for nil map: %v", err)
	}
}

func TestValidateOverrides_EmptyMap(t *testing.T) {
	known := KnownTokens()
	if err := ValidateOverrides("theme.overrides", map[string]string{}, known); err != nil {
		t.Errorf("unexpected error for empty map: %v", err)
	}
}

func TestValidateOverrides_NoSuggestion(t *testing.T) {
	known := KnownTokens()
	overrides := map[string]string{
		"my-brand-color": "#ff0000",
	}
	err := ValidateOverrides("theme.dark_overrides", overrides, known)
	if err == nil {
		t.Fatal("expected error for unknown key")
	}
	if !strings.Contains(err.Error(), "theme.dark_overrides") {
		t.Errorf("error should include the label: %v", err)
	}
	if strings.Contains(err.Error(), "did you mean") {
		t.Errorf("should not suggest for distant key: %v", err)
	}
}

func TestSuggestToken_CloseMatch(t *testing.T) {
	known := KnownTokens()
	if s := SuggestToken("bg-sidbar", known); s != "bg-sidebar" {
		t.Errorf("expected bg-sidebar, got %q", s)
	}
}

func TestSuggestToken_NoMatch(t *testing.T) {
	known := KnownTokens()
	if s := SuggestToken("completely-unrelated-name", known); s != "" {
		t.Errorf("expected empty suggestion, got %q", s)
	}
}

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "", 3},
		{"", "abc", 3},
		{"abc", "abc", 0},
		{"abc", "abd", 1},
		{"bg-sidebar", "bg-sidbar", 1},
		{"font-sans", "fontsans", 1},
	}
	for _, tt := range tests {
		if got := levenshtein(tt.a, tt.b); got != tt.want {
			t.Errorf("levenshtein(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}
