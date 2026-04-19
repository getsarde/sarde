package template

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"

	"github.com/coderoo-dev/coderoo/internal/engine"
)

// resolveTemplate finds a template file through the multi-layer lookup chain.
// For default-layout pages, the lookup order is (5 layers):
//  1. {ProjectDir}/layouts/{collection}/{name}
//  2. {ProjectDir}/themes/{theme}/layouts/{collection}/{name}
//  3. {ProjectDir}/layouts/_default/{name}
//  4. {ProjectDir}/themes/{theme}/layouts/_default/{name}
//  5. embedded: _default/{name}
//
// For docs-layout pages, _docs/ layers are inserted between collection and _default (8 layers).
// Returns the template contents, the resolved path (for error messages), and any error.
func resolveTemplate(resolver *engine.ThemeResolver, collection string, layout engine.LayoutType, name string) ([]byte, string, error) {
	candidates := buildTemplateCandidates(resolver, collection, layout, name)

	for _, c := range candidates {
		data, err := c.read()
		if err == nil {
			return data, c.label, nil
		}
	}

	return nil, "", fmt.Errorf("template %q not found for collection=%q layout=%q (tried %d locations)", name, collection, layout, len(candidates))
}

// resolvePartial finds a partial template through the 3-layer lookup chain:
//  1. {ProjectDir}/layouts/partials/{name}
//  2. {ProjectDir}/themes/{theme}/layouts/partials/{name}
//  3. embedded: partials/{name}
func resolvePartial(resolver *engine.ThemeResolver, name string) ([]byte, string, error) {
	candidates := buildPartialCandidates(resolver, name)

	for _, c := range candidates {
		data, err := c.read()
		if err == nil {
			return data, c.label, nil
		}
	}

	return nil, "", fmt.Errorf("partial %q not found (tried %d locations)", name, len(candidates))
}

// resolveAllPartials discovers all unique partial names across all layers.
// User partials override theme partials which override embedded partials.
func resolveAllPartials(resolver *engine.ThemeResolver) map[string][]byte {
	partials := make(map[string][]byte)

	// Load in reverse priority order so higher-priority layers overwrite.
	// 1. Embedded partials
	if resolver.EmbeddedFS != nil {
		loadPartialsFromFS(resolver.EmbeddedFS, "partials", partials)
	}

	// 2. Theme partials
	if resolver.ThemeName != "" {
		themeDir := filepath.Join(resolver.ProjectDir, "themes", resolver.ThemeName, "layouts", "partials")
		loadPartialsFromDir(themeDir, partials)
	}

	// 3. User partials (highest priority)
	userDir := filepath.Join(resolver.ProjectDir, "layouts", "partials")
	loadPartialsFromDir(userDir, partials)

	return partials
}

// candidate represents a single location to check for a template file.
type candidate struct {
	label string        // human-readable path for error messages
	read  func() ([]byte, error) // reads the file
}

// fsCandidate creates a candidate that reads from the OS filesystem.
func fsCandidate(fspath string) candidate {
	return candidate{
		label: fspath,
		read:  func() ([]byte, error) { return os.ReadFile(fspath) },
	}
}

// embeddedCandidate creates a candidate that reads from an embedded FS.
func embeddedCandidate(efs fs.FS, fspath string) candidate {
	return candidate{
		label: "embedded:" + fspath,
		read:  func() ([]byte, error) { return fs.ReadFile(efs, fspath) },
	}
}

// buildTemplateCandidates returns the ordered list of locations to check.
func buildTemplateCandidates(resolver *engine.ThemeResolver, collection string, layout engine.LayoutType, name string) []candidate {
	var candidates []candidate

	projDir := resolver.ProjectDir
	theme := resolver.ThemeName

	// Layer 1-2: Collection-specific (user, then theme)
	if collection != "" {
		candidates = append(candidates, fsCandidate(filepath.Join(projDir, "layouts", collection, name)))
		if theme != "" {
			candidates = append(candidates, fsCandidate(filepath.Join(projDir, "themes", theme, "layouts", collection, name)))
		}
	}

	// Layer 3-4 (docs only): _docs/ specific
	if layout == engine.LayoutDocs {
		candidates = append(candidates, fsCandidate(filepath.Join(projDir, "layouts", "_docs", name)))
		if theme != "" {
			candidates = append(candidates, fsCandidate(filepath.Join(projDir, "themes", theme, "layouts", "_docs", name)))
		}
	}

	// Layer 5-6: _default (user, then theme)
	candidates = append(candidates, fsCandidate(filepath.Join(projDir, "layouts", "_default", name)))
	if theme != "" {
		candidates = append(candidates, fsCandidate(filepath.Join(projDir, "themes", theme, "layouts", "_default", name)))
	}

	// Layer 7-8: Embedded fallback
	if resolver.EmbeddedFS != nil {
		if layout == engine.LayoutDocs {
			candidates = append(candidates, embeddedCandidate(resolver.EmbeddedFS, path.Join("_docs", name)))
		}
		candidates = append(candidates, embeddedCandidate(resolver.EmbeddedFS, path.Join("_default", name)))
	}

	return candidates
}

// buildPartialCandidates returns the ordered list of locations to check for a partial.
func buildPartialCandidates(resolver *engine.ThemeResolver, name string) []candidate {
	var candidates []candidate

	projDir := resolver.ProjectDir
	theme := resolver.ThemeName

	// User partials
	candidates = append(candidates, fsCandidate(filepath.Join(projDir, "layouts", "partials", name)))

	// Theme partials
	if theme != "" {
		candidates = append(candidates, fsCandidate(filepath.Join(projDir, "themes", theme, "layouts", "partials", name)))
	}

	// Embedded partials
	if resolver.EmbeddedFS != nil {
		candidates = append(candidates, embeddedCandidate(resolver.EmbeddedFS, path.Join("partials", name)))
	}

	return candidates
}

// loadPartialsFromFS loads all .html files from a directory in an fs.FS.
func loadPartialsFromFS(efs fs.FS, dir string, out map[string][]byte) {
	entries, err := fs.ReadDir(efs, dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".html" {
			continue
		}
		data, err := fs.ReadFile(efs, path.Join(dir, e.Name()))
		if err == nil {
			out[e.Name()] = data
		}
	}
}

// loadPartialsFromDir loads all .html files from an OS directory.
func loadPartialsFromDir(dir string, out map[string][]byte) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".html" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err == nil {
			out[e.Name()] = data
		}
	}
}
