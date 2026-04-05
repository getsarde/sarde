package project

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/coderoo-dev/coderoo/internal/content"
	"gopkg.in/yaml.v3"
)

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

	// Write atomically: temp file then rename.
	tmpPath := absPath + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(sb.String()), 0o644); err != nil {
		return err
	}
	return os.Rename(tmpPath, absPath)
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

// scaffoldFrontmatter creates default frontmatter for a new content file.
func scaffoldFrontmatter(title string) map[string]any {
	return map[string]any{
		"title": title,
		"date":  time.Now().Format(time.RFC3339),
		"draft": true,
	}
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
