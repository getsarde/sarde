package config

import (
	"github.com/getsarde/sarde/embedded"
	"gopkg.in/yaml.v3"
)

// Defaults returns a fully populated SiteConfig from the embedded default YAML.
// This is layer 1 of the 5-layer config cascade.
// Panics if the embedded YAML is invalid (programmer error).
func Defaults() *SiteConfig {
	cfg := &SiteConfig{}
	if err := yaml.Unmarshal(embedded.DefaultsYAML, cfg); err != nil {
		panic("config: invalid embedded defaults: " + err.Error())
	}
	return cfg
}
