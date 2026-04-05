package embedded

import _ "embed"

// DefaultsYAML contains the embedded default site configuration.
//
//go:embed defaults/site.yaml
var DefaultsYAML []byte
