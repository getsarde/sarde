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
