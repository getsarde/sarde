package theme

import (
	"fmt"
	htmltemplate "html/template"
	"sort"
	"strings"
)

// GenerateCSS produces CSS custom property blocks for light and dark modes.
// All token keys are prefixed with --fb- and sorted alphabetically.
func GenerateCSS(lightTokens, darkTokens map[string]string) string {
	var sb strings.Builder

	if len(lightTokens) > 0 {
		sb.WriteString(":root {\n")
		writeTokens(&sb, lightTokens)
		sb.WriteString("}\n")
	}

	if len(darkTokens) > 0 {
		sb.WriteString(":root.dark {\n")
		writeTokens(&sb, darkTokens)
		sb.WriteString("}\n")
	}

	return sb.String()
}

// GenerateStyleTag wraps the generated CSS in a <style> element.
func GenerateStyleTag(lightTokens, darkTokens map[string]string) htmltemplate.HTML {
	css := GenerateCSS(lightTokens, darkTokens)
	if css == "" {
		return ""
	}
	return htmltemplate.HTML(fmt.Sprintf("<style id=\"fb-theme\">\n%s</style>", css))
}

// writeTokens writes sorted, --fb- prefixed CSS custom properties to a builder.
func writeTokens(sb *strings.Builder, tokens map[string]string) {
	keys := make([]string, 0, len(tokens))
	for k := range tokens {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		sb.WriteString(fmt.Sprintf("  --fb-%s: %s;\n", k, tokens[k]))
	}
}
