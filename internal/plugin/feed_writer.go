package plugin

import (
	"encoding/xml"
	"fmt"

	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/plugin/cfgutil"
)

type feedBuilder func(col *engine.Collection, baseURL string, limit int) ([]byte, error)

func writeFeedFiles(ctx *BuildDoneContext, cfg map[string]any, filename, label string, build feedBuilder) error {
	limit := cfgutil.Int(cfg, "limit", 20)
	baseURL := ctx.BaseURL()

	feedCollections := feedEnabledCollections(cfg, ctx.Collections)

	feedCount := 0
	for _, colName := range feedCollections {
		col, ok := ctx.Collections[colName]
		if !ok || col == nil {
			continue
		}

		data, err := build(col, baseURL, limit)
		if err != nil {
			return fmt.Errorf("marshaling %s for %s: %w", label, colName, err)
		}

		output := []byte(xml.Header + string(data))
		path := colName + "/" + filename
		if err := ctx.WriteFile(path, output); err != nil {
			return err
		}
		feedCount++
	}

	if feedCount > 0 {
		ctx.Log(fmt.Sprintf("Generated %d %s feed(s)", feedCount, label))
	}
	return nil
}
