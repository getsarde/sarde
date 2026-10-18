package validate

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/getsarde/sarde/internal/suggest"
)

// Error describes a single validation failure. The json tags define the
// machine-readable shape emitted by `--format json` error envelopes (consumed
// by Sarde Studio); keep them stable.
type Error struct {
	Path    string   `json:"path"`              // dotted field path, e.g. "collections.docs.layout"
	Value   string   `json:"value,omitempty"`   // the invalid value (stringified for display)
	Message string   `json:"message"`           // human-readable explanation with valid options
	Allowed []string `json:"allowed,omitempty"` // valid values when the check was an enumeration
	// Suggestion is the allowed value closest to Value, when it looks like a
	// typo of one (an edit distance of 2 or less).
	Suggestion string `json:"suggestion,omitempty"`
}

func (e Error) Error() string {
	switch {
	case e.Value != "" && e.Suggestion != "":
		return fmt.Sprintf("%s: %s (got %q; did you mean %q?)", e.Path, e.Message, e.Value, e.Suggestion)
	case e.Value != "":
		return fmt.Sprintf("%s: %s (got %q)", e.Path, e.Message, e.Value)
	}
	return fmt.Sprintf("%s: %s", e.Path, e.Message)
}

// Checker accumulates validation errors as you check fields.
type Checker struct {
	errors []Error
}

// OneOf checks that value is one of the allowed strings.
// Skips if value is empty (zero value means "use default"). The error lists
// the allowed values sorted and without duplicates, so it reads the same on
// every run, and suggests the closest one when value looks like a typo.
func (c *Checker) OneOf(path, value string, allowed []string) {
	if value == "" {
		return
	}
	if slices.Contains(allowed, value) {
		return
	}
	list := slices.Clone(allowed)
	slices.Sort(list)
	list = slices.Compact(list)
	c.errors = append(c.errors, Error{
		Path:       path,
		Value:      value,
		Message:    fmt.Sprintf("must be one of: %s", strings.Join(list, ", ")),
		Allowed:    list,
		Suggestion: suggest.Closest(value, list),
	})
}

// IntRange checks that value is within [min, max].
// Skips if value is 0 (zero value means "use default").
func (c *Checker) IntRange(path string, value, min, max int) {
	if value == 0 {
		return
	}
	if value < min || value > max {
		c.errors = append(c.errors, Error{
			Path:    path,
			Value:   strconv.Itoa(value),
			Message: fmt.Sprintf("must be between %d and %d", min, max),
		})
	}
}

// IntMin checks that value is >= min. Skips if 0.
func (c *Checker) IntMin(path string, value, min int) {
	if value == 0 {
		return
	}
	if value < min {
		c.errors = append(c.errors, Error{
			Path:    path,
			Value:   strconv.Itoa(value),
			Message: fmt.Sprintf("must be at least %d", min),
		})
	}
}

// LessOrEqual checks a <= b. Skips if either is 0.
func (c *Checker) LessOrEqual(pathA string, a int, pathB string, b int) {
	if a == 0 || b == 0 {
		return
	}
	if a > b {
		c.errors = append(c.errors, Error{
			Path:    pathA,
			Value:   strconv.Itoa(a),
			Message: fmt.Sprintf("must be <= %s (%d)", pathB, b),
		})
	}
}

// Required checks that value is non-empty.
func (c *Checker) Required(path, value string) {
	if strings.TrimSpace(value) == "" {
		c.errors = append(c.errors, Error{
			Path:    path,
			Message: "is required",
		})
	}
}

// FloatMin checks that value is >= min. Skips if min is nil.
func (c *Checker) FloatMin(path string, value float64, min *float64) {
	if min == nil {
		return
	}
	if value < *min {
		c.errors = append(c.errors, Error{
			Path:    path,
			Value:   fmt.Sprintf("%.4g", value),
			Message: fmt.Sprintf("must be at least %.4g", *min),
		})
	}
}

// FloatMax checks that value is <= max. Skips if max is nil.
func (c *Checker) FloatMax(path string, value float64, max *float64) {
	if max == nil {
		return
	}
	if value > *max {
		c.errors = append(c.errors, Error{
			Path:    path,
			Value:   fmt.Sprintf("%.4g", value),
			Message: fmt.Sprintf("must be at most %.4g", *max),
		})
	}
}

// MaxLength checks that a string's length does not exceed max. Skips if max is nil.
func (c *Checker) MaxLength(path string, value string, max *int) {
	if max == nil {
		return
	}
	if len(value) > *max {
		c.errors = append(c.errors, Error{
			Path:    path,
			Value:   fmt.Sprintf("%d chars", len(value)),
			Message: fmt.Sprintf("length must be at most %d", *max),
		})
	}
}

// Check adds a custom error if cond is false.
func (c *Checker) Check(path, value string, cond bool, msg string) {
	if !cond {
		c.errors = append(c.errors, Error{
			Path:    path,
			Value:   value,
			Message: msg,
		})
	}
}

// Errors returns all accumulated errors. Nil if everything is valid.
func (c *Checker) Errors() []Error { return c.errors }

// HasErrors returns true if any errors were recorded.
func (c *Checker) HasErrors() bool { return len(c.errors) > 0 }

// FormatErrors joins all errors into a single multi-line string.
func FormatErrors(errs []Error) string {
	var b strings.Builder
	for i, e := range errs {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(e.Error())
	}
	return b.String()
}
