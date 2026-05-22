package shortcode

import (
	"bytes"
	"fmt"
	htmltemplate "html/template"
	"strings"

	"github.com/coderoo-dev/coderoo/internal/engine"
)

const maxRecursionDepth = 10

// ShortcodeContext is the data object passed to every shortcode template.
type ShortcodeContext struct {
	Params map[string]string
	Inner  htmltemplate.HTML
	Page   *engine.Page
	Site   *engine.SiteContext
}

// Processor applies shortcode substitution to raw markdown before Goldmark runs.
type Processor struct {
	registry *Registry
}

// NewProcessor creates a Processor backed by the given registry.
func NewProcessor(registry *Registry) *Processor {
	return &Processor{registry: registry}
}

// Process applies all shortcode substitutions to the markdown source.
// mdRenderer is passed per-call for goroutine safety (caller passes its pool-borrowed renderer).
func (p *Processor) Process(
	markdown string,
	page *engine.Page,
	site *engine.SiteContext,
	mdRenderer engine.MarkdownRenderer,
) (string, []engine.ValidationWarning) {
	if !strings.Contains(markdown, "{{<") {
		return markdown, nil
	}

	sanitized, blocks := stripCodeBlocks(markdown)

	var allWarnings []engine.ValidationWarning
	result := sanitized

	for depth := 0; depth < maxRecursionDepth; depth++ {
		next, changed, warnings := p.processOnce(result, page, site, mdRenderer)
		allWarnings = append(allWarnings, warnings...)
		if !changed {
			break
		}
		result = next
		if depth == maxRecursionDepth-1 && changed {
			allWarnings = append(allWarnings, engine.ValidationWarning{
				File:    page.FilePath,
				Field:   "shortcodes",
				Message: "maximum shortcode recursion depth exceeded",
				Level:   "warning",
			})
		}
	}

	result = restoreCodeBlocks(result, blocks)
	return result, allWarnings
}

func (p *Processor) processOnce(
	src string,
	page *engine.Page,
	site *engine.SiteContext,
	mdRenderer engine.MarkdownRenderer,
) (string, bool, []engine.ValidationWarning) {
	var warnings []engine.ValidationWarning
	changed := false

	// Process self-closing shortcodes first.
	result := reSelfClosing.ReplaceAllStringFunc(src, func(match string) string {
		sub := reSelfClosing.FindStringSubmatch(match)
		if sub == nil {
			return match
		}
		name := sub[1]
		params := ParseParams(sub[2])

		output, warn := p.renderShortcode(name, params, "", page, site)
		if warn != nil {
			warnings = append(warnings, *warn)
			if output == "" {
				return match
			}
		}
		changed = true
		return output
	})

	// Process paired (opening + closing) shortcodes.
	result = p.replacePaired(result, page, site, mdRenderer, &warnings, &changed)

	return result, changed, warnings
}

func (p *Processor) replacePaired(
	src string,
	page *engine.Page,
	site *engine.SiteContext,
	mdRenderer engine.MarkdownRenderer,
	warnings *[]engine.ValidationWarning,
	changed *bool,
) string {
	var out strings.Builder
	pos := 0

	for pos < len(src) {
		loc := reOpening.FindStringIndex(src[pos:])
		if loc == nil {
			out.WriteString(src[pos:])
			break
		}

		// Write everything before this opening tag.
		out.WriteString(src[pos : pos+loc[0]])

		openMatch := reOpening.FindStringSubmatch(src[pos+loc[0]:])
		if openMatch == nil {
			out.WriteString(src[pos+loc[0] : pos+loc[1]])
			pos += loc[1]
			continue
		}

		name := openMatch[1]
		params := ParseParams(openMatch[2])
		openEnd := pos + loc[1]

		// Find the matching closing tag with bracket balancing.
		closeStart, closeEnd, found := p.findMatchingClose(src, openEnd, name)
		if !found {
			*warnings = append(*warnings, engine.ValidationWarning{
				File:    page.FilePath,
				Field:   name,
				Message: fmt.Sprintf("unclosed shortcode %q", name),
				Level:   "warning",
			})
			out.WriteString(src[pos+loc[0] : pos+loc[1]])
			pos = openEnd
			continue
		}

		inner := src[openEnd:closeStart]

		// Render inner markdown to HTML.
		var innerHTML htmltemplate.HTML
		if strings.TrimSpace(inner) != "" && mdRenderer != nil {
			result, err := mdRenderer.Render(inner)
			if err == nil {
				innerHTML = htmltemplate.HTML(result.HTML)
			} else {
				innerHTML = htmltemplate.HTML(inner)
			}
		}

		output, warn := p.renderShortcode(name, params, innerHTML, page, site)
		if warn != nil {
			*warnings = append(*warnings, *warn)
			if output == "" {
				out.WriteString(src[pos+loc[0] : closeEnd])
				pos = closeEnd
				continue
			}
		}

		*changed = true
		out.WriteString(output)
		pos = closeEnd
	}

	return out.String()
}

// findMatchingClose locates the closing tag for the given shortcode name,
// handling nested same-name shortcodes with bracket balancing.
func (p *Processor) findMatchingClose(src string, start int, name string) (closeStart, closeEnd int, found bool) {
	depth := 1
	pos := start

	for pos < len(src) {
		// Check for another opening of the same name.
		openLoc := reOpening.FindStringIndex(src[pos:])
		closeLoc := reClosing.FindStringIndex(src[pos:])

		if closeLoc == nil {
			return 0, 0, false
		}

		// If there's an opening tag before the next closing tag, check if it's same name.
		if openLoc != nil && openLoc[0] < closeLoc[0] {
			openMatch := reOpening.FindStringSubmatch(src[pos+openLoc[0]:])
			if openMatch != nil && openMatch[1] == name {
				depth++
			}
			pos += openLoc[1]
			continue
		}

		// Process the closing tag.
		closeMatch := reClosing.FindStringSubmatch(src[pos+closeLoc[0]:])
		if closeMatch != nil && closeMatch[1] == name {
			depth--
			if depth == 0 {
				return pos + closeLoc[0], pos + closeLoc[1], true
			}
		}
		pos += closeLoc[1]
	}

	return 0, 0, false
}

func (p *Processor) renderShortcode(
	name string,
	params map[string]string,
	inner htmltemplate.HTML,
	page *engine.Page,
	site *engine.SiteContext,
) (string, *engine.ValidationWarning) {
	tmpl := p.registry.Resolve(name)
	if tmpl == nil {
		return "", &engine.ValidationWarning{
			File:    page.FilePath,
			Field:   name,
			Message: fmt.Sprintf("unknown shortcode %q", name),
			Level:   "warning",
		}
	}

	ctx := ShortcodeContext{
		Params: params,
		Inner:  inner,
		Page:   page,
		Site:   site,
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, ctx); err != nil {
		return "", &engine.ValidationWarning{
			File:    page.FilePath,
			Field:   name,
			Message: fmt.Sprintf("shortcode %q execution error: %v", name, err),
			Level:   "warning",
		}
	}

	return buf.String(), nil
}
