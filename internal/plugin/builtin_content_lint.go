package plugin

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/coderoo-dev/coderoo/internal/config"
	"github.com/coderoo-dev/coderoo/internal/engine"
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

	rules := lint.Rules

	for _, page := range ctx.Pages {
		if page.Draft {
			continue
		}

		lines := strings.Split(page.RawContent, "\n")
		var issues []lintIssue

		// Heading length check.
		maxLen := rules.HeadingMaxLength
		if maxLen > 0 {
			issues = append(issues, checkHeadingLength(lines, maxLen)...)
		}

		// Heading increment check.
		if config.BoolVal(rules.HeadingIncrement, false) {
			issues = append(issues, checkHeadingIncrement(lines)...)
		}

		// Image alt required.
		if config.BoolVal(rules.ImageAltRequired, false) {
			issues = append(issues, checkImageAlt(lines)...)
		}

		// No empty links.
		if config.BoolVal(rules.NoEmptyLinks, false) {
			issues = append(issues, checkEmptyLinks(lines)...)
		}

		// Required frontmatter fields.
		if len(rules.FrontmatterRequired) > 0 {
			issues = append(issues, checkFrontmatterRequired(page, rules.FrontmatterRequired)...)
		}

		for _, issue := range issues {
			ctx.AddWarning(engine.ValidationWarning{
				File:    page.FilePath,
				Field:   "lint",
				Message: fmt.Sprintf("line %d: %s", issue.Line, issue.Message),
			})
		}
	}

	return nil
}

var headingRegex = regexp.MustCompile(`^(#{1,6})\s+(.*)`)

func checkHeadingLength(lines []string, maxLen int) []lintIssue {
	var issues []lintIssue
	for i, line := range lines {
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

func checkHeadingIncrement(lines []string) []lintIssue {
	var issues []lintIssue
	prevLevel := 0
	for i, line := range lines {
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

func checkImageAlt(lines []string) []lintIssue {
	var issues []lintIssue
	for i, line := range lines {
		if emptyAltRegex.MatchString(line) {
			issues = append(issues, lintIssue{
				Line:    i + 1,
				Message: "image missing alt text",
			})
		}
	}
	return issues
}

var emptyLinkRegex = regexp.MustCompile(`\[\]\(`)

func checkEmptyLinks(lines []string) []lintIssue {
	var issues []lintIssue
	for i, line := range lines {
		// Skip image links (![](..)).
		cleaned := emptyAltRegex.ReplaceAllString(line, "")
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
