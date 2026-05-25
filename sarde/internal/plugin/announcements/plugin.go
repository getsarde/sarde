// Package announcements provides a built-in plugin that renders dismissible
// announcement banners at the bottom of every page. Supports multiple
// announcements with i18n, three display modes (stack/first/rotate),
// date scheduling, and page targeting.
package announcements

import (
	"crypto/sha256"
	"embed"
	"fmt"
	"html"
	"html/template"
	"io/fs"
	"strconv"
	"strings"

	"github.com/frostybee/sarde/internal/i18n"
	"github.com/frostybee/sarde/internal/plugin"
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

type pluginConfig struct {
	displayMode         string // "stack" | "first" | "rotate"
	rotateInterval      int    // ms, min 500, default 5000
	showRotateIndicator bool
}

type announcementItem struct {
	id          string
	bannerType  string
	messageKey  string
	dismissible bool
	active      bool
	startDate   string
	endDate     string
	showOn      []string
	hideOn      []string
}

// New constructs the announcements plugin from its config map, StringTable, and
// a pointer to the current language (captured by closure, resolved at render time).
// st and langPtr may be nil — the plugin degrades to raw message keys.
func New(cfg map[string]any, st *i18n.StringTable, langPtr *string) *plugin.Plugin {
	pcfg := cfgPluginConfig(cfg)
	items := cfgItems(cfg)
	active := hasActiveItems(items)

	renderBanner := func(lang string) template.HTML {
		if !active {
			return ""
		}

		var banners strings.Builder
		for _, item := range items {
			if !item.active {
				continue
			}

			msg := resolveString(st, lang, item.messageKey)
			if msg == "" {
				continue
			}

			validTypes := map[string]bool{"info": true, "warning": true, "success": true, "danger": true}
			bt := item.bannerType
			if !validTypes[bt] {
				bt = "info"
			}

			banners.WriteString(fmt.Sprintf(
				`<div class="sarde-announcement-banner sarde-announcement-%s" data-announcement-id="%s"`,
				bt, html.EscapeString(item.id),
			))
			if item.startDate != "" {
				banners.WriteString(fmt.Sprintf(` data-start-date="%s"`, html.EscapeString(item.startDate)))
			}
			if item.endDate != "" {
				banners.WriteString(fmt.Sprintf(` data-end-date="%s"`, html.EscapeString(item.endDate)))
			}
			if len(item.showOn) > 0 {
				banners.WriteString(fmt.Sprintf(` data-show-on="%s"`, html.EscapeString(strings.Join(item.showOn, ","))))
			}
			if len(item.hideOn) > 0 {
				banners.WriteString(fmt.Sprintf(` data-hide-on="%s"`, html.EscapeString(strings.Join(item.hideOn, ","))))
			}
			banners.WriteString(`>`)

			banners.WriteString(fmt.Sprintf(
				`<span class="sarde-announcement-content">%s</span>`,
				html.EscapeString(msg),
			))
			if item.dismissible {
				dismissLabel := resolveString(st, lang, "announcements.dismiss")
				banners.WriteString(fmt.Sprintf(
					`<button class="sarde-announcement-dismiss" data-announcement-id="%s" aria-label="%s">&times;</button>`,
					html.EscapeString(item.id), html.EscapeString(dismissLabel),
				))
			}
			banners.WriteString(`</div>`)
		}

		if banners.Len() == 0 {
			return ""
		}

		var container strings.Builder
		container.WriteString(`<div class="sarde-announcement-container"`)
		container.WriteString(fmt.Sprintf(` data-display-mode="%s"`, html.EscapeString(pcfg.displayMode)))
		if pcfg.displayMode == "rotate" {
			container.WriteString(fmt.Sprintf(` data-rotate-interval="%d"`, pcfg.rotateInterval))
			container.WriteString(fmt.Sprintf(` data-show-rotate-indicator="%s"`, strconv.FormatBool(pcfg.showRotateIndicator)))
		}
		container.WriteString(`>`)
		container.WriteString(banners.String())
		container.WriteString(`</div>`)

		return template.HTML(container.String())
	}

	return &plugin.Plugin{
		Name: "announcements",
		Hooks: plugin.PluginHooks{
			ConfigSetup: func(ctx *plugin.ConfigSetupContext) error {
				ctx.AddTemplateFunc(plugin.LangAwareAnnouncementFunc, func(lang string) template.HTML {
					return renderBanner(lang)
				})
				ctx.AddTemplateFunc("announcementBanner", func() template.HTML {
					lang := ""
					if langPtr != nil {
						lang = *langPtr
					}
					return renderBanner(lang)
				})
				return nil
			},
			BeforeRender: func(ctx *plugin.BeforeRenderContext) error {
				if !active {
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

func resolveString(st *i18n.StringTable, lang, key string) string {
	if st == nil || lang == "" {
		return key
	}
	return st.Resolve(lang, key)
}

func hasActiveItems(items []announcementItem) bool {
	for _, item := range items {
		if item.active {
			return true
		}
	}
	return false
}

func cfgPluginConfig(cfg map[string]any) pluginConfig {
	pc := pluginConfig{
		displayMode:         "stack",
		rotateInterval:      5000,
		showRotateIndicator: true,
	}
	if cfg == nil {
		return pc
	}

	if mode := cfgString(cfg, "display_mode", ""); mode != "" {
		validModes := map[string]bool{"stack": true, "first": true, "rotate": true}
		if validModes[mode] {
			pc.displayMode = mode
		}
	}

	if v, ok := cfg["rotate_interval"]; ok {
		switch val := v.(type) {
		case int:
			pc.rotateInterval = val
		case float64:
			pc.rotateInterval = int(val)
		}
	}
	if pc.rotateInterval < 500 {
		pc.rotateInterval = 500
	}

	if v, ok := cfg["show_rotate_indicator"]; ok {
		if b, ok := v.(bool); ok {
			pc.showRotateIndicator = b
		}
	}

	return pc
}

func cfgItems(cfg map[string]any) []announcementItem {
	if cfg == nil {
		return nil
	}
	rawItems, ok := cfg["items"]
	if !ok {
		return nil
	}
	list, ok := rawItems.([]any)
	if !ok {
		return nil
	}

	items := make([]announcementItem, 0, len(list))
	for _, entry := range list {
		m, ok := entry.(map[string]any)
		if !ok {
			continue
		}
		items = append(items, announcementItem{
			id:          cfgString(m, "id", "default"),
			bannerType:  cfgString(m, "type", "info"),
			messageKey:  cfgString(m, "message", ""),
			dismissible: cfgBool(m, "dismissible", true),
			active:      cfgBool(m, "active", true),
			startDate:   cfgString(m, "start_date", ""),
			endDate:     cfgString(m, "end_date", ""),
			showOn:      cfgStringSlice(m, "show_on"),
			hideOn:      cfgStringSlice(m, "hide_on"),
		})
	}
	return items
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

func cfgStringSlice(cfg map[string]any, key string) []string {
	if cfg == nil {
		return nil
	}
	v, ok := cfg[key]
	if !ok {
		return nil
	}
	list, ok := v.([]any)
	if !ok {
		return nil
	}
	result := make([]string, 0, len(list))
	for _, item := range list {
		if s, ok := item.(string); ok {
			result = append(result, s)
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
