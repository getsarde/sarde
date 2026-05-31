package content

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/frostybee/sarde/internal/devlog"
	"github.com/frostybee/sarde/internal/engine"
)

// ContentFile holds metadata about a discovered content file.
type ContentFile struct {
	FilePath       string          // absolute path
	RelPath        string          // relative to content dir (forward slashes)
	Kind           engine.NodeKind // home, section, page, bundle, standalone
	CollectionName string          // top-level dir name, "" for root-level files
	Slug           string          // derived from filename
	Weight         int             // from numeric prefix
	IsBundle       bool            // true if index.md with sibling assets
	BundleAssets   []string        // sibling non-.md files (bundles only)
	Lang           string          // language code (set by i18n detector)
	LangRelPath    string          // relative path within language root (for translation matching)
	Version        string          // version ID (set by version detector), e.g. "v1"
	VersionRelPath string          // path within the version root (cross-version key)
}

// Scanner walks the content directory and returns file paths grouped by collection.
type Scanner struct {
	Languages   map[string]bool            // configured language codes (nil = single-language)
	DefaultLang string                     // default language code
	VersionIDs  map[string]map[string]bool // collection name → set of version IDs (nil = no versioning)
}

// Discover walks contentDir and returns file paths grouped by collection name.
// Root-level files (standalone, home) are grouped under the "" key.
func (s *Scanner) Discover(contentDir string) (map[string][]string, error) {
	files, err := s.DiscoverFiles(contentDir)
	if err != nil {
		return nil, err
	}

	result := make(map[string][]string)
	for _, f := range files {
		result[f.CollectionName] = append(result[f.CollectionName], f.FilePath)
	}
	return result, nil
}

// DiscoverFiles walks contentDir and returns a richer ContentFile for each .md file found.
func (s *Scanner) DiscoverFiles(contentDir string) ([]ContentFile, error) {
	contentDir = filepath.Clean(contentDir)
	var files []ContentFile

	// First pass: collect all entries to detect bundles
	dirAssets := make(map[string][]string) // dir → list of non-.md files

	err := filepath.WalkDir(contentDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".md" {
			// Track non-markdown files for bundle detection
			dir := filepath.Dir(path)
			dirAssets[dir] = append(dirAssets[dir], path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Second pass: classify .md files
	err = filepath.WalkDir(contentDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || strings.ToLower(filepath.Ext(path)) != ".md" {
			return nil
		}

		rel, relErr := filepath.Rel(contentDir, path)
		if relErr != nil {
			// Can't make path relative to contentDir (e.g. different Windows
			// drive). Skip it rather than producing a page with an empty slug.
			devlog.Warn("content", "skipping %s: %v", path, relErr)
			return nil
		}
		rel = filepath.ToSlash(rel)
		// Use path-based operations on the already-normalized forward-slash rel
		lastSlash := strings.LastIndex(rel, "/")
		var base, dir string
		if lastSlash < 0 {
			base = rel
			dir = ""
		} else {
			base = rel[lastSlash+1:]
			dir = rel[:lastSlash]
		}

		// Determine collection name (first path segment)
		collectionName := ""
		if dir != "" {
			parts := strings.SplitN(dir, "/", 2)
			collectionName = parts[0]
		}

		// Classify node kind
		kind := classifyKind(base, dir, path, dirAssets)

		// Derive slug and weight from filename
		slug, weight := FilenameSlug(base)

		// For _index.md and index.md, slug comes from parent directory
		if base == "_index.md" || base == "index.md" {
			if dir != "" {
				parts := strings.Split(dir, "/")
				dirSlug, dirWeight := FilenameSlug(parts[len(parts)-1] + ".md")
				slug = dirSlug
				if dirWeight > 0 && weight == 0 {
					weight = dirWeight
				}
			} else {
				slug = ""
			}
		}

		cf := ContentFile{
			FilePath:       path,
			RelPath:        rel,
			Kind:           kind,
			CollectionName: collectionName,
			Slug:           slug,
			Weight:         weight,
			IsBundle:       kind == engine.KindBundle,
		}

		if cf.IsBundle {
			cf.BundleAssets = dirAssets[filepath.Dir(path)]
		}

		// i18n: classify language if multi-language is configured
		if len(s.Languages) > 0 {
			ClassifyLang(&cf, s.Languages, s.DefaultLang)
		}

		// Versioning: classify version if any collection has versioning
		if len(s.VersionIDs) > 0 {
			ClassifyVersion(&cf, s.VersionIDs)
		}

		files = append(files, cf)
		return nil
	})

	return files, err
}

// ClassifyNode determines the NodeKind for a given file path.
func ClassifyNode(contentDir, filePath string) engine.NodeKind {
	contentDir = filepath.Clean(contentDir)
	rel, _ := filepath.Rel(contentDir, filePath)
	rel = filepath.ToSlash(rel)
	base := filepath.Base(rel)
	dir := filepath.Dir(rel)
	if dir == "." {
		dir = ""
	}

	// Check for sibling assets (for bundle detection)
	dirAssets := make(map[string][]string)
	absDir := filepath.Dir(filePath)
	entries, _ := os.ReadDir(absDir)
	for _, e := range entries {
		if !e.IsDir() && strings.ToLower(filepath.Ext(e.Name())) != ".md" {
			dirAssets[absDir] = append(dirAssets[absDir], filepath.Join(absDir, e.Name()))
		}
	}

	return classifyKind(base, dir, filePath, dirAssets)
}

// ClassifyFile constructs a ContentFile for a single file path without walking
// the entire content directory. Used by incremental rebuild.
func (s *Scanner) ClassifyFile(contentDir, filePath string) (ContentFile, error) {
	contentDir = filepath.Clean(contentDir)

	rel, err := filepath.Rel(contentDir, filePath)
	if err != nil {
		return ContentFile{}, err
	}
	rel = filepath.ToSlash(rel)

	lastSlash := strings.LastIndex(rel, "/")
	var base, dir string
	if lastSlash < 0 {
		base = rel
		dir = ""
	} else {
		base = rel[lastSlash+1:]
		dir = rel[:lastSlash]
	}

	collectionName := ""
	if dir != "" {
		parts := strings.SplitN(dir, "/", 2)
		collectionName = parts[0]
	}

	// Check for sibling assets (for bundle detection).
	dirAssets := make(map[string][]string)
	absDir := filepath.Dir(filePath)
	entries, _ := os.ReadDir(absDir)
	for _, e := range entries {
		if !e.IsDir() && strings.ToLower(filepath.Ext(e.Name())) != ".md" {
			dirAssets[absDir] = append(dirAssets[absDir], filepath.Join(absDir, e.Name()))
		}
	}

	kind := classifyKind(base, dir, filePath, dirAssets)
	slug, weight := FilenameSlug(base)

	if base == "_index.md" || base == "index.md" {
		if dir != "" {
			parts := strings.Split(dir, "/")
			dirSlug, dirWeight := FilenameSlug(parts[len(parts)-1] + ".md")
			slug = dirSlug
			if dirWeight > 0 && weight == 0 {
				weight = dirWeight
			}
		} else {
			slug = ""
		}
	}

	cf := ContentFile{
		FilePath:       filePath,
		RelPath:        rel,
		Kind:           kind,
		CollectionName: collectionName,
		Slug:           slug,
		Weight:         weight,
		IsBundle:       kind == engine.KindBundle,
	}
	if cf.IsBundle {
		cf.BundleAssets = dirAssets[absDir]
	}
	if len(s.Languages) > 0 {
		ClassifyLang(&cf, s.Languages, s.DefaultLang)
	}
	if len(s.VersionIDs) > 0 {
		ClassifyVersion(&cf, s.VersionIDs)
	}
	return cf, nil
}

func classifyKind(base, dir, absPath string, dirAssets map[string][]string) engine.NodeKind {
	switch {
	case base == "_index.md" && dir == "":
		return engine.KindHome
	case base == "_index.md":
		return engine.KindSection
	case base == "index.md" && dir != "" && len(dirAssets[filepath.Dir(absPath)]) > 0:
		return engine.KindBundle
	case dir == "":
		return engine.KindStandalone
	default:
		return engine.KindPage
	}
}
