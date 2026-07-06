package importer

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/getsarde/sarde/internal/content"
)

// ImportResult holds statistics from an import operation.
type ImportResult struct {
	NotesConverted int      `json:"notes_converted"`
	ImagesCopied   int      `json:"images_copied"`
	LinksConverted int      `json:"links_converted"`
	ItemsSkipped   int      `json:"items_skipped"`
	Warnings       []string `json:"warnings,omitempty"`
}

// Supported image extensions for copying.
var imageExts = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true,
	".gif": true, ".svg": true, ".webp": true,
}

// Regex patterns for Obsidian syntax conversion.
var (
	reComment         = regexp.MustCompile(`%%[^%]*%%`)
	reDataview        = regexp.MustCompile("(?s)```dataview.*?```")
	reImageEmbed      = regexp.MustCompile(`!\[\[([^\]|]+\.(?i:png|jpg|jpeg|gif|svg|webp))(?:\|[^\]]*)?\]\]`)
	reWikilinkAliased = regexp.MustCompile(`\[\[([^\]|]+)\|([^\]]+)\]\]`)
	reWikilink        = regexp.MustCompile(`\[\[([^\]|]+)\]\]`)
	// reCalloutHeader matches an Obsidian callout header line and captures the
	// type and optional title. The optional +/- is Obsidian's foldable marker.
	reCalloutHeader = regexp.MustCompile(`^>\s*\[!((?i:note|tip|warning|danger|info|caution|important))\][+-]?\s*(.*)$`)
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
	copied, skipped, warnings := copyImages(vaultPath, filepath.Join(outDir, "assets"))
	result.ImagesCopied = copied
	result.ItemsSkipped += skipped
	result.Warnings = warnings

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

	// 3. Convert image embeds. copyImages flattens the vault's folder layout
	// into assets/, so subfolder embeds like ![[attachments/x.png]] must be
	// referenced by basename. A |size suffix (![[x.png|300]]) is dropped.
	text = reImageEmbed.ReplaceAllStringFunc(text, func(match string) string {
		m := reImageEmbed.FindStringSubmatch(match)
		filename := path.Base(filepath.ToSlash(m[1]))
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
	text = convertCallouts(text)

	return text, links
}

// convertCallouts rewrites Obsidian blockquote callouts as aside blocks.
// Sarde's aside parser requires an explicit closing ":::" fence and does not
// understand blockquote continuation prefixes, so this must be a line pass:
// the header opens a fence, following "> " lines are unquoted into the body,
// and the first non-blockquote line closes the fence. Without the closing
// fence the aside block would swallow the rest of the document.
func convertCallouts(text string) string {
	lines := strings.Split(text, "\n")
	out := make([]string, 0, len(lines))
	inCallout := false

	for _, line := range lines {
		if m := reCalloutHeader.FindStringSubmatch(line); m != nil {
			if inCallout {
				out = append(out, ":::", "")
			}
			fence := ":::" + strings.ToLower(m[1])
			if title := strings.TrimSpace(m[2]); title != "" {
				fence += "[" + title + "]"
			}
			out = append(out, fence)
			inCallout = true
			continue
		}
		if inCallout {
			if strings.HasPrefix(line, ">") {
				body := strings.TrimPrefix(line, ">")
				body = strings.TrimPrefix(body, " ")
				out = append(out, body)
				continue
			}
			out = append(out, ":::")
			inCallout = false
		}
		out = append(out, line)
	}
	if inCallout {
		out = append(out, ":::")
	}
	return strings.Join(out, "\n")
}

// resolveWikilink looks up a page name in the wikilink map, falling back to slugification.
func resolveWikilink(page string, wikilinkMap map[string]string) string {
	key := strings.ToLower(strings.TrimSpace(page))
	if slug, ok := wikilinkMap[key]; ok {
		return slug
	}
	return content.Slugify(page)
}

// copiedImage tracks the first image written under a given basename so later
// collisions can be compared by content and attributed in warnings.
type copiedImage struct {
	hash   [sha256.Size]byte
	source string // vault-relative path of the first copy
}

// copyImages walks the vault and copies image files to the flat assets
// directory (markdown embeds reference images by bare filename, so the layout
// must stay flat). Duplicate basenames with identical content are copied once;
// differing content is written under a deduped "<stem>-<n><ext>" name with a
// warning, since markdown references to that basename resolve to the first copy.
func copyImages(vaultPath, assetsDir string) (copied, skipped int, warnings []string) {
	// Keys are lowercased: the output filesystem may be case-insensitive
	// (Windows/macOS), where "Logo.png" and "logo.png" would silently collide.
	seen := make(map[string]copiedImage)

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

		relSource := path
		if rel, relErr := filepath.Rel(vaultPath, path); relErr == nil {
			relSource = filepath.ToSlash(rel)
		}
		hash := sha256.Sum256(data)
		name := info.Name()
		key := strings.ToLower(name)

		if prev, ok := seen[key]; ok {
			if prev.hash == hash {
				// Same content already copied once; not a new image.
				return nil
			}
			// Different content under the same name: keep the first copy
			// (walk order is deterministic), write this one deduped.
			stem := strings.TrimSuffix(name, filepath.Ext(name))
			dedupedName := ""
			for n := 2; ; n++ {
				candidate := fmt.Sprintf("%s-%d%s", stem, n, filepath.Ext(name))
				if _, taken := seen[strings.ToLower(candidate)]; !taken {
					dedupedName = candidate
					break
				}
			}
			warnings = append(warnings, fmt.Sprintf(
				"image name collision: %q (from %s) differs from the copy already imported from %s; saved as %q — markdown references to %q resolve to the first copy",
				name, relSource, prev.source, dedupedName, name))
			name = dedupedName
			key = strings.ToLower(name)
		}

		dst := filepath.Join(assetsDir, name)
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			skipped++
			return nil
		}
		seen[key] = copiedImage{hash: hash, source: relSource}
		copied++
		return nil
	})
	return
}
