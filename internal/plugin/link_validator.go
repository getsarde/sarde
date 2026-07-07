package plugin

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/content"
	"github.com/getsarde/sarde/internal/engine"
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
	// Full builds already run the authoritative internal/links report
	// (SiteBuilder.validateLinks); running here too re-validated the same
	// collected links against the same config and printed every warning
	// twice. This plugin is the lightweight checker for incremental rebuilds
	// only, where the LinkGraph path deliberately does not run.
	if !ctx.Incremental {
		return nil
	}

	settings := ctx.Config.LinkValidation
	if !config.BoolVal(settings.Enabled, true) {
		return nil
	}
	if ctx.PageIndex == nil || len(ctx.ValidationData) == 0 {
		return nil
	}

	checkAnchors := config.BoolVal(settings.CheckAnchors, true)
	checkImages := config.BoolVal(settings.CheckImages, true)
	warnRelative := settings.EffectiveOnRelativeLinks() != "ignore"
	warnLocal := settings.EffectiveOnLocalLinks() != "ignore"
	sameSitePolicy := settings.SameSitePolicy
	if sameSitePolicy == "" {
		sameSitePolicy = "ignore"
	}

	excludePatterns := settings.Exclude
	if len(excludePatterns) == 0 {
		excludePatterns = settings.Ignore
	}

	siteURL := ctx.BaseURL()

	// Only re-validate links from pages that actually changed this pass;
	// unchanged pages' links were validated by a prior build. This mirrors
	// content_lint's scoping: a link from an unchanged page targeting a
	// heading removed from a changed page won't be re-flagged until the
	// next full build or an edit to that file.
	changedSet := make(map[string]struct{}, len(ctx.ChangedPages))
	for _, p := range ctx.ChangedPages {
		changedSet[p.Permalink] = struct{}{}
	}

	var errorCount, linkCount int
	for permalink, entry := range ctx.ValidationData {
		if _, changed := changedSet[permalink]; !changed {
			continue
		}
		for _, link := range entry.Links {
			linkCount++
			if link.IsImage && !checkImages {
				continue
			}
			if validateLink(ctx, ctx.PageIndex, ctx.Resolver, permalink, entry.FilePath, entry.Lang, link.Href,
				checkAnchors, warnRelative, warnLocal, sameSitePolicy,
				siteURL, excludePatterns) {
				errorCount++
			}
		}
	}

	if errorCount > 0 {
		ctx.Log(fmt.Sprintf("Found %d broken link(s)", errorCount))
	} else {
		ctx.Log(fmt.Sprintf("Validated %d links", linkCount))
	}

	if config.BoolVal(settings.FailBuild, false) && errorCount > 0 {
		return fmt.Errorf("link validation failed: %d broken link(s) found", errorCount)
	}

	return nil
}

func validateLink(ctx *BuildDoneContext, idx *content.PageIndex,
	resolver *engine.URLResolver, pagePermalink, filePath, lang, href string,
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
		// "warn" and "error" both surface the finding; "error" additionally
		// counts toward the fail_build threshold via the caller's errorCount
		// (mirroring internal/links/report.go's handling on full builds).
		if sameSitePolicy == "warn" || sameSitePolicy == "error" {
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

	lookupPath := normalizedPath
	if resolver != nil {
		lookupPath = resolver.URL(normalizedPath, lang, "")
	}

	if idx.HasPage(lookupPath) {
		if anchor != "" && checkAnchors {
			if !idx.HasHeading(lookupPath, anchor) {
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
