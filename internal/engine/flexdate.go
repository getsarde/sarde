package engine

import (
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// FlexDate is a frontmatter date field that tolerates an unset value.
//
// An empty, whitespace-only, or null value decodes as the zero time, meaning
// "not set", instead of failing the parse. Editors that clear a date field
// commonly write `date: ''`, and a single such file must not abort the build.
//
// Accepted forms are the YAML native timestamp, RFC 3339, and plain
// YYYY-MM-DD (with an optional time component). Anything else is an error.
type FlexDate struct {
	time.Time
}

var flexDateLayouts = []string{
	"2006-01-02",
	time.RFC3339,
	"2006-01-02T15:04:05",
	"2006-01-02 15:04:05",
	"2006-01-02 15:04",
}

func (d *FlexDate) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind == yaml.ScalarNode {
		if value.Tag == "!!null" || strings.TrimSpace(value.Value) == "" {
			d.Time = time.Time{}
			return nil
		}
	}

	var t time.Time
	if err := value.Decode(&t); err == nil {
		d.Time = t
		return nil
	}

	raw := strings.TrimSpace(value.Value)
	for _, layout := range flexDateLayouts {
		if t, err := time.Parse(layout, raw); err == nil {
			d.Time = t
			return nil
		}
	}
	return fmt.Errorf("invalid date %q: want YYYY-MM-DD or an RFC 3339 timestamp", raw)
}

// MarshalYAML emits the zero time as an absent value so a round-trip through
// YAML does not turn "unset" into a year-1 timestamp.
func (d FlexDate) MarshalYAML() (any, error) {
	if d.Time.IsZero() {
		return nil, nil
	}
	return d.Time, nil
}
