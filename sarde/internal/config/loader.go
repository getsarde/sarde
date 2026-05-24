package config

import (
	"errors"
	"io/fs"
	"os"

	"gopkg.in/yaml.v3"
)

// LoadFile reads a YAML config file and unmarshals it into a SiteConfig.
// Returns (nil, nil) if the file does not exist — missing config is not an
// error under the zero-config philosophy. Returns (nil, error) for invalid YAML.
func LoadFile(path string) (*SiteConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	cfg := &SiteConfig{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}
