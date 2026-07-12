package project

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/consts"
)

// CreateVersion cuts a new version for a versioned collection. If a prior
// last_version already existed, the collection's current root content is
// copied (not moved) into content/<collection>/<oldLastVersion>/, freezing
// that version's content while root content continues to represent the new
// unreleased version. versionID is then registered as the new last_version.
func (pm *ProjectManager) CreateVersion(collection, versionID, label string) error {
	pm.mu.Lock()
	if pm.state == StateClosed {
		pm.mu.Unlock()
		return fmt.Errorf("no project open")
	}
	projectDir := pm.projectDir
	contentDir := pm.contentDir()
	pm.mu.Unlock()

	if collection == "" {
		return fmt.Errorf("collection name is required")
	}
	if versionID == "" {
		return fmt.Errorf("version id is required")
	}
	if strings.ContainsAny(versionID, "/\\") || strings.Contains(versionID, "..") {
		return fmt.Errorf("invalid version id %q", versionID)
	}

	siteYAMLPath := filepath.Join(projectDir, consts.FileSiteConfig)
	raw, err := readRawYAML(siteYAMLPath)
	if err != nil {
		return err
	}

	collections := ensureMapSection(raw, "collections")
	colCfg, ok := collections[collection].(map[string]any)
	if !ok {
		colCfg = make(map[string]any)
		collections[collection] = colCfg
	}
	versioning := ensureMapSection(colCfg, "versioning")
	versioning["enabled"] = true

	oldLastVersion, _ := versioning["last_version"].(string)

	entries, _ := versioning["versions"].([]any)
	existingIDs := make(map[string]bool, len(entries))
	for _, e := range entries {
		em, ok := e.(map[string]any)
		if !ok {
			continue
		}
		id, _ := em["id"].(string)
		if id == "" {
			continue
		}
		if id == versionID {
			return fmt.Errorf("version %q already exists", versionID)
		}
		existingIDs[id] = true
	}

	if oldLastVersion != "" {
		colContentDir := filepath.Join(contentDir, collection)
		if err := archiveVersionContent(colContentDir, oldLastVersion, existingIDs); err != nil {
			return fmt.Errorf("archiving version %q: %w", oldLastVersion, err)
		}
	}

	entries = append(entries, map[string]any{"id": versionID, "label": label})
	versioning["versions"] = entries
	versioning["last_version"] = versionID

	vc, err := decodeVersioningConfig(versioning)
	if err != nil {
		return fmt.Errorf("invalid versioning config: %w", err)
	}
	if err := config.ValidateVersioning(collection, vc); err != nil {
		return err
	}

	if err := writeRawYAML(siteYAMLPath, raw); err != nil {
		return err
	}

	return pm.reresolveAndBroadcast(projectDir)
}

// DeleteVersion removes a version entry and its archived content directory.
// Rejects deleting the collection's current last_version.
func (pm *ProjectManager) DeleteVersion(collection, versionID string) error {
	pm.mu.Lock()
	if pm.state == StateClosed {
		pm.mu.Unlock()
		return fmt.Errorf("no project open")
	}
	projectDir := pm.projectDir
	contentDir := pm.contentDir()
	pm.mu.Unlock()

	if collection == "" || versionID == "" {
		return fmt.Errorf("collection and version id are required")
	}

	siteYAMLPath := filepath.Join(projectDir, consts.FileSiteConfig)
	raw, err := readRawYAML(siteYAMLPath)
	if err != nil {
		return err
	}

	collections := ensureMapSection(raw, "collections")
	colCfg, ok := collections[collection].(map[string]any)
	if !ok {
		return fmt.Errorf("collection %q has no versioning configured", collection)
	}
	versioning, ok := colCfg["versioning"].(map[string]any)
	if !ok {
		return fmt.Errorf("collection %q has no versioning configured", collection)
	}

	if lastVersion, _ := versioning["last_version"].(string); lastVersion == versionID {
		return fmt.Errorf("cannot delete the current latest version %q", versionID)
	}

	entries, _ := versioning["versions"].([]any)
	found := false
	remaining := make([]any, 0, len(entries))
	for _, e := range entries {
		if em, ok := e.(map[string]any); ok {
			if id, _ := em["id"].(string); id == versionID {
				found = true
				continue
			}
		}
		remaining = append(remaining, e)
	}
	if !found {
		return fmt.Errorf("version %q not found", versionID)
	}
	versioning["versions"] = remaining

	vc, err := decodeVersioningConfig(versioning)
	if err != nil {
		return fmt.Errorf("invalid versioning config: %w", err)
	}
	if err := config.ValidateVersioning(collection, vc); err != nil {
		return err
	}

	if err := writeRawYAML(siteYAMLPath, raw); err != nil {
		return err
	}

	versionDir := filepath.Join(contentDir, collection, versionID)
	if err := os.RemoveAll(versionDir); err != nil {
		return fmt.Errorf("removing version directory: %w", err)
	}

	return pm.reresolveAndBroadcast(projectDir)
}

// UpdateVersionEntry updates the label, banner, or redirect of an existing
// version entry. Each pointer arg is applied only when non-nil, leaving the
// existing value untouched otherwise.
func (pm *ProjectManager) UpdateVersionEntry(collection, versionID string, label, banner, redirect *string) error {
	pm.mu.Lock()
	if pm.state == StateClosed {
		pm.mu.Unlock()
		return fmt.Errorf("no project open")
	}
	projectDir := pm.projectDir
	pm.mu.Unlock()

	if collection == "" || versionID == "" {
		return fmt.Errorf("collection and version id are required")
	}

	siteYAMLPath := filepath.Join(projectDir, consts.FileSiteConfig)
	raw, err := readRawYAML(siteYAMLPath)
	if err != nil {
		return err
	}

	collections := ensureMapSection(raw, "collections")
	colCfg, ok := collections[collection].(map[string]any)
	if !ok {
		return fmt.Errorf("collection %q has no versioning configured", collection)
	}
	versioning, ok := colCfg["versioning"].(map[string]any)
	if !ok {
		return fmt.Errorf("collection %q has no versioning configured", collection)
	}

	entries, _ := versioning["versions"].([]any)
	found := false
	for _, e := range entries {
		em, ok := e.(map[string]any)
		if !ok {
			continue
		}
		if id, _ := em["id"].(string); id != versionID {
			continue
		}
		found = true
		if label != nil {
			em["label"] = *label
		}
		if banner != nil {
			em["banner"] = *banner
		}
		if redirect != nil {
			em["redirect"] = *redirect
		}
		break
	}
	if !found {
		return fmt.Errorf("version %q not found", versionID)
	}

	vc, err := decodeVersioningConfig(versioning)
	if err != nil {
		return fmt.Errorf("invalid versioning config: %w", err)
	}
	if err := config.ValidateVersioning(collection, vc); err != nil {
		return err
	}

	if err := writeRawYAML(siteYAMLPath, raw); err != nil {
		return err
	}

	return pm.reresolveAndBroadcast(projectDir)
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// decodeVersioningConfig round-trips a raw versioning map through YAML into
// a typed config.VersioningConfig so it can be validated with
// config.ValidateVersioning before being persisted.
func decodeVersioningConfig(raw map[string]any) (*config.VersioningConfig, error) {
	data, err := yaml.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var vc config.VersioningConfig
	if err := yaml.Unmarshal(data, &vc); err != nil {
		return nil, err
	}
	return &vc, nil
}

// archiveVersionContent copies the current top-level entries of a
// collection's content directory into destDir/<oldVersionID>/, skipping
// already-archived version subdirectories and the collection's config file.
// Source files are left in place: they continue to represent the new
// unreleased version going forward.
func archiveVersionContent(colContentDir, oldVersionID string, existingIDs map[string]bool) error {
	entries, err := os.ReadDir(colContentDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	destDir := filepath.Join(colContentDir, oldVersionID)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}

	for _, e := range entries {
		name := e.Name()
		if name == oldVersionID || existingIDs[name] {
			continue
		}
		if name == "config.yaml" || name == "config.yml" {
			continue
		}
		src := filepath.Join(colContentDir, name)
		dst := filepath.Join(destDir, name)
		if err := copyPath(src, dst); err != nil {
			return err
		}
	}
	return nil
}

// copyPath copies a file or directory tree from src to dst.
func copyPath(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		if err := os.MkdirAll(dst, info.Mode()); err != nil {
			return err
		}
		entries, err := os.ReadDir(src)
		if err != nil {
			return err
		}
		for _, e := range entries {
			if err := copyPath(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
				return err
			}
		}
		return nil
	}
	return copyFileContents(src, dst, info.Mode())
}

func copyFileContents(src, dst string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
