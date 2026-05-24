package taxonomy

import (
	"errors"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/frostybee/sarde/internal/consts"
)

// TermMetadata holds optional per-term configuration from data/<taxonomy>.yml.
type TermMetadata struct {
	Label       string `yaml:"label"`
	Description string `yaml:"description"`
	Color       string `yaml:"color"`
	Icon        string `yaml:"icon"`
	Hidden      bool   `yaml:"hidden"`
	Priority    int    `yaml:"priority"`
}

// LoadTermMetadata reads data/<taxonomyName>.yml from the project data dir.
// Returns a map of slug to TermMetadata. A missing file returns an empty map.
func LoadTermMetadata(projectDir, taxonomyName string) (map[string]TermMetadata, error) {
	path := filepath.Join(projectDir, consts.DirData, taxonomyName+".yml")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var meta map[string]TermMetadata
	if err := yaml.Unmarshal(data, &meta); err != nil {
		return nil, err
	}
	return meta, nil
}
