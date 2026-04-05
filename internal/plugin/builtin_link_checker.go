package plugin

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/coderoo-dev/coderoo/internal/engine"
)

func newLinkCheckerPlugin(cfg map[string]any) *Plugin {
	return &Plugin{
		Name: "link_checker",
		Hooks: PluginHooks{
			BuildDone: func(ctx *BuildDoneContext) error {
				return linkCheckerBuildDone(ctx, cfg)
			},
		},
	}
}

var hrefRegex = regexp.MustCompile(`(?i)<a[^>]+href="([^"]*)"`)
var srcRegex = regexp.MustCompile(`(?i)<img[^>]+src="([^"]*)"`)

func linkCheckerBuildDone(ctx *BuildDoneContext, cfg map[string]any) error {
	checkImages := cfgBool(cfg, "check_images", true)
	ignorePatterns := cfgStringSlice(cfg, "ignore")

	// Walk all HTML files in the output directory.
	return filepath.Walk(ctx.OutputDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() || !strings.HasSuffix(path, ".html") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		rel, _ := filepath.Rel(ctx.OutputDir, path)
		html := string(data)

		// Check <a href="..."> links.
		for _, match := range hrefRegex.FindAllStringSubmatch(html, -1) {
			href := match[1]
			checkLink(ctx, href, rel, ignorePatterns)
		}

		// Check <img src="..."> if enabled.
		if checkImages {
			for _, match := range srcRegex.FindAllStringSubmatch(html, -1) {
				src := match[1]
				checkLink(ctx, src, rel, ignorePatterns)
			}
		}

		return nil
	})
}

func checkLink(ctx *BuildDoneContext, href, sourceFile string, ignorePatterns []string) {
	// Skip external links, anchors, data URIs, mailto, etc.
	if href == "" || strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") ||
		strings.HasPrefix(href, "//") || strings.HasPrefix(href, "#") ||
		strings.HasPrefix(href, "data:") || strings.HasPrefix(href, "mailto:") ||
		strings.HasPrefix(href, "javascript:") || strings.HasPrefix(href, "tel:") {
		return
	}

	if shouldExclude(href, ignorePatterns) {
		return
	}

	// Strip anchor from path.
	path := href
	if idx := strings.Index(path, "#"); idx >= 0 {
		path = path[:idx]
	}
	if path == "" {
		return
	}

	// Resolve the target file in the output directory.
	target := filepath.Join(ctx.OutputDir, filepath.FromSlash(path))

	// Check if the target exists (as file or as directory with index.html).
	if _, err := os.Stat(target); err == nil {
		return
	}
	if strings.HasSuffix(path, "/") {
		candidate := filepath.Join(target, "index.html")
		if _, err := os.Stat(candidate); err == nil {
			return
		}
	}
	// Try appending /index.html for clean URLs.
	candidate := filepath.Join(target, "index.html")
	if _, err := os.Stat(candidate); err == nil {
		return
	}

	// Broken link.
	ctx.AddWarning(engine.ValidationWarning{
		File:    sourceFile,
		Field:   "link",
		Message: "broken link: " + href,
	})
}
