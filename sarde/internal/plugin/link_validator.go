package plugin

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/frostybee/sarde/internal/config"
	"github.com/frostybee/sarde/internal/content"
	"github.com/frostybee/sarde/internal/engine"
)

const (
	errInvalidLink  = "invalid link"
	errInvalidHash  = "invalid hash"
	errRelativeLink = "relative link"
	errLocalLink    = "local link"
	errSameSite     = "same site"
)

func newLinkValidatorPlugin(cfg map[string]any) *Plugin {
	return &Plugin{
		Name: "link_validator",
		Hooks: PluginHooks{
			BuildDone: func(ctx *BuildDoneContext) error {
				return linkValidatorBuildDone(ctx)
			},
		},
	}
}

func linkValidatorBuildDone(ctx *BuildDoneContext) error {
	settings := ctx.Config.LinkValidation
	if !config.BoolVal(settings.Enabled, true) {
		return nil
	}
	if ctx.PageIndex == nil || len(ctx.ValidationData) == 0 {
		return nil
	}

	checkAnchors := config.BoolVal(settings.CheckAnchors, true)
	checkImages := config.BoolVal(settings.CheckImages, true)
	warnRelative := config.BoolVal(settings.WarnRelativeLinks, true)
	warnLocal := config.BoolVal(settings.WarnLocalLinks, true)
	sameSitePolicy := settings.SameSitePolicy
	if sameSitePolicy == "" {
		sameSitePolicy = "ignore"
	}

	excludePatterns := settings.Exclude
	if len(excludePatterns) == 0 {
		excludePatterns = settings.Ignore
	}

	var siteURL string
	if ctx.Site != nil {
		siteURL = strings.TrimRight(ctx.Site.BaseURL, "/")
	}

	var errorCount int
	for permalink, entry := range ctx.ValidationData {
		for _, link := range entry.Links {
			if link.IsImage && !checkImages {
				continue
			}
			if validateLink(ctx, ctx.PageIndex, permalink, entry.FilePath, link.Href,
				checkAnchors, warnRelative, warnLocal, sameSitePolicy,
				siteURL, excludePatterns) {
				errorCount++
			}
		}
	}

	if errorCount > 0 {
		ctx.Log(fmt.Sprintf("Found %d broken link(s)", errorCount))
	} else {
		linkCount := 0
		for _, entry := range ctx.ValidationData {
			linkCount += len(entry.Links)
		}
		ctx.Log(fmt.Sprintf("Validated %d links", linkCount))
	}

	if config.BoolVal(settings.FailBuild, false) && errorCount > 0 {
		return fmt.Errorf("link validation failed: %d broken link(s) found", errorCount)
	}

	return nil
}

// validateLink checks a single link and adds a warning if invalid.
// Returns true if a warning was emitted.
func validateLink(ctx *BuildDoneContext, idx *content.PageIndex,
	pagePermalink, filePath, href string,
	checkAnchors, warnRelative, warnLocal bool,
	sameSitePolicy, siteURL string, excludePatterns []string) bool {

	if href == "" {
		return false
	}

	if isSpecialScheme(href) {
		return false
	}

	if shouldExclude(href, excludePatterns) {
		return false
	}

	if warnLocal && isLocalLink(href) {
		addLinkWarning(ctx, filePath, href, errLocalLink)
		return true
	}

	if sameSitePolicy != "ignore" && siteURL != "" && isSameSite(href, siteURL) {
		if sameSitePolicy == "warn" {
			addLinkWarning(ctx, filePath, href, errSameSite)
			return true
		}
		href = stripOrigin(href, siteURL)
	}

	if isExternalURL(href) {
		return false
	}

	if isRelativeLink(href) {
		if warnRelative {
			addLinkWarning(ctx, filePath, href, errRelativeLink)
			return true
		}
		return false
	}

	path, anchor := splitAnchor(href)

	if path == "" && anchor != "" {
		if checkAnchors && !idx.HasHeading(pagePermalink, anchor) {
			addLinkWarning(ctx, filePath, href, errInvalidHash)
			return true
		}
		return false
	}

	normalizedPath := content.NormalizePermalink(path)

	if idx.HasPage(normalizedPath) {
		if anchor != "" && checkAnchors {
			if !idx.HasHeading(normalizedPath, anchor) {
				addLinkWarning(ctx, filePath, href, errInvalidHash)
				return true
			}
		}
		return false
	}

	if idx.HasAsset(path) {
		return false
	}

	addLinkWarning(ctx, filePath, href, errInvalidLink)
	return true
}

func addLinkWarning(ctx *BuildDoneContext, filePath, href, errType string) {
	ctx.AddWarning(engine.ValidationWarning{
		File:    filePath,
		Field:   "link",
		Message: fmt.Sprintf("%s: %s", errType, href),
	})
}

func isSpecialScheme(href string) bool {
	lower := strings.ToLower(href)
	return strings.HasPrefix(lower, "mailto:") ||
		strings.HasPrefix(lower, "tel:") ||
		strings.HasPrefix(lower, "data:") ||
		strings.HasPrefix(lower, "javascript:") ||
		strings.HasPrefix(lower, "vbscript:")
}

func isExternalURL(href string) bool {
	return strings.HasPrefix(href, "http://") ||
		strings.HasPrefix(href, "https://") ||
		strings.HasPrefix(href, "//")
}

func isLocalLink(href string) bool {
	lower := strings.ToLower(href)
	return strings.HasPrefix(lower, "http://localhost") ||
		strings.HasPrefix(lower, "https://localhost") ||
		strings.HasPrefix(lower, "http://127.0.0.1") ||
		strings.HasPrefix(lower, "https://127.0.0.1")
}

func isSameSite(href, siteURL string) bool {
	if siteURL == "" {
		return false
	}
	parsed, err := url.Parse(href)
	if err != nil {
		return false
	}
	siteParsed, err := url.Parse(siteURL)
	if err != nil {
		return false
	}
	return parsed.Scheme == siteParsed.Scheme && parsed.Host == siteParsed.Host
}

func stripOrigin(href, siteURL string) string {
	siteURL = strings.TrimRight(siteURL, "/")
	if strings.HasPrefix(href, siteURL) {
		remainder := href[len(siteURL):]
		if remainder == "" {
			return "/"
		}
		return remainder
	}
	return href
}

func isRelativeLink(href string) bool {
	return strings.HasPrefix(href, "./") || strings.HasPrefix(href, "../")
}

func splitAnchor(href string) (path, anchor string) {
	if idx := strings.Index(href, "#"); idx >= 0 {
		path = href[:idx]
		anchor = href[idx+1:]
	} else {
		path = href
	}
	if idx := strings.Index(path, "?"); idx >= 0 {
		path = path[:idx]
	}
	return
}
