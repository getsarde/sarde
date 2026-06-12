package editor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSafeWrite_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "sub", "a.md")
	if err := SafeWrite(p, []byte("hello"), SafeWriteOptions{}); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(p)
	if string(got) != "hello" {
		t.Errorf("got %q", got)
	}
}

func TestSafeWrite_Backup(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.md")
	os.WriteFile(p, []byte("v1"), 0o644)

	if err := SafeWrite(p, []byte("v2"), SafeWriteOptions{Backup: true}); err != nil {
		t.Fatal(err)
	}

	bak, err := os.ReadFile(p + ".bak")
	if err != nil {
		t.Fatalf(".bak not created: %v", err)
	}
	if string(bak) != "v1" {
		t.Errorf(".bak = %q, want v1", bak)
	}
	cur, _ := os.ReadFile(p)
	if string(cur) != "v2" {
		t.Errorf("current = %q, want v2", cur)
	}
}

func TestSafeWrite_Revision(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.md")
	os.WriteFile(p, []byte("v1"), 0o644)

	if err := SafeWrite(p, []byte("v2"), SafeWriteOptions{Revision: true}); err != nil {
		t.Fatal(err)
	}

	revs := ListRevisions(p)
	if len(revs) != 1 {
		t.Fatalf("want 1 revision, got %d", len(revs))
	}
	got, _ := os.ReadFile(revs[0].Path)
	if string(got) != "v1" {
		t.Errorf("revision body = %q, want v1", got)
	}
}

func TestValidateWithinRoot(t *testing.T) {
	root := t.TempDir()
	if err := ValidateWithinRoot(root, "blog/post.md"); err != nil {
		t.Errorf("valid path rejected: %v", err)
	}
	if err := ValidateWithinRoot(root, "../etc/passwd"); err == nil {
		t.Error("traversal accepted")
	}
	if err := ValidateWithinRoot(root, "blog/../../etc"); err == nil {
		t.Error("nested traversal accepted")
	}
	if err := ValidateWithinRoot(root, ""); err == nil {
		t.Error("empty path accepted")
	}
}

func TestRestoreRevision(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.md")

	os.WriteFile(p, []byte("v1"), 0o644)
	SafeWrite(p, []byte("v2"), SafeWriteOptions{Revision: true})

	revs := ListRevisions(p)
	if len(revs) != 1 {
		t.Fatalf("want 1 revision, got %d", len(revs))
	}
	if err := RestoreRevision(p, revs[0].Path); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(p)
	if string(got) != "v1" {
		t.Errorf("restored = %q, want v1", got)
	}
	// After restore, we should have two revisions: the original v1 snapshot
	// plus the v2 snapshot taken during restore.
	if len(ListRevisions(p)) != 2 {
		t.Errorf("want 2 revisions after restore, got %d", len(ListRevisions(p)))
	}
}

func TestPruneRevisions(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.md")
	os.WriteFile(p, []byte("v"), 0o644)

	for i := 0; i < 5; i++ {
		CreateRevision(p)
	}
	if err := PruneRevisions(p, 2); err != nil {
		t.Fatal(err)
	}
	if got := len(ListRevisions(p)); got != 2 {
		t.Errorf("got %d revisions, want 2", got)
	}
}

// Regression: for extensionless files the suffix check is vacuous, so decoy
// files like "README.123.backup" used to be listed as revisions (Sscanf
// accepted the "123" prefix of the middle segment).
func TestListRevisions_ExtensionlessIgnoresDecoys(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "README")
	if err := os.WriteFile(path, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CreateRevision(path); err != nil {
		t.Fatalf("CreateRevision: %v", err)
	}

	revDir := filepath.Join(dir, ".revisions")
	os.WriteFile(filepath.Join(revDir, "README.123.backup"), []byte("decoy"), 0o644)
	os.WriteFile(filepath.Join(revDir, "README.notanumber"), []byte("decoy"), 0o644)

	revs := ListRevisions(path)
	if len(revs) != 1 {
		var names []string
		for _, r := range revs {
			names = append(names, filepath.Base(r.Path))
		}
		t.Errorf("ListRevisions = %d entries %v, want exactly the genuine revision", len(revs), names)
	}
}

func TestListRevisions_RejectsNonNumericMiddle(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "doc.md")
	if err := os.WriteFile(path, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CreateRevision(path); err != nil {
		t.Fatalf("CreateRevision: %v", err)
	}

	revDir := filepath.Join(dir, ".revisions")
	os.WriteFile(filepath.Join(revDir, "doc.123abc.md"), []byte("decoy"), 0o644)

	revs := ListRevisions(path)
	if len(revs) != 1 {
		t.Errorf("ListRevisions = %d entries, want 1 (decoy with non-numeric middle excluded)", len(revs))
	}
}
