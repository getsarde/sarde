package download

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type SourceKind int

const (
	SourceGitHub SourceKind = iota
	SourceZipURL
	SourceTarGzURL
	SourceLocalDir
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
	return fmt.Sprintf("https://github.com/%s/%s/archive/refs/heads/%s.zip",
		url.PathEscape(r.Owner), url.PathEscape(r.Repo), url.PathEscape(r.Branch))
}

func DownloadFile(srcURL string) (string, error) {
	client := &http.Client{Timeout: 60 * time.Second}

	resp, err := client.Get(srcURL)
	if err != nil {
		return "", fmt.Errorf("downloading %s: %w", srcURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("downloading %s: HTTP %d", srcURL, resp.StatusCode)
	}

	if resp.ContentLength > maxDownloadSize {
		return "", fmt.Errorf("download too large: %d bytes (max %d)", resp.ContentLength, maxDownloadSize)
	}

	tmp, err := os.CreateTemp("", "coderoo-theme-*")
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
		name := filepath.ToSlash(f.Name)
		name = stripLeading(name, stripComponents)
		if name == "" {
			continue
		}

		dest := filepath.Join(destDir, filepath.FromSlash(name))
		if !isInsideDir(destDir, dest) {
			continue
		}

		if f.FileInfo().IsDir() {
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

	out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode()|0o200)
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
			out, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode)|0o200)
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
