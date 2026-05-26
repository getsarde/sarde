package content

import (
	"bytes"
	"encoding/json"
	"strings"

	"github.com/frostybee/sarde/internal/engine"
	"github.com/pelletier/go-toml/v2"
	"gopkg.in/yaml.v3"
)

// Parser implements engine.FrontmatterParser.
// It auto-detects YAML (---), TOML (+++), and JSON ({}) frontmatter delimiters.
type Parser struct{}

// Parse splits raw file bytes into a frontmatter map and Markdown body.
// Returns an empty map and the full content as body if no frontmatter is found.
func (p *Parser) Parse(raw []byte) (map[string]interface{}, string, error) {
	content := string(bytes.TrimLeft(raw, "\xef\xbb\xbf")) // strip BOM
	content = strings.TrimLeft(content, "\n\r")

	if content == "" {
		return map[string]interface{}{}, "", nil
	}

	switch {
	case strings.HasPrefix(content, "---"):
		fm, _, body, err := parseDelimited(content, "---")
		return fm, body, err
	case strings.HasPrefix(content, "+++"):
		return parseTOML(content)
	case strings.HasPrefix(content, "{"):
		return parseJSON(content)
	default:
		return map[string]interface{}{}, content, nil
	}
}

func parseDelimited(content, delimiter string) (map[string]interface{}, []byte, string, error) {
	// Find the closing delimiter after the opening one
	rest := content[len(delimiter):]
	// Skip the newline after opening delimiter
	if idx := strings.IndexByte(rest, '\n'); idx >= 0 {
		rest = rest[idx+1:]
	}
	closingIdx := strings.Index(rest, "\n"+delimiter)
	if closingIdx < 0 {
		// No closing delimiter — treat entire content as body
		return map[string]interface{}{}, nil, content, nil
	}

	fmText := rest[:closingIdx]
	body := rest[closingIdx+len("\n"+delimiter):]
	// Strip leading newline from body
	body = strings.TrimPrefix(body, "\n")
	body = strings.TrimPrefix(body, "\r\n")

	fmBytes := []byte(fmText)
	fm := make(map[string]interface{})
	if err := yaml.Unmarshal(fmBytes, &fm); err != nil {
		return nil, nil, "", err
	}
	return fm, fmBytes, body, nil
}

func parseTOML(content string) (map[string]interface{}, string, error) {
	rest := content[3:] // skip "+++"
	if idx := strings.IndexByte(rest, '\n'); idx >= 0 {
		rest = rest[idx+1:]
	}
	closingIdx := strings.Index(rest, "\n+++")
	if closingIdx < 0 {
		return map[string]interface{}{}, content, nil
	}

	fmText := rest[:closingIdx]
	body := rest[closingIdx+4:] // skip "\n+++"
	body = strings.TrimPrefix(body, "\n")
	body = strings.TrimPrefix(body, "\r\n")

	fm := make(map[string]interface{})
	if err := toml.Unmarshal([]byte(fmText), &fm); err != nil {
		return nil, "", err
	}
	return fm, body, nil
}

func parseJSON(content string) (map[string]interface{}, string, error) {
	// Find the closing brace, accounting for nesting
	depth := 0
	endIdx := -1
	for i, ch := range content {
		switch ch {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				endIdx = i
				goto found
			}
		}
	}
found:
	if endIdx < 0 {
		return map[string]interface{}{}, content, nil
	}

	fmText := content[:endIdx+1]
	body := content[endIdx+1:]
	body = strings.TrimPrefix(body, "\n")
	body = strings.TrimPrefix(body, "\r\n")

	fm := make(map[string]interface{})
	if err := json.Unmarshal([]byte(fmText), &fm); err != nil {
		return nil, "", err
	}
	return fm, body, nil
}

// ParseFrontmatter is a convenience function that parses raw file bytes into
// a typed Frontmatter struct and the Markdown body. It handles all three
// frontmatter formats (YAML, TOML, JSON) uniformly by converting through YAML.
func ParseFrontmatter(raw []byte) (*engine.Frontmatter, string, error) {
	_, fm, body, err := ParseAll(raw)
	return fm, body, err
}

// parseRaw normalises raw file bytes and dispatches to the format-specific
// parser. For YAML input it also returns the raw frontmatter bytes so callers
// can unmarshal directly into a typed struct. For TOML/JSON, fmText is nil.
func parseRaw(raw []byte) (fmMap map[string]interface{}, fmText []byte, body string, err error) {
	content := string(bytes.TrimLeft(raw, "\xef\xbb\xbf"))
	content = strings.TrimLeft(content, "\n\r")

	if content == "" {
		return map[string]interface{}{}, nil, "", nil
	}

	switch {
	case strings.HasPrefix(content, "---"):
		return parseDelimited(content, "---")
	case strings.HasPrefix(content, "+++"):
		fm, body, err := parseTOML(content)
		return fm, nil, body, err
	case strings.HasPrefix(content, "{"):
		fm, body, err := parseJSON(content)
		return fm, nil, body, err
	default:
		return map[string]interface{}{}, nil, content, nil
	}
}

// ParseAll parses raw file bytes into both an untyped map (for schema validation)
// and a typed Frontmatter struct. For YAML input (the common case), the struct
// is unmarshaled directly from the raw frontmatter bytes, avoiding a redundant
// marshal+unmarshal round-trip.
func ParseAll(raw []byte) (map[string]interface{}, *engine.Frontmatter, string, error) {
	fmMap, fmText, body, err := parseRaw(raw)
	if err != nil {
		return nil, nil, "", err
	}

	fm := &engine.Frontmatter{}
	if len(fmMap) > 0 {
		if fmText != nil {
			if err := yaml.Unmarshal(fmText, fm); err != nil {
				return nil, nil, "", err
			}
		} else {
			yamlBytes, err := yaml.Marshal(fmMap)
			if err != nil {
				return nil, nil, "", err
			}
			if err := yaml.Unmarshal(yamlBytes, fm); err != nil {
				return nil, nil, "", err
			}
		}
	}

	return fmMap, fm, body, nil
}
