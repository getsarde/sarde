package importer

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/coderoo-dev/coderoo/internal/content"
)

// ImportResult holds statistics from an import operation.
type ImportResult struct {
	NotesConverted int `json:"notes_converted"`
	ImagesCopied   int `json:"images_copied"`
	LinksConverted int `json:"links_converted"`
	ItemsSkipped   int `json:"items_skipped"`
}

// Supported image extensions for copying.
var imageExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true,
	".gif": true, ".svg": true, ".webp": true,
}

// Regex patterns for Obsidian syntax conversion.
var (
	reComment          = regexp.MustCompile(`%%[^%]*%%`)
	reDataview         = regexp.MustCompile("(?s)```dataview.*?```")
	reImageEmbed       = regexp.MustCompile(`!\[\[([^\]]+\.(?:png|jpg|jpeg|gif|svg|webp))\]\]`)
	reWikilinkAliased  = regexp.MustCompile(`\[\[([^\]|]+)\|([^\]]+)\]\]`)
	reWikilink         = regexp.MustCompile(`\[\[([^\]|]+)\]\]`)
	reCallout          = regexp.MustCompile(`(?m)^> \[!(note|tip|warning|danger|info|caution|important)\].*$`)
)

// ImportObsidian converts an Obsidian vault into a collection of markdown files.
// Files are written to contentDir/collection/ with numeric prefixes.
// Images are copied to contentDir/collection/assets/.
func ImportObsidian(vaultPath, collection, contentDir string) (*ImportResult, error) {
	result := &ImportResult{}

	// Collect markdown files from vault.
	mdFiles, err := collectMarkdownFiles(vaultPath)
	if err != nil {
		return nil, fmt.Errorf("scanning vault: %w", err)
	}
	if len(mdFiles) == 0 {
		return result, nil
	}

	// Build wikilink map: lowercase name → slug.
	wikilinkMap := buildWikilinkMap(mdFiles)

	// Sort for deterministic numbering.
	sort.Strings(mdFiles)

	// Create output directory.
	outDir := filepath.Join(contentDir, collection)
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating output dir: %w", err)
	}

	// Process each note.
	for i, mdFile := range mdFiles {
		raw, err := os.ReadFile(mdFile)
		if err != nil {
			result.ItemsSkipped++
			continue
		}

		converted, links := convertNote(string(raw), collection, wikilinkMap)
		result.LinksConverted += links

		// Build output filename with numeric prefix.
		base := strings.TrimSuffix(filepath.Base(mdFile), ".md")
		slug := content.Slugify(base)
		outName := fmt.Sprintf("%02d-%s.md", i+1, slug)
		outPath := filepath.Join(outDir, outName)

		if err := os.WriteFile(outPath, []byte(converted), 0o644); err != nil {
			result.ItemsSkipped++
			continue
		}
		result.NotesConverted++
	}

	// Copy image files.
	copied, skipped := copyImages(vaultPath, filepath.Join(outDir, "assets"))
	result.ImagesCopied = copied
	result.ItemsSkipped += skipped

	return result, nil
}

// collectMarkdownFiles walks the vault and returns paths to all .md files,
// skipping hidden directories (starting with .).
func collectMarkdownFiles(root string) ([]string, error) {
	var files []string
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() && strings.HasPrefix(info.Name(), ".") {
			return filepath.SkipDir
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".md") {
			files = append(files, path)
		}
		return nil
	})
	return files, err
}

// buildWikilinkMap creates a lookup from lowercase filename (no ext) to slugified name.
func buildWikilinkMap(files []string) map[string]string {
	m := make(map[string]string)
	for _, f := range files {
		name := strings.TrimSuffix(filepath.Base(f), ".md")
		key := strings.ToLower(strings.TrimSpace(name))
		m[key] = content.Slugify(name)
	}
	return m
}

// convertNote applies all Obsidian → standard markdown transformations.
// Returns the converted text and the number of links converted.
func convertNote(text, collection string, wikilinkMap map[string]string) (string, int) {
	links := 0

	// 1. Strip comments.
	text = reComment.ReplaceAllString(text, "")

	// 2. Strip dataview blocks.
	text = reDataview.ReplaceAllString(text, "")

	// 3. Convert image embeds.
	text = reImageEmbed.ReplaceAllStringFunc(text, func(match string) string {
		m := reImageEmbed.FindStringSubmatch(match)
		filename := m[1]
		alt := strings.TrimSuffix(filename, filepath.Ext(filename))
		return fmt.Sprintf("![%s](assets/%s)", alt, filename)
	})

	// 4. Convert aliased wikilinks.
	text = reWikilinkAliased.ReplaceAllStringFunc(text, func(match string) string {
		m := reWikilinkAliased.FindStringSubmatch(match)
		page := m[1]
		display := m[2]
		slug := resolveWikilink(page, wikilinkMap)
		links++
		return fmt.Sprintf("[%s](/%s/%s)", display, collection, slug)
	})

	// 5. Convert plain wikilinks.
	text = reWikilink.ReplaceAllStringFunc(text, func(match string) string {
		m := reWikilink.FindStringSubmatch(match)
		page := m[1]
		slug := resolveWikilink(page, wikilinkMap)
		links++
		return fmt.Sprintf("[%s](/%s/%s)", page, collection, slug)
	})

	// 6. Convert callouts.
	text = reCallout.ReplaceAllStringFunc(text, func(match string) string {
		m := reCallout.FindStringSubmatch(match)
		return ":::" + m[1]
	})

	return text, links
}

// resolveWikilink looks up a page name in the wikilink map, falling back to slugification.
func resolveWikilink(page string, wikilinkMap map[string]string) string {
	key := strings.ToLower(strings.TrimSpace(page))
	if slug, ok := wikilinkMap[key]; ok {
		return slug
	}
	return content.Slugify(page)
}

// copyImages walks the vault and copies image files to the assets directory.
func copyImages(vaultPath, assetsDir string) (copied, skipped int) {
	filepath.Walk(vaultPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if strings.HasPrefix(info.Name(), ".") {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(info.Name()))
		if !imageExts[ext] {
			return nil
		}

		if err := os.MkdirAll(assetsDir, 0o755); err != nil {
			skipped++
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			skipped++
			return nil
		}

		dst := filepath.Join(assetsDir, info.Name())
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			skipped++
			return nil
		}
		copied++
		return nil
	})
	return
}
