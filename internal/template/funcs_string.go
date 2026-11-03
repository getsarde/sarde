package template

import (
	"bytes"
	"fmt"
	htmltemplate "html/template"
	"strings"
	"sync"
	"unicode"

	"github.com/getsarde/sarde/internal/content"
	"github.com/getsarde/sarde/internal/content/markdown/icons"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	"github.com/yuin/goldmark/util"
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
	// Clamp negative counts: runes[:n] with n < 0 would panic with a runtime
	// error that escapes template recovery and kills the whole build.
	if n < 0 {
		n = 0
	}
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

	// inlineMDResolverKey carries the site's URL resolver into one
	// markdownify call, so the shared Goldmark instance can add the base path.
	inlineMDResolverKey = parser.NewContextKey()
)

func getInlineMD() goldmark.Markdown {
	inlineMDOnce.Do(func() {
		inlineMD = goldmark.New(
			goldmark.WithExtensions(extension.GFM),
			goldmark.WithParserOptions(parser.WithASTTransformers(
				util.Prioritized(sitePathTransformer{}, 100),
			)),
		)
	})
	return inlineMD
}

// sitePathTransformer gives a site-absolute link or image in a markdownify
// value the base path, as the page renderer does for body content. There is
// no page here to look a link up against, so it gets no language prefix.
type sitePathTransformer struct{}

func (sitePathTransformer) Transform(doc *ast.Document, _ text.Reader, pc parser.Context) {
	r, _ := pc.Get(inlineMDResolverKey).(*engine.URLResolver)
	if r == nil {
		return
	}
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch n := n.(type) {
		case *ast.Link:
			n.Destination = []byte(r.AssetURL(string(n.Destination)))
		case *ast.Image:
			n.Destination = []byte(r.AssetURL(string(n.Destination)))
		}
		return ast.WalkContinue, nil
	})
}

func fnMarkdownify(v any) htmltemplate.HTML {
	return markdownifyWith(nil, v)
}

// markdownifyWith renders v as inline Markdown. A non-nil resolver adds the
// base path to site-absolute links and images.
func markdownifyWith(r *engine.URLResolver, v any) htmltemplate.HTML {
	s := fmt.Sprint(v)
	pc := parser.NewContext()
	if r != nil {
		pc.Set(inlineMDResolverKey, r)
	}
	var buf bytes.Buffer
	if err := getInlineMD().Convert([]byte(s), &buf, parser.WithContext(pc)); err != nil {
		return htmltemplate.HTML(htmltemplate.HTMLEscapeString(s))
	}
	result := strings.TrimSpace(buf.String())
	result = strings.TrimPrefix(result, "<p>")
	result = strings.TrimSuffix(result, "</p>")
	return htmltemplate.HTML(result)
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

// fnHighlight is the highlight fallback without a code block engine (the
// shortcode function map, or a failed render): the code in <pre><code>.
func fnHighlight(code, lang string) htmltemplate.HTML {
	escaped := htmltemplate.HTMLEscapeString(code)
	return htmltemplate.HTML(fmt.Sprintf(`<pre><code class="language-%s">%s</code></pre>`, htmltemplate.HTMLEscapeString(lang), escaped))
}

// fnInitials returns up to two uppercase letters for a monogram: the first
// letter of each of the first two words, or the first two letters of a
// single word. Words without a letter (numbers, punctuation) are skipped, and
// HTML tags are stripped first, so "Go Essentials" gives "GE" and
// "Reference" gives "RE".
func fnInitials(s string) string {
	var firsts [][]rune
	for _, w := range strings.Fields(fnPlainify(s)) {
		var letters []rune
		for _, r := range w {
			if unicode.IsLetter(r) {
				letters = append(letters, r)
			}
		}
		if len(letters) > 0 {
			firsts = append(firsts, letters)
		}
	}
	var out []rune
	switch {
	case len(firsts) == 0:
		return ""
	case len(firsts) == 1:
		out = firsts[0]
		if len(out) > 2 {
			out = out[:2]
		}
	default:
		out = []rune{firsts[0][0], firsts[1][0]}
	}
	return strings.ToUpper(string(out))
}
