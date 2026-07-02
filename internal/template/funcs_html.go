package template

import (
	"html"
	htmltemplate "html/template"
	"sort"
	"strings"

	"github.com/getsarde/sarde/internal/engine"
)

// fnRenderHeadTags renders a slice of engine.HeadTag as raw <head> markup,
// skipping any tag not present in engine.AllowedHeadTags. Attribute keys are
// sorted for deterministic output.
func fnRenderHeadTags(v any) htmltemplate.HTML {
	tags, ok := v.([]engine.HeadTag)
	if !ok || len(tags) == 0 {
		return ""
	}
	var sb strings.Builder
	for _, h := range tags {
		if !engine.AllowedHeadTags[h.Tag] {
			continue
		}
		sb.WriteString("<")
		sb.WriteString(h.Tag)
		keys := make([]string, 0, len(h.Attrs))
		for k := range h.Attrs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			sb.WriteString(" ")
			sb.WriteString(html.EscapeString(k))
			sb.WriteString(`="`)
			sb.WriteString(html.EscapeString(h.Attrs[k]))
			sb.WriteString(`"`)
		}
		if h.Content != "" {
			sb.WriteString(">")
			sb.WriteString(html.EscapeString(h.Content))
			sb.WriteString("</")
			sb.WriteString(h.Tag)
			sb.WriteString(">\n")
		} else {
			sb.WriteString(">\n")
		}
	}
	return htmltemplate.HTML(sb.String())
}

// fnRenderAttrs renders a map of HTML attributes as a sorted, escaped
// attribute string (e.g. ` class="x" id="y"`).
func fnRenderAttrs(attrs map[string]string) htmltemplate.HTML {
	if len(attrs) == 0 {
		return ""
	}
	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sb strings.Builder
	for _, k := range keys {
		sb.WriteString(" ")
		sb.WriteString(html.EscapeString(k))
		sb.WriteString(`="`)
		sb.WriteString(html.EscapeString(attrs[k]))
		sb.WriteString(`"`)
	}
	return htmltemplate.HTML(sb.String())
}
