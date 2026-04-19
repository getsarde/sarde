package plugin

import (
	"testing"

	"github.com/coderoo-dev/coderoo/internal/config"
	"github.com/coderoo-dev/coderoo/internal/engine"
)

func TestReadingTime_DefaultWPM(t *testing.T) {
	pages := []*engine.Page{
		{Title: "Test", WordCount: 400, ReadingTime: 2},
	}

	cfg := config.Defaults()
	ctx := &ContentLoadedContext{
		Config: cfg,
		Pages:  &pages,
		store:  NewStore(),
	}

	// Default WPM (200) — should be a no-op.
	err := readingTimeContentLoaded(ctx, nil)
	if err != nil {
		t.Fatalf("readingTimeContentLoaded failed: %v", err)
	}

	if pages[0].ReadingTime != 2 {
		t.Errorf("ReadingTime = %d, want 2 (unchanged)", pages[0].ReadingTime)
	}
}

func TestReadingTime_CustomWPM(t *testing.T) {
	pages := []*engine.Page{
		{Title: "Test", WordCount: 500, ReadingTime: 3},
	}

	cfg := config.Defaults()
	ctx := &ContentLoadedContext{
		Config: cfg,
		Pages:  &pages,
		store:  NewStore(),
	}

	// 250 WPM: 500 words / 250 = 2 minutes.
	pluginCfg := map[string]any{"wpm": 250}
	err := readingTimeContentLoaded(ctx, pluginCfg)
	if err != nil {
		t.Fatalf("readingTimeContentLoaded failed: %v", err)
	}

	if pages[0].ReadingTime != 2 {
		t.Errorf("ReadingTime = %d, want 2", pages[0].ReadingTime)
	}
}

func TestReadingTime_MinimumOne(t *testing.T) {
	pages := []*engine.Page{
		{Title: "Short", WordCount: 10, ReadingTime: 1},
	}

	cfg := config.Defaults()
	ctx := &ContentLoadedContext{
		Config: cfg,
		Pages:  &pages,
		store:  NewStore(),
	}

	pluginCfg := map[string]any{"wpm": 1000}
	readingTimeContentLoaded(ctx, pluginCfg)

	if pages[0].ReadingTime != 1 {
		t.Errorf("ReadingTime = %d, want 1 (minimum)", pages[0].ReadingTime)
	}
}
