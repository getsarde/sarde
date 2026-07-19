package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/build"
	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/content"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/theme"
)

// GetConfig returns the current site configuration.
func (pm *ProjectManager) GetConfig() *config.SiteConfig {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.config
}

// UpdateSettings applies partial config updates and re-resolves.
func (pm *ProjectManager) UpdateSettings(input SettingsInput) error {
	pm.mu.Lock()
	if pm.state == StateClosed {
		pm.mu.Unlock()
		return fmt.Errorf("no project open")
	}
	projectDir := pm.projectDir
	pm.mu.Unlock()

	siteYAMLPath := filepath.Join(projectDir, consts.FileSiteConfig)
	data, err := os.ReadFile(siteYAMLPath)
	if err != nil {
		return fmt.Errorf("reading sarde.yaml: %w", err)
	}

	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		raw = make(map[string]any)
	}

	siteSection, _ := raw["site"].(map[string]any)
	if siteSection == nil {
		siteSection = make(map[string]any)
		raw["site"] = siteSection
	}

	if input.Title != nil {
		siteSection["title"] = *input.Title
	}
	if input.URL != nil {
		siteSection["url"] = *input.URL
	}
	if input.Language != nil {
		siteSection["language"] = *input.Language
	}
	if input.Description != nil {
		siteSection["description"] = *input.Description
	}

	out, err := yaml.Marshal(raw)
	if err != nil {
		return fmt.Errorf("marshaling sarde.yaml: %w", err)
	}
	if err := os.WriteFile(siteYAMLPath, out, 0o644); err != nil {
		return err
	}

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

// GetCollections returns metadata about all collections.
func (pm *ProjectManager) GetCollections() ([]CollectionInfo, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if pm.state == StateClosed {
		return nil, fmt.Errorf("no project open")
	}

	return pm.getCollectionsLocked()
}

// getCollectionsLocked is the lock-free version for internal use when lock is already held.
// Uses a lightweight directory scan instead of the full DiscoverFiles pipeline.
func (pm *ProjectManager) getCollectionsLocked() ([]CollectionInfo, error) {
	contentDir := pm.contentDir()
	entries, err := os.ReadDir(contentDir)
	if err != nil {
		return nil, err
	}

	var collections []CollectionInfo
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") || strings.HasPrefix(e.Name(), "_") {
			continue
		}
		name := e.Name()
		count := countMarkdownFiles(filepath.Join(contentDir, name))
		collections = append(collections, CollectionInfo{
			Name:      name,
			Title:     content.FilenameToTitle(name),
			PageCount: count,
		})
	}
	return collections, nil
}

func countMarkdownFiles(dir string) int {
	count := 0
	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		// Mirror the scanner's discovery filter so the page counts shown in
		// the UI match what actually builds (nested .trash/_drafts dirs and
		// hidden files don't count).
		if d.IsDir() {
			if path != dir && content.IsIgnoredDirName(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !content.IsIgnoredFileName(d.Name()) && strings.HasSuffix(d.Name(), ".md") {
			count++
		}
		return nil
	})
	return count
}

func (pm *ProjectManager) resolveConfig(projectDir string) (*config.SiteConfig, *engine.ThemeConfig, error) {
	configPath := filepath.Join(projectDir, consts.FileSiteConfig)
	cfg, err := config.Resolve(config.ResolveOptions{
		ConfigPath:   configPath,
		EnvPrefix:    "SARDE",
		Strict:       true,
		KnownPlugins: build.KnownPluginNames(projectDir),
	})
	if err != nil {
		return nil, nil, fmt.Errorf("resolving config: %w", err)
	}

	// Load theme.
	var thm *theme.Theme
	if cfg.Theme.Name != "" && cfg.Theme.Name != "default" {
		thm, _ = theme.LoadFromDir(filepath.Join(projectDir, "themes", cfg.Theme.Name))
	}
	if thm == nil {
		thm, _ = theme.LoadFromFS(embedded.ThemeFS(), ".")
	}

	// Fold theme shortcut fields into overrides before token resolution.
	foldThemeShortcuts(cfg)

	// Validate token names in overrides.
	known := theme.KnownTokens()
	if err := theme.ValidateOverrides("theme.overrides", cfg.Theme.Overrides, known); err != nil {
		return nil, nil, err
	}
	if err := theme.ValidateOverrides("theme.dark_overrides", cfg.Theme.DarkOverrides, known); err != nil {
		return nil, nil, err
	}

	// Resolve tokens.
	lightTokens := theme.ResolveTokens(theme.DefaultTokens(), thm, cfg.Theme.Preset, cfg.Theme.Overrides)
	lightTokens = theme.DeriveTokens(lightTokens)
	darkTokens := theme.ResolveDarkTokens(theme.DefaultDarkTokens(), thm, cfg.Theme.Preset, darkOverrides(cfg))
	styleTag := theme.GenerateStyleTag(lightTokens, darkTokens)

	name := "Default"
	slug := "default"
	if thm != nil {
		if thm.Name != "" {
			name = thm.Name
		}
		if thm.Slug != "" {
			slug = thm.Slug
		}
	}

	themeCfg := &engine.ThemeConfig{
		Name:        name,
		Slug:        slug,
		Tokens:      lightTokens,
		DarkTokens:  darkTokens,
		DarkEnabled: config.BoolVal(cfg.Theme.Dark, true),
		StyleTag:    styleTag,
	}

	return cfg, themeCfg, nil
}

func foldThemeShortcuts(cfg *config.SiteConfig) {
	accentVal := cfg.Theme.AccentColor
	if accentVal == "" {
		accentVal = cfg.Theme.PrimaryColor
	}
	shortcuts := map[string]string{
		"accent":    accentVal,
		"font-sans": cfg.Theme.FontFamily,
		"font-mono": cfg.Theme.FontMono,
	}
	if cfg.Theme.Overrides == nil {
		cfg.Theme.Overrides = make(map[string]string)
	}
	for token, val := range shortcuts {
		if val != "" {
			if _, exists := cfg.Theme.Overrides[token]; !exists {
				cfg.Theme.Overrides[token] = val
			}
		}
	}
}

func darkOverrides(cfg *config.SiteConfig) map[string]string {
	if len(cfg.Theme.DarkOverrides) > 0 {
		return cfg.Theme.DarkOverrides
	}
	return cfg.Theme.Overrides
}
