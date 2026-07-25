package collection

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/content"
)

func TestBuildCollectionsWithOptions_ParallelMatchesSerialOrdering(t *testing.T) {
	dir := t.TempDir()
	writeCollectionFixture(t, dir, "content/blog/config.yaml", "frontmatter_schema:\n  fields:\n    required_field:\n      type: string\n      required: true\n")
	writeCollectionFixture(t, dir, "content/blog/_index.md", "---\ntitle: Blog\nrequired_field: ok\n---\n")
	for i := 1; i <= 20; i++ {
		name := fmt.Sprintf("%02d-page.md", i)
		writeCollectionFixture(t, dir, "content/blog/"+name, "---\ntitle: "+name+"\n---\n# "+name+"\n")
	}

	contentDir := filepath.Join(dir, "content")
	scanner := &content.Scanner{}
	files, err := scanner.DiscoverFiles(contentDir)
	if err != nil {
		t.Fatalf("DiscoverFiles failed: %v", err)
	}
	cfg := config.Defaults()

	serialCollections, serialWarnings, err := BuildCollectionsWithOptions(files, cfg, contentDir, nil, BuildOptions{Parallel: false})
	if err != nil {
		t.Fatalf("serial BuildCollectionsWithOptions failed: %v", err)
	}
	parallelCollections, parallelWarnings, err := BuildCollectionsWithOptions(files, cfg, contentDir, nil, BuildOptions{Parallel: true, WorkerCount: 2})
	if err != nil {
		t.Fatalf("parallel BuildCollectionsWithOptions failed: %v", err)
	}

	serialPages := serialCollections["blog"].Pages
	parallelPages := parallelCollections["blog"].Pages
	if len(serialPages) != len(parallelPages) {
		t.Fatalf("page count mismatch: serial=%d parallel=%d", len(serialPages), len(parallelPages))
	}
	for i := range serialPages {
		if serialPages[i].RelPermalink != parallelPages[i].RelPermalink {
			t.Fatalf("page %d permalink mismatch: serial=%s parallel=%s", i, serialPages[i].RelPermalink, parallelPages[i].RelPermalink)
		}
	}
	if len(serialWarnings) != len(parallelWarnings) {
		t.Fatalf("warning count mismatch: serial=%d parallel=%d", len(serialWarnings), len(parallelWarnings))
	}
	for i := range serialWarnings {
		if serialWarnings[i].File != parallelWarnings[i].File || serialWarnings[i].Field != parallelWarnings[i].Field {
			t.Fatalf("warning %d mismatch: serial=%+v parallel=%+v", i, serialWarnings[i], parallelWarnings[i])
		}
	}
}

func writeCollectionFixture(t *testing.T, base, rel, body string) {
	t.Helper()
	writePath := filepath.Join(base, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(writePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(writePath, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
