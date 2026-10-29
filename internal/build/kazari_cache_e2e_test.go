package build

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/config"
)

// Changing a Kazari option that alters only the code block HTML, such as
// lineNumbers, must re-render pages served from the page cache. The code
// block CSS does not change, so a renderer fingerprint built from the CSS
// alone kept the cached HTML.
func TestBuild_KazariConfigChangeRerendersCachedPages(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "content/_index.md", "---\ntitle: Home\n---\n# Home\n")
	writeFixture(t, dir, "content/docs/guide.md", "---\ntitle: Guide\n---\n```go\nfunc main() {\n\tprintln(\"hi\")\n}\n```\n")
	kazariFile := filepath.Join(dir, "kazari.config.yaml")

	build := func(lineNumbers string) string {
		t.Helper()
		if err := os.WriteFile(kazariFile, []byte("lineNumbers: "+lineNumbers+"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		cfg := config.Defaults()
		cfg.Build.Minify = config.BoolPtr(false)
		builder := NewSiteBuilder(BuildOptions{
			ProjectDir:  dir,
			Config:      cfg,
			ThemeConfig: buildThemeConfig(),
			EmbeddedFS:  embedded.ThemeFS(),
		})
		if _, err := builder.Build(); err != nil {
			t.Fatalf("Build (lineNumbers: %s): %v", lineNumbers, err)
		}
		return readFixture(t, filepath.Join(dir, "dist"), "docs/guide/index.html")
	}

	if html := build("false"); strings.Contains(html, `class="kz-ln"`) {
		t.Fatal("line numbers rendered with lineNumbers: false")
	}
	if html := build("true"); !strings.Contains(html, `class="kz-ln"`) {
		t.Error("the second build served the cached page: no line numbers after setting lineNumbers: true")
	}
}
