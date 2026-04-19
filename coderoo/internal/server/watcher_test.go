package server

import (
	"path/filepath"
	"testing"
)

func TestClassifyChange(t *testing.T) {
	w := &Watcher{projectDir: "/project"}

	tests := []struct {
		name string
		path string
		want ChangeKind
	}{
		{"content markdown", "/project/content/blog/hello.md", ChangeContent},
		{"content index", "/project/content/_index.md", ChangeContent},
		{"layout template", "/project/layouts/blog/single.html", ChangeTemplate},
		{"theme file", "/project/themes/default/layouts/base.html", ChangeTemplate},
		{"css in static", "/project/static/style.css", ChangeCSS},
		{"css in assets", "/project/assets/main.css", ChangeCSS},
		{"js in static", "/project/static/app.js", ChangeStatic},
		{"site config", "/project/site.yaml", ChangeConfig},
		{"theme config", "/project/theme.yaml", ChangeConfig},
		{"nav config", "/project/nav.yaml", ChangeConfig},
		{"data file", "/project/data/menu.yaml", ChangeStatic},
		{"random file", "/project/readme.txt", ChangeStatic},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := w.classifyChange(filepath.FromSlash(tt.path))
			if got != tt.want {
				t.Errorf("classifyChange(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestShouldIgnore(t *testing.T) {
	w := &Watcher{projectDir: "/project"}

	tests := []struct {
		name   string
		path   string
		ignore bool
	}{
		{"git dir", "/project/.git/HEAD", true},
		{"cache dir", "/project/.cache/data", true},
		{"dist dir", "/project/dist/index.html", true},
		{"public dir", "/project/public/index.html", true},
		{"hidden file", "/project/content/.DS_Store", true},
		{"swap file", "/project/content/hello.md.swp", true},
		{"tilde file", "/project/content/hello.md~", true},
		{"tmp file", "/project/content/hello.tmp", true},
		{"normal content", "/project/content/blog/hello.md", false},
		{"layout file", "/project/layouts/base.html", false},
		{"config file", "/project/site.yaml", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := w.shouldIgnore(filepath.FromSlash(tt.path))
			if got != tt.ignore {
				t.Errorf("shouldIgnore(%q) = %v, want %v", tt.path, got, tt.ignore)
			}
		})
	}
}

func TestShouldIgnoreDir(t *testing.T) {
	w := &Watcher{}

	tests := []struct {
		name   string
		dir    string
		ignore bool
	}{
		{"git", ".git", true},
		{"cache", ".cache", true},
		{"dist", "dist", true},
		{"public", "public", true},
		{"node_modules", "node_modules", true},
		{"hidden", ".hidden", true},
		{"content", "content", false},
		{"layouts", "layouts", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := w.shouldIgnoreDir(tt.dir)
			if got != tt.ignore {
				t.Errorf("shouldIgnoreDir(%q) = %v, want %v", tt.dir, got, tt.ignore)
			}
		})
	}
}
