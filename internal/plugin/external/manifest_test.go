package external

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeManifest(t *testing.T, dir, content string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "plugin.yaml"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

const validManifest = `
name: SlideViewer
slug: slideviewer
version: 1.2.0
description: Presentation viewer
author: frostybee
premium: true
purchase_url: https://example.com/buy
inject:
  when: layout
  layout: presentation
  styles: [css/slideviewer.css]
  module_scripts: [js/SlideViewer.js]
output:
  prefix: assets/vendor/slideviewer/
  include: [js/, css/]
`

func TestLoadManifestValid(t *testing.T) {
	dir := writeManifest(t, filepath.Join(t.TempDir(), "slideviewer"), validManifest)
	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}
	if err := m.Validate("slideviewer"); err != nil {
		t.Fatalf("Validate: %v", err)
	}
	if m.Name != "SlideViewer" || m.Version != "1.2.0" || !m.Premium {
		t.Errorf("unexpected manifest fields: %+v", m)
	}
	if len(m.Inject.Styles) != 1 || m.Inject.Styles[0] != "css/slideviewer.css" {
		t.Errorf("unexpected styles: %v", m.Inject.Styles)
	}
}

func TestLoadManifestUnknownField(t *testing.T) {
	dir := writeManifest(t, filepath.Join(t.TempDir(), "x"), "name: X\nslug: x\nversion: 1.0.0\nbogus_field: 1\n")
	if _, err := LoadManifest(dir); err == nil {
		t.Fatal("expected error for unknown manifest field")
	}
}

func TestManifestValidate(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		dirName string
		wantErr string
	}{
		{
			name:    "missing name",
			yaml:    "slug: x\nversion: 1.0.0\n",
			dirName: "x",
			wantErr: "name",
		},
		{
			name:    "missing slug",
			yaml:    "name: X\nversion: 1.0.0\n",
			dirName: "x",
			wantErr: "slug",
		},
		{
			name:    "missing version",
			yaml:    "name: X\nslug: x\n",
			dirName: "x",
			wantErr: "version",
		},
		{
			name:    "slug dir mismatch",
			yaml:    "name: X\nslug: x\nversion: 1.0.0\n",
			dirName: "other",
			wantErr: "does not match",
		},
		{
			name:    "invalid slug characters",
			yaml:    "name: X\nslug: X.Y\nversion: 1.0.0\n",
			dirName: "X.Y",
			wantErr: "invalid slug",
		},
		{
			name:    "unknown when rule",
			yaml:    "name: X\nslug: x\nversion: 1.0.0\ninject:\n  when: sometimes\n",
			dirName: "x",
			wantErr: "unknown inject.when",
		},
		{
			name:    "layout rule missing layout",
			yaml:    "name: X\nslug: x\nversion: 1.0.0\ninject:\n  when: layout\n",
			dirName: "x",
			wantErr: "requires inject.layout",
		},
		{
			name:    "layout rule bad layout",
			yaml:    "name: X\nslug: x\nversion: 1.0.0\ninject:\n  when: layout\n  layout: nope\n",
			dirName: "x",
			wantErr: "unknown inject.layout",
		},
		{
			name:    "collection rule missing collection",
			yaml:    "name: X\nslug: x\nversion: 1.0.0\ninject:\n  when: collection\n",
			dirName: "x",
			wantErr: "requires inject.collection",
		},
		{
			name:    "style path traversal",
			yaml:    "name: X\nslug: x\nversion: 1.0.0\ninject:\n  styles: [../../etc/passwd]\n",
			dirName: "x",
			wantErr: "..",
		},
		{
			name:    "absolute style path",
			yaml:    "name: X\nslug: x\nversion: 1.0.0\ninject:\n  styles: [/etc/passwd]\n",
			dirName: "x",
			wantErr: "relative",
		},
		{
			name:    "output prefix traversal",
			yaml:    "name: X\nslug: x\nversion: 1.0.0\noutput:\n  prefix: ../outside/\n",
			dirName: "x",
			wantErr: "..",
		},
		{
			name:    "output include traversal",
			yaml:    "name: X\nslug: x\nversion: 1.0.0\noutput:\n  include: [../js/]\n",
			dirName: "x",
			wantErr: "..",
		},
		{
			name:    "valid minimal",
			yaml:    "name: X\nslug: x\nversion: 1.0.0\n",
			dirName: "x",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := writeManifest(t, filepath.Join(t.TempDir(), tt.dirName), tt.yaml)
			m, err := LoadManifest(dir)
			if err != nil {
				t.Fatalf("LoadManifest: %v", err)
			}
			err = m.Validate(tt.dirName)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("Validate: unexpected error %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("Validate: expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Validate error %q does not contain %q", err, tt.wantErr)
			}
		})
	}
}

func TestManifestWhenDefaultsToAlwaysWithAssets(t *testing.T) {
	m := &Manifest{Name: "X", Slug: "x", Version: "1.0.0"}
	m.Inject.Styles = []string{"css/x.css"}
	if err := m.Validate("x"); err != nil {
		t.Fatal(err)
	}
	if m.Inject.When != "always" {
		t.Errorf("expected When normalized to 'always', got %q", m.Inject.When)
	}
}

func TestEffectivePrefix(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		want   string
	}{
		{"default", "", "assets/vendor/x/"},
		{"custom with trailing slash", "assets/vendor/custom/", "assets/vendor/custom/"},
		{"custom without trailing slash", "assets/vendor/custom", "assets/vendor/custom/"},
		{"leading slash stripped", "/assets/vendor/custom/", "assets/vendor/custom/"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := &Manifest{Slug: "x"}
			m.Output.Prefix = tt.prefix
			if got := m.EffectivePrefix(); got != tt.want {
				t.Errorf("EffectivePrefix() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestEffectivePrefix_SnakeSlugKebabized: identifiers are snake_case, URLs are
// kebab-case; the default vendor prefix kebab-izes the slug, while an explicit
// output.prefix passes through untouched.
func TestEffectivePrefix_SnakeSlugKebabized(t *testing.T) {
	m := &Manifest{Slug: "my_widget"}
	if got := m.EffectivePrefix(); got != "assets/vendor/my-widget/" {
		t.Errorf("EffectivePrefix() = %q, want assets/vendor/my-widget/", got)
	}

	m.Output.Prefix = "assets/vendor/my_widget/"
	if got := m.EffectivePrefix(); got != "assets/vendor/my_widget/" {
		t.Errorf("explicit output.prefix must pass through untouched, got %q", got)
	}
}

func TestIncludeFilter(t *testing.T) {
	m := &Manifest{Slug: "x"}
	if m.IncludeFilter() != nil {
		t.Fatal("expected nil filter with no include entries")
	}

	m.Output.Include = []string{"js/", "css/main.css", "vendor"}
	filter := m.IncludeFilter()
	tests := []struct {
		rel  string
		want bool
	}{
		{"js/app.js", true},
		{"js/sub/deep.js", true},
		{"css/main.css", true},
		{"css/other.css", false},
		{"vendor/lib.js", true},
		{"vendor", true},
		{"fonts/x.woff2", false},
		{"jsx/app.js", false},
	}
	for _, tt := range tests {
		if got := filter(tt.rel); got != tt.want {
			t.Errorf("filter(%q) = %v, want %v", tt.rel, got, tt.want)
		}
	}
}
