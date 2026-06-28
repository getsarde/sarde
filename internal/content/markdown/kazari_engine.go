package markdown

import (
	"context"
	"fmt"

	"github.com/frostybee/kazari"
	kazarichroma "github.com/frostybee/kazari/chroma"
	kazarinuri "github.com/frostybee/kazari/nuri"
	"github.com/frostybee/nuri"
	"github.com/frostybee/nuri/bundle/core"
	"github.com/getsarde/sarde/internal/config"
)

// nuriToChroma maps Nuri/TextMate theme names to their closest Chroma
// style equivalents. Only names that differ are listed; themes with
// identical names (dracula, nord, github-dark, etc.) resolve directly.
var nuriToChroma = map[string]string{
	"github-light":               "github",
	"github-light-default":       "github",
	"github-light-high-contrast": "github",
	"github-dark-default":        "github-dark",
	"github-dark-dimmed":         "github-dark",
	"github-dark-high-contrast":  "github-dark",
	"gruvbox-dark-hard":          "gruvbox",
	"gruvbox-dark-medium":        "gruvbox",
	"gruvbox-dark-soft":          "gruvbox",
	"gruvbox-light-hard":         "gruvbox-light",
	"gruvbox-light-medium":       "gruvbox-light",
	"gruvbox-light-soft":         "gruvbox-light",
	"one-dark-pro":               "onedark",
	"tokyo-night":                "tokyonight-night",
}

// BuildKazariEngine creates a Kazari code block engine. The highlighter
// backend is selected by cfg.Engine: "chroma" for fast Go-native
// highlighting (ideal for dev/live-reload), "nuri" (default) for accurate
// TextMate-based highlighting matching VS Code output.
func BuildKazariEngine(ctx context.Context, cfg *config.CodeblocksSettings, projectDir string) (*kazari.Engine, error) {
	var hl kazari.Highlighter

	switch cfg.Engine {
	case "chroma":
		hl = kazarichroma.New(kazarichroma.WithStyleMap(nuriToChroma))
	default:
		nuriHL, err := nuri.New(ctx, nuri.WithFS(core.FS()))
		if err != nil {
			return nil, fmt.Errorf("kazari: nuri init: %w", err)
		}
		hl = kazarinuri.New(ctx, nuriHL)
	}

	darkModeSelector := cfg.DarkModeSelector
	if darkModeSelector == "" {
		darkModeSelector = "[data-theme=\"dark\"]"
	}

	engine := kazari.New(
		kazari.WithHighlighter(hl),
		kazari.WithThemes(cfg.LightTheme, cfg.DarkTheme),
		kazari.WithDarkMode(kazari.SelectorMode(darkModeSelector)),
		kazari.WithMermaidPassThrough(true),
		kazari.WithConfigDir(projectDir),
	)
	return engine, nil
}
