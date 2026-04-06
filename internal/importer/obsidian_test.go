package importer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestConvertNote(t *testing.T) {
	wikilinkMap := map[string]string{
		"getting started":  "getting-started",
		"advanced topics":  "advanced-topics",
	}

	tests := []struct {
		name     string
		input    string
		want     string
		wantLinks int
	}{
		{
			"plain wikilink",
			"See [[Getting Started]] for more.",
			"See [Getting Started](/docs/getting-started) for more.",
			1,
		},
		{
			"aliased wikilink",
			"Read [[Advanced Topics|the advanced guide]].",
			"Read [the advanced guide](/docs/advanced-topics).",
			1,
		},
		{
			"image embed",
			"![[diagram.png]]",
			"![diagram](assets/diagram.png)",
			0,
		},
		{
			"callout note",
			"> [!note] Important info",
			":::note",
			0,
		},
		{
			"callout warning",
			"> [!warning] Be careful",
			":::warning",
			0,
		},
		{
			"comment stripping",
			"Hello %%secret%% world",
			"Hello  world",
			0,
		},
		{
			"dataview stripping",
			"Before\n```dataview\nTABLE file.name\n```\nAfter",
			"Before\n\nAfter",
			0,
		},
		{
			"unknown wikilink falls back to slugify",
			"See [[Some New Page]] here.",
			"See [Some New Page](/docs/some-new-page) here.",
			1,
		},
		{
			"combined transformations",
			"# Title\n%%draft%%\nSee [[Getting Started]] and ![[photo.jpg]]\n> [!tip] Good idea",
			"# Title\n\nSee [Getting Started](/docs/getting-started) and ![photo](assets/photo.jpg)\n:::tip",
			1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, links := convertNote(tt.input, "docs", wikilinkMap)
			if got != tt.want {
				t.Errorf("convertNote() =\n%q\nwant\n%q", got, tt.want)
			}
			if links != tt.wantLinks {
				t.Errorf("links = %d, want %d", links, tt.wantLinks)
			}
		})
	}
}

func TestImportObsidian(t *testing.T) {
	// Create a test vault.
	vaultDir := t.TempDir()
	contentDir := t.TempDir()

	// Write test notes.
	os.WriteFile(filepath.Join(vaultDir, "Introduction.md"), []byte("# Intro\nSee [[Setup]] for details."), 0o644)
	os.WriteFile(filepath.Join(vaultDir, "Setup.md"), []byte("# Setup\n![[logo.png]]\nDone."), 0o644)

	// Write a test image.
	os.WriteFile(filepath.Join(vaultDir, "logo.png"), []byte("fake-png-data"), 0o644)

	// Create a hidden dir that should be skipped.
	os.MkdirAll(filepath.Join(vaultDir, ".obsidian"), 0o755)
	os.WriteFile(filepath.Join(vaultDir, ".obsidian", "config.json"), []byte("{}"), 0o644)

	result, err := ImportObsidian(vaultDir, "docs", contentDir)
	if err != nil {
		t.Fatalf("ImportObsidian() error: %v", err)
	}

	if result.NotesConverted != 2 {
		t.Errorf("NotesConverted = %d, want 2", result.NotesConverted)
	}
	if result.ImagesCopied != 1 {
		t.Errorf("ImagesCopied = %d, want 1", result.ImagesCopied)
	}
	if result.LinksConverted != 1 {
		t.Errorf("LinksConverted = %d, want 1", result.LinksConverted)
	}

	// Check output files exist with numeric prefixes.
	outDir := filepath.Join(contentDir, "docs")
	entries, _ := os.ReadDir(outDir)
	mdCount := 0
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".md") {
			mdCount++
		}
	}
	if mdCount != 2 {
		t.Errorf("output .md files = %d, want 2", mdCount)
	}

	// Check image was copied.
	if _, err := os.Stat(filepath.Join(outDir, "assets", "logo.png")); err != nil {
		t.Error("image not copied to assets/")
	}

	// Check wikilink was converted in Introduction.md.
	intro, _ := os.ReadFile(filepath.Join(outDir, "01-introduction.md"))
	if !strings.Contains(string(intro), "[Setup](/docs/setup)") {
		t.Errorf("wikilink not converted, got: %s", intro)
	}
}

func TestCollectMarkdownFiles_SkipsHidden(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "note.md"), []byte("hello"), 0o644)
	os.MkdirAll(filepath.Join(dir, ".obsidian"), 0o755)
	os.WriteFile(filepath.Join(dir, ".obsidian", "hidden.md"), []byte("hidden"), 0o644)

	files, err := collectMarkdownFiles(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Errorf("got %d files, want 1 (hidden should be skipped)", len(files))
	}
}
