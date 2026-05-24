package theme

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/frostybee/sarde/internal/consts"
	"gopkg.in/yaml.v3"
)

// LoadFromFS loads a theme from an fs.FS (typically the embedded filesystem).
// path is the directory within the FS containing theme.yaml.
// Returns (nil, nil) if theme.yaml does not exist (zero-config behavior).
func LoadFromFS(fsys fs.FS, path string) (*Theme, error) {
	themeFile := path + "/" + consts.FileThemeConfig
	if path == "." || path == "" {
		themeFile = consts.FileThemeConfig
	}

	data, err := fs.ReadFile(fsys, themeFile)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading theme.yaml from FS: %w", err)
	}

	return parseTheme(data)
}

// LoadFromDir loads a theme from a filesystem directory.
// Returns (nil, nil) if theme.yaml does not exist.
func LoadFromDir(dir string) (*Theme, error) {
	path := filepath.Join(dir, consts.FileThemeConfig)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading theme.yaml: %w", err)
	}

	return parseTheme(data)
}

// parseTheme unmarshals YAML data into a Theme struct.
func parseTheme(data []byte) (*Theme, error) {
	var t Theme
	if err := yaml.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("parsing theme.yaml: %w", err)
	}
	return &t, nil
}
