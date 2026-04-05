package template

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	toml "github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

// loadDataFile loads a data file from the project's data/ directory.
// It tries extensions in order: .yaml, .yml, .json, .toml.
// Results are cached in the provided sync.Map for the lifetime of the build.
// Returns nil (not error) if the file is not found — templates can check for nil.
func loadDataFile(projectDir, name string, cache *sync.Map) any {
	if projectDir == "" {
		return nil
	}

	// Check cache first.
	if v, ok := cache.Load(name); ok {
		return v
	}

	dataDir := filepath.Join(projectDir, "data")
	extensions := []struct {
		ext    string
		parser func([]byte) (any, error)
	}{
		{".yaml", parseYAML},
		{".yml", parseYAML},
		{".json", parseJSON},
		{".toml", parseTOML},
	}

	for _, ext := range extensions {
		path := filepath.Join(dataDir, name+ext.ext)
		raw, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		result, err := ext.parser(raw)
		if err != nil {
			// Cache the error as nil so we don't retry on every template invocation.
			cache.Store(name, nil)
			return nil
		}
		cache.Store(name, result)
		return result
	}

	// Not found — cache nil.
	cache.Store(name, nil)
	return nil
}

func parseYAML(raw []byte) (any, error) {
	var result any
	if err := yaml.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("YAML parse: %w", err)
	}
	return result, nil
}

func parseJSON(raw []byte) (any, error) {
	var result any
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("JSON parse: %w", err)
	}
	return result, nil
}

func parseTOML(raw []byte) (any, error) {
	var result any
	if err := toml.Unmarshal(raw, &result); err != nil {
		return nil, fmt.Errorf("TOML parse: %w", err)
	}
	return result, nil
}
