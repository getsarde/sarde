// Package announcements provides a built-in plugin that renders a dismissible
// announcement banner at the bottom of every page. The banner HTML is
// server-rendered via a template function; JS handles localStorage-based
// dismissal tracking.
package announcements

import (
	"crypto/sha256"
	"embed"
	"fmt"
	"html"
	"html/template"
	"io/fs"
	"strings"

	"github.com/coderoo-dev/coderoo/internal/plugin"
)

//go:embed all:assets
var assetsFS embed.FS

const pluginPrefix = "assets/plugins/announcements/"

var (
	jsData  []byte
	cssData []byte
	jsURL   string
	cssURL  string
)

func init() {
	jsData, _ = fs.ReadFile(assetsFS, "assets/announcements.js")
	cssData, _ = fs.ReadFile(assetsFS, "assets/announcements.css")

	if len(jsData) > 0 {
		h := sha256.Sum256(jsData)
		jsURL = fmt.Sprintf("/assets/plugins/announcements/announcements.%x.js", h[:4])
	}
	if len(cssData) > 0 {
		h := sha256.Sum256(cssData)
		cssURL = fmt.Sprintf("/assets/plugins/announcements/announcements.%x.css", h[:4])
	}
}

// New constructs the announcements plugin from its config map.
func New(cfg map[string]any) *plugin.Plugin {
	message := cfgString(cfg, "message", "")
	bannerType := cfgString(cfg, "type", "info")
	dismissible := cfgBool(cfg, "dismissible", true)
	active := cfgBool(cfg, "active", true)
	id := cfgString(cfg, "id", "default")

	bannerHTML := buildBannerHTML(message, bannerType, dismissible, id, active)

	return &plugin.Plugin{
		Name: "announcements",
		Hooks: plugin.PluginHooks{
			ConfigSetup: func(ctx *plugin.ConfigSetupContext) error {
				ctx.AddTemplateFunc("announcementBanner", func() template.HTML {
					return bannerHTML
				})
				return nil
			},
			BeforeRender: func(ctx *plugin.BeforeRenderContext) error {
				if !active || message == "" {
					return nil
				}
				rd := ctx.RouteData
				if cssURL != "" {
					rd.Styles = append(rd.Styles, cssURL)
				}
				if jsURL != "" {
					rd.Scripts = append(rd.Scripts, jsURL)
				}
				return nil
			},
			BuildDone: func(ctx *plugin.BuildDoneContext) error {
				if jsData != nil {
					if err := ctx.WriteFile(pluginPrefix+"announcements."+hashHex(jsData)+".js", jsData); err != nil {
						return err
					}
				}
				if cssData != nil {
					if err := ctx.WriteFile(pluginPrefix+"announcements."+hashHex(cssData)+".css", cssData); err != nil {
						return err
					}
				}
				return nil
			},
		},
	}
}

func buildBannerHTML(message, bannerType string, dismissible bool, id string, active bool) template.HTML {
	if !active || message == "" {
		return ""
	}

	validTypes := map[string]bool{"info": true, "warning": true, "success": true, "danger": true}
	if !validTypes[bannerType] {
		bannerType = "info"
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf(
		`<div class="announcement-banner announcement-%s" data-announcement-id="%s">`,
		bannerType, html.EscapeString(id),
	))
	sb.WriteString(fmt.Sprintf(
		`<span class="announcement-content">%s</span>`,
		html.EscapeString(message),
	))
	if dismissible {
		sb.WriteString(fmt.Sprintf(
			`<button class="announcement-dismiss" data-announcement-id="%s" aria-label="Dismiss announcement">&times;</button>`,
			html.EscapeString(id),
		))
	}
	sb.WriteString(`</div>`)

	return template.HTML(sb.String())
}

func hashHex(data []byte) string {
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h[:4])
}

func cfgString(cfg map[string]any, key, fallback string) string {
	if cfg == nil {
		return fallback
	}
	v, ok := cfg[key]
	if !ok {
		return fallback
	}
	s, ok := v.(string)
	if !ok {
		return fallback
	}
	return s
}

func cfgBool(cfg map[string]any, key string, fallback bool) bool {
	if cfg == nil {
		return fallback
	}
	v, ok := cfg[key]
	if !ok {
		return fallback
	}
	b, ok := v.(bool)
	if !ok {
		return fallback
	}
	return b
}
