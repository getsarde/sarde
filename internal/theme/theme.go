// Package theme handles theme discovery, loading, token resolution, and CSS generation.
package theme

// Theme represents a loaded theme with its configuration and token definitions.
type Theme struct {
	Name        string            `yaml:"name"`
	Slug        string            `yaml:"slug"`
	Version     string            `yaml:"version"`
	Author      string            `yaml:"author"`
	Description string            `yaml:"description"`
	License     string            `yaml:"license"`
	Tokens      map[string]string `yaml:"tokens"`
	DarkTokens  map[string]string `yaml:"dark_tokens"`
	Presets     map[string]Preset `yaml:"presets"`
}

// Preset represents a theme preset variation (e.g., "ocean", "forest").
type Preset struct {
	Name       string            `yaml:"name"`
	Tokens     map[string]string `yaml:"tokens"`
	DarkTokens map[string]string `yaml:"dark_tokens"`
}
