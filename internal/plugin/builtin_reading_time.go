package plugin

import (
	"math"
	"strings"
)

func newReadingTimePlugin(cfg map[string]any) *Plugin {
	return &Plugin{
		Name: "reading_time",
		Hooks: PluginHooks{
			ContentLoaded: func(ctx *ContentLoadedContext) error {
				return readingTimeContentLoaded(ctx, cfg)
			},
		},
	}
}

func readingTimeContentLoaded(ctx *ContentLoadedContext, cfg map[string]any) error {
	wpm := cfgInt(cfg, "wpm", 200)

	// Default WPM matches the transformer — no work needed.
	if wpm == 200 {
		return nil
	}

	// Re-compute reading time with custom WPM.
	for _, page := range *ctx.Pages {
		if page.WordCount > 0 {
			page.ReadingTime = int(math.Ceil(float64(page.WordCount) / float64(wpm)))
			if page.ReadingTime < 1 {
				page.ReadingTime = 1
			}
		}
	}

	return nil
}

// countWords counts words in text (simple whitespace split).
// Used only if WordCount isn't already set.
func countWords(text string) int {
	return len(strings.Fields(text))
}
