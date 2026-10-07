package sitetemplate

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestHideRootFiles(t *testing.T) {
	src := fstest.MapFS{
		"README.md":         {Data: []byte("repo readme")},
		"LICENSE":           {Data: []byte("mit")},
		"sarde.yaml":        {Data: []byte("site:\n")},
		"docs/README.md":    {Data: []byte("nested readme stays")},
		"content/LICENSE":   {Data: []byte("nested license stays")},
		"content/_index.md": {Data: []byte("# Home")},
		".github/wf/d.yml":  {Data: []byte("on: push")},
	}
	h := HideRootFiles(src, "README.md", "LICENSE")

	if err := fstest.TestFS(h, "sarde.yaml", "docs/README.md", "content/LICENSE", "content/_index.md", ".github/wf/d.yml"); err != nil {
		t.Fatal(err)
	}
	for _, hidden := range []string{"README.md", "LICENSE"} {
		if _, err := fs.Stat(h, hidden); err == nil {
			t.Errorf("%s should be hidden from Stat", hidden)
		}
	}
	entries, _ := fs.ReadDir(h, ".")
	for _, e := range entries {
		if e.Name() == "README.md" || e.Name() == "LICENSE" {
			t.Errorf("%s listed in root ReadDir", e.Name())
		}
	}

	dir := t.TempDir()
	if err := os.CopyFS(dir, h); err != nil {
		t.Fatal(err)
	}
	for _, missing := range []string{"README.md", "LICENSE"} {
		if _, err := os.Stat(filepath.Join(dir, missing)); err == nil {
			t.Errorf("%s was copied", missing)
		}
	}
	for _, present := range []string{"sarde.yaml", "docs/README.md", "content/LICENSE", ".github/wf/d.yml"} {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(present))); err != nil {
			t.Errorf("%s missing after CopyFS: %v", present, err)
		}
	}
}
