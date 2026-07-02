package server

import (
	"path/filepath"
	"testing"
	"time"
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
		{"css in assets", "/project/assets/main.css", ChangeStatic},
		{"js in static", "/project/static/app.js", ChangeStatic},
		{"site config", "/project/sarde.yaml", ChangeConfig},
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
		{"config file", "/project/sarde.yaml", false},
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

func TestShouldIgnore_CustomOutputDir(t *testing.T) {
	// A non-default build.output (e.g. "www") must be ignored so the build's
	// own writes don't trigger a rebuild loop.
	w := &Watcher{projectDir: "/project", outputDir: "www"}

	tests := []struct {
		name   string
		path   string
		ignore bool
	}{
		{"custom output file", "/project/www/index.html", true},
		{"custom output nested", "/project/www/blog/post/index.html", true},
		{"default dist still ignored", "/project/dist/index.html", true},
		{"content not ignored", "/project/content/blog/hello.md", false},
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

func TestShouldIgnoreDir_OutputDirMatchedByPath(t *testing.T) {
	// The output dir must be matched by path relative to the project, not by
	// bare name: a same-named directory under content/ must stay watched.
	w := &Watcher{projectDir: filepath.FromSlash("/project"), outputDir: "site"}

	tests := []struct {
		name   string
		path   string
		ignore bool
	}{
		{"output dir itself", "/project/site", true},
		{"nested under output", "/project/site/blog", true},
		{"same name under content", "/project/content/site", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := w.shouldIgnoreDir(filepath.FromSlash(tt.path))
			if got != tt.ignore {
				t.Errorf("shouldIgnoreDir(%q) = %v, want %v", tt.path, got, tt.ignore)
			}
		})
	}
}

func TestIsRootLevelNonConfig(t *testing.T) {
	w := &Watcher{projectDir: filepath.FromSlash("/project")}

	tests := []struct {
		name string
		path string
		skip bool
	}{
		{"site config at root", "/project/sarde.yaml", false},
		{"theme config at root", "/project/theme.yaml", false},
		{"nav config at root", "/project/nav.yaml", false},
		{"stray file at root", "/project/README.md", true},
		{"watched dir entry at root", "/project/content", true},
		{"file inside subdir", "/project/content/blog/post.md", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := w.isRootLevelNonConfig(filepath.FromSlash(tt.path))
			if got != tt.skip {
				t.Errorf("isRootLevelNonConfig(%q) = %v, want %v", tt.path, got, tt.skip)
			}
		})
	}
}

func TestMergeChanges_ContentOnlyUnionsPaths(t *testing.T) {
	got := mergeChanges([]FileChange{
		{Path: "/project/content/a.md", Kind: ChangeContent},
		{Path: "/project/content/b.md", Kind: ChangeContent, Paths: []string{"/project/content/b.md", "/project/content/a.md"}},
	})

	if got.Kind != ChangeContent {
		t.Fatalf("Kind = %q, want %q", got.Kind, ChangeContent)
	}
	if len(got.Paths) != 2 {
		t.Fatalf("Paths = %#v, want deduped union of 2 content paths", got.Paths)
	}
}

func TestMergeChanges_ContentPlusCSSEscalatesToStatic(t *testing.T) {
	// The incremental content path would drop the CSS file (only a full build
	// copies static files), so mixed batches must escalate to a full build.
	got := mergeChanges([]FileChange{
		{Path: "/project/static/style.css", Kind: ChangeCSS},
		{Path: "/project/content/blog/post.md", Kind: ChangeContent},
	})

	if got.Kind != ChangeStatic {
		t.Fatalf("Kind = %q, want %q (escalated full build)", got.Kind, ChangeStatic)
	}
}

func TestMergeChanges_ContentPlusStaticEscalatesToStatic(t *testing.T) {
	got := mergeChanges([]FileChange{
		{Path: "/project/content/blog/post.md", Kind: ChangeContent},
		{Path: "/project/static/logo.png", Kind: ChangeStatic},
	})

	if got.Kind != ChangeStatic {
		t.Fatalf("Kind = %q, want %q (escalated full build)", got.Kind, ChangeStatic)
	}
}

func TestMergeChanges_ConfigBeatsContent(t *testing.T) {
	got := mergeChanges([]FileChange{
		{Path: "/project/content/blog/post.md", Kind: ChangeContent},
		{Path: "/project/sarde.yaml", Kind: ChangeConfig},
	})

	if got.Kind != ChangeConfig {
		t.Fatalf("Kind = %q, want %q", got.Kind, ChangeConfig)
	}
	if len(got.Paths) != 1 || got.Paths[0] != "/project/content/blog/post.md" {
		t.Fatalf("Paths = %#v, want collected content path", got.Paths)
	}
}

func TestMergeChanges_KeepsEarliestDetectedAt(t *testing.T) {
	early := time.Now().Add(-2 * time.Second)
	late := time.Now()
	// The earliest timestamp sits on the lower-priority change; a higher
	// priority change arriving later must not discard it.
	got := mergeChanges([]FileChange{
		{Path: "/project/static/logo.png", Kind: ChangeStatic, DetectedAt: early},
		{Path: "/project/layouts/base.html", Kind: ChangeTemplate, DetectedAt: late},
	})

	if got.Kind != ChangeTemplate {
		t.Fatalf("Kind = %q, want %q", got.Kind, ChangeTemplate)
	}
	if !got.DetectedAt.Equal(early) {
		t.Fatalf("DetectedAt = %v, want earliest %v", got.DetectedAt, early)
	}
}

func TestWatcherStop_CancelsPendingDebounce(t *testing.T) {
	fired := make(chan struct{}, 1)
	w := NewWatcher(t.TempDir(), "", 30*time.Millisecond, func([]FileChange) {
		fired <- struct{}{}
	})

	// Arm the debounce timer, then stop before it fires.
	w.debounceChange(FileChange{Path: "content/a.md", Kind: ChangeContent})
	w.Stop()

	select {
	case <-fired:
		t.Fatal("onChange fired after Stop")
	case <-time.After(100 * time.Millisecond):
	}
}
