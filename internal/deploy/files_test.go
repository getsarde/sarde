package deploy

import (
	"context"
	"crypto/sha1" //nolint:gosec // test mirrors the content address
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCollectFiles_ExcludesLockAndSorts(t *testing.T) {
	dist := writeDist(t, map[string]string{
		"index.html":      "home",
		"b/c.html":        "c",
		"a/b/c.html":      "nested",
		"sub/.sarde.lock": "a nested lock is ordinary content",
		"assets/app.js":   "js",
	})
	// Different case on a case-insensitive disk is the same lock file.
	files, err := collectFiles(dist, nil)
	if err != nil {
		t.Fatal(err)
	}
	var rels []string
	for _, f := range files {
		rels = append(rels, f.Rel)
		if strings.Contains(f.Rel, `\`) {
			t.Errorf("backslash in %q", f.Rel)
		}
	}
	want := []string{"a/b/c.html", "assets/app.js", "b/c.html", "index.html", "sub/.sarde.lock"}
	if strings.Join(rels, ",") != strings.Join(want, ",") {
		t.Errorf("files = %v, want %v", rels, want)
	}
}

func TestCollectFiles_UppercaseLockSkipped(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".SARDE.LOCK"), []byte("x"), 0o644)
	os.WriteFile(filepath.Join(dir, "index.html"), []byte("x"), 0o644)
	files, err := collectFiles(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Rel != "index.html" {
		t.Errorf("files = %+v, want only index.html", files)
	}
}

func TestCollectFiles_SkipsSymlinks(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "real.html")
	os.WriteFile(target, []byte("x"), 0o644)
	if err := os.Symlink(target, filepath.Join(dir, "link.html")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	rep := &recordingReporter{}
	files, err := collectFiles(dir, rep)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 || files[0].Rel != "real.html" {
		t.Errorf("files = %+v", files)
	}
	if len(rep.logs) == 0 || !strings.Contains(rep.logs[0], "link.html") {
		t.Errorf("expected a warning naming the symlink, got %v", rep.logs)
	}
}

func TestHashAll_SHA1(t *testing.T) {
	dist := writeDist(t, map[string]string{"a.txt": "hello", "b.txt": "world"})
	files, _ := collectFiles(dist, nil)
	rep := &recordingReporter{}
	sums, err := hashAll(context.Background(), files, func(f File) (string, error) { return sha1File(f.Abs) }, rep)
	if err != nil {
		t.Fatal(err)
	}
	for i, content := range []string{"hello", "world"} {
		want := sha1.Sum([]byte(content)) //nolint:gosec
		if sums[i] != hex.EncodeToString(want[:]) {
			t.Errorf("sum %d = %s", i, sums[i])
		}
	}
	if len(rep.progress) != 2 {
		t.Errorf("progress events = %d, want 2", len(rep.progress))
	}
}

func TestPlainDir(t *testing.T) {
	cases := map[string]string{
		`\\?\C:\sites\docs`:            `C:\sites\docs`,
		`\\?\UNC\server\share\docs`:    `\\server\share\docs`,
		`C:\sites\docs`:                `C:\sites\docs`,
		`/home/me/site`:                `/home/me/site`,
		`\\server\share\already-plain`: `\\server\share\already-plain`,
	}
	for in, want := range cases {
		if got := plainDir(in); got != want {
			t.Errorf("plainDir(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestNodeExtname(t *testing.T) {
	cases := map[string]string{
		"index.html":         ".html",
		"a/b/app.min.js":     ".js",
		".nojekyll":          "",
		"sub/.nojekyll":      "",
		".index.md":          ".md",
		"index.":             ".",
		"README":             "",
		"dir.with.dots/file": "",
	}
	for in, want := range cases {
		if got := nodeExtname(in); got != want {
			t.Errorf("nodeExtname(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestContentTypeFor_PinnedWebTypes(t *testing.T) {
	cases := map[string]string{
		"app.js":         "text/javascript; charset=utf-8",
		"a/B.CSS":        "text/css; charset=utf-8",
		"index.html":     "text/html; charset=utf-8",
		"logo.svg":       "image/svg+xml",
		"blob.unknownxy": "application/octet-stream",
	}
	for in, want := range cases {
		if got := contentTypeFor(in); got != want {
			t.Errorf("contentTypeFor(%q) = %q, want %q", in, got, want)
		}
	}
}
