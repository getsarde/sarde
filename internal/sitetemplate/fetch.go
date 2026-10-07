package sitetemplate

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/getsarde/sarde/internal/atomicwrite"
	"github.com/getsarde/sarde/internal/download"
	"github.com/getsarde/sarde/internal/version"
)

// Fetcher downloads template archives. The zero value is not usable; call
// DefaultFetcher, or fill the fields in tests.
type Fetcher struct {
	// CacheDir holds one archive per owner/repo/ref. Empty disables caching.
	CacheDir string
	// Version is the engine version refs are pinned to.
	Version string
	// Download fetches a URL into a temporary file (the caller removes it).
	Download func(ctx context.Context, url string) (string, error)
}

// Result is a fetched template, served straight from its zip archive.
type Result struct {
	// FS is the template folder: the repository subpath applied, and
	// README.md and LICENSE at its root hidden.
	FS fs.FS
	// Ref is the ref actually used, "main" after a tag fallback.
	Ref string
	// Notices are things the user should know: a missing tag, a cached copy
	// used offline. The caller prints them.
	Notices []string
	// Cleanup closes the archive. Call it once the files are copied.
	Cleanup func()
}

// DefaultFetcher caches under ~/.sarde/templates and downloads from GitHub.
func DefaultFetcher() *Fetcher {
	f := &Fetcher{Version: version.Version, Download: download.DownloadFileContext}
	if home, err := os.UserHomeDir(); err == nil {
		f.CacheDir = filepath.Join(home, ".sarde", "templates")
	}
	return f
}

// Fetch resolves the ref for spec, obtains the archive (cache or network),
// and returns the template folder as an fs.FS.
func (f *Fetcher) Fetch(ctx context.Context, spec Spec) (*Result, error) {
	r := ResolveRef(spec, f.Version)
	var notices []string

	zipPath, fromTemp, ref, err := f.archive(ctx, spec, r, &notices)
	if err != nil {
		return nil, err
	}

	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		if fromTemp != "" {
			os.Remove(fromTemp)
		}
		return nil, fmt.Errorf("opening template archive: %w", err)
	}
	cleanup := func() {
		zr.Close()
		if fromTemp != "" {
			os.Remove(fromTemp)
		}
	}

	root, err := templateRoot(&zr.Reader, spec, ref)
	if err != nil {
		cleanup()
		return nil, err
	}
	return &Result{
		FS:      HideRootFiles(root, "README.md", "LICENSE"),
		Ref:     ref,
		Notices: notices,
		Cleanup: cleanup,
	}, nil
}

// archive returns the path of the zip to read and the ref it holds.
// fromTemp is the temporary file to delete afterwards, or "" when the cache
// file is used directly.
func (f *Fetcher) archive(ctx context.Context, spec Spec, r Resolved, notices *[]string) (zipPath, fromTemp, ref string, err error) {
	if r.Pinned {
		// A tag never moves, so a cached copy is authoritative.
		if p := f.cachePath(spec, r.Ref); p != "" {
			if _, statErr := os.Stat(p); statErr == nil {
				return p, "", r.Ref, nil
			}
		}
		tmp, dlErr := f.Download(ctx, download.ArchiveURLFor(spec.Owner, spec.Repo, r.Ref, r.Kind))
		if dlErr == nil {
			return f.store(spec, r.Ref, tmp), tmp, r.Ref, nil
		}
		switch {
		case isNotFound(dlErr):
			*notices = append(*notices, fmt.Sprintf("tag %s not found in github.com/%s/%s; using main", r.Ref, spec.Owner, spec.Repo))
			r = Resolved{Ref: "main", Kind: download.RefBranch}
		case isNetworkError(dlErr):
			if p, when, ok := f.cached(spec, "main"); ok {
				*notices = append(*notices, fmt.Sprintf("GitHub unreachable (%v); using the copy of %s/%s@main cached on %s", dlErr, spec.Owner, spec.Repo, when))
				return p, "", "main", nil
			}
			return "", "", "", f.offlineError(spec, dlErr)
		default:
			return "", "", "", dlErr
		}
	}

	// Network first: main moves, and an explicit ref is the user's call.
	tmp, dlErr := f.Download(ctx, download.ArchiveURLFor(spec.Owner, spec.Repo, r.Ref, r.Kind))
	if dlErr == nil {
		return f.store(spec, r.Ref, tmp), tmp, r.Ref, nil
	}
	if isNotFound(dlErr) {
		return "", "", "", fmt.Errorf("template github.com/%s/%s at %q not found (HTTP 404); check the repository name and ref", spec.Owner, spec.Repo, r.Ref)
	}
	if isNetworkError(dlErr) {
		if p, when, ok := f.cached(spec, r.Ref); ok {
			*notices = append(*notices, fmt.Sprintf("GitHub unreachable (%v); using the copy of %s/%s@%s cached on %s", dlErr, spec.Owner, spec.Repo, r.Ref, when))
			return p, "", r.Ref, nil
		}
		return "", "", "", f.offlineError(spec, dlErr)
	}
	return "", "", "", dlErr
}

// store copies a downloaded archive into the cache. The download is served
// from tmp either way, so a cache write failure is not an error.
func (f *Fetcher) store(spec Spec, ref, tmp string) string {
	if p := f.cachePath(spec, ref); p != "" {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err == nil {
			_ = atomicwrite.CopyFile(tmp, p, 0o644)
		}
	}
	return tmp
}

func (f *Fetcher) cachePath(spec Spec, ref string) string {
	if f.CacheDir == "" {
		return ""
	}
	return filepath.Join(f.CacheDir, spec.Owner, spec.Repo, url.PathEscape(ref)+".zip")
}

// cached reports whether an archive for ref is cached, and when it was saved.
func (f *Fetcher) cached(spec Spec, ref string) (path, when string, ok bool) {
	p := f.cachePath(spec, ref)
	if p == "" {
		return "", "", false
	}
	info, err := os.Stat(p)
	if err != nil {
		return "", "", false
	}
	return p, info.ModTime().Format("2006-01-02"), true
}

func (f *Fetcher) offlineError(spec Spec, cause error) error {
	where := "the template cache"
	if f.CacheDir != "" {
		where = f.CacheDir
	}
	// The cause already names the archive URL, so it leads.
	msg := fmt.Sprintf("%s. No cached copy under %s; check the connection, or clone %s",
		strings.TrimSuffix(cause.Error(), "."), where, spec.CloneURL())
	if spec.Subpath != "" {
		msg += fmt.Sprintf(" and copy its %q folder", spec.Subpath)
	}
	return errors.New(msg)
}

func isNotFound(err error) bool {
	var se *download.HTTPStatusError
	return errors.As(err, &se) && se.StatusCode == http.StatusNotFound
}

// isNetworkError is any failure that never got an HTTP status and was not a
// cancellation: DNS, connection refused, TLS, timeouts.
func isNetworkError(err error) bool {
	var se *download.HTTPStatusError
	if errors.As(err, &se) {
		return false
	}
	return !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded)
}

// templateRoot locates the template folder inside a GitHub archive. GitHub
// wraps every archive in one top-level folder named after the repository
// and ref, with a tag's leading "v" dropped, so the name is read from the
// entries rather than computed.
func templateRoot(zr *zip.Reader, spec Spec, ref string) (fs.FS, error) {
	top := ""
	for _, file := range zr.File {
		name := strings.TrimPrefix(strings.ReplaceAll(file.Name, "\\", "/"), "/")
		first, _, _ := strings.Cut(name, "/")
		if first == "" {
			continue
		}
		if top == "" {
			top = first
		} else if first != top {
			return nil, fmt.Errorf("template archive has more than one top-level folder (%q and %q)", top, first)
		}
	}
	if top == "" {
		return nil, errors.New("template archive is empty")
	}

	root := path.Join(top, spec.Subpath)
	sub, err := fs.Sub(zr, root)
	if err != nil {
		return nil, fmt.Errorf("reading template archive: %w", err)
	}
	entries, err := fs.ReadDir(sub, ".")
	if err != nil || len(entries) == 0 {
		folder := spec.Subpath
		if folder == "" {
			folder = "."
		}
		return nil, fmt.Errorf("folder %q not found in github.com/%s/%s at %s", folder, spec.Owner, spec.Repo, ref)
	}

	// os.CopyFS cannot copy symlinks out of a zip; fail early with a name.
	var symlink string
	_ = fs.WalkDir(sub, ".", func(p string, d fs.DirEntry, err error) error {
		if err == nil && d.Type()&fs.ModeSymlink != 0 && symlink == "" {
			symlink = p
			return fs.SkipAll
		}
		return nil
	})
	if symlink != "" {
		return nil, fmt.Errorf("template contains a symlink (%s); symlinks are not supported", symlink)
	}
	return sub, nil
}
