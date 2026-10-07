package sitetemplate

import (
	"archive/zip"
	"bytes"
	"context"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/getsarde/sarde/internal/download"
)

// buildZip returns a GitHub-style archive: every entry under one top folder.
func buildZip(t *testing.T, top string, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, body := range files {
		f, err := w.Create(top + "/" + name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

var courseFiles = map[string]string{
	"course/README.md":                    "about the template",
	"course/LICENSE":                      "mit",
	"course/sarde.yaml":                   "site:\n  title: Course\n",
	"course/content/_index.md":            "# Home",
	"course/.github/workflows/deploy.yml": "on: push",
	"LICENSE":                             "root license",
}

// templateServer serves archives by request path and counts requests.
type templateServer struct {
	*httptest.Server
	archives map[string][]byte
	requests atomic.Int32
}

func newTemplateServer(t *testing.T, archives map[string][]byte) *templateServer {
	t.Helper()
	ts := &templateServer{archives: archives}
	ts.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ts.requests.Add(1)
		data, ok := ts.archives[r.URL.Path]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Write(data)
	}))
	t.Cleanup(ts.Close)
	prev := download.GitHubBaseURL
	download.GitHubBaseURL = ts.URL
	t.Cleanup(func() { download.GitHubBaseURL = prev })
	return ts
}

func newFetcher(t *testing.T, version string) *Fetcher {
	t.Helper()
	return &Fetcher{CacheDir: t.TempDir(), Version: version, Download: download.DownloadFileContext}
}

func mustFetch(t *testing.T, f *Fetcher, spec string) *Result {
	t.Helper()
	s, err := Parse(spec)
	if err != nil {
		t.Fatal(err)
	}
	res, err := f.Fetch(context.Background(), s)
	if err != nil {
		t.Fatalf("Fetch(%s): %v", spec, err)
	}
	t.Cleanup(res.Cleanup)
	return res
}

func TestFetch_TagIsCachedAndAuthoritative(t *testing.T) {
	srv := newTemplateServer(t, map[string][]byte{
		"/getsarde/sarde-templates/archive/refs/tags/v1.5.zip": buildZip(t, "sarde-templates-1.5", courseFiles),
	})
	f := newFetcher(t, "1.5.0")

	res := mustFetch(t, f, "course")
	if res.Ref != "v1.5" || len(res.Notices) != 0 {
		t.Fatalf("ref=%q notices=%v", res.Ref, res.Notices)
	}
	if _, err := os.Stat(filepath.Join(f.CacheDir, "getsarde", "sarde-templates", "v1.5.zip")); err != nil {
		t.Fatalf("archive not cached: %v", err)
	}
	if _, err := fs.Stat(res.FS, "README.md"); err == nil {
		t.Error("README.md at the template root must be hidden")
	}
	if _, err := fs.Stat(res.FS, ".github/workflows/deploy.yml"); err != nil {
		t.Errorf("workflow missing: %v", err)
	}
	data, _ := fs.ReadFile(res.FS, "sarde.yaml")
	if !strings.Contains(string(data), "title: Course") {
		t.Errorf("sarde.yaml content wrong: %q", data)
	}

	srv.Close()
	before := srv.requests.Load()
	res2 := mustFetch(t, f, "course")
	if srv.requests.Load() != before || len(res2.Notices) != 0 {
		t.Errorf("a cached tag must not touch the network; requests=%d notices=%v", srv.requests.Load()-before, res2.Notices)
	}
}

func TestFetch_MissingTagFallsBackToMain(t *testing.T) {
	newTemplateServer(t, map[string][]byte{
		"/getsarde/sarde-templates/archive/refs/heads/main.zip": buildZip(t, "sarde-templates-main", courseFiles),
	})
	f := newFetcher(t, "1.5.0")
	res := mustFetch(t, f, "course")
	if res.Ref != "main" {
		t.Errorf("ref = %q, want main", res.Ref)
	}
	if len(res.Notices) != 1 || !strings.Contains(res.Notices[0], "v1.5") || !strings.Contains(res.Notices[0], "using main") {
		t.Errorf("notices = %v", res.Notices)
	}
	if _, err := os.Stat(filepath.Join(f.CacheDir, "getsarde", "sarde-templates", "main.zip")); err != nil {
		t.Errorf("main archive not cached: %v", err)
	}
}

func TestFetch_MainIsNetworkFirstAndOfflineUsesCache(t *testing.T) {
	srv := newTemplateServer(t, map[string][]byte{
		"/getsarde/sarde-templates/archive/refs/heads/main.zip": buildZip(t, "sarde-templates-main", courseFiles),
	})
	f := newFetcher(t, "dev")
	mustFetch(t, f, "course")
	mustFetch(t, f, "course")
	if n := srv.requests.Load(); n != 2 {
		t.Errorf("main should be fetched every time while online, got %d requests", n)
	}

	srv.Close()
	res := mustFetch(t, f, "course")
	if len(res.Notices) != 1 || !strings.Contains(res.Notices[0], "cached on") {
		t.Errorf("offline fetch should say it used the cache: %v", res.Notices)
	}
}

func TestFetch_OfflineWithoutCacheNamesCloneURL(t *testing.T) {
	srv := newTemplateServer(t, nil)
	srv.Close()
	f := newFetcher(t, "dev")
	s, _ := Parse("course")
	_, err := f.Fetch(context.Background(), s)
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, want := range []string{"https://github.com/getsarde/sarde-templates.git", "No cached copy", `"course" folder`} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error missing %q: %v", want, err)
		}
	}
}

func TestFetch_NotFoundAndMissingFolder(t *testing.T) {
	newTemplateServer(t, map[string][]byte{
		"/acme/site/archive/refs/heads/main.zip": buildZip(t, "site-main", map[string]string{"README.md": "x"}),
	})
	f := newFetcher(t, "dev")

	s, _ := Parse("acme/nope")
	if _, err := f.Fetch(context.Background(), s); err == nil || !strings.Contains(err.Error(), "HTTP 404") {
		t.Errorf("missing repo: %v", err)
	}
	s, _ = Parse("acme/site/missing")
	if _, err := f.Fetch(context.Background(), s); err == nil || !strings.Contains(err.Error(), `folder "missing" not found`) {
		t.Errorf("missing folder: %v", err)
	}
}

func TestFetch_ExplicitRefAndSubpathEscaping(t *testing.T) {
	srv := newTemplateServer(t, map[string][]byte{
		"/acme/site/archive/release/1.5.zip": buildZip(t, "site-release-1.5", map[string]string{"tpl/sarde.yaml": "x"}),
	})
	f := newFetcher(t, "1.4.0")
	res := mustFetch(t, f, "acme/site/tpl#release/1.5")
	if res.Ref != "release/1.5" || srv.requests.Load() != 1 {
		t.Errorf("ref=%q requests=%d", res.Ref, srv.requests.Load())
	}
	if _, err := os.Stat(filepath.Join(f.CacheDir, "acme", "site", "release%2F1.5.zip")); err != nil {
		t.Errorf("slash in ref must be escaped in the cache name: %v", err)
	}
}

func TestFetch_SymlinkEntryIsRejected(t *testing.T) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	hdr := &zip.FileHeader{Name: "site-main/tpl/link"}
	hdr.SetMode(fs.ModeSymlink | 0o777)
	lw, _ := w.CreateHeader(hdr)
	lw.Write([]byte("../etc/passwd"))
	fw, _ := w.Create("site-main/tpl/sarde.yaml")
	fw.Write([]byte("x"))
	w.Close()

	newTemplateServer(t, map[string][]byte{"/acme/site/archive/refs/heads/main.zip": buf.Bytes()})
	s, _ := Parse("acme/site/tpl")
	_, err := newFetcher(t, "dev").Fetch(context.Background(), s)
	if err == nil || !strings.Contains(err.Error(), "symlink") {
		t.Errorf("expected a symlink error, got %v", err)
	}
}
