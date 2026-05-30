package plugin

import (
	"encoding/json"
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
			redirects[alias] = page.URL()
		}
	}

	if len(redirects) == 0 {
		ctx.Log("No redirects to generate")
		return nil
	}

	// Sort keys for deterministic output.
	sources := make([]string, 0, len(redirects))
	for from := range redirects {
		sources = append(sources, from)
	}
	sort.Strings(sources)

	// Determine which formats to emit.
	format := ctx.Config.Deploy.RedirectFormat
	netlifyEnabled := format == "" || format == "all" || format == "netlify"
	vercelEnabled := format == "" || format == "all" || format == "vercel"

	// Write HTML meta-refresh files (universal fallback) and build _redirects content.
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
		if netlifyEnabled {
			fmt.Fprintf(&redirectsFile, "%s  %s  301\n", from, to)
		}
	}

	// Write _redirects file.
	if netlifyEnabled {
		if err := ctx.WriteFile("_redirects", []byte(redirectsFile.String())); err != nil {
			return err
		}
	}

	// Write vercel.json.
	if vercelEnabled {
		if err := writeVercelRedirects(ctx, redirects, sources); err != nil {
			return err
		}
	}

	ctx.Log(fmt.Sprintf("Generated %d redirect(s)", len(redirects)))
	return nil
}

type vercelRedirect struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
	Permanent   bool   `json:"permanent"`
}

type vercelConfig struct {
	Redirects []vercelRedirect `json:"redirects"`
}

func writeVercelRedirects(ctx *BuildDoneContext, redirects map[string]string, sources []string) error {
	cfg := vercelConfig{
		Redirects: make([]vercelRedirect, 0, len(sources)),
	}
	for _, from := range sources {
		cfg.Redirects = append(cfg.Redirects, vercelRedirect{
			Source:      from,
			Destination: redirects[from],
			Permanent:   true,
		})
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling vercel.json: %w", err)
	}
	return ctx.WriteFile("vercel.json", data)
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
