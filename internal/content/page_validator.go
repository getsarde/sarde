package content

import (
	"errors"
	"fmt"
	"strings"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/frostybee/edict"
)

var knownFrontmatterKeys = map[string]bool{
	"title": true, "slug": true, "date": true, "updated": true,
	"publish_date": true, "expiry_date": true, "aliases": true,
	"layout": true, "type": true, "template": true,
	"draft": true, "description": true, "image": true,
	"summary": true, "render": true, "pagefind": true,
	"show_updated": true, "edit_url": true,
	"sidebar": true, "toc": true, "prev": true, "next": true,
	"tags": true, "categories": true, "show_tags": true, "transparent": true,
	"hero": true, "icon": true, "head": true, "banner": true,
	"cascade": true, "params": true,
}

var knownSidebarKeys = map[string]bool{
	"order": true, "label": true, "hidden": true,
	"group": true, "attrs": true, "badge": true,
	"icon": true,
}

var knownTOCKeys = map[string]bool{
	"enabled": true, "min_level": true, "max_level": true,
}

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
	result = append(result, validateSlug(fm)...)
	result = append(result, validateHeadTags(fm)...)
	return result
}

// DetectUnknownFields warns about frontmatter keys not recognized by Sarde.
// Schema-defined custom fields, taxonomy keys from taxCfg, and children of
// cascade/params are excluded.
func DetectUnknownFields(fmMap map[string]any, schema *engine.FrontmatterSchema, taxCfg map[string]config.TaxonomyConfig, filePath string) []engine.ValidationWarning {
	var warnings []engine.ValidationWarning

	for key := range fmMap {
		if knownFrontmatterKeys[key] {
			continue
		}
		if key == "cascade" || key == "params" {
			continue
		}
		if schema != nil {
			if _, ok := schema.Fields[key]; ok {
				continue
			}
		}
		if _, ok := taxCfg[key]; ok {
			continue
		}
		warnings = append(warnings, engine.ValidationWarning{
			File:    filePath,
			Field:   key,
			Message: fmt.Sprintf("unknown frontmatter key %q", key),
			Level:   "warning",
		})
	}

	if sidebar, ok := fmMap["sidebar"].(map[string]any); ok {
		for key := range sidebar {
			if !knownSidebarKeys[key] {
				warnings = append(warnings, engine.ValidationWarning{
					File:    filePath,
					Field:   "sidebar." + key,
					Message: fmt.Sprintf("unknown sidebar key %q", key),
					Level:   "warning",
				})
			}
		}
	}

	if toc, ok := fmMap["toc"].(map[string]any); ok {
		for key := range toc {
			if !knownTOCKeys[key] {
				warnings = append(warnings, engine.ValidationWarning{
					File:    filePath,
					Field:   "toc." + key,
					Message: fmt.Sprintf("unknown toc key %q", key),
					Level:   "warning",
				})
			}
		}
	}

	return warnings
}

func validateIdentity(fm *engine.Frontmatter) []engine.ValidationWarning {
	v := edict.FromStruct(fm)
	v.Field("Title").Required()
	return collectWarnings(v)
}

func validateTOC(page *engine.Page) []engine.ValidationWarning {
	v := edict.FromStruct(page)
	v.Warn("TOC.MinLevel").IntRange(1, 6)
	v.Warn("TOC.MaxLevel").IntRange(1, 6)
	v.Warn("TOC.MinLevel").LessOrEqual("TOC.MaxLevel")
	return collectWarnings(v)
}

func validateSidebar(page *engine.Page) []engine.ValidationWarning {
	v := edict.FromStruct(page)
	v.Warn("Sidebar.Order").IntMin(0)
	warnings := collectWarnings(v)

	// Badge.Variant is a named string type (BadgeVariant), so Edict's
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
	v := edict.FromStruct(fm)
	v.Warn("Layout").OneOf(validLayoutStrings...)
	return collectWarnings(v)
}

func validateSlug(fm *engine.Frontmatter) []engine.ValidationWarning {
	if fm.Slug != "" && strings.TrimSpace(fm.Slug) == "" {
		return []engine.ValidationWarning{{
			Field:   "slug",
			Message: "slug is whitespace-only; this would produce a broken URL",
			Level:   "warning",
		}}
	}
	return nil
}

func validateHeadTags(fm *engine.Frontmatter) []engine.ValidationWarning {
	var warnings []engine.ValidationWarning
	for i, ht := range fm.Head {
		if ht.Tag == "" {
			continue
		}
		if !engine.AllowedHeadTags[ht.Tag] {
			warnings = append(warnings, engine.ValidationWarning{
				Field:   fmt.Sprintf("head[%d].tag", i),
				Message: fmt.Sprintf("head tag %q is not allowed; permitted tags: meta, link, script, style, noscript, base", ht.Tag),
				Level:   "warning",
			})
		}
	}
	return warnings
}

// collectWarnings runs the validator and converts Edict results to ValidationWarnings.
// Handles both error+warning and warning-only cases: Validate() returns nil when
// only warnings fire, so we must also call v.Warnings().
func collectWarnings(v *edict.Validator) []engine.ValidationWarning {
	err := v.Validate()
	var issues []edict.Error
	if err != nil {
		var res *edict.Results
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
		if w.Severity == edict.SeverityError {
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
