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

// kazariCSS builds a chroma engine for dir and returns its CSS.
func kazariCSS(t *testing.T, cfg *config.CodeblocksSettings, dir string) string {
	t.Helper()
	cfg.Engine = "chroma"
	eng, err := BuildKazariEngine(context.Background(), cfg, dir)
	if err != nil {
		t.Fatal(err)
	}
	return eng.CSS()
}

// kazariDir returns a project directory with the given kazari.config.yaml.
func kazariDir(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "kazari.config.yaml"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

const fileThemes = "themes:\n  light: dracula\n  dark: monokai\n"

// The themes entry in kazari.config.yaml wins over the deprecated
// light_theme and dark_theme in sarde.yaml. Theme names do not appear in the
// CSS, so the result is compared with an engine built without the file.
func TestBuildKazariEngine_ConfigFileThemesWin(t *testing.T) {
	file := kazariCSS(t, &config.CodeblocksSettings{LightTheme: "dracula", DarkTheme: "monokai"}, t.TempDir())
	sarde := kazariCSS(t, &config.CodeblocksSettings{LightTheme: "github-light", DarkTheme: "github-dark"}, t.TempDir())
	if file == sarde {
		t.Fatal("the two theme pairs render the same CSS; the test cannot tell them apart")
	}

	got := kazariCSS(t, &config.CodeblocksSettings{LightTheme: "github-light", DarkTheme: "github-dark"}, kazariDir(t, fileThemes))
	if got != file {
		t.Error("sarde.yaml themes override the themes entry in kazari.config.yaml")
	}
}

// Without a themes entry in kazari.config.yaml, light_theme and dark_theme
// from sarde.yaml still apply.
func TestBuildKazariEngine_SardeThemesApplyWithoutFileThemes(t *testing.T) {
	want := kazariCSS(t, &config.CodeblocksSettings{LightTheme: "dracula", DarkTheme: "monokai"}, t.TempDir())
	got := kazariCSS(t, &config.CodeblocksSettings{LightTheme: "dracula", DarkTheme: "monokai"}, kazariDir(t, "copyButton: true\n"))
	if got != want {
		t.Error("sarde.yaml themes do not apply when kazari.config.yaml sets none")
	}
}

func TestEffectiveCodeThemes(t *testing.T) {
	cfg := &config.CodeblocksSettings{LightTheme: "github-light", DarkTheme: "github-dark"}
	cases := []struct {
		name                string
		dir                 string
		wantLight, wantDark string
	}{
		{"no file", t.TempDir(), "github-light", "github-dark"},
		{"file with themes", kazariDir(t, fileThemes), "dracula", "monokai"},
		{"file without themes", kazariDir(t, "copyButton: true\n"), "github-light", "github-dark"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			light, dark := EffectiveCodeThemes(cfg, tc.dir)
			if light != tc.wantLight || dark != tc.wantDark {
				t.Errorf("EffectiveCodeThemes = %q, %q, want %q, %q", light, dark, tc.wantLight, tc.wantDark)
			}
		})
	}
}
