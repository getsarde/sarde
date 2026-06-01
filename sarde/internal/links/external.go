package links

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
)

// ExternalCheckConfig holds the resolved configuration for external link checking.
type ExternalCheckConfig struct {
	Enabled     bool
	Concurrency int
	Timeout     time.Duration
	CachePath   string // absolute path to linkcache.json
	CacheTTL    time.Duration
	OnBroken    string   // "warn" | "error" | "ignore"
	Ignore      []string // URL glob patterns to skip
	Method      string   // "head-then-get" | "head" | "get"
}

// ExternalResult is the probed outcome for a single URL.
type ExternalResult struct {
	URL        string    `json:"url"`
	StatusCode int       `json:"status_code"`
	OK         bool      `json:"ok"`
	Checked    time.Time `json:"checked"`
	Error      string    `json:"error,omitempty"`
}

type linkCache struct {
	Entries map[string]ExternalResult `json:"entries"`
}

// CheckExternalLinks probes external URLs recorded in the graph, updating
// broken ones to StatusExternalBroken. Results are cached to disk.
func CheckExternalLinks(graph *LinkGraph, cfg ExternalCheckConfig) error {
	if !cfg.Enabled {
		return nil
	}

	cache := loadLinkCache(cfg.CachePath)

	refs := graph.ExternalRefs()
	uniqueURLs := dedupeExternalURLs(refs)

	var toProbe []string
	cutoff := time.Now().Add(-cfg.CacheTTL)
	for _, u := range uniqueURLs {
		if matchesIgnorePattern(u, cfg.Ignore) {
			continue
		}
		if entry, ok := cache.Entries[u]; ok && entry.Checked.After(cutoff) {
			continue
		}
		toProbe = append(toProbe, u)
	}

	if len(toProbe) > 0 {
		timeout := cfg.Timeout
		if timeout <= 0 {
			timeout = 10 * time.Second
		}
		concurrency := cfg.Concurrency
		if concurrency <= 0 {
			concurrency = 8
		}
		method := cfg.Method
		if method == "" {
			method = "head-then-get"
		}

		client := &http.Client{Timeout: timeout}
		var mu sync.Mutex
		sem := make(chan struct{}, concurrency)
		g := new(errgroup.Group)

		for _, u := range toProbe {
			u := u
			g.Go(func() error {
				sem <- struct{}{}
				defer func() { <-sem }()
				res := probeURL(context.Background(), client, u, method)
				mu.Lock()
				cache.Entries[u] = res
				mu.Unlock()
				return nil
			})
		}
		_ = g.Wait()
	}

	if err := saveLinkCache(cfg.CachePath, cache); err != nil {
		return fmt.Errorf("saving link cache: %w", err)
	}

	for _, u := range uniqueURLs {
		if matchesIgnorePattern(u, cfg.Ignore) {
			continue
		}
		if entry, ok := cache.Entries[u]; ok && !entry.OK {
			graph.MarkExternalBroken(u)
		}
	}

	return nil
}

func dedupeExternalURLs(refs []LinkRef) []string {
	seen := make(map[string]struct{}, len(refs))
	var urls []string
	for _, ref := range refs {
		if ref.Status != StatusExternal {
			continue
		}
		if _, ok := seen[ref.RawDest]; ok {
			continue
		}
		seen[ref.RawDest] = struct{}{}
		urls = append(urls, ref.RawDest)
	}
	return urls
}

func matchesIgnorePattern(rawURL string, patterns []string) bool {
	for _, p := range patterns {
		if matched, _ := path.Match(p, rawURL); matched {
			return true
		}
	}
	return false
}

func probeURL(ctx context.Context, client *http.Client, url, method string) ExternalResult {
	now := time.Now()

	httpMethod := http.MethodHead
	if method == "get" {
		httpMethod = http.MethodGet
	}

	req, err := http.NewRequestWithContext(ctx, httpMethod, url, nil)
	if err != nil {
		return ExternalResult{URL: url, OK: false, Checked: now, Error: err.Error()}
	}
	req.Header.Set("User-Agent", "sarde-link-checker/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return ExternalResult{URL: url, OK: false, Checked: now, Error: err.Error()}
	}
	resp.Body.Close()

	if (resp.StatusCode == 405 || resp.StatusCode == 403) && method == "head-then-get" {
		req2, err2 := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err2 != nil {
			return ExternalResult{URL: url, StatusCode: resp.StatusCode, OK: false, Checked: now, Error: err2.Error()}
		}
		req2.Header.Set("User-Agent", "sarde-link-checker/1.0")
		resp2, err2 := client.Do(req2)
		if err2 != nil {
			return ExternalResult{URL: url, StatusCode: resp.StatusCode, OK: false, Checked: now, Error: err2.Error()}
		}
		resp2.Body.Close()
		resp = resp2
	}

	ok := resp.StatusCode >= 200 && resp.StatusCode < 400
	return ExternalResult{URL: url, StatusCode: resp.StatusCode, OK: ok, Checked: now}
}

func loadLinkCache(path string) linkCache {
	c := linkCache{Entries: make(map[string]ExternalResult)}
	data, err := os.ReadFile(path)
	if err != nil {
		return c
	}
	var loaded linkCache
	if err := json.Unmarshal(data, &loaded); err != nil {
		return c
	}
	if loaded.Entries != nil {
		c.Entries = loaded.Entries
	}
	return c
}

func saveLinkCache(cachePath string, c linkCache) error {
	dir := filepath.Dir(cachePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cachePath, data, 0644)
}
