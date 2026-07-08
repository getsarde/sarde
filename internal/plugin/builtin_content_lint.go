package plugin

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/engine"
)

func newContentLintPlugin(cfg map[string]any) *Plugin {
	return &Plugin{
		Name: "content_lint",
		Hooks: PluginHooks{
			BuildDone: func(ctx *BuildDoneContext) error {
				return contentLintBuildDone(ctx)
			},
		},
	}
}

type lintIssue struct {
	Line    int
	Message string
}

func contentLintBuildDone(ctx *BuildDoneContext) error {
	lint := ctx.Config.ContentLint
	if lint.Enabled != nil && !*lint.Enabled {
		return nil
	}

	// On incremental rebuilds only the changed pages are re-linted; warnings
	// are per-rebuild and ephemeral in the dev server, so issues on unchanged
	// pages are simply not re-reported until the next full build or edit to
	// the offending file.
	pages := ctx.Pages
	if ctx.Incremental {
		pages = ctx.ChangedPages
	}
	warnings := LintPages(pages, lint)
	for _, w := range warnings {
		ctx.AddWarning(w)
	}

	if len(warnings) > 0 {
		ctx.Log(fmt.Sprintf("Found %d lint issue(s)", len(warnings)))
	}
	return nil
}

// LintPages runs content lint rules on a set of pages and returns warnings.
// This is exported so that the validate command can call it directly without
// going through the plugin BuildDone hook.
func LintPages(pages []*engine.Page, lint config.ContentLintSettings) []engine.ValidationWarning {
	rules := lint.Rules
	var warnings []engine.ValidationWarning

	for _, page := range pages {
		if page.Draft {
			continue
		}

		lines := strings.Split(page.RawContent, "\n")
		fenced := fencedLines(lines)
		var issues []lintIssue

		if rules.HeadingMaxLength > 0 {
			issues = append(issues, checkHeadingLength(lines, fenced, rules.HeadingMaxLength)...)
		}
		if config.BoolVal(rules.HeadingIncrement, false) {
			issues = append(issues, checkHeadingIncrement(lines, fenced)...)
		}
		if config.BoolVal(rules.ImageAltRequired, false) {
			issues = append(issues, checkImageAlt(lines, fenced)...)
		}
		if config.BoolVal(rules.NoEmptyLinks, false) {
			issues = append(issues, checkEmptyLinks(lines, fenced)...)
		}
		// Line-scanning rules report lines relative to RawContent, which has
		// frontmatter stripped; shift to on-disk line numbers.
		for i := range issues {
			issues[i].Line += page.FrontmatterLines
		}
		if len(rules.FrontmatterRequired) > 0 {
			issues = append(issues, checkFrontmatterRequired(page, rules.FrontmatterRequired)...)
		}

		for _, issue := range issues {
			warnings = append(warnings, engine.ValidationWarning{
				File:    page.FilePath,
				Field:   "lint",
				Message: fmt.Sprintf("line %d: %s", issue.Line, issue.Message),
			})
		}
	}

	return warnings
}

var headingRegex = regexp.MustCompile(`^(#{1,6})\s+(.*)`)

// fencedLines marks the lines that belong to fenced code blocks, including
// the fence delimiter lines themselves. A fence opens with three or more
// backticks or tildes and closes on a run of the same character at least as
// long as the opener with nothing else on the line; an unclosed fence extends
// to the end of the content.
func fencedLines(lines []string) []bool {
	mask := make([]bool, len(lines))
	var fenceChar byte
	fenceLen := 0
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if fenceChar == 0 {
			if n := leadingRun(trimmed, '`'); n >= 3 {
				fenceChar, fenceLen = '`', n
				mask[i] = true
			} else if n := leadingRun(trimmed, '~'); n >= 3 {
				fenceChar, fenceLen = '~', n
				mask[i] = true
			}
			continue
		}
		mask[i] = true
		if n := leadingRun(trimmed, fenceChar); n >= fenceLen && n == len(trimmed) {
			fenceChar, fenceLen = 0, 0
		}
	}
	return mask
}

func leadingRun(s string, ch byte) int {
	n := 0
	for n < len(s) && s[n] == ch {
		n++
	}
	return n
}

// stripCodeSpans blanks inline code spans (backtick-delimited, closing run of
// the same length as the opener) so the image/link regexes cannot match
// example syntax shown as inline code. Spans are replaced with spaces to keep
// column positions; unmatched backtick runs are left untouched.
func stripCodeSpans(line string) string {
	if !strings.Contains(line, "`") {
		return line
	}
	out := []byte(line)
	i := 0
	for i < len(out) {
		if out[i] != '`' {
			i++
			continue
		}
		open := i
		for i < len(out) && out[i] == '`' {
			i++
		}
		end := closingRunEnd(out, i, i-open)
		if end < 0 {
			continue
		}
		for k := open; k < end; k++ {
			out[k] = ' '
		}
		i = end
	}
	return string(out)
}

// closingRunEnd returns the index just past the next run of exactly n
// backticks at or after position i, or -1 if none exists.
func closingRunEnd(b []byte, i, n int) int {
	for i < len(b) {
		if b[i] != '`' {
			i++
			continue
		}
		start := i
		for i < len(b) && b[i] == '`' {
			i++
		}
		if i-start == n {
			return i
		}
	}
	return -1
}

func checkHeadingLength(lines []string, fenced []bool, maxLen int) []lintIssue {
	var issues []lintIssue
	for i, line := range lines {
		if fenced[i] {
			continue
		}
		m := headingRegex.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		text := strings.TrimSpace(m[2])
		if len(text) > maxLen {
			issues = append(issues, lintIssue{
				Line:    i + 1,
				Message: fmt.Sprintf("heading exceeds %d characters (%d)", maxLen, len(text)),
			})
		}
	}
	return issues
}

func checkHeadingIncrement(lines []string, fenced []bool) []lintIssue {
	var issues []lintIssue
	prevLevel := 0
	for i, line := range lines {
		if fenced[i] {
			continue
		}
		m := headingRegex.FindStringSubmatch(line)
		if m == nil {
			continue
		}
		level := len(m[1])
		if prevLevel > 0 && level > prevLevel+1 {
			issues = append(issues, lintIssue{
				Line:    i + 1,
				Message: fmt.Sprintf("heading level skipped from h%d to h%d", prevLevel, level),
			})
		}
		prevLevel = level
	}
	return issues
}

var emptyAltRegex = regexp.MustCompile(`!\[\]\(`)

func checkImageAlt(lines []string, fenced []bool) []lintIssue {
	var issues []lintIssue
	for i, line := range lines {
		if fenced[i] {
			continue
		}
		if emptyAltRegex.MatchString(stripCodeSpans(line)) {
			issues = append(issues, lintIssue{
				Line:    i + 1,
				Message: "image missing alt text",
			})
		}
	}
	return issues
}

var emptyLinkRegex = regexp.MustCompile(`\[\]\(`)

func checkEmptyLinks(lines []string, fenced []bool) []lintIssue {
	var issues []lintIssue
	for i, line := range lines {
		if fenced[i] {
			continue
		}
		// Skip image links (![](..)).
		cleaned := emptyAltRegex.ReplaceAllString(stripCodeSpans(line), "")
		if emptyLinkRegex.MatchString(cleaned) {
			issues = append(issues, lintIssue{
				Line:    i + 1,
				Message: "link has empty text",
			})
		}
	}
	return issues
}

func checkFrontmatterRequired(page *engine.Page, required []string) []lintIssue {
	var issues []lintIssue
	for _, field := range required {
		if !hasFrontmatterField(page, field) {
			issues = append(issues, lintIssue{
				Line:    1,
				Message: fmt.Sprintf("required frontmatter field %q is missing", field),
			})
		}
	}
	return issues
}

func hasFrontmatterField(page *engine.Page, field string) bool {
	switch field {
	case "title":
		return page.Title != ""
	case "description":
		return page.Description != ""
	case "date":
		return !page.Date.IsZero()
	case "image":
		return page.Image != ""
	case "tags":
		return len(page.Tags) > 0
	case "categories":
		return len(page.Categories) > 0
	default:
		if page.Params != nil {
			_, ok := page.Params[field]
			return ok
		}
		return false
	}
}
