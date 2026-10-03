package markdown

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getsarde/sarde/internal/config"
)

// A darkMode entry in kazari.config.yaml must not override Sarde's selector:
// code blocks have to switch with the theme toggle, which sets data-theme on
// <html>. Sites scaffolded before this fix carry selector ".dark".
func TestBuildKazariEngine_DarkModeIgnoresConfigFile(t *testing.T) {
	dir := t.TempDir()
	cfgFile := "darkMode:\n  kind: selector\n  selector: \".dark\"\n"
	if err := os.WriteFile(filepath.Join(dir, "kazari.config.yaml"), []byte(cfgFile), 0o644); err != nil {
		t.Fatal(err)
	}

	eng, err := BuildKazariEngine(context.Background(), &config.CodeblocksSettings{
		Engine:     "chroma",
		LightTheme: "github-light",
		DarkTheme:  "github-dark",
	}, dir)
	if err != nil {
		t.Fatal(err)
	}

	css := eng.CSS()
	// Kazari writes the selector minified, without the quotes.
	if !strings.Contains(css, `[data-theme=dark] .kazari-block`) && !strings.Contains(css, `[data-theme="dark"] .kazari-block`) {
		t.Error("code block CSS does not scope dark mode to the theme toggle's attribute")
	}
	if strings.Contains(css, ".dark .kazari-block") {
		t.Error("code block CSS still uses the .dark selector from kazari.config.yaml")
	}
}

func TestDarkModeSelector(t *testing.T) {
	if got := DarkModeSelector(&config.CodeblocksSettings{}); got != `[data-theme="dark"]` {
		t.Errorf("default = %q", got)
	}
	if got := DarkModeSelector(&config.CodeblocksSettings{DarkModeSelector: ".night"}); got != ".night" {
		t.Errorf("configured = %q", got)
	}
}
