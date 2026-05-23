package announcements_test

import (
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coderoo-dev/coderoo/internal/config"
	"github.com/coderoo-dev/coderoo/internal/engine"
	"github.com/coderoo-dev/coderoo/internal/i18n"
	"github.com/coderoo-dev/coderoo/internal/plugin"
	"github.com/coderoo-dev/coderoo/internal/plugin/announcements"
)

func testStringTable(t *testing.T) *i18n.StringTable {
	t.Helper()
	dir := t.TempDir()
	i18nDir := filepath.Join(dir, "i18n")
	os.MkdirAll(i18nDir, 0o755)
	os.WriteFile(filepath.Join(i18nDir, "en.yaml"), []byte(`
announcements:
  dismiss: "Dismiss announcement"
  maint: "Site under maintenance"
  new_feature: "New feature available"
`), 0o644)
	os.WriteFile(filepath.Join(i18nDir, "fr.yaml"), []byte(`
announcements:
  dismiss: "Fermer l'annonce"
  maint: "Site en maintenance"
  new_feature: "Nouvelle fonctionnalité disponible"
`), 0o644)
	st, err := i18n.LoadStrings(nil, dir, "", "en")
	if err != nil {
		t.Fatalf("LoadStrings: %v", err)
	}
	return st
}

func getBannerFunc(t *testing.T, p *plugin.Plugin) func() template.HTML {
	t.Helper()
	mgr := plugin.NewManager()
	mgr.Register(p)
	cfg := config.Defaults()
	cfg.Plugins.Enabled = []string{"announcements"}
	if err := mgr.RunConfigSetup(cfg); err != nil {
		t.Fatalf("RunConfigSetup: %v", err)
	}
	funcs := mgr.TemplateFuncs()
	fn, ok := funcs["announcementBanner"]
	if !ok {
		t.Fatal("announcementBanner not registered")
	}
	return fn.(func() template.HTML)
}

func runBeforeRender(t *testing.T, p *plugin.Plugin) *engine.RouteData {
	t.Helper()
	mgr := plugin.NewManager()
	mgr.Register(p)
	cfg := config.Defaults()
	cfg.Plugins.Enabled = []string{"announcements"}
	rd := &engine.RouteData{}
	if err := mgr.RunBeforeRender(cfg, &engine.Page{}, rd, &engine.SiteContext{}); err != nil {
		t.Fatalf("RunBeforeRender: %v", err)
	}
	return rd
}

func TestNew_NilConfig(t *testing.T) {
	p := announcements.New(nil, nil, nil)
	bannerFunc := getBannerFunc(t, p)
	result := bannerFunc()
	if result != "" {
		t.Errorf("expected empty HTML for nil config, got %q", result)
	}
}

func TestNew_MultipleActiveItems(t *testing.T) {
	st := testStringTable(t)
	lang := "en"
	cfg := map[string]any{
		"items": []any{
			map[string]any{"id": "maint", "type": "warning", "message": "announcements.maint", "active": true, "dismissible": true},
			map[string]any{"id": "feature", "type": "info", "message": "announcements.new_feature", "active": true, "dismissible": false},
		},
	}

	p := announcements.New(cfg, st, &lang)
	bannerFunc := getBannerFunc(t, p)
	result := string(bannerFunc())

	for _, want := range []string{
		"announcement-container",
		`data-announcement-id="maint"`,
		`data-announcement-id="feature"`,
		"Site under maintenance",
		"New feature available",
		"announcement-warning",
		"announcement-info",
	} {
		if !strings.Contains(result, want) {
			t.Errorf("expected %q in output", want)
		}
	}
}

func TestNew_InactiveItemSkipped(t *testing.T) {
	st := testStringTable(t)
	lang := "en"
	cfg := map[string]any{
		"items": []any{
			map[string]any{"id": "active-one", "message": "announcements.maint", "active": true},
			map[string]any{"id": "inactive-one", "message": "announcements.new_feature", "active": false},
		},
	}

	p := announcements.New(cfg, st, &lang)
	bannerFunc := getBannerFunc(t, p)
	result := string(bannerFunc())

	if !strings.Contains(result, `data-announcement-id="active-one"`) {
		t.Error("expected active item in output")
	}
	if strings.Contains(result, `data-announcement-id="inactive-one"`) {
		t.Error("inactive item should not appear in output")
	}
}

func TestNew_EmptyItems(t *testing.T) {
	st := testStringTable(t)
	lang := "en"
	cfg := map[string]any{
		"items": []any{},
	}

	p := announcements.New(cfg, st, &lang)
	bannerFunc := getBannerFunc(t, p)
	result := bannerFunc()

	if result != "" {
		t.Errorf("expected empty HTML for empty items, got %q", result)
	}
}

func TestNew_I18nResolvesAtCallTime(t *testing.T) {
	st := testStringTable(t)
	lang := "en"
	cfg := map[string]any{
		"items": []any{
			map[string]any{"id": "test", "message": "announcements.maint", "active": true},
		},
	}

	p := announcements.New(cfg, st, &lang)
	bannerFunc := getBannerFunc(t, p)

	result1 := string(bannerFunc())
	if !strings.Contains(result1, "Site under maintenance") {
		t.Error("expected English message")
	}

	lang = "fr"
	result2 := string(bannerFunc())
	if !strings.Contains(result2, "Site en maintenance") {
		t.Error("expected French message after lang change")
	}
}

func TestNew_NilStringTable(t *testing.T) {
	lang := "en"
	cfg := map[string]any{
		"items": []any{
			map[string]any{"id": "test", "message": "raw.key.fallback", "active": true, "dismissible": true},
		},
	}

	p := announcements.New(cfg, nil, &lang)
	bannerFunc := getBannerFunc(t, p)
	result := string(bannerFunc())

	if !strings.Contains(result, "raw.key.fallback") {
		t.Error("expected raw key as fallback when StringTable is nil")
	}
}

func TestNew_DismissAriaLabel(t *testing.T) {
	st := testStringTable(t)
	lang := "en"
	cfg := map[string]any{
		"items": []any{
			map[string]any{"id": "test", "message": "announcements.maint", "active": true, "dismissible": true},
		},
	}

	p := announcements.New(cfg, st, &lang)
	bannerFunc := getBannerFunc(t, p)
	result := string(bannerFunc())

	if !strings.Contains(result, `aria-label="Dismiss announcement"`) {
		t.Error("expected i18n-resolved dismiss aria-label")
	}

	lang = "fr"
	resultFr := string(bannerFunc())
	if !strings.Contains(resultFr, `aria-label="Fermer l&#39;annonce"`) {
		t.Errorf("expected French dismiss label, got relevant portion: %s",
			extractAriaLabel(resultFr))
	}
}

func extractAriaLabel(html string) string {
	idx := strings.Index(html, "aria-label=")
	if idx < 0 {
		return "<not found>"
	}
	end := idx + 60
	if end > len(html) {
		end = len(html)
	}
	return html[idx:end]
}

func TestBeforeRender_InjectsWhenActive(t *testing.T) {
	cfg := map[string]any{
		"items": []any{
			map[string]any{"id": "test", "message": "msg", "active": true},
		},
	}

	p := announcements.New(cfg, nil, nil)
	rd := runBeforeRender(t, p)

	if len(rd.Styles) == 0 {
		t.Error("expected CSS URL injected")
	}
	if len(rd.Scripts) == 0 {
		t.Error("expected JS URL injected")
	}
}

func TestBeforeRender_NoInjectAllInactive(t *testing.T) {
	cfg := map[string]any{
		"items": []any{
			map[string]any{"id": "test", "message": "msg", "active": false},
		},
	}

	p := announcements.New(cfg, nil, nil)
	rd := runBeforeRender(t, p)

	if len(rd.Styles) > 0 {
		t.Error("should not inject CSS when all inactive")
	}
	if len(rd.Scripts) > 0 {
		t.Error("should not inject JS when all inactive")
	}
}

// ── Display mode tests ──

func TestNew_DisplayModeStack(t *testing.T) {
	cfg := map[string]any{
		"items": []any{
			map[string]any{"id": "test", "message": "msg", "active": true},
		},
	}

	p := announcements.New(cfg, nil, nil)
	result := string(getBannerFunc(t, p)())

	if !strings.Contains(result, `data-display-mode="stack"`) {
		t.Error("expected data-display-mode=stack on container")
	}
	if strings.Contains(result, "data-rotate-interval") {
		t.Error("stack mode should not have rotate-interval attribute")
	}
}

func TestNew_DisplayModeRotate(t *testing.T) {
	cfg := map[string]any{
		"display_mode":    "rotate",
		"rotate_interval": 3000,
		"items": []any{
			map[string]any{"id": "test", "message": "msg", "active": true},
		},
	}

	p := announcements.New(cfg, nil, nil)
	result := string(getBannerFunc(t, p)())

	if !strings.Contains(result, `data-display-mode="rotate"`) {
		t.Error("expected data-display-mode=rotate")
	}
	if !strings.Contains(result, `data-rotate-interval="3000"`) {
		t.Error("expected data-rotate-interval=3000")
	}
	if !strings.Contains(result, `data-show-rotate-indicator="true"`) {
		t.Error("expected data-show-rotate-indicator=true")
	}
}

func TestNew_DisplayModeFirst(t *testing.T) {
	cfg := map[string]any{
		"display_mode": "first",
		"items": []any{
			map[string]any{"id": "test", "message": "msg", "active": true},
		},
	}

	p := announcements.New(cfg, nil, nil)
	result := string(getBannerFunc(t, p)())

	if !strings.Contains(result, `data-display-mode="first"`) {
		t.Error("expected data-display-mode=first")
	}
}

func TestNew_RotateIntervalClamped(t *testing.T) {
	cfg := map[string]any{
		"display_mode":    "rotate",
		"rotate_interval": 100,
		"items": []any{
			map[string]any{"id": "test", "message": "msg", "active": true},
		},
	}

	p := announcements.New(cfg, nil, nil)
	result := string(getBannerFunc(t, p)())

	if !strings.Contains(result, `data-rotate-interval="500"`) {
		t.Errorf("expected rotate interval clamped to 500, got: %s", result)
	}
}

func TestNew_ItemDateAttributes(t *testing.T) {
	cfg := map[string]any{
		"items": []any{
			map[string]any{
				"id":         "dated",
				"message":    "msg",
				"active":     true,
				"start_date": "2025-01-01T00:00:00Z",
				"end_date":   "2027-12-31T23:59:59Z",
			},
		},
	}

	p := announcements.New(cfg, nil, nil)
	result := string(getBannerFunc(t, p)())

	if !strings.Contains(result, `data-start-date="2025-01-01T00:00:00Z"`) {
		t.Error("expected data-start-date attribute")
	}
	if !strings.Contains(result, `data-end-date="2027-12-31T23:59:59Z"`) {
		t.Error("expected data-end-date attribute")
	}
}

func TestNew_ItemPageTargeting(t *testing.T) {
	cfg := map[string]any{
		"items": []any{
			map[string]any{
				"id":      "targeted",
				"message": "msg",
				"active":  true,
				"show_on": []any{"/docs/**", "/blog/**"},
				"hide_on": []any{"/admin/**"},
			},
		},
	}

	p := announcements.New(cfg, nil, nil)
	result := string(getBannerFunc(t, p)())

	if !strings.Contains(result, `data-show-on="/docs/**,/blog/**"`) {
		t.Error("expected data-show-on with comma-joined patterns")
	}
	if !strings.Contains(result, `data-hide-on="/admin/**"`) {
		t.Error("expected data-hide-on attribute")
	}
}

func TestNew_DefaultAttributesOmitted(t *testing.T) {
	cfg := map[string]any{
		"items": []any{
			map[string]any{"id": "minimal", "message": "msg", "active": true},
		},
	}

	p := announcements.New(cfg, nil, nil)
	result := string(getBannerFunc(t, p)())

	if strings.Contains(result, "data-show-on") {
		t.Error("empty show_on should not emit data-show-on")
	}
	if strings.Contains(result, "data-hide-on") {
		t.Error("empty hide_on should not emit data-hide-on")
	}
	if strings.Contains(result, "data-start-date") {
		t.Error("empty start_date should not emit data-start-date")
	}
	if strings.Contains(result, "data-end-date") {
		t.Error("empty end_date should not emit data-end-date")
	}
}
