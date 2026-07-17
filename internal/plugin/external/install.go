package external

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/download"
)

// Install fetches a plugin from source (local zip, local directory, GitHub
// repository, or zip/tar.gz URL), validates its manifest in a staging
// directory, and moves it into {projectDir}/plugins/{slug}/. The destination
// directory name always equals the manifest slug; there is no name override
// because license lookup and enable/disable both key off the slug. reserved
// holds built-in plugin names that external slugs may not use.
func Install(projectDir, source string, reserved []string) (*Manifest, error) {
	kind := download.InferSourceKind(source)
	if kind == download.SourceUnknown {
		return nil, fmt.Errorf("cannot determine source type for %q; provide a zip file, GitHub URL, zip/tar.gz URL, or local directory", source)
	}

	staging, err := os.MkdirTemp("", "sarde-plugin-*")
	if err != nil {
		return nil, fmt.Errorf("creating staging directory: %w", err)
	}
	defer os.RemoveAll(staging)

	if err := fetchToStaging(source, kind, staging); err != nil {
		return nil, err
	}

	root, err := findPluginRoot(staging)
	if err != nil {
		return nil, err
	}

	m, err := LoadManifest(root)
	if err != nil {
		return nil, err
	}
	if err := m.Validate(m.Slug); err != nil {
		return nil, err
	}
	for _, name := range reserved {
		if name == m.Slug {
			return nil, fmt.Errorf("slug %q conflicts with a built-in plugin name", m.Slug)
		}
	}

	destDir := filepath.Join(projectDir, consts.DirPlugins, m.Slug)
	if _, err := os.Stat(destDir); err == nil {
		return nil, fmt.Errorf("plugin %q already exists; remove it first with 'sarde plugin remove %s'", m.Slug, m.Slug)
	}
	if err := os.MkdirAll(filepath.Dir(destDir), 0o755); err != nil {
		return nil, err
	}

	// Rename is atomic when staging and project share a volume; fall back to
	// a copy when they do not.
	if err := os.Rename(root, destDir); err != nil {
		if err := download.CopyDir(root, destDir); err != nil {
			return nil, fmt.Errorf("installing plugin into %s: %w", destDir, err)
		}
	}
	return m, nil
}

// Remove deletes {projectDir}/plugins/{slug}.
func Remove(projectDir, slug string) error {
	if !ValidSlug(slug) {
		return fmt.Errorf("invalid plugin slug %q", slug)
	}
	dir := filepath.Join(projectDir, consts.DirPlugins, slug)
	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("plugin %q is not installed", slug)
	}
	return os.RemoveAll(dir)
}

func fetchToStaging(source string, kind download.SourceKind, staging string) error {
	switch kind {
	case download.SourceLocalDir:
		return download.CopyDir(source, staging)
	case download.SourceLocalZipFile:
		return download.ExtractZip(source, staging, 0)
	case download.SourceGitHub:
		ref, err := download.ParseGitHubURL(source)
		if err != nil {
			return err
		}
		tmpPath, err := download.DownloadFile(ref.ArchiveURL())
		if err != nil {
			return fmt.Errorf("downloading plugin: %w", err)
		}
		defer os.Remove(tmpPath)
		if err := download.ExtractZip(tmpPath, staging, 1); err != nil {
			return err
		}
		if ref.Subpath != "" {
			// Narrow the staging root to the subpath by relocating it.
			sub := filepath.Join(staging, filepath.FromSlash(ref.Subpath))
			if _, err := os.Stat(sub); err != nil {
				return fmt.Errorf("subpath %q not found in repository", ref.Subpath)
			}
			keep := staging + "-sub"
			if err := os.Rename(sub, keep); err != nil {
				return err
			}
			if err := os.RemoveAll(staging); err != nil {
				return err
			}
			return os.Rename(keep, staging)
		}
		return nil
	case download.SourceZipURL:
		tmpPath, err := download.DownloadFile(source)
		if err != nil {
			return fmt.Errorf("downloading plugin: %w", err)
		}
		defer os.Remove(tmpPath)
		return download.ExtractZip(tmpPath, staging, 0)
	case download.SourceTarGzURL:
		tmpPath, err := download.DownloadFile(source)
		if err != nil {
			return fmt.Errorf("downloading plugin: %w", err)
		}
		defer os.Remove(tmpPath)
		return download.ExtractTarGz(tmpPath, staging, 0)
	}
	return fmt.Errorf("unsupported source kind")
}

// findPluginRoot locates the directory containing plugin.yaml: the staging
// root itself, or a single wrapping top-level directory (the common shape of
// zips archived with a folder inside).
func findPluginRoot(staging string) (string, error) {
	if _, err := os.Stat(filepath.Join(staging, consts.FilePluginManifest)); err == nil {
		return staging, nil
	}
	entries, err := os.ReadDir(staging)
	if err != nil {
		return "", err
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, e.Name())
		}
	}
	if len(dirs) == 1 {
		nested := filepath.Join(staging, dirs[0])
		if _, err := os.Stat(filepath.Join(nested, consts.FilePluginManifest)); err == nil {
			return nested, nil
		}
	}
	return "", fmt.Errorf("no %s found in the archive; not a valid sarde plugin", consts.FilePluginManifest)
}
