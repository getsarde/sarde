package cfgutil

import "testing"

func TestFloat(t *testing.T) {
	tests := []struct {
		name     string
		cfg      map[string]any
		key      string
		fallback float64
		expected float64
	}{
		{"nil config", nil, "opacity", 0.5, 0.5},
		{"missing key", map[string]any{}, "opacity", 0.5, 0.5},
		{"float64 value", map[string]any{"opacity": 0.07}, "opacity", 0.5, 0.07},
		{"int value", map[string]any{"opacity": 1}, "opacity", 0.5, 1.0},
		{"int64 value", map[string]any{"opacity": int64(2)}, "opacity", 0.5, 2.0},
		{"wrong type", map[string]any{"opacity": "high"}, "opacity", 0.5, 0.5},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Float(tt.cfg, tt.key, tt.fallback); got != tt.expected {
				t.Errorf("Float() = %v, want %v", got, tt.expected)
			}
		})
	}
}
