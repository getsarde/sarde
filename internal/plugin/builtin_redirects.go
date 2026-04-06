package plugin

import (
	"fmt"
	"path"
	"sort"
	"strings"
)

func newRedirectsPlugin(cfg map[string]any) *Plugin {
	return &Plugin{
		Name: "redirects",
		Hooks: PluginHooks{
			BuildDone: func(ctx *BuildDoneContext) error {
				return redirectsBuildDone(ctx)
			},
		},
	}
}

func redirectsBuildDone(ctx *BuildDoneContext) error {
	// Collect all redirects: source → target.
	redirects := make(map[string]string)

	// 1. Global redirects from config.
	for from, to := range ctx.Config.Redirects {
		redirects[from] = to
	}

	// 2. Page aliases (override global if conflict).
	for _, page := range ctx.Pages {
		for _, alias := range page.Aliases {
			redirects[alias] = page.RelPermalink
		}
	}

	if len(redirects) == 0 {
		return nil
	}

	// Sort keys for deterministic output.
	sources := make([]string, 0, len(redirects))
	for from := range redirects {
		sources = append(sources, from)
	}
	sort.Strings(sources)

	// Write HTML redirect files and build _redirects content.
	var redirectsFile strings.Builder
	for _, from := range sources {
		to := redirects[from]

		// Write HTML meta-refresh file.
		htmlPath := from
		if strings.HasSuffix(htmlPath, "/") {
			htmlPath += "index.html"
		} else if !strings.HasSuffix(htmlPath, ".html") {
			htmlPath += "/index.html"
		}
		htmlPath = strings.TrimPrefix(htmlPath, "/")
		if err := ctx.WriteFile(htmlPath, redirectHTML(to)); err != nil {
			return fmt.Errorf("writing redirect %s: %w", from, err)
		}

		// Append to _redirects file (Netlify/Cloudflare format).
		fmt.Fprintf(&redirectsFile, "%s  %s  301\n", from, to)
	}

	return ctx.WriteFile("_redirects", []byte(redirectsFile.String()))
}

// redirectHTML generates a minimal HTML redirect page.
func redirectHTML(target string) []byte {
	// Sanitize target for safe embedding in HTML attributes.
	safe := strings.ReplaceAll(target, `"`, "%22")
	safe = strings.ReplaceAll(safe, `<`, "%3C")
	safe = strings.ReplaceAll(safe, `>`, "%3E")
	title := "Redirecting to " + path.Base(strings.TrimSuffix(safe, "/"))

	return []byte(fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
<meta charset="utf-8">
<title>%s</title>
<meta http-equiv="refresh" content="0; url=%s">
<link rel="canonical" href="%s">
</head>
<body>
<p>Redirecting to <a href="%s">%s</a>...</p>
</body>
</html>
`, title, safe, safe, safe, safe))
}
