package template

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/getsarde/sarde/internal/consts"
)

func fakeEmbeddedCSS() fstest.MapFS {
	return fstest.MapFS{
		"css/tokens.css": {Data: []byte("/* embedded tokens */")},
		"css/base.css":   {Data: []byte("/* embedded base */")},
		"css/blog.css":   {Data: []byte("/* embedded blog */")},
	}
}

func writeThemeCSS(t *testing.T, projectDir, themeName, rel, content string) {
	t.Helper()
	p := filepath.Join(projectDir, consts.DirThemes, themeName, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestAssembleThemeCSS_NoThemeUsesEmbedded(t *testing.T) {
	got := assembleThemeCSS(fakeEmbeddedCSS(), t.TempDir(), "")
	for _, want := range []string{cssLayerPrefix, "/* embedded tokens */", "/* embedded base */", "/* embedded blog */"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in bundle", want)
		}
	}
}

func TestAssembleThemeCSS_PartialThemeKeepsEmbeddedRest(t *testing.T) {
	dir := t.TempDir()
	writeThemeCSS(t, dir, "magazine", "css/blog.css", "/* theme blog */")

	got := assembleThemeCSS(fakeEmbeddedCSS(), dir, "magazine")

	if !strings.Contains(got, "/* theme blog */") {
		t.Error("theme blog.css not used")
	}
	if strings.Contains(got, "/* embedded blog */") {
		t.Error("embedded blog.css should be replaced by the theme copy")
	}
	if !strings.Contains(got, "/* embedded tokens */") || !strings.Contains(got, "/* embedded base */") {
		t.Error("stylesheets missing from the theme must fall back to the embedded copies")
	}
}

func TestAssembleThemeCSS_OrderFollowsCSSOrder(t *testing.T) {
	dir := t.TempDir()
	writeThemeCSS(t, dir, "magazine", "css/tokens.css", "/* theme tokens */")

	got := assembleThemeCSS(fakeEmbeddedCSS(), dir, "magazine")

	iTokens := strings.Index(got, "/* theme tokens */")
	iBase := strings.Index(got, "/* embedded base */")
	iBlog := strings.Index(got, "/* embedded blog */")
	if iTokens < 0 || iBase < 0 || iBlog < 0 || !(iTokens < iBase && iBase < iBlog) {
		t.Errorf("bundle order wrong: tokens=%d base=%d blog=%d", iTokens, iBase, iBlog)
	}
}

func TestAssembleThemeCSS_ThemeDirIgnoredWhenNameEmpty(t *testing.T) {
	dir := t.TempDir()
	writeThemeCSS(t, dir, "default", "css/blog.css", "/* theme blog */")

	got := assembleThemeCSS(fakeEmbeddedCSS(), dir, "")
	if strings.Contains(got, "/* theme blog */") {
		t.Error("theme files must not be read when no theme name is set")
	}
}

func TestAssembleThemeCSS_NothingReadableReturnsEmpty(t *testing.T) {
	if got := assembleThemeCSS(nil, t.TempDir(), ""); got != "" {
		t.Errorf("expected empty bundle, got %q", got)
	}
}
