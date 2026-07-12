package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/consts"
	"github.com/spf13/cobra"
)

var docVersionCmd = &cobra.Command{
	Use:           "doc-version",
	Short:         "Manage docs versioning for a collection",
	SilenceUsage:  true,
	SilenceErrors: true,
}

var docVersionCreateCmd = &cobra.Command{
	Use:           "create <project-dir> <collection> <version-id> <label>",
	Short:         "Cut a new version for a collection",
	Args:          cobra.ExactArgs(4),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runDocVersionCreate,
}

var docVersionDeleteCmd = &cobra.Command{
	Use:           "delete <project-dir> <collection> <version-id>",
	Short:         "Delete a version from a collection",
	Args:          cobra.ExactArgs(3),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runDocVersionDelete,
}

var docVersionUpdateCmd = &cobra.Command{
	Use:           "update <project-dir> <collection> <version-id>",
	Short:         "Update a version entry's label, banner, or redirect",
	Args:          cobra.ExactArgs(3),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runDocVersionUpdate,
}

func init() {
	docVersionUpdateCmd.Flags().String("label", "", "New version label")
	docVersionUpdateCmd.Flags().String("banner", "", "Banner: none, unmaintained, unreleased")
	docVersionUpdateCmd.Flags().String("redirect", "", "Redirect: same-page, root")

	docVersionCmd.AddCommand(docVersionCreateCmd)
	docVersionCmd.AddCommand(docVersionDeleteCmd)
	docVersionCmd.AddCommand(docVersionUpdateCmd)
	rootCmd.AddCommand(docVersionCmd)
}

// runDocVersionCreate cuts a new version for a collection. If a prior
// last_version already existed, the collection's current root content is
// copied (not moved) into content/<collection>/<oldLastVersion>/, freezing
// that version's content while root content continues to represent the new
// unreleased version.
func runDocVersionCreate(cmd *cobra.Command, args []string) error {
	projectDir := projectDirFromArgs(args)
	collection := args[1]
	versionID := strings.TrimSpace(args[2])
	label := args[3]

	if collection == "" {
		return printJSONError(fmt.Errorf("collection name is required"))
	}
	if versionID == "" {
		return printJSONError(fmt.Errorf("version id is required"))
	}
	if strings.ContainsAny(versionID, "/\\") || strings.Contains(versionID, "..") {
		return printJSONError(fmt.Errorf("invalid version id %q", versionID))
	}

	siteYAMLPath := filepath.Join(projectDir, consts.FileSiteConfig)
	raw, err := docVersionReadRawYAML(siteYAMLPath)
	if err != nil {
		return printJSONError(err)
	}

	collections := docVersionEnsureMapSection(raw, "collections")
	colCfg, ok := collections[collection].(map[string]any)
	if !ok {
		colCfg = make(map[string]any)
		collections[collection] = colCfg
	}
	versioning := docVersionEnsureMapSection(colCfg, "versioning")
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
			return printJSONError(fmt.Errorf("version %q already exists", versionID))
		}
		existingIDs[id] = true
	}

	if oldLastVersion != "" {
		colContentDir := filepath.Join(projectDir, consts.DirContent, collection)
		if err := docVersionArchiveContent(colContentDir, oldLastVersion, existingIDs); err != nil {
			return printJSONError(fmt.Errorf("archiving version %q: %w", oldLastVersion, err))
		}
	}

	entries = append(entries, map[string]any{"id": versionID, "label": label})
	versioning["versions"] = entries
	versioning["last_version"] = versionID

	if err := docVersionValidate(collection, versioning); err != nil {
		return printJSONError(err)
	}

	if err := docVersionWriteRawYAML(siteYAMLPath, raw); err != nil {
		return printJSONError(err)
	}

	printJSONResult(map[string]any{"ok": true, "versionId": versionID, "lastVersion": versionID})
	return nil
}

// runDocVersionDelete removes a version entry and its archived content
// directory. Rejects deleting the collection's current last_version.
func runDocVersionDelete(cmd *cobra.Command, args []string) error {
	projectDir := projectDirFromArgs(args)
	collection := args[1]
	versionID := strings.TrimSpace(args[2])

	siteYAMLPath := filepath.Join(projectDir, consts.FileSiteConfig)
	raw, err := docVersionReadRawYAML(siteYAMLPath)
	if err != nil {
		return printJSONError(err)
	}

	collections := docVersionEnsureMapSection(raw, "collections")
	colCfg, ok := collections[collection].(map[string]any)
	if !ok {
		return printJSONError(fmt.Errorf("collection %q has no versioning configured", collection))
	}
	versioning, ok := colCfg["versioning"].(map[string]any)
	if !ok {
		return printJSONError(fmt.Errorf("collection %q has no versioning configured", collection))
	}

	if lastVersion, _ := versioning["last_version"].(string); lastVersion == versionID {
		return printJSONError(fmt.Errorf("cannot delete the current latest version %q", versionID))
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
		return printJSONError(fmt.Errorf("version %q not found", versionID))
	}
	versioning["versions"] = remaining

	if err := docVersionValidate(collection, versioning); err != nil {
		return printJSONError(err)
	}

	if err := docVersionWriteRawYAML(siteYAMLPath, raw); err != nil {
		return printJSONError(err)
	}

	versionDir := filepath.Join(projectDir, consts.DirContent, collection, versionID)
	if err := os.RemoveAll(versionDir); err != nil {
		return printJSONError(fmt.Errorf("removing version directory: %w", err))
	}

	printJSONResult(map[string]any{"ok": true, "versionId": versionID})
	return nil
}

// runDocVersionUpdate updates the label, banner, or redirect of an existing
// version entry. Only flags explicitly passed on the command line are applied.
func runDocVersionUpdate(cmd *cobra.Command, args []string) error {
	projectDir := projectDirFromArgs(args)
	collection := args[1]
	versionID := strings.TrimSpace(args[2])

	siteYAMLPath := filepath.Join(projectDir, consts.FileSiteConfig)
	raw, err := docVersionReadRawYAML(siteYAMLPath)
	if err != nil {
		return printJSONError(err)
	}

	collections := docVersionEnsureMapSection(raw, "collections")
	colCfg, ok := collections[collection].(map[string]any)
	if !ok {
		return printJSONError(fmt.Errorf("collection %q has no versioning configured", collection))
	}
	versioning, ok := colCfg["versioning"].(map[string]any)
	if !ok {
		return printJSONError(fmt.Errorf("collection %q has no versioning configured", collection))
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
		if cmd.Flags().Changed("label") {
			v, _ := cmd.Flags().GetString("label")
			em["label"] = v
		}
		if cmd.Flags().Changed("banner") {
			v, _ := cmd.Flags().GetString("banner")
			em["banner"] = v
		}
		if cmd.Flags().Changed("redirect") {
			v, _ := cmd.Flags().GetString("redirect")
			em["redirect"] = v
		}
		break
	}
	if !found {
		return printJSONError(fmt.Errorf("version %q not found", versionID))
	}

	if err := docVersionValidate(collection, versioning); err != nil {
		return printJSONError(err)
	}

	if err := docVersionWriteRawYAML(siteYAMLPath, raw); err != nil {
		return printJSONError(err)
	}

	printJSONResult(map[string]any{"ok": true, "versionId": versionID})
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// docVersionValidate round-trips a raw versioning map through YAML into a
// typed config.VersioningConfig and validates it with config.ValidateVersioning.
func docVersionValidate(collection string, versioning map[string]any) error {
	data, err := yaml.Marshal(versioning)
	if err != nil {
		return err
	}
	var vc config.VersioningConfig
	if err := yaml.Unmarshal(data, &vc); err != nil {
		return err
	}
	return config.ValidateVersioning(collection, &vc)
}

// docVersionArchiveContent copies the current top-level entries of a
// collection's content directory into destDir/<oldVersionID>/, skipping
// already-archived version subdirectories and the collection's config file.
// Source files are left in place: they continue to represent the new
// unreleased version going forward.
func docVersionArchiveContent(colContentDir, oldVersionID string, existingIDs map[string]bool) error {
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
		if err := docVersionCopyPath(src, dst); err != nil {
			return err
		}
	}
	return nil
}

// docVersionCopyPath copies a file or directory tree from src to dst.
func docVersionCopyPath(src, dst string) error {
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
			if err := docVersionCopyPath(filepath.Join(src, e.Name()), filepath.Join(dst, e.Name())); err != nil {
				return err
			}
		}
		return nil
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode())
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

// docVersionReadRawYAML reads and unmarshals a YAML file into a generic map,
// treating a missing/unparseable file as an empty document.
func docVersionReadRawYAML(path string) (map[string]any, error) {
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

// docVersionWriteRawYAML marshals and writes a generic map back to a YAML file.
func docVersionWriteRawYAML(path string, raw map[string]any) error {
	out, err := yaml.Marshal(raw)
	if err != nil {
		return fmt.Errorf("marshaling %s: %w", filepath.Base(path), err)
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", filepath.Base(path), err)
	}
	return nil
}

// docVersionEnsureMapSection returns raw[key] as a map[string]any, creating
// and inserting one if absent or of an unexpected type.
func docVersionEnsureMapSection(raw map[string]any, key string) map[string]any {
	section, ok := raw[key].(map[string]any)
	if !ok {
		section = make(map[string]any)
		raw[key] = section
	}
	return section
}
