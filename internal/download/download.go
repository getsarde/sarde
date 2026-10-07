package download

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/getsarde/sarde/internal/devlog"
	"github.com/getsarde/sarde/internal/version"
)

// GitHubBaseURL is the host archive URLs are built on. Tests point it at an
// httptest server; production code never changes it.
var GitHubBaseURL = "https://github.com"

// RefKind says how a Git ref should be requested from GitHub's archive
// endpoint. RefAny lets GitHub resolve a branch, tag, or commit by name.
type RefKind int

const (
	RefAny RefKind = iota
	RefBranch
	RefTag
)

// ArchiveURLFor returns the zip archive URL for a ref of a GitHub repository.
// Each path segment of the ref is escaped on its own, so a branch such as
// "release/1.5" keeps its slash.
func ArchiveURLFor(owner, repo, ref string, kind RefKind) string {
	segs := strings.Split(ref, "/")
	for i, seg := range segs {
		segs[i] = url.PathEscape(seg)
	}
	escaped := strings.Join(segs, "/")
	base := fmt.Sprintf("%s/%s/%s/archive", strings.TrimSuffix(GitHubBaseURL, "/"), url.PathEscape(owner), url.PathEscape(repo))
	switch kind {
	case RefBranch:
		return base + "/refs/heads/" + escaped + ".zip"
	case RefTag:
		return base + "/refs/tags/" + escaped + ".zip"
	default:
		return base + "/" + escaped + ".zip"
	}
}

// HTTPStatusError reports a download that reached the server but was
// answered with a status other than 200.
type HTTPStatusError struct {
	URL        string
	StatusCode int
}

func (e *HTTPStatusError) Error() string {
	return fmt.Sprintf("downloading %s: HTTP %d", e.URL, e.StatusCode)
}

type SourceKind int

const (
	SourceGitHub SourceKind = iota
	SourceZipURL
	SourceTarGzURL
	SourceLocalDir
	SourceLocalZipFile
	SourceUnknown
)

type GitHubRef struct {
	Owner   string
	Repo    string
	Branch  string
	Subpath string
}

const maxDownloadSize = 100 << 20 // 100 MB

func InferSourceKind(src string) SourceKind {
	if isGitHubURL(src) {
		return SourceGitHub
	}

	lower := strings.ToLower(src)
	if strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://") {
		if strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz") {
			return SourceTarGzURL
		}
		if strings.HasSuffix(lower, ".zip") {
			return SourceZipURL
		}
		return SourceUnknown
	}

	info, err := os.Stat(src)
	if err == nil && info.IsDir() {
		return SourceLocalDir
	}
	if err == nil && !info.IsDir() && strings.HasSuffix(lower, ".zip") {
		return SourceLocalZipFile
	}

	return SourceUnknown
}

func isGitHubURL(src string) bool {
	s := strings.TrimPrefix(strings.TrimPrefix(src, "https://"), "http://")
	return strings.HasPrefix(s, "github.com/")
}

func ParseGitHubURL(raw string) (*GitHubRef, error) {
	s := strings.TrimPrefix(strings.TrimPrefix(raw, "https://"), "http://")
	s = strings.TrimSuffix(s, "/")

	if !strings.HasPrefix(s, "github.com/") {
		return nil, fmt.Errorf("not a GitHub URL: %s", raw)
	}

	parts := strings.Split(strings.TrimPrefix(s, "github.com/"), "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("invalid GitHub URL: need github.com/<owner>/<repo>, got %s", raw)
	}

	ref := &GitHubRef{
		Owner:  parts[0],
		Repo:   strings.TrimSuffix(parts[1], ".git"),
		Branch: "main",
	}

	// github.com/owner/repo/tree/<branch>[/<subpath>...]
	if len(parts) >= 4 && parts[2] == "tree" {
		ref.Branch = parts[3]
		if len(parts) > 4 {
			ref.Subpath = strings.Join(parts[4:], "/")
		}
	}

	return ref, nil
}

func (r *GitHubRef) ArchiveURL() string {
	return ArchiveURLFor(r.Owner, r.Repo, r.Branch, RefBranch)
}

// DownloadFile fetches srcURL into a temporary file and returns its path.
// The caller removes the file.
func DownloadFile(srcURL string) (string, error) {
	return DownloadFileContext(context.Background(), srcURL)
}

// DownloadFileContext is DownloadFile with cancellation. A non-200 response
// is returned as an *HTTPStatusError so callers can tell "not found" from a
// network failure.
func DownloadFileContext(ctx context.Context, srcURL string) (string, error) {
	if strings.HasPrefix(strings.ToLower(srcURL), "http://") {
		devlog.Warn("download", "downloading over insecure HTTP: %s; consider using HTTPS", srcURL)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srcURL, nil)
	if err != nil {
		return "", fmt.Errorf("downloading %s: %w", srcURL, err)
	}
	req.Header.Set("User-Agent", "sarde/"+version.Version)

	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("downloading %s: %w", srcURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", &HTTPStatusError{URL: srcURL, StatusCode: resp.StatusCode}
	}

	// Best-effort early rejection when the server declares a length. For
	// chunked/unknown-length responses ContentLength is -1 and this never
	// fires; the post-copy size check below (paired with the LimitReader) is
	// the actual enforcement.
	if resp.ContentLength > maxDownloadSize {
		return "", fmt.Errorf("download too large: %d bytes (max %d)", resp.ContentLength, maxDownloadSize)
	}

	tmp, err := os.CreateTemp("", "sarde-download-*")
	if err != nil {
		return "", fmt.Errorf("creating temp file: %w", err)
	}

	_, err = io.Copy(tmp, io.LimitReader(resp.Body, maxDownloadSize+1))
	tmp.Close()
	if err != nil {
		os.Remove(tmp.Name())
		return "", fmt.Errorf("writing download to %s: %w", tmp.Name(), err)
	}

	info, _ := os.Stat(tmp.Name())
	if info != nil && info.Size() > maxDownloadSize {
		os.Remove(tmp.Name())
		return "", fmt.Errorf("download too large: exceeded %d bytes", maxDownloadSize)
	}

	return tmp.Name(), nil
}

func ExtractZip(zipPath, destDir string, stripComponents int) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("opening zip %s: %w", zipPath, err)
	}
	defer r.Close()

	for _, f := range r.File {
		// Normalize separators explicitly: Windows-created archives (e.g.
		// PowerShell Compress-Archive) store entry names with backslashes,
		// which filepath.ToSlash only fixes when running on Windows.
		name := strings.ReplaceAll(f.Name, "\\", "/")
		isDir := f.FileInfo().IsDir() || strings.HasSuffix(name, "/")
		name = stripLeading(name, stripComponents)
		if name == "" {
			continue
		}

		dest := filepath.Join(destDir, filepath.FromSlash(name))
		if !isInsideDir(destDir, dest) {
			continue
		}

		if isDir {
			os.MkdirAll(dest, 0o755)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}

		if err := extractZipFile(f, dest); err != nil {
			return err
		}
	}

	return nil
}

func extractZipFile(f *zip.File, dest string) error {
	rc, err := f.Open()
	if err != nil {
		return fmt.Errorf("opening zip entry %s: %w", f.Name, err)
	}
	defer rc.Close()

	// Windows-created archives often record mode 0; OR in 0o644 so extracted
	// files are always readable (mode 0|0o200 would be write-only).
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode().Perm()|0o644)
	if err != nil {
		return fmt.Errorf("creating %s: %w", dest, err)
	}
	defer out.Close()

	_, err = io.Copy(out, rc)
	return err
}

func ExtractTarGz(tarPath, destDir string, stripComponents int) error {
	f, err := os.Open(tarPath)
	if err != nil {
		return fmt.Errorf("opening %s: %w", tarPath, err)
	}
	defer f.Close()

	gz, err := gzip.NewReader(f)
	if err != nil {
		return fmt.Errorf("gzip reader: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("reading tar: %w", err)
		}

		name := filepath.ToSlash(hdr.Name)
		name = stripLeading(name, stripComponents)
		if name == "" {
			continue
		}

		dest := filepath.Join(destDir, filepath.FromSlash(name))
		if !isInsideDir(destDir, dest) {
			continue
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(dest, 0o755)
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
				return err
			}
			// Same readability guarantee as extractZipFile: entries with
			// recorded mode 0 must not come out write-only.
			out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode).Perm()|0o644)
			if err != nil {
				return fmt.Errorf("creating %s: %w", dest, err)
			}
			_, err = io.Copy(out, tr)
			out.Close()
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func CopyDir(src, dest string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dest, rel)

		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}

		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func stripLeading(name string, n int) string {
	for i := 0; i < n; i++ {
		idx := strings.Index(name, "/")
		if idx < 0 {
			return ""
		}
		name = name[idx+1:]
	}
	return name
}

func isInsideDir(base, target string) bool {
	absBase, _ := filepath.Abs(base)
	absTarget, _ := filepath.Abs(target)
	rel, err := filepath.Rel(absBase, absTarget)
	if err != nil {
		return false
	}
	return !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".."
}
