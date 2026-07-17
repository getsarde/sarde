package project

import (
	"fmt"
	"path/filepath"

	"github.com/getsarde/sarde/internal/build"
	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/license"
	"github.com/getsarde/sarde/internal/plugin/external"
)

// PluginInfo describes an installed external plugin for the desktop app.
// Fields carries blueprint metadata for a future plugin settings UI.
type PluginInfo struct {
	Slug        string                    `json:"slug"`
	Name        string                    `json:"name"`
	Version     string                    `json:"version"`
	Description string                    `json:"description"`
	Author      string                    `json:"author"`
	Kind        string                    `json:"kind"` // always "external" for now
	Premium     bool                      `json:"premium"`
	LicenseOK   bool                      `json:"licenseOk"`
	LicenseMsg  string                    `json:"licenseMsg,omitempty"`
	Enabled     bool                      `json:"enabled"`
	PurchaseURL string                    `json:"purchaseUrl,omitempty"`
	Fields      []external.BlueprintField `json:"fields,omitempty"`
}

// ListPlugins returns all external plugins found in the project's plugins/
// directory, including ones with unreadable manifests (surfaced via
// LicenseMsg so the UI can show the problem).
func (pm *ProjectManager) ListPlugins() ([]PluginInfo, error) {
	projectDir, err := pm.openProjectDir()
	if err != nil {
		return nil, err
	}

	pm.mu.RLock()
	disabled := make(map[string]bool)
	if pm.config != nil {
		for _, name := range pm.config.Plugins.Disabled {
			disabled[name] = true
		}
	}
	pm.mu.RUnlock()

	var infos []PluginInfo
	for _, dir := range external.DiscoverDirs(projectDir) {
		slug := filepath.Base(dir)
		info := PluginInfo{Slug: slug, Kind: "external", Enabled: !disabled[slug]}

		m, err := external.LoadManifest(dir)
		if err != nil {
			info.Enabled = false
			info.LicenseMsg = err.Error()
			infos = append(infos, info)
			continue
		}

		info.Name = m.Name
		info.Version = m.Version
		info.Description = m.Description
		info.Author = m.Author
		info.Premium = m.Premium
		info.PurchaseURL = m.PurchaseURL
		if m.Premium {
			if err := license.VerifyFor(projectDir, slug, m.Version); err != nil {
				info.LicenseMsg = err.Error()
			} else {
				info.LicenseOK = true
			}
		} else {
			info.LicenseOK = true
		}
		if fields, err := external.LoadBlueprint(dir); err == nil {
			info.Fields = fields
		}
		infos = append(infos, info)
	}
	return infos, nil
}

// EnablePlugin removes slug from plugins.disabled in sarde.yaml.
func (pm *ProjectManager) EnablePlugin(slug string) error {
	return pm.setPluginDisabled(slug, false)
}

// DisablePlugin adds slug to plugins.disabled in sarde.yaml.
func (pm *ProjectManager) DisablePlugin(slug string) error {
	return pm.setPluginDisabled(slug, true)
}

// InstallPlugin installs an external plugin from source (zip path, URL,
// GitHub reference, or local directory) and returns its info.
func (pm *ProjectManager) InstallPlugin(source string) (*PluginInfo, error) {
	projectDir, err := pm.openProjectDir()
	if err != nil {
		return nil, err
	}

	m, err := external.Install(projectDir, source, build.KnownPluginNames(""))
	if err != nil {
		return nil, err
	}

	if err := pm.reresolveAndBroadcast(projectDir); err != nil {
		return nil, err
	}

	info := &PluginInfo{
		Slug: m.Slug, Name: m.Name, Version: m.Version,
		Description: m.Description, Author: m.Author,
		Kind: "external", Premium: m.Premium, PurchaseURL: m.PurchaseURL,
		Enabled: true,
	}
	if m.Premium {
		if err := license.VerifyFor(projectDir, m.Slug, m.Version); err != nil {
			info.LicenseMsg = err.Error()
		} else {
			info.LicenseOK = true
		}
	} else {
		info.LicenseOK = true
	}
	return info, nil
}

// RemovePlugin deletes plugins/{slug} from the project. License files are
// kept: they live outside the plugin directory.
func (pm *ProjectManager) RemovePlugin(slug string) error {
	projectDir, err := pm.openProjectDir()
	if err != nil {
		return err
	}
	if err := external.Remove(projectDir, slug); err != nil {
		return err
	}
	return pm.reresolveAndBroadcast(projectDir)
}

// openProjectDir returns the project directory or an error when no project
// is open.
func (pm *ProjectManager) openProjectDir() (string, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	if pm.state == StateClosed {
		return "", fmt.Errorf("no project open")
	}
	return pm.projectDir, nil
}

// setPluginDisabled surgically edits the plugins.disabled list in sarde.yaml.
func (pm *ProjectManager) setPluginDisabled(slug string, disable bool) error {
	if !external.ValidSlug(slug) {
		return fmt.Errorf("invalid plugin slug %q", slug)
	}
	projectDir, err := pm.openProjectDir()
	if err != nil {
		return err
	}

	siteYAMLPath := filepath.Join(projectDir, consts.FileSiteConfig)
	raw, err := readRawYAML(siteYAMLPath)
	if err != nil {
		return err
	}

	pluginsSection := ensureMapSection(raw, "plugins")
	var current []any
	if list, ok := pluginsSection["disabled"].([]any); ok {
		current = list
	}

	updated := make([]any, 0, len(current)+1)
	present := false
	for _, item := range current {
		if name, ok := item.(string); ok && name == slug {
			present = true
			if !disable {
				continue // enabling: drop the entry
			}
		}
		updated = append(updated, item)
	}
	if disable && !present {
		updated = append(updated, slug)
	}

	if len(updated) == 0 {
		delete(pluginsSection, "disabled")
		if len(pluginsSection) == 0 {
			delete(raw, "plugins")
		}
	} else {
		pluginsSection["disabled"] = updated
	}

	if err := writeRawYAML(siteYAMLPath, raw); err != nil {
		return err
	}
	return pm.reresolveAndBroadcast(projectDir)
}
