package project

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/consts"
)

var languageCodePattern = regexp.MustCompile(`^[a-z]{2,3}(-[a-z0-9]+)*$`)

// AddLanguage registers a new language in i18n.languages. The first time a
// second language is configured, default_language is set to the existing
// site language (or "en") if not already set.
func (pm *ProjectManager) AddLanguage(code string, cfg config.LanguageConfig) error {
	pm.mu.Lock()
	if pm.state == StateClosed {
		pm.mu.Unlock()
		return fmt.Errorf("no project open")
	}
	projectDir := pm.projectDir
	pm.mu.Unlock()

	code = strings.ToLower(strings.TrimSpace(code))
	if !languageCodePattern.MatchString(code) {
		return fmt.Errorf("invalid language code %q", code)
	}

	siteYAMLPath := filepath.Join(projectDir, consts.FileSiteConfig)
	raw, err := readRawYAML(siteYAMLPath)
	if err != nil {
		return err
	}

	i18nSection := ensureMapSection(raw, "i18n")
	languages := ensureMapSection(i18nSection, "languages")
	wasEmpty := len(languages) == 0

	entry := map[string]any{}
	if cfg.Name != "" {
		entry["name"] = cfg.Name
	}
	if cfg.Title != "" {
		entry["title"] = cfg.Title
	}
	if cfg.Weight != 0 {
		entry["weight"] = cfg.Weight
	}
	if cfg.Dir != "" {
		entry["dir"] = cfg.Dir
	}
	languages[code] = entry

	if wasEmpty {
		if dl, ok := i18nSection["default_language"].(string); !ok || dl == "" {
			defaultLang := "en"
			if siteSection, ok := raw["site"].(map[string]any); ok {
				if lang, ok := siteSection["language"].(string); ok && lang != "" {
					defaultLang = lang
				}
			}
			i18nSection["default_language"] = defaultLang
		}
	}

	if err := writeRawYAML(siteYAMLPath, raw); err != nil {
		return err
	}

	return pm.reresolveAndBroadcast(projectDir)
}

// RemoveLanguage removes a language from i18n.languages. Rejects removing
// the default language. Optionally deletes its content/<code>/ directory.
func (pm *ProjectManager) RemoveLanguage(code string, deleteContent bool) error {
	pm.mu.Lock()
	if pm.state == StateClosed {
		pm.mu.Unlock()
		return fmt.Errorf("no project open")
	}
	projectDir := pm.projectDir
	contentDir := pm.contentDir()
	pm.mu.Unlock()

	code = strings.ToLower(strings.TrimSpace(code))

	siteYAMLPath := filepath.Join(projectDir, consts.FileSiteConfig)
	raw, err := readRawYAML(siteYAMLPath)
	if err != nil {
		return err
	}

	i18nSection := ensureMapSection(raw, "i18n")
	if defaultLang, _ := i18nSection["default_language"].(string); defaultLang == code {
		return fmt.Errorf("cannot remove default language %q", code)
	}

	languages := ensureMapSection(i18nSection, "languages")
	if _, ok := languages[code]; !ok {
		return fmt.Errorf("language %q is not configured", code)
	}
	delete(languages, code)

	if err := writeRawYAML(siteYAMLPath, raw); err != nil {
		return err
	}

	if deleteContent {
		if err := os.RemoveAll(filepath.Join(contentDir, code)); err != nil {
			return fmt.Errorf("removing content directory: %w", err)
		}
	}

	return pm.reresolveAndBroadcast(projectDir)
}

// ---------------------------------------------------------------------------
// Raw sarde.yaml helpers — shared low-level map[string]any manipulation.
// ---------------------------------------------------------------------------

// readRawYAML reads and unmarshals a YAML file into a generic map, treating
// a missing/unparseable file as an empty document (zero-config philosophy).
func readRawYAML(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return make(map[string]any), nil
	}
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil || raw == nil {
		raw = make(map[string]any)
	}
	return raw, nil
}

// writeRawYAML marshals and writes a generic map back to a YAML file.
func writeRawYAML(path string, raw map[string]any) error {
	out, err := yaml.Marshal(raw)
	if err != nil {
		return fmt.Errorf("marshaling %s: %w", filepath.Base(path), err)
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", filepath.Base(path), err)
	}
	return nil
}

// ensureMapSection returns raw[key] as a map[string]any, creating and
// inserting one if absent or of an unexpected type.
func ensureMapSection(raw map[string]any, key string) map[string]any {
	section, ok := raw[key].(map[string]any)
	if !ok {
		section = make(map[string]any)
		raw[key] = section
	}
	return section
}

// reresolveAndBroadcast re-resolves the project config from disk, publishes
// it, and broadcasts config:changed. Caller must not hold pm.mu.
func (pm *ProjectManager) reresolveAndBroadcast(projectDir string) error {
	cfg, themeCfg, err := pm.resolveConfig(projectDir)
	if err != nil {
		return err
	}

	pm.mu.Lock()
	pm.setProjectConfig(projectDir, cfg, themeCfg)
	pm.mu.Unlock()

	pm.eventHub.Broadcast(Event{Type: "config:changed"})
	return nil
}
