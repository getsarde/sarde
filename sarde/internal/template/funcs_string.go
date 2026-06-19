package template

import (
	"bytes"
	"fmt"
	htmltemplate "html/template"
	"strings"
	"sync"

	"github.com/frostybee/sarde/internal/content"
	"github.com/frostybee/sarde/internal/content/markdown/icons"
	"github.com/frostybee/sarde/internal/engine"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

func fnTitle(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
		}
	}
	return strings.Join(words, " ")
}

func fnTruncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	if n <= 3 {
		return string(runes[:n])
	}
	return string(runes[:n-3]) + "..."
}

var (
	inlineMDOnce sync.Once
	inlineMD     goldmark.Markdown
)

func getInlineMD() goldmark.Markdown {
	inlineMDOnce.Do(func() {
		inlineMD = goldmark.New(
			goldmark.WithExtensions(extension.GFM),
		)
	})
	return inlineMD
}

func fnMarkdownify(s string) htmltemplate.HTML {
	var buf bytes.Buffer
	if err := getInlineMD().Convert([]byte(s), &buf); err != nil {
		return htmltemplate.HTML(htmltemplate.HTMLEscapeString(s))
	}
	return htmltemplate.HTML(buf.String())
}

func fnPlainify(s string) string {
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func lookupPage(pi *content.PageIndex, site *engine.SiteContext, slug string) *engine.Page {
	if pi != nil {
		if p := pi.LookupByPermalink(slug); p != nil {
			return p
		}
		if p := pi.LookupBySlug(slug); p != nil {
			return p
		}
	}
	if site != nil {
		for _, p := range site.Pages {
			if p.Slug == slug || p.RelPermalink == slug {
				return p
			}
		}
	}
	return nil
}

func fnSafeHTML(s string) htmltemplate.HTML {
	return htmltemplate.HTML(s)
}

// fnIcon renders an inline SVG icon via icons.Render. The first variadic arg is
// a CSS class; the remaining args are key/value attribute pairs (a trailing odd
// arg is ignored):
//
//	{{ icon "rocket" }}
//	{{ icon "rocket" "sarde-icon-lg" }}
//	{{ icon "arrow-up" "sarde-icon" "rotate" "90" "title" "Up" }}
//
// The SVG body is trusted build-time icon data and icons.Render escapes all
// attribute values, so wrapping the result in template.HTML is safe.
func fnIcon(name string, args ...string) htmltemplate.HTML {
	var class string
	attrs := make(map[string]string)
	if len(args) > 0 {
		class = args[0]
		pairs := args[1:]
		for i := 0; i+1 < len(pairs); i += 2 {
			attrs[pairs[i]] = pairs[i+1]
		}
	}
	return htmltemplate.HTML(icons.Render(name, class, attrs))
}

func fnHighlight(code, lang string) htmltemplate.HTML {
	// Stub: wraps code in <pre><code>. Full Chroma syntax highlighting is not yet implemented.
	escaped := htmltemplate.HTMLEscapeString(code)
	return htmltemplate.HTML(fmt.Sprintf(`<pre><code class="language-%s">%s</code></pre>`, htmltemplate.HTMLEscapeString(lang), escaped))
}
