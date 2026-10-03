package build

import (
	"os"
	"path/filepath"
	"testing"
)

func TestWarnKazariDarkMode(t *testing.T) {
	const selector = `[data-theme="dark"]`
	tests := []struct {
		name string
		file string // file name in the project root; empty for none
		body string
		warn bool
	}{
		{"no config file", "", "", false},
		{"no darkMode entry", "kazari.config.yaml", "copyButton: true\n", false},
		{"matching selector", "kazari.config.yaml", "darkMode:\n  kind: selector\n  selector: '[data-theme=\"dark\"]'\n", false},
		{"matching selector without quotes", "kazari.config.yaml", "darkMode:\n  kind: selector\n  selector: '[data-theme=dark]'\n", false},
		{"old scaffold selector", "kazari.config.yaml", "darkMode:\n  kind: selector\n  selector: \".dark\"\n", true},
		{"media query", "kazari.config.yml", "darkMode:\n  kind: mediaQuery\n", true},
		{"json file", "kazari.config.json", `{"darkMode": {"kind": "selector", "selector": ".dark"}}`, true},
		{"unparseable file", "kazari.config.yaml", "darkMode: [\n", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.file != "" {
				if err := os.WriteFile(filepath.Join(dir, tt.file), []byte(tt.body), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			got := warnKazariDarkMode(dir, selector)
			if (len(got) > 0) != tt.warn {
				t.Fatalf("warnings = %+v, want warning: %v", got, tt.warn)
			}
			if tt.warn && got[0].File != tt.file {
				t.Errorf("warning names %q, want %q", got[0].File, tt.file)
			}
		})
	}
}
