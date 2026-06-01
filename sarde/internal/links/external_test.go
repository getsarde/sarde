package links

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDedupeExternalURLs(t *testing.T) {
	refs := []LinkRef{
		{RawDest: "https://example.com/a", Status: StatusExternal},
		{RawDest: "https://example.com/a", Status: StatusExternal},
		{RawDest: "https://example.com/b", Status: StatusExternal},
		{RawDest: "http://localhost:3000/api", Status: StatusExternal},
		{RawDest: "./internal.md", Status: StatusOK},
	}
	urls := dedupeExternalURLs(refs)

	if len(urls) != 3 {
		t.Fatalf("expected 3 unique URLs, got %d: %v", len(urls), urls)
	}
	want := map[string]bool{
		"https://example.com/a":       true,
		"https://example.com/b":       true,
		"http://localhost:3000/api":    true,
	}
	for _, u := range urls {
		if !want[u] {
			t.Errorf("unexpected URL %q", u)
		}
	}
}

func TestMatchesIgnorePattern(t *testing.T) {
	tests := []struct {
		url      string
		patterns []string
		want     bool
	}{
		{"https://example.com/foo", []string{"https://example.com/*"}, true},
		{"https://example.com/foo/bar", []string{"https://example.com/*"}, false},
		{"https://other.com/foo", []string{"https://example.com/*"}, false},
		{"https://example.com/foo", []string{}, false},
		{"https://example.com/foo", []string{"https://example.com/foo"}, true},
	}
	for _, tt := range tests {
		if got := matchesIgnorePattern(tt.url, tt.patterns); got != tt.want {
			t.Errorf("matchesIgnorePattern(%q, %v) = %v, want %v", tt.url, tt.patterns, got, tt.want)
		}
	}
}

func TestLoadLinkCache_Missing(t *testing.T) {
	c := loadLinkCache(filepath.Join(t.TempDir(), "nonexistent.json"))
	if len(c.Entries) != 0 {
		t.Errorf("expected empty cache, got %d entries", len(c.Entries))
	}
}

func TestLoadLinkCache_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cache.json")

	original := linkCache{
		Entries: map[string]ExternalResult{
			"https://example.com": {URL: "https://example.com", StatusCode: 200, OK: true, Checked: time.Now().Truncate(time.Second)},
			"https://dead.com":    {URL: "https://dead.com", StatusCode: 404, OK: false, Checked: time.Now().Truncate(time.Second)},
		},
	}
	if err := saveLinkCache(path, original); err != nil {
		t.Fatalf("saveLinkCache failed: %v", err)
	}

	loaded := loadLinkCache(path)
	if len(loaded.Entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(loaded.Entries))
	}
	for url, want := range original.Entries {
		got, ok := loaded.Entries[url]
		if !ok {
			t.Errorf("missing entry for %q", url)
			continue
		}
		if got.StatusCode != want.StatusCode || got.OK != want.OK {
			t.Errorf("entry %q: got status=%d ok=%v, want status=%d ok=%v",
				url, got.StatusCode, got.OK, want.StatusCode, want.OK)
		}
	}
}

func TestProbeURL_HeadSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	defer srv.Close()

	res := probeURL(context.Background(), srv.Client(), srv.URL, "head-then-get")
	if !res.OK {
		t.Errorf("expected OK=true, got false (status=%d, error=%q)", res.StatusCode, res.Error)
	}
	if res.StatusCode != 200 {
		t.Errorf("expected 200, got %d", res.StatusCode)
	}
}

func TestProbeURL_HeadFallbackGet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(405)
			return
		}
		w.WriteHeader(200)
	}))
	defer srv.Close()

	res := probeURL(context.Background(), srv.Client(), srv.URL, "head-then-get")
	if !res.OK {
		t.Errorf("expected OK=true after GET fallback, got false (status=%d, error=%q)", res.StatusCode, res.Error)
	}
	if res.StatusCode != 200 {
		t.Errorf("expected 200 from GET fallback, got %d", res.StatusCode)
	}
}

func TestProbeURL_BrokenLink(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer srv.Close()

	res := probeURL(context.Background(), srv.Client(), srv.URL, "head-then-get")
	if res.OK {
		t.Error("expected OK=false for 404")
	}
	if res.StatusCode != 404 {
		t.Errorf("expected 404, got %d", res.StatusCode)
	}
}

func TestProbeURL_ConnectionError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close()

	res := probeURL(context.Background(), http.DefaultClient, srv.URL, "head-then-get")
	if res.OK {
		t.Error("expected OK=false for connection error")
	}
	if res.Error == "" {
		t.Error("expected non-empty Error for connection error")
	}
}

func TestCheckExternalLinks_AllCached(t *testing.T) {
	graph := NewLinkGraph()
	graph.Record(LinkRef{RawDest: "https://good.example.com/page", Status: StatusExternal, Kind: KindExternal})
	graph.Record(LinkRef{RawDest: "https://dead.example.com/gone", Status: StatusExternal, Kind: KindExternal})

	dir := t.TempDir()
	cachePath := filepath.Join(dir, "linkcache.json")
	cache := linkCache{
		Entries: map[string]ExternalResult{
			"https://good.example.com/page": {URL: "https://good.example.com/page", StatusCode: 200, OK: true, Checked: time.Now()},
			"https://dead.example.com/gone": {URL: "https://dead.example.com/gone", StatusCode: 404, OK: false, Checked: time.Now()},
		},
	}
	if err := saveLinkCache(cachePath, cache); err != nil {
		t.Fatal(err)
	}

	if err := CheckExternalLinks(graph, ExternalCheckConfig{
		Enabled:     true,
		Concurrency: 4,
		Timeout:     5 * time.Second,
		CachePath:   cachePath,
		CacheTTL:    24 * time.Hour,
		OnBroken:    "warn",
		Method:      "head-then-get",
	}); err != nil {
		t.Fatal(err)
	}

	refs := graph.Refs()
	for _, ref := range refs {
		switch ref.RawDest {
		case "https://good.example.com/page":
			if ref.Status != StatusExternal {
				t.Errorf("good URL: expected StatusExternal, got %d", ref.Status)
			}
		case "https://dead.example.com/gone":
			if ref.Status != StatusExternalBroken {
				t.Errorf("dead URL: expected StatusExternalBroken, got %d", ref.Status)
			}
		}
	}
}

func TestCheckExternalLinks_Disabled(t *testing.T) {
	graph := NewLinkGraph()
	graph.Record(LinkRef{RawDest: "https://example.com", Status: StatusExternal, Kind: KindExternal})

	if err := CheckExternalLinks(graph, ExternalCheckConfig{Enabled: false}); err != nil {
		t.Fatal(err)
	}

	refs := graph.Refs()
	if refs[0].Status != StatusExternal {
		t.Error("expected status unchanged when disabled")
	}
}

func TestCheckExternalLinks_WithHTTPServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			w.WriteHeader(200)
		case "/gone":
			w.WriteHeader(404)
		default:
			w.WriteHeader(500)
		}
	}))
	defer srv.Close()

	graph := NewLinkGraph()
	graph.Record(LinkRef{RawDest: srv.URL + "/ok", Status: StatusExternal, Kind: KindExternal})
	graph.Record(LinkRef{RawDest: srv.URL + "/gone", Status: StatusExternal, Kind: KindExternal})

	dir := t.TempDir()
	cachePath := filepath.Join(dir, "linkcache.json")

	if err := CheckExternalLinks(graph, ExternalCheckConfig{
		Enabled:     true,
		Concurrency: 2,
		Timeout:     5 * time.Second,
		CachePath:   cachePath,
		CacheTTL:    24 * time.Hour,
		OnBroken:    "warn",
		Method:      "head-then-get",
	}); err != nil {
		t.Fatal(err)
	}

	for _, ref := range graph.Refs() {
		switch ref.RawDest {
		case srv.URL + "/ok":
			if ref.Status != StatusExternal {
				t.Errorf("/ok: expected StatusExternal, got %d", ref.Status)
			}
		case srv.URL + "/gone":
			if ref.Status != StatusExternalBroken {
				t.Errorf("/gone: expected StatusExternalBroken, got %d", ref.Status)
			}
		}
	}

	if _, err := os.Stat(cachePath); err != nil {
		t.Errorf("cache file not created: %v", err)
	}
}
