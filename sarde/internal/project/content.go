package project

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/frostybee/sarde/internal/consts"
	"github.com/frostybee/sarde/internal/content"
	"github.com/frostybee/sarde/internal/editor"
	"gopkg.in/yaml.v3"
)

// revisionRetention caps the number of snapshots kept per file.
const revisionRetention = 20

// contentDirForCollection returns the absolute path to a collection's content directory.
func contentDirForCollection(projectDir, collection string) string {
	return filepath.Join(projectDir, consts.DirContent, collection)
}

var parser = &content.Parser{}

// readContentFile reads a content file and returns parsed frontmatter and body.
func readContentFile(contentDir, relPath string) (map[string]any, string, error) {
	absPath := filepath.Join(contentDir, filepath.FromSlash(relPath))
	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, "", err
	}
	return parser.Parse(data)
}

// writeContentFile serializes frontmatter as YAML and writes the content file atomically.
func writeContentFile(contentDir, relPath string, fm map[string]any, body string) error {
	absPath := filepath.Join(contentDir, filepath.FromSlash(relPath))

	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return err
	}

	var sb strings.Builder
	if len(fm) > 0 {
		sb.WriteString("---\n")
		fmBytes, err := yaml.Marshal(fm)
		if err != nil {
			return fmt.Errorf("marshaling frontmatter: %w", err)
		}
		sb.Write(fmBytes)
		sb.WriteString("---\n")
	}
	sb.WriteString(body)

	// Atomic write + .bak backup + revision snapshot for existing files.
	if err := editor.SafeWrite(absPath, []byte(sb.String()), editor.SafeWriteOptions{
		Backup:   true,
		Revision: true,
	}); err != nil {
		return err
	}
	// Best-effort cap on revision growth.
	_ = editor.PruneRevisions(absPath, revisionRetention)
	return nil
}

// validateContentPath ensures the path is within the content directory and does not traverse.
func validateContentPath(contentDir, relPath string) error {
	if relPath == "" {
		return fmt.Errorf("empty path")
	}

	// Prevent directory traversal.
	cleaned := filepath.Clean(filepath.FromSlash(relPath))
	if strings.Contains(cleaned, "..") {
		return fmt.Errorf("path traversal not allowed: %s", relPath)
	}

	// Ensure the resolved path is within the content directory.
	absPath := filepath.Join(contentDir, cleaned)
	absContent, _ := filepath.Abs(contentDir)
	absTarget, _ := filepath.Abs(absPath)
	if !strings.HasPrefix(absTarget, absContent+string(filepath.Separator)) && absTarget != absContent {
		return fmt.Errorf("path outside content directory: %s", relPath)
	}

	return nil
}

var nonAlnumHyphen = regexp.MustCompile(`[^a-z0-9-]+`)
var multiHyphen = regexp.MustCompile(`-{2,}`)

// slugify generates a URL-safe slug from a title.
func slugify(title string) string {
	s := strings.ToLower(title)
	s = strings.ReplaceAll(s, " ", "-")
	s = nonAlnumHyphen.ReplaceAllString(s, "")
	s = multiHyphen.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	return s
}

// scaffoldFrontmatter creates frontmatter for a new content file by merging:
//  1. Built-in base fields (draft: true)
//  2. Schema defaults from content/<collection>/config.yaml
//  3. Archetype file from archetypes/<collection>.md (or archetypes/default.md)
//  4. User-supplied title and current date (always applied last)
//
// projectDir and collection may be empty; the function degrades gracefully.
func scaffoldFrontmatter(projectDir, collection, title string) map[string]any {
	fm := map[string]any{
		"draft": true,
	}

	// Layer 2: schema defaults from content/<collection>/config.yaml
	if projectDir != "" && collection != "" {
		colDir := contentDirForCollection(projectDir, collection)
		if schema, err := content.LoadSchema(colDir); err == nil && schema != nil {
			fm = content.ApplyDefaults(fm, schema)
		}
	}

	// Layer 3: archetype file fields
	if projectDir != "" && collection != "" {
		if arch := loadArchetype(projectDir, collection); arch != nil {
			for k, v := range arch {
				fm[k] = v
			}
		}
	}

	// Layer 4: always override with user-supplied title and current date
	fm["title"] = title
	if _, hasDate := fm["date"]; !hasDate {
		fm["date"] = time.Now().Format(time.RFC3339)
	} else {
		// Archetype date placeholder (empty string) → replace with real date
		if d, ok := fm["date"].(string); ok && d == "" {
			fm["date"] = time.Now().Format(time.RFC3339)
		}
	}

	return fm
}

// extractSummary extracts metadata from parsed frontmatter for ContentFile/ContentSummary.
func extractSummary(fm map[string]any, body string) (title string, draft bool, date time.Time, weight int, wordCount int, readingTime int) {
	if t, ok := fm["title"].(string); ok {
		title = t
	}
	if d, ok := fm["draft"].(bool); ok {
		draft = d
	}
	if d, ok := fm["date"].(string); ok {
		if parsed, err := time.Parse(time.RFC3339, d); err == nil {
			date = parsed
		}
	}
	if w, ok := fm["weight"].(int); ok {
		weight = w
	}

	wordCount = len(strings.Fields(body))
	if wordCount > 0 {
		readingTime = int(math.Ceil(float64(wordCount) / 200.0))
		if readingTime < 1 {
			readingTime = 1
		}
	}
	return
}
