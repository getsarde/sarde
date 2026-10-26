package validate

import (
	"strings"
	"testing"
)

func float64Ptr(v float64) *float64 { return &v }
func intPtr(v int) *int             { return &v }

// ---------------------------------------------------------------------------
// OneOf
// ---------------------------------------------------------------------------

func TestOneOf_ListsSortedUniqueValuesAndSuggests(t *testing.T) {
	var c Checker
	c.OneOf("plugins.enabled[0]", "sitemp", []string{"sitemap", "rss", "sitemap", "atom"})
	if !c.HasErrors() {
		t.Fatal("expected an error")
	}
	e := c.Errors()[0]
	if e.Message != "must be one of: atom, rss, sitemap" {
		t.Errorf("message = %q, want the sorted list without the duplicate", e.Message)
	}
	if strings.Join(e.Allowed, ",") != "atom,rss,sitemap" {
		t.Errorf("allowed = %v", e.Allowed)
	}
	if e.Suggestion != "sitemap" {
		t.Errorf("suggestion = %q, want sitemap", e.Suggestion)
	}
	want := `plugins.enabled[0]: must be one of: atom, rss, sitemap (got "sitemp"; did you mean "sitemap"?)`
	if e.Error() != want {
		t.Errorf("Error() = %q, want %q", e.Error(), want)
	}

	var far Checker
	far.OneOf("deploy.provider", "heroku", []string{"github", "netlify"})
	if got := far.Errors()[0]; got.Suggestion != "" || strings.Contains(got.Error(), "did you mean") {
		t.Errorf("a distant value must get no hint: %q", got.Error())
	}
}

func TestOneOf(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		allowed []string
		wantErr bool
	}{
		{"valid value", "inline", []string{"inline", "sprite"}, false},
		{"invalid value", "banana", []string{"inline", "sprite"}, true},
		{"empty skipped", "", []string{"inline", "sprite"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var c Checker
			c.OneOf("field", tt.value, tt.allowed)
			if tt.wantErr && !c.HasErrors() {
				t.Error("expected error, got none")
			}
			if !tt.wantErr && c.HasErrors() {
				t.Errorf("unexpected error: %v", c.Errors())
			}
			if tt.wantErr && c.HasErrors() {
				e := c.Errors()[0]
				if !strings.Contains(e.Message, "must be one of") {
					t.Errorf("message = %q, want 'must be one of'", e.Message)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// IntRange
// ---------------------------------------------------------------------------

func TestIntRange(t *testing.T) {
	tests := []struct {
		name    string
		value   int
		min     int
		max     int
		wantErr bool
	}{
		{"in range", 3, 1, 6, false},
		{"at min", 1, 1, 6, false},
		{"at max", 6, 1, 6, false},
		{"below range", -1, 1, 6, true},
		{"above range", 9, 1, 6, true},
		{"zero skipped", 0, 1, 6, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var c Checker
			c.IntRange("field", tt.value, tt.min, tt.max)
			if tt.wantErr != c.HasErrors() {
				t.Errorf("HasErrors = %v, want %v", c.HasErrors(), tt.wantErr)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// IntMin
// ---------------------------------------------------------------------------

func TestIntMin(t *testing.T) {
	tests := []struct {
		name    string
		value   int
		min     int
		wantErr bool
	}{
		{"above min", 5, 1, false},
		{"at min", 1, 1, false},
		{"below min", -3, 1, true},
		{"zero skipped", 0, 1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var c Checker
			c.IntMin("field", tt.value, tt.min)
			if tt.wantErr != c.HasErrors() {
				t.Errorf("HasErrors = %v, want %v", c.HasErrors(), tt.wantErr)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// LessOrEqual
// ---------------------------------------------------------------------------

func TestLessOrEqual(t *testing.T) {
	tests := []struct {
		name    string
		a, b    int
		wantErr bool
	}{
		{"a < b", 2, 5, false},
		{"a == b", 3, 3, false},
		{"a > b", 5, 2, true},
		{"a zero skipped", 0, 5, false},
		{"b zero skipped", 5, 0, false},
		{"both zero skipped", 0, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var c Checker
			c.LessOrEqual("a", tt.a, "b", tt.b)
			if tt.wantErr != c.HasErrors() {
				t.Errorf("HasErrors = %v, want %v", c.HasErrors(), tt.wantErr)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Required
// ---------------------------------------------------------------------------

func TestRequired(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		wantErr bool
	}{
		{"non-empty", "hello", false},
		{"empty", "", true},
		{"whitespace only", "   ", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var c Checker
			c.Required("field", tt.value)
			if tt.wantErr != c.HasErrors() {
				t.Errorf("HasErrors = %v, want %v", c.HasErrors(), tt.wantErr)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// FloatMin
// ---------------------------------------------------------------------------

func TestFloatMin(t *testing.T) {
	tests := []struct {
		name    string
		value   float64
		min     *float64
		wantErr bool
	}{
		{"above min", 5.0, float64Ptr(1.0), false},
		{"at min", 1.0, float64Ptr(1.0), false},
		{"below min", 0.5, float64Ptr(1.0), true},
		{"nil min skipped", 0.5, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var c Checker
			c.FloatMin("field", tt.value, tt.min)
			if tt.wantErr != c.HasErrors() {
				t.Errorf("HasErrors = %v, want %v", c.HasErrors(), tt.wantErr)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// FloatMax
// ---------------------------------------------------------------------------

func TestFloatMax(t *testing.T) {
	tests := []struct {
		name    string
		value   float64
		max     *float64
		wantErr bool
	}{
		{"below max", 5.0, float64Ptr(10.0), false},
		{"at max", 10.0, float64Ptr(10.0), false},
		{"above max", 15.0, float64Ptr(10.0), true},
		{"nil max skipped", 15.0, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var c Checker
			c.FloatMax("field", tt.value, tt.max)
			if tt.wantErr != c.HasErrors() {
				t.Errorf("HasErrors = %v, want %v", c.HasErrors(), tt.wantErr)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// MaxLength
// ---------------------------------------------------------------------------

func TestMaxLength(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		max     *int
		wantErr bool
	}{
		{"within limit", "short", intPtr(10), false},
		{"at limit", "1234567890", intPtr(10), false},
		{"over limit", "12345678901", intPtr(10), true},
		{"nil max skipped", "anything", nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var c Checker
			c.MaxLength("field", tt.value, tt.max)
			if tt.wantErr != c.HasErrors() {
				t.Errorf("HasErrors = %v, want %v", c.HasErrors(), tt.wantErr)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Check
// ---------------------------------------------------------------------------

func TestCheck(t *testing.T) {
	var c Checker
	c.Check("field", "val", true, "should pass")
	if c.HasErrors() {
		t.Error("true condition should not add error")
	}

	c.Check("field", "val", false, "custom message")
	if !c.HasErrors() {
		t.Error("false condition should add error")
	}
	if c.Errors()[0].Message != "custom message" {
		t.Errorf("Message = %q, want %q", c.Errors()[0].Message, "custom message")
	}
}

// ---------------------------------------------------------------------------
// FormatErrors
// ---------------------------------------------------------------------------

func TestFormatErrors(t *testing.T) {
	errs := []Error{
		{Path: "a.b", Value: "x", Message: "invalid"},
		{Path: "c", Message: "is required"},
	}
	got := FormatErrors(errs)
	if !strings.Contains(got, "a.b: invalid (got \"x\")") {
		t.Errorf("missing first error in:\n%s", got)
	}
	if !strings.Contains(got, "c: is required") {
		t.Errorf("missing second error in:\n%s", got)
	}
	if strings.Count(got, "\n") != 1 {
		t.Errorf("expected 1 newline, got %d", strings.Count(got, "\n"))
	}
}

func TestFormatErrors_ListsAllowedValuesOnce(t *testing.T) {
	var c Checker
	known := []string{"search", "seo", "sitemap"}
	c.OneOf("plugins.enabled[0]", "serch", known)
	c.OneOf("plugins.enabled[1]", "sittemap", known)
	c.OneOf("layout", "wide", []string{"default", "docs"})
	got := FormatErrors(c.Errors())
	if n := strings.Count(got, "must be one of: search, seo, sitemap"); n != 1 {
		t.Errorf("the plugin list appears %d times, want 1:\n%s", n, got)
	}
	for _, want := range []string{
		`plugins.enabled[0]: must be one of: search, seo, sitemap (got "serch"; did you mean "search"?)`,
		`plugins.enabled[1]: must be one of the values listed above (got "sittemap"; did you mean "sitemap"?)`,
		`layout: must be one of: default, docs (got "wide")`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	// The errors themselves keep their full message for JSON output.
	if c.Errors()[1].Message != "must be one of: search, seo, sitemap" {
		t.Errorf("FormatErrors changed the error: %q", c.Errors()[1].Message)
	}
}
