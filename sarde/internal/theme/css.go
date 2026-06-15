package theme

import (
	"fmt"
	htmltemplate "html/template"
	"sort"
	"strings"
)

// GenerateCSS produces CSS custom property blocks for light and dark modes.
// All token keys are prefixed with --sd- and sorted alphabetically.
func GenerateCSS(lightTokens, darkTokens map[string]string) string {
	var sb strings.Builder

	if len(lightTokens) > 0 {
		sb.WriteString(":root {\n")
		writeTokens(&sb, lightTokens)
		sb.WriteString("}\n")
	}

	if len(darkTokens) > 0 {
		sb.WriteString(":root[data-theme=\"dark\"] {\n")
		writeTokens(&sb, darkTokens)
		sb.WriteString("}\n")
	}

	return sb.String()
}

// GenerateStyleTag wraps the generated CSS in a <style> element.
// It emits both the legacy two-block output (baseline) and a light-dark()
// @supports block (progressive enhancement for modern browsers).
func GenerateStyleTag(lightTokens, darkTokens map[string]string) htmltemplate.HTML {
	legacy := GenerateCSS(lightTokens, darkTokens)
	lightDark := GenerateLightDarkCSS(lightTokens, darkTokens)
	css := legacy + lightDark
	if css == "" {
		return ""
	}
	return htmltemplate.HTML(fmt.Sprintf("<style id=\"sd-theme\">\n%s</style>", css))
}

// GenerateLightDarkCSS produces a @supports block that uses the CSS light-dark()
// function for tokens that have both a light and dark value. This is emitted as
// progressive enhancement — browsers that support light-dark() use it, others
// ignore the @supports block and fall back to the legacy two-block output.
func GenerateLightDarkCSS(lightTokens, darkTokens map[string]string) string {
	if len(lightTokens) == 0 {
		return ""
	}

	// Collect all keys from the light map (dark-only keys are skipped — they're
	// already defined in tokens.css and don't need light-dark() wrapping).
	keys := make([]string, 0, len(lightTokens))
	for k := range lightTokens {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var sb strings.Builder
	sb.WriteString("@supports (color: light-dark(red, blue)) {\n")
	sb.WriteString(":root {\n")
	sb.WriteString("  color-scheme: light dark;\n")

	for _, k := range keys {
		lightVal := lightTokens[k]
		if darkVal, hasDark := darkTokens[k]; hasDark && darkVal != lightVal {
			sb.WriteString(fmt.Sprintf("  --sd-%s: light-dark(%s, %s);\n", k, lightVal, darkVal))
		} else {
			sb.WriteString(fmt.Sprintf("  --sd-%s: %s;\n", k, lightVal))
		}
	}

	sb.WriteString("}\n")
	sb.WriteString(":root[data-theme=\"dark\"] {\n")
	sb.WriteString("  color-scheme: dark;\n")
	sb.WriteString("}\n")
	sb.WriteString("}\n")
	return sb.String()
}

// writeTokens writes sorted, --sd- prefixed CSS custom properties to a builder.
func writeTokens(sb *strings.Builder, tokens map[string]string) {
	keys := make([]string, 0, len(tokens))
	for k := range tokens {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		sb.WriteString(fmt.Sprintf("  --sd-%s: %s;\n", k, tokens[k]))
	}
}
