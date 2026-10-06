package template

import (
	"fmt"
	"html"
	htmltemplate "html/template"
	"sort"
	"strings"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/engine"
)

// fnRenderHeadTags renders head tags as raw <head> markup, skipping any tag
// not present in engine.AllowedHeadTags. It takes frontmatter tags
// ([]engine.HeadTag), site config tags ([]config.HeadTag) and tags set
// through a section cascade (the decoded YAML, a []any of maps). Attribute
// keys are sorted for deterministic output. The content of script, style
// and noscript is raw text, written as given (escaping it would turn the
// quotes in CSS or JS into entities), except that a closing tag inside it
// is neutralized so it cannot end the element early.
func fnRenderHeadTags(v any) htmltemplate.HTML {
	tags := headTagsOf(v)
	if len(tags) == 0 {
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
			sb.WriteString(headTagContent(h.Tag, h.Content))
			sb.WriteString("</")
			sb.WriteString(h.Tag)
			sb.WriteString(">\n")
		} else {
			sb.WriteString(">\n")
		}
	}
	return htmltemplate.HTML(sb.String())
}

// headTagsOf normalizes the shapes head tags arrive in to engine.HeadTag.
func headTagsOf(v any) []engine.HeadTag {
	switch tags := v.(type) {
	case []engine.HeadTag:
		return tags
	case []config.HeadTag:
		out := make([]engine.HeadTag, len(tags))
		for i, t := range tags {
			out[i] = engine.HeadTag(t)
		}
		return out
	case []any:
		out := make([]engine.HeadTag, 0, len(tags))
		for _, item := range tags {
			m, ok := item.(map[string]any)
			if !ok {
				continue
			}
			tag := engine.HeadTag{
				Tag:     strings.ToLower(strings.TrimSpace(stringOf(m["tag"]))),
				Content: stringOf(m["content"]),
			}
			if attrs, ok := m["attrs"].(map[string]any); ok {
				tag.Attrs = make(map[string]string, len(attrs))
				for k, val := range attrs {
					tag.Attrs[k] = stringOf(val)
				}
			}
			out = append(out, tag)
		}
		return out
	}
	return nil
}

func stringOf(v any) string {
	switch s := v.(type) {
	case nil:
		return ""
	case string:
		return s
	default:
		return fmt.Sprint(s)
	}
}

// rawTextTags hold content the browser does not decode: escaping it would
// put entities into the CSS or JS.
var rawTextTags = map[string]bool{"script": true, "style": true, "noscript": true}

// headTagContent is the content to write between a head tag's open and
// close tags. Raw-text tags keep their content, with any closing tag for
// the element broken up (`</script` becomes `<\/script`, which JS and CSS
// read the same inside a string); other tags are HTML-escaped.
func headTagContent(tag, content string) string {
	if !rawTextTags[tag] {
		return html.EscapeString(content)
	}
	closing := "</" + tag
	lower := strings.ToLower(content)
	if !strings.Contains(lower, closing) {
		return content
	}
	var sb strings.Builder
	for {
		i := strings.Index(lower, closing)
		if i < 0 {
			sb.WriteString(content)
			return sb.String()
		}
		sb.WriteString(content[:i])
		sb.WriteString(`<\/`)
		content = content[i+2:]
		lower = lower[i+2:]
	}
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
