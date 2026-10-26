package server

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Renaming the static root while `sarde dev` runs must work: on Windows a
// watch handle on a folder inside it blocks the rename of the folder.
// Renaming it back and editing a stylesheet must still reach the dev server.
func TestWatcher_StaticRootCanBeRenamedWhileWatched(t *testing.T) {
	projectDir := t.TempDir()
	css := filepath.Join(projectDir, "public", "css")
	if err := os.MkdirAll(css, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(css, "x.css"), []byte("a{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := make(chan []FileChange, 16)
	w := NewWatcher(projectDir, "", 50*time.Millisecond, func(batch []FileChange) { got <- batch })
	if err := w.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer w.Stop()

	public := filepath.Join(projectDir, "public")
	moved := filepath.Join(projectDir, "public-old")
	if err := os.Rename(public, moved); err != nil {
		t.Fatalf("renaming the watched static root: %v", err)
	}
	time.Sleep(200 * time.Millisecond)
	if err := os.Rename(moved, public); err != nil {
		t.Fatalf("renaming it back: %v", err)
	}
	time.Sleep(200 * time.Millisecond)
	drain(got)

	if err := os.WriteFile(filepath.Join(css, "x.css"), []byte("b{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	deadline := time.After(5 * time.Second)
	for {
		select {
		case batch := <-got:
			for _, c := range batch {
				if filepath.Base(c.Path) == "x.css" && c.Kind == ChangeCSS {
					return
				}
			}
		case <-deadline:
			t.Fatal("no CSS change after the static root came back")
		}
	}
}

func TestWatcher_NewStaticSubfolderIsSeen(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(projectDir, "public"), 0o755); err != nil {
		t.Fatal(err)
	}
	got := make(chan []FileChange, 16)
	w := NewWatcher(projectDir, "", 50*time.Millisecond, func(batch []FileChange) { got <- batch })
	if err := w.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}
	defer w.Stop()

	css := filepath.Join(projectDir, "public", "styles", "deep")
	if err := os.MkdirAll(css, 0o755); err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)
	if err := os.WriteFile(filepath.Join(css, "site.css"), []byte("a{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	deadline := time.After(5 * time.Second)
	for {
		select {
		case batch := <-got:
			for _, c := range batch {
				if filepath.Base(c.Path) == "site.css" && c.Kind == ChangeCSS {
					return
				}
			}
		case <-deadline:
			t.Fatal("no CSS change for a file in a new subfolder of public/")
		}
	}
}

func drain(ch chan []FileChange) {
	for {
		select {
		case <-ch:
		default:
			return
		}
	}
}
