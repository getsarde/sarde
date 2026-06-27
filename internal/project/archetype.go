package project

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/getsarde/sarde/internal/content"
	"gopkg.in/yaml.v3"
)

// ScaffoldFile creates a new content file at absPath with archetype/schema-aware frontmatter.
// It is exported for use by the CLI, which operates outside the ProjectManager.
func ScaffoldFile(projectDir, collection, title, absPath string) error {
	fm := scaffoldFrontmatter(projectDir, collection, title)

	var sb strings.Builder
	sb.WriteString("---\n")
	fmBytes, err := yaml.Marshal(fm)
	if err != nil {
		return fmt.Errorf("marshaling frontmatter: %w", err)
	}
	sb.Write(fmBytes)
	sb.WriteString("---\n")

	return os.WriteFile(absPath, []byte(sb.String()), 0o644)
}

// loadArchetype reads archetypes/<collection>.md from the project root and returns
// its parsed frontmatter. Falls back to archetypes/default.md if no collection-specific
// file exists. Returns nil (no error) when no archetype file is found.
func loadArchetype(projectDir, collection string) map[string]any {
	candidates := []string{
		filepath.Join(projectDir, "archetypes", collection+".md"),
		filepath.Join(projectDir, "archetypes", "default.md"),
	}

	p := &content.Parser{}
	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		fm, _, err := p.Parse(data)
		if err != nil || len(fm) == 0 {
			continue
		}
		return fm
	}
	return nil
}
