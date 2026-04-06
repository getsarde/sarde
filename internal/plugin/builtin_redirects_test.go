package plugin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/coderoo-dev/coderoo/internal/config"
	"github.com/coderoo-dev/coderoo/internal/engine"
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
}

func TestRedirectsBuildDone_PageAliases(t *testing.T) {
	outDir := t.TempDir()
	var warnings []engine.ValidationWarning

	ctx := &BuildDoneContext{
		Config: &config.SiteConfig{},
		OutputDir: outDir,
		Pages: []*engine.Page{
			{
				Title:        "New Page",
				RelPermalink: "/docs/new-page/",
				Aliases:      []string{"/docs/old-name/", "/docs/renamed/"},
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
				Title:        "Page",
				RelPermalink: "/alias-target/",
				Aliases:      []string{"/conflict/"},
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
