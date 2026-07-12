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
	"github.com/getsarde/sarde/internal/content"
)

var languageCodePattern = regexp.MustCompile(`^[a-z]{2,3}(-[a-z0-9]+)*$`)

// TranslationCoverage reports content coverage for one collection x language pair.
type TranslationCoverage struct {
	Total      int `json:"total"`
	Translated int `json:"translated"`
}

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

// ScaffoldLanguageContent creates content/<code>/<collection>/ directories
// with _index.md stubs mirroring the default language's collections, and
// seeds i18n/<code>.yaml from the default language's translation file.
func (pm *ProjectManager) ScaffoldLanguageContent(code string) error {
	pm.mu.Lock()
	if pm.state == StateClosed {
		pm.mu.Unlock()
		return fmt.Errorf("no project open")
	}
	projectDir := pm.projectDir
	contentDir := pm.contentDir()
	cfg := pm.config
	pm.mu.Unlock()

	code = strings.ToLower(strings.TrimSpace(code))
	if !languageCodePattern.MatchString(code) {
		return fmt.Errorf("invalid language code %q", code)
	}

	defaultLang := "en"
	if cfg != nil && cfg.I18n.DefaultLanguage != "" {
		defaultLang = cfg.I18n.DefaultLanguage
	}

	// Discover collection names from the default-language content tree, if
	// present; otherwise fall back to the content root (pre-i18n layout).
	scanDir := contentDir
	if fi, err := os.Stat(filepath.Join(contentDir, defaultLang)); err == nil && fi.IsDir() {
		scanDir = filepath.Join(contentDir, defaultLang)
	}

	entries, err := os.ReadDir(scanDir)
	if err != nil {
		return fmt.Errorf("reading content directory: %w", err)
	}

	targetRoot := filepath.Join(contentDir, code)
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") || strings.HasPrefix(e.Name(), "_") {
			continue
		}
		colDir := filepath.Join(targetRoot, e.Name())
		if err := os.MkdirAll(colDir, 0o755); err != nil {
			return fmt.Errorf("creating collection directory: %w", err)
		}
		indexPath := filepath.Join(colDir, "_index.md")
		if _, err := os.Stat(indexPath); os.IsNotExist(err) {
			title := content.FilenameToTitle(e.Name())
			stub := fmt.Sprintf("---\ntitle: %q\n---\n", title)
			if err := os.WriteFile(indexPath, []byte(stub), 0o644); err != nil {
				return fmt.Errorf("writing index stub: %w", err)
			}
		}
	}

	// Seed i18n/<code>.yaml from the default language's file, or create empty.
	i18nDir := filepath.Join(projectDir, "i18n")
	if err := os.MkdirAll(i18nDir, 0o755); err != nil {
		return fmt.Errorf("creating i18n directory: %w", err)
	}
	targetFile := filepath.Join(i18nDir, code+".yaml")
	if _, err := os.Stat(targetFile); os.IsNotExist(err) {
		defaultFile := filepath.Join(i18nDir, defaultLang+".yaml")
		data, err := os.ReadFile(defaultFile)
		if err != nil {
			data = []byte{}
		}
		if err := os.WriteFile(targetFile, data, 0o644); err != nil {
			return fmt.Errorf("writing i18n file: %w", err)
		}
	}

	pm.eventHub.Broadcast(Event{Type: "config:changed"})
	return nil
}

// TranslationStatus returns, per collection and language, the total number
// of pages in the default language and how many exist in each language.
func (pm *ProjectManager) TranslationStatus() (map[string]map[string]TranslationCoverage, error) {
	pm.mu.RLock()
	if pm.state == StateClosed {
		pm.mu.RUnlock()
		return nil, fmt.Errorf("no project open")
	}
	contentDir := pm.contentDir()
	cfg := pm.config
	pm.mu.RUnlock()

	if cfg == nil {
		return map[string]map[string]TranslationCoverage{}, nil
	}

	defaultLang := cfg.I18n.GetDefaultLanguage()

	languages := []string{defaultLang}
	for _, code := range cfg.I18n.LanguageCodes() {
		if code != defaultLang {
			languages = append(languages, code)
		}
	}

	// Discover collections from the default-language tree, if present;
	// otherwise fall back to the content root (pre-i18n layout).
	defaultDir := contentDir
	if fi, err := os.Stat(filepath.Join(contentDir, defaultLang)); err == nil && fi.IsDir() {
		defaultDir = filepath.Join(contentDir, defaultLang)
	}

	entries, err := os.ReadDir(defaultDir)
	if err != nil {
		return nil, fmt.Errorf("reading content directory: %w", err)
	}

	result := make(map[string]map[string]TranslationCoverage)
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") || strings.HasPrefix(e.Name(), "_") {
			continue
		}
		colName := e.Name()
		total := countMarkdownFiles(filepath.Join(defaultDir, colName))

		perLang := make(map[string]TranslationCoverage, len(languages))
		perLang[defaultLang] = TranslationCoverage{Total: total, Translated: total}

		for _, lang := range languages {
			if lang == defaultLang {
				continue
			}
			translated := countMarkdownFiles(filepath.Join(contentDir, lang, colName))
			perLang[lang] = TranslationCoverage{Total: total, Translated: translated}
		}

		result[colName] = perLang
	}

	return result, nil
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
