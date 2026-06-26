package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/engine"
)

func TestRedirectsBuildDone_GlobalRedirects(t *testing.T) {
	outDir := t.TempDir()
	var warnings []engine.ValidationWarning

	ctx := &BuildDoneContext{
		Config: &config.SiteConfig{
			Redirects: map[string]string{
				"/old-page":  "/new-page/",
				"/legacy/":   "/modern/",
			},
		},
		OutputDir: outDir,
		Pages:     []*engine.Page{},
	}
	ctx.SetWarnings(&warnings)

	if err := redirectsBuildDone(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check HTML redirect file for /old-page.
	data, err := os.ReadFile(filepath.Join(outDir, "old-page", "index.html"))
	if err != nil {
		t.Fatalf("redirect HTML not written: %v", err)
	}
	if !strings.Contains(string(data), `url=/new-page/`) {
		t.Errorf("redirect HTML missing target URL, got: %s", data)
	}
	if !strings.Contains(string(data), `rel="canonical"`) {
		t.Error("redirect HTML missing canonical link")
	}

	// Check _redirects file.
	rdata, err := os.ReadFile(filepath.Join(outDir, "_redirects"))
	if err != nil {
		t.Fatalf("_redirects not written: %v", err)
	}
	content := string(rdata)
	if !strings.Contains(content, "/old-page  /new-page/  301") {
		t.Errorf("_redirects missing entry, got: %s", content)
	}

	// Check vercel.json (default format = "all" writes it).
	vdata, err := os.ReadFile(filepath.Join(outDir, "vercel.json"))
	if err != nil {
		t.Fatalf("vercel.json not written: %v", err)
	}
	var vc vercelConfig
	if err := json.Unmarshal(vdata, &vc); err != nil {
		t.Fatalf("vercel.json is not valid JSON: %v", err)
	}
	if len(vc.Redirects) != 2 {
		t.Errorf("vercel.json has %d redirects, want 2", len(vc.Redirects))
	}
}

func TestRedirectsBuildDone_PageAliases(t *testing.T) {
	outDir := t.TempDir()
	var warnings []engine.ValidationWarning

	ctx := &BuildDoneContext{
		Config: &config.SiteConfig{},
		OutputDir: outDir,
		Pages: []*engine.Page{
			{
				PageIdentity: engine.PageIdentity{Title: "New Page", RelPermalink: "/docs/new-page/"},
				PageTaxonomy: engine.PageTaxonomy{Aliases: []string{"/docs/old-name/", "/docs/renamed/"}},
			},
		},
	}
	ctx.SetWarnings(&warnings)

	if err := redirectsBuildDone(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check both alias HTML files.
	for _, alias := range []string{"docs/old-name/index.html", "docs/renamed/index.html"} {
		data, err := os.ReadFile(filepath.Join(outDir, alias))
		if err != nil {
			t.Fatalf("alias redirect %s not written: %v", alias, err)
		}
		if !strings.Contains(string(data), `/docs/new-page/`) {
			t.Errorf("alias redirect %s missing target URL", alias)
		}
	}
}

func TestRedirectsBuildDone_AliasOverridesGlobal(t *testing.T) {
	outDir := t.TempDir()
	var warnings []engine.ValidationWarning

	ctx := &BuildDoneContext{
		Config: &config.SiteConfig{
			Redirects: map[string]string{
				"/conflict/": "/global-target/",
			},
		},
		OutputDir: outDir,
		Pages: []*engine.Page{
			{
				PageIdentity: engine.PageIdentity{Title: "Page", RelPermalink: "/alias-target/"},
				PageTaxonomy: engine.PageTaxonomy{Aliases: []string{"/conflict/"}},
			},
		},
	}
	ctx.SetWarnings(&warnings)

	if err := redirectsBuildDone(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Alias should win over global.
	data, err := os.ReadFile(filepath.Join(outDir, "conflict", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `/alias-target/`) {
		t.Errorf("alias should override global, got: %s", data)
	}
}

func TestRedirectsBuildDone_NoRedirects(t *testing.T) {
	outDir := t.TempDir()
	var warnings []engine.ValidationWarning

	ctx := &BuildDoneContext{
		Config:    &config.SiteConfig{},
		OutputDir: outDir,
		Pages:     []*engine.Page{},
	}
	ctx.SetWarnings(&warnings)

	if err := redirectsBuildDone(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// No files should be written.
	if _, err := os.Stat(filepath.Join(outDir, "_redirects")); !os.IsNotExist(err) {
		t.Error("_redirects should not be written when no redirects")
	}
}

func TestRedirectsBuildDone_NetlifyOnlyFormat(t *testing.T) {
	outDir := t.TempDir()
	var warnings []engine.ValidationWarning

	ctx := &BuildDoneContext{
		Config: &config.SiteConfig{
			Deploy:    config.DeployConfig{RedirectFormat: "netlify"},
			Redirects: map[string]string{"/old": "/new"},
		},
		OutputDir: outDir,
		Pages:     []*engine.Page{},
	}
	ctx.SetWarnings(&warnings)

	if err := redirectsBuildDone(ctx); err != nil {
		t.Fatal(err)
	}

	// _redirects should exist.
	if _, err := os.Stat(filepath.Join(outDir, "_redirects")); err != nil {
		t.Error("_redirects should be written for netlify format")
	}

	// vercel.json should NOT exist.
	if _, err := os.Stat(filepath.Join(outDir, "vercel.json")); !os.IsNotExist(err) {
		t.Error("vercel.json should not be written for netlify format")
	}

	// HTML redirect should still exist.
	if _, err := os.Stat(filepath.Join(outDir, "old", "index.html")); err != nil {
		t.Error("HTML redirect should always be written")
	}
}

func TestRedirectsBuildDone_VercelOnlyFormat(t *testing.T) {
	outDir := t.TempDir()
	var warnings []engine.ValidationWarning

	ctx := &BuildDoneContext{
		Config: &config.SiteConfig{
			Deploy:    config.DeployConfig{RedirectFormat: "vercel"},
			Redirects: map[string]string{"/old": "/new"},
		},
		OutputDir: outDir,
		Pages:     []*engine.Page{},
	}
	ctx.SetWarnings(&warnings)

	if err := redirectsBuildDone(ctx); err != nil {
		t.Fatal(err)
	}

	// vercel.json should exist and be valid.
	vdata, err := os.ReadFile(filepath.Join(outDir, "vercel.json"))
	if err != nil {
		t.Fatal("vercel.json should be written for vercel format")
	}
	var vc vercelConfig
	if err := json.Unmarshal(vdata, &vc); err != nil {
		t.Fatalf("vercel.json invalid: %v", err)
	}
	if len(vc.Redirects) != 1 {
		t.Errorf("vercel.json has %d redirects, want 1", len(vc.Redirects))
	}
	if vc.Redirects[0].Source != "/old" || vc.Redirects[0].Destination != "/new" {
		t.Errorf("vercel redirect = %+v, want /old → /new", vc.Redirects[0])
	}
	if !vc.Redirects[0].Permanent {
		t.Error("vercel redirect should be permanent")
	}

	// _redirects should NOT exist.
	if _, err := os.Stat(filepath.Join(outDir, "_redirects")); !os.IsNotExist(err) {
		t.Error("_redirects should not be written for vercel format")
	}

	// HTML redirect should still exist.
	if _, err := os.Stat(filepath.Join(outDir, "old", "index.html")); err != nil {
		t.Error("HTML redirect should always be written")
	}
}

func TestRedirectsBuildDone_HtmlOnlyFormat(t *testing.T) {
	outDir := t.TempDir()
	var warnings []engine.ValidationWarning

	ctx := &BuildDoneContext{
		Config: &config.SiteConfig{
			Deploy:    config.DeployConfig{RedirectFormat: "html"},
			Redirects: map[string]string{"/old": "/new"},
		},
		OutputDir: outDir,
		Pages:     []*engine.Page{},
	}
	ctx.SetWarnings(&warnings)

	if err := redirectsBuildDone(ctx); err != nil {
		t.Fatal(err)
	}

	// HTML redirect should exist.
	if _, err := os.Stat(filepath.Join(outDir, "old", "index.html")); err != nil {
		t.Error("HTML redirect should be written for html format")
	}

	// Neither _redirects nor vercel.json should exist.
	if _, err := os.Stat(filepath.Join(outDir, "_redirects")); !os.IsNotExist(err) {
		t.Error("_redirects should not be written for html format")
	}
	if _, err := os.Stat(filepath.Join(outDir, "vercel.json")); !os.IsNotExist(err) {
		t.Error("vercel.json should not be written for html format")
	}
}
