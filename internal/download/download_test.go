package download

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestInferSourceKind(t *testing.T) {
	tests := []struct {
		src  string
		want SourceKind
	}{
		{"github.com/user/theme", SourceGitHub},
		{"https://github.com/user/theme", SourceGitHub},
		{"https://github.com/user/theme/tree/main/sub", SourceGitHub},
		{"https://example.com/theme.zip", SourceZipURL},
		{"https://example.com/theme.tar.gz", SourceTarGzURL},
		{"https://example.com/theme.tgz", SourceTarGzURL},
		{"https://example.com/unknown", SourceUnknown},
		{"nonexistent-path", SourceUnknown},
	}

	for _, tt := range tests {
		got := InferSourceKind(tt.src)
		if got != tt.want {
			t.Errorf("InferSourceKind(%q) = %d, want %d", tt.src, got, tt.want)
		}
	}
}

func TestInferSourceKind_LocalDir(t *testing.T) {
	dir := t.TempDir()
	if got := InferSourceKind(dir); got != SourceLocalDir {
		t.Errorf("InferSourceKind(tempdir) = %d, want SourceLocalDir", got)
	}
}

func TestParseGitHubURL(t *testing.T) {
	tests := []struct {
		input   string
		owner   string
		repo    string
		branch  string
		subpath string
	}{
		{"github.com/user/theme", "user", "theme", "main", ""},
		{"https://github.com/user/theme", "user", "theme", "main", ""},
		{"https://github.com/user/theme.git", "user", "theme", "main", ""},
		{"https://github.com/user/repo/tree/develop", "user", "repo", "develop", ""},
		{"https://github.com/user/mono/tree/main/themes/dark", "user", "mono", "main", "themes/dark"},
	}

	for _, tt := range tests {
		ref, err := ParseGitHubURL(tt.input)
		if err != nil {
			t.Errorf("ParseGitHubURL(%q) error: %v", tt.input, err)
			continue
		}
		if ref.Owner != tt.owner || ref.Repo != tt.repo || ref.Branch != tt.branch || ref.Subpath != tt.subpath {
			t.Errorf("ParseGitHubURL(%q) = {%s, %s, %s, %s}, want {%s, %s, %s, %s}",
				tt.input, ref.Owner, ref.Repo, ref.Branch, ref.Subpath,
				tt.owner, tt.repo, tt.branch, tt.subpath)
		}
	}
}

func TestParseGitHubURL_Invalid(t *testing.T) {
	invalid := []string{
		"example.com/user/theme",
		"github.com/",
		"github.com/user",
		"",
	}
	for _, s := range invalid {
		if _, err := ParseGitHubURL(s); err == nil {
			t.Errorf("ParseGitHubURL(%q) should have returned error", s)
		}
	}
}

func TestGitHubArchiveURL(t *testing.T) {
	ref := &GitHubRef{Owner: "acme", Repo: "dark-theme", Branch: "main"}
	want := "https://github.com/acme/dark-theme/archive/refs/heads/main.zip"
	if got := ref.ArchiveURL(); got != want {
		t.Errorf("ArchiveURL() = %q, want %q", got, want)
	}
}

// TestDownloadFile_InsecureHTTP_StillAttemptsDownload verifies that
// DownloadFile does not block or panic on plain http:// URLs — it only
// warns (via devlog.Warn, which writes to stderr and isn't captured here).
// Using a URL that can't be reached confirms the warning path executes and
// then falls through to the normal request/error handling unchanged.
func TestDownloadFile_InsecureHTTP_StillAttemptsDownload(t *testing.T) {
	_, err := DownloadFile("http://127.0.0.1:0/nonexistent-theme.zip")
	if err == nil {
		t.Fatal("expected an error connecting to an unreachable http:// URL")
	}
}

// TestDownloadFile_HTTPS_NoWarningPath verifies the https:// case takes the
// non-warning branch without panicking, by exercising an unreachable https
// URL and confirming it still returns a (connection) error as normal.
func TestDownloadFile_HTTPS_NoWarningPath(t *testing.T) {
	_, err := DownloadFile("https://127.0.0.1:0/nonexistent-theme.zip")
	if err == nil {
		t.Fatal("expected an error connecting to an unreachable https:// URL")
	}
}

func TestExtractZip(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "test.zip")

	createTestZip(t, zipPath, map[string]string{
		"theme.yaml":       "name: test",
		"css/tokens.css":   "body {}",
		"layouts/base.html": "<html>",
	})

	destDir := filepath.Join(dir, "out")
	if err := ExtractZip(zipPath, destDir, 0); err != nil {
		t.Fatalf("ExtractZip: %v", err)
	}

	for _, rel := range []string{"theme.yaml", "css/tokens.css", "layouts/base.html"} {
		if _, err := os.Stat(filepath.Join(destDir, filepath.FromSlash(rel))); err != nil {
			t.Errorf("expected %s to exist: %v", rel, err)
		}
	}
}

func TestExtractZip_StripComponents(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "test.zip")

	createTestZip(t, zipPath, map[string]string{
		"repo-main/theme.yaml":     "name: test",
		"repo-main/css/tokens.css": "body {}",
	})

	destDir := filepath.Join(dir, "out")
	if err := ExtractZip(zipPath, destDir, 1); err != nil {
		t.Fatalf("ExtractZip strip=1: %v", err)
	}

	if _, err := os.Stat(filepath.Join(destDir, "theme.yaml")); err != nil {
		t.Error("theme.yaml not found after stripping 1 component")
	}
	if _, err := os.Stat(filepath.Join(destDir, "css", "tokens.css")); err != nil {
		t.Error("css/tokens.css not found after stripping 1 component")
	}
}

func TestExtractTarGz(t *testing.T) {
	dir := t.TempDir()
	tarPath := filepath.Join(dir, "test.tar.gz")

	createTestTarGz(t, tarPath, map[string]string{
		"theme.yaml": "name: test",
		"css/a.css":  "body {}",
	})

	destDir := filepath.Join(dir, "out")
	if err := ExtractTarGz(tarPath, destDir, 0); err != nil {
		t.Fatalf("ExtractTarGz: %v", err)
	}

	if _, err := os.Stat(filepath.Join(destDir, "theme.yaml")); err != nil {
		t.Error("theme.yaml not found")
	}
}

func TestExtractTarGz_StripComponents(t *testing.T) {
	dir := t.TempDir()
	tarPath := filepath.Join(dir, "test.tar.gz")

	createTestTarGz(t, tarPath, map[string]string{
		"repo-main/theme.yaml": "name: stripped",
	})

	destDir := filepath.Join(dir, "out")
	if err := ExtractTarGz(tarPath, destDir, 1); err != nil {
		t.Fatalf("ExtractTarGz strip=1: %v", err)
	}

	if _, err := os.Stat(filepath.Join(destDir, "theme.yaml")); err != nil {
		t.Error("theme.yaml not found after stripping")
	}
}

func TestCopyDir(t *testing.T) {
	src := t.TempDir()
	os.MkdirAll(filepath.Join(src, "sub"), 0o755)
	os.WriteFile(filepath.Join(src, "a.txt"), []byte("hello"), 0o644)
	os.WriteFile(filepath.Join(src, "sub", "b.txt"), []byte("world"), 0o644)

	dest := filepath.Join(t.TempDir(), "copy")
	if err := CopyDir(src, dest); err != nil {
		t.Fatalf("CopyDir: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dest, "a.txt"))
	if err != nil || string(data) != "hello" {
		t.Errorf("a.txt: got %q, err %v", data, err)
	}
	data, err = os.ReadFile(filepath.Join(dest, "sub", "b.txt"))
	if err != nil || string(data) != "world" {
		t.Errorf("sub/b.txt: got %q, err %v", data, err)
	}
}

func TestStripLeading(t *testing.T) {
	tests := []struct {
		name string
		n    int
		want string
	}{
		{"repo-main/theme.yaml", 1, "theme.yaml"},
		{"repo-main/css/a.css", 1, "css/a.css"},
		{"a/b/c.txt", 2, "c.txt"},
		{"single", 1, ""},
		{"a/b", 0, "a/b"},
	}
	for _, tt := range tests {
		got := stripLeading(tt.name, tt.n)
		if got != tt.want {
			t.Errorf("stripLeading(%q, %d) = %q, want %q", tt.name, tt.n, got, tt.want)
		}
	}
}

func TestIsInsideDir(t *testing.T) {
	base := filepath.Join(os.TempDir(), "base")
	if isInsideDir(base, filepath.Join(base, "..", "escape")) {
		t.Error("should reject path outside base")
	}
	if !isInsideDir(base, filepath.Join(base, "sub", "file.txt")) {
		t.Error("should accept path inside base")
	}
}

// Archive entries with recorded mode 0 (e.g. Windows-created zips, GitHub
// archives) must extract as readable files, not write-only 0o200.
func TestExtractZip_ZeroModeEntriesAreReadable(t *testing.T) {
	dir := t.TempDir()
	zipPath := filepath.Join(dir, "zero.zip")

	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	w := zip.NewWriter(f)
	// Unix creator version with all permission bits zero — Mode() returns 0.
	hdr := &zip.FileHeader{Name: "theme.yaml", Method: zip.Deflate}
	hdr.CreatorVersion = 3 << 8
	fw, err := w.CreateHeader(hdr)
	if err != nil {
		t.Fatal(err)
	}
	fw.Write([]byte("name: zero"))
	w.Close()
	f.Close()

	destDir := filepath.Join(dir, "out")
	if err := ExtractZip(zipPath, destDir, 0); err != nil {
		t.Fatalf("ExtractZip: %v", err)
	}

	extracted := filepath.Join(destDir, "theme.yaml")
	data, err := os.ReadFile(extracted)
	if err != nil {
		t.Fatalf("extracted file not readable: %v", err)
	}
	if string(data) != "name: zero" {
		t.Errorf("content = %q, want %q", data, "name: zero")
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(extracted)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode()&0o444 == 0 {
			t.Errorf("mode %v lacks read permission", info.Mode())
		}
	}
}

func TestExtractTarGz_ZeroModeEntriesAreReadable(t *testing.T) {
	dir := t.TempDir()
	tarPath := filepath.Join(dir, "zero.tar.gz")

	f, err := os.Create(tarPath)
	if err != nil {
		t.Fatal(err)
	}
	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)
	content := "name: zero"
	tw.WriteHeader(&tar.Header{Name: "theme.yaml", Size: int64(len(content)), Mode: 0})
	tw.Write([]byte(content))
	tw.Close()
	gw.Close()
	f.Close()

	destDir := filepath.Join(dir, "out")
	if err := ExtractTarGz(tarPath, destDir, 0); err != nil {
		t.Fatalf("ExtractTarGz: %v", err)
	}

	extracted := filepath.Join(destDir, "theme.yaml")
	data, err := os.ReadFile(extracted)
	if err != nil {
		t.Fatalf("extracted file not readable: %v", err)
	}
	if string(data) != content {
		t.Errorf("content = %q, want %q", data, content)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(extracted)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode()&0o444 == 0 {
			t.Errorf("mode %v lacks read permission", info.Mode())
		}
	}
}

// --- helpers ---

func createTestZip(t *testing.T, path string, files map[string]string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	w := zip.NewWriter(f)
	for name, content := range files {
		fw, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		fw.Write([]byte(content))
	}
	w.Close()
}

func createTestTarGz(t *testing.T, path string, files map[string]string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	for name, content := range files {
		tw.WriteHeader(&tar.Header{
			Name: name,
			Size: int64(len(content)),
			Mode: 0o644,
		})
		tw.Write([]byte(content))
	}

	tw.Close()
	gw.Close()
}
