package content

import (
	"errors"
	"fmt"
	"strings"

	"github.com/frostybee/sarde/internal/engine"
	"github.com/frostybee/valiant"
)

var validLayoutStrings = []string{
	"default", "docs", "splash", "wide", "full",
	"centered", "split", "presentation",
}

var validBadgeVariants = []string{
	"default", "note", "tip", "success", "caution", "danger",
}

func ValidatePageFields(page *engine.Page, fm *engine.Frontmatter) []engine.ValidationWarning {
	var result []engine.ValidationWarning
	result = append(result, validateIdentity(fm)...)
	result = append(result, validateTOC(page)...)
	result = append(result, validateSidebar(page)...)
	result = append(result, validateLayout(fm)...)
	return result
}

func validateIdentity(fm *engine.Frontmatter) []engine.ValidationWarning {
	v := valiant.FromStruct(fm)
	v.Field("Title").Required()
	return collectWarnings(v)
}

func validateTOC(page *engine.Page) []engine.ValidationWarning {
	v := valiant.FromStruct(page)
	v.Warn("TOC.MinLevel").IntRange(1, 6)
	v.Warn("TOC.MaxLevel").IntRange(1, 6)
	v.Warn("TOC.MinLevel").LessOrEqual("TOC.MaxLevel")
	return collectWarnings(v)
}

func validateSidebar(page *engine.Page) []engine.ValidationWarning {
	v := valiant.FromStruct(page)
	v.Warn("Sidebar.Order").IntMin(0)
	warnings := collectWarnings(v)

	// Badge.Variant is a named string type (BadgeVariant), so Valiant's
	// OneOf (which asserts string) cannot be used directly.
	if variant := page.Sidebar.Badge.Variant; variant != "" {
		valid := false
		for _, allowed := range validBadgeVariants {
			if string(variant) == allowed {
				valid = true
				break
			}
		}
		if !valid {
			warnings = append(warnings, engine.ValidationWarning{
				Field:   "sidebar.badge.variant",
				Message: fmt.Sprintf("must be one of: %s (got %q)", strings.Join(validBadgeVariants, ", "), variant),
				Level:   "warn",
			})
		}
	}
	return warnings
}

func validateLayout(fm *engine.Frontmatter) []engine.ValidationWarning {
	if fm.Layout == "" {
		return nil
	}
	v := valiant.FromStruct(fm)
	v.Warn("Layout").OneOf(validLayoutStrings...)
	return collectWarnings(v)
}

// collectWarnings runs the validator and converts Valiant results to ValidationWarnings.
// Handles both error+warning and warning-only cases: Validate() returns nil when
// only warnings fire, so we must also call v.Warnings().
func collectWarnings(v *valiant.Validator) []engine.ValidationWarning {
	err := v.Validate()
	var issues []valiant.Error
	if err != nil {
		var res *valiant.Results
		if errors.As(err, &res) {
			issues = res.AllIssues()
		}
	} else {
		issues = v.Warnings()
	}
	if len(issues) == 0 {
		return nil
	}
	warnings := make([]engine.ValidationWarning, 0, len(issues))
	for _, w := range issues {
		level := "warn"
		if w.Severity == valiant.SeverityError {
			level = "error"
		}
		warnings = append(warnings, engine.ValidationWarning{
			Field:   goPathToYAML(w.Path),
			Message: w.Message,
			Level:   level,
		})
	}
	return warnings
}

var yamlPaths = map[string]string{
	"Title":                 "title",
	"TOC.MinLevel":          "toc.min_level",
	"TOC.MaxLevel":          "toc.max_level",
	"Sidebar.Order":         "sidebar.order",
	"Sidebar.Badge.Variant": "sidebar.badge.variant",
	"Layout":                "layout",
}

func goPathToYAML(path string) string {
	if y, ok := yamlPaths[path]; ok {
		return y
	}
	return path
}
