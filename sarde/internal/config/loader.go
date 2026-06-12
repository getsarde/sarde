package config

import (
	"errors"
	"fmt"
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

// LoadFileStrict reads a YAML config file with unknown-field detection enabled.
// Any field in the YAML that doesn't map to a SiteConfig struct field causes
// an error. Used only for the user's sarde.yaml (layer 3).
func LoadFileStrict(path string) (*SiteConfig, error) {
	f, err := os.Open(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	cfg := &SiteConfig{}
	dec := yaml.NewDecoder(f)
	dec.KnownFields(true)
	if err := dec.Decode(cfg); err != nil {
		return nil, fmt.Errorf("sarde.yaml: %w", err)
	}
	return cfg, nil
}
