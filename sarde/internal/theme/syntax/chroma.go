package syntax

import (
	"bytes"
	"fmt"
	"strings"

	chromahtml "github.com/alecthomas/chroma/v2/formatters/html"
	"github.com/alecthomas/chroma/v2/styles"
)

// GenerateChromaCSS produces CSS rules for syntax highlighting from two
// Chroma style names. The light theme CSS is emitted unscoped; the dark
// theme CSS is wrapped in .dark { } using CSS nesting. Both are wrapped
// in @layer sarde.components so they participate in the cascade correctly.
func GenerateChromaCSS(lightTheme, darkTheme string) (string, error) {
	lightCSS, err := generateForStyle(lightTheme)
	if err != nil {
		return "", fmt.Errorf("light theme: %w", err)
	}
	darkCSS, err := generateForStyle(darkTheme)
	if err != nil {
		return "", fmt.Errorf("dark theme: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("@layer sarde.components {\n")
	sb.WriteString(lightCSS)
	sb.WriteString(".dark {\n")
	sb.WriteString(darkCSS)
	sb.WriteString("}\n")
	sb.WriteString("}\n")
	return sb.String(), nil
}

// GenerateStyleCSS produces CSS rules for a single Chroma style.
func GenerateStyleCSS(name string) (string, error) {
	return generateForStyle(name)
}

// IsKnownStyle reports whether name matches a built-in Chroma style.
func IsKnownStyle(name string) bool {
	return isKnownStyle(name)
}

func generateForStyle(name string) (string, error) {
	if !isKnownStyle(name) {
		return "", fmt.Errorf("unknown chroma style %q (available: %s)",
			name, strings.Join(styles.Names(), ", "))
	}
	style := styles.Get(name)
	f := chromahtml.New(chromahtml.WithClasses(true))
	var buf bytes.Buffer
	if err := f.WriteCSS(&buf, style); err != nil {
		return "", fmt.Errorf("generating CSS for %q: %w", name, err)
	}
	return buf.String(), nil
}

func isKnownStyle(name string) bool {
	for _, n := range styles.Names() {
		if n == name {
			return true
		}
	}
	return false
}
