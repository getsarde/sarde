package plugin

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/content/markdown/extensions/linkrender"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/links"
)

const (
	errInvalidLink  = "invalid link"
	errAmbiguous    = "ambiguous link (" + links.AmbiguousHint + ")"
	errInvalidHash  = "invalid hash"
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
	sameSitePolicy := settings.SameSitePolicy
	if sameSitePolicy == "" {
		sameSitePolicy = "ignore"
	}

	excludePatterns := settings.Exclude
	if len(excludePatterns) == 0 {
		excludePatterns = settings.Ignore
	}

	warnLocal := settings.EffectiveOnLocalLinks() != "ignore"

	siteURL := ctx.BaseURL()
	escapePrefix := settings.SiteRootEscapePrefix

	changedSet := make(map[string]struct{}, len(ctx.ChangedPages))
	for _, p := range ctx.ChangedPages {
		changedSet[p.Permalink] = struct{}{}
	}

	resolveCtx := linkrender.ResolveContext{
		PageIndex:   ctx.PageIndex,
		URLResolver: ctx.Resolver.URL,
		Collections: ctx.Collections,
	}

	var errorCount, linkCount int
	for permalink, entry := range ctx.ValidationData {
		if _, changed := changedSet[permalink]; !changed {
			continue
		}
		page := ctx.PageIndex.LookupByPermalink(permalink)
		if page == nil {
			continue
		}
		for _, link := range entry.Links {
			linkCount++
			if link.IsImage && !checkImages {
				continue
			}
			if validateLink(ctx, page, resolveCtx, link.Href,
				checkAnchors, warnLocal, sameSitePolicy,
				siteURL, escapePrefix, excludePatterns) {
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

func validateLink(ctx *BuildDoneContext, page *engine.Page,
	resolveCtx linkrender.ResolveContext, href string,
	checkAnchors, warnLocal bool,
	sameSitePolicy, siteURL, escapePrefix string,
	excludePatterns []string) bool {

	if href == "" {
		return false
	}

	if isSpecialScheme(href) {
		return false
	}

	if ShouldExcludePath(href, excludePatterns) {
		return false
	}

	if warnLocal && isLocalLink(href) {
		addLinkWarning(ctx, page.FilePath, href, errLocalLink)
		return true
	}

	if sameSitePolicy != "ignore" && siteURL != "" && isSameSite(href, siteURL) {
		if sameSitePolicy == "warn" || sameSitePolicy == "error" {
			addLinkWarning(ctx, page.FilePath, href, errSameSite)
			return true
		}
		href = stripOrigin(href, siteURL)
	}

	result := linkrender.CheckHref(href, page, resolveCtx,
		ctx.PageIndex, ctx.Resolver, escapePrefix)

	switch result.Status {
	case links.StatusOK:
		if result.HasFragment && checkAnchors {
			fragment := extractFragment(href)
			if fragment != "" && !ctx.PageIndex.HasHeading(result.TargetPermalink, fragment) {
				addLinkWarning(ctx, page.FilePath, href, errInvalidHash)
				return true
			}
		}
		return false

	case links.StatusExternal:
		return false

	case links.StatusAmbiguous:
		addLinkWarning(ctx, page.FilePath, href, errAmbiguous)
		return true

	default:
		addLinkWarning(ctx, page.FilePath, href, errInvalidLink)
		return true
	}
}

func extractFragment(href string) string {
	if i := strings.IndexByte(href, '#'); i >= 0 {
		frag := href[i+1:]
		if i := strings.IndexByte(frag, '?'); i >= 0 {
			frag = frag[:i]
		}
		return frag
	}
	return ""
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
