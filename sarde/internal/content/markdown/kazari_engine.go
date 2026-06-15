package markdown

import (
	"context"
	"fmt"

	"github.com/frostybee/kazari"
	kazarinuri "github.com/frostybee/kazari/nuri"
	"github.com/frostybee/nuri"
	"github.com/frostybee/nuri/bundle/core"
	"github.com/frostybee/sarde/internal/config"
)

// BuildKazariEngine creates a Kazari code block engine backed by the Nuri
// TextMate tokenizer. Called once per build; the engine is reused across
// incremental rebuilds.
func BuildKazariEngine(ctx context.Context, cfg *config.CodeblocksSettings) (*kazari.Engine, error) {
	nuriHL, err := nuri.New(ctx, nuri.WithFS(core.FS()))
	if err != nil {
		return nil, fmt.Errorf("kazari: nuri init: %w", err)
	}

	darkModeSelector := cfg.DarkModeSelector
	if darkModeSelector == "" {
		darkModeSelector = ".dark"
	}

	engine := kazari.New(
		kazari.WithHighlighter(kazarinuri.New(ctx, nuriHL)),
		kazari.WithThemes(cfg.LightTheme, cfg.DarkTheme),
		kazari.WithDarkMode(kazari.SelectorMode(darkModeSelector)),
		kazari.WithMermaidPassThrough(true),
	)
	return engine, nil
}
