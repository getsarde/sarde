package maputil

import "testing"

func TestGetStringOr(t *testing.T) {
	m := map[string]any{"name": "alice", "count": 42}

	if v := GetStringOr(m, "name", ""); v != "alice" {
		t.Errorf("got %q, want %q", v, "alice")
	}
	if v := GetStringOr(m, "missing", "default"); v != "default" {
		t.Errorf("got %q, want %q", v, "default")
	}
	if v := GetStringOr(m, "count", "fallback"); v != "fallback" {
		t.Errorf("wrong type should return default, got %q", v)
	}
	if v := GetStringOr(nil, "key", "def"); v != "def" {
		t.Errorf("nil map should return default, got %q", v)
	}
}

func TestGetIntOr(t *testing.T) {
	m := map[string]any{"a": 10, "b": int64(20), "c": float64(30.9), "s": "nope"}

	if v := GetIntOr(m, "a", 0); v != 10 {
		t.Errorf("int: got %d, want 10", v)
	}
	if v := GetIntOr(m, "b", 0); v != 20 {
		t.Errorf("int64: got %d, want 20", v)
	}
	if v := GetIntOr(m, "c", 0); v != 30 {
		t.Errorf("float64: got %d, want 30", v)
	}
	if v := GetIntOr(m, "s", 99); v != 99 {
		t.Errorf("wrong type should return default, got %d", v)
	}
	if v := GetIntOr(m, "missing", 5); v != 5 {
		t.Errorf("missing key: got %d, want 5", v)
	}
}

func TestGetBoolOr(t *testing.T) {
	m := map[string]any{"flag": true, "num": 1}

	if v := GetBoolOr(m, "flag", false); v != true {
		t.Errorf("got %v, want true", v)
	}
	if v := GetBoolOr(m, "num", false); v != false {
		t.Errorf("wrong type should return default, got %v", v)
	}
	if v := GetBoolOr(m, "missing", true); v != true {
		t.Errorf("missing key: got %v, want true", v)
	}
}

func TestGetStringSlice(t *testing.T) {
	m := map[string]any{
		"tags":  []any{"go", "ssg", 42},
		"empty": []any{},
		"wrong": "not a slice",
	}

	tags := GetStringSlice(m, "tags")
	if len(tags) != 2 || tags[0] != "go" || tags[1] != "ssg" {
		t.Errorf("tags = %v, want [go ssg]", tags)
	}
	if v := GetStringSlice(m, "empty"); len(v) != 0 {
		t.Errorf("empty slice: got %v", v)
	}
	if v := GetStringSlice(m, "wrong"); v != nil {
		t.Errorf("wrong type should return nil, got %v", v)
	}
	if v := GetStringSlice(m, "missing"); v != nil {
		t.Errorf("missing key should return nil, got %v", v)
	}
}

func TestToInt(t *testing.T) {
	tests := []struct {
		name string
		val  any
		want int
		ok   bool
	}{
		{"int", 42, 42, true},
		{"int64", int64(100), 100, true},
		{"float64", float64(3.7), 3, true},
		{"string", "nope", 0, false},
		{"nil", nil, 0, false},
		{"bool", true, 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ToInt(tt.val)
			if ok != tt.ok || got != tt.want {
				t.Errorf("ToInt(%v) = (%d, %v), want (%d, %v)", tt.val, got, ok, tt.want, tt.ok)
			}
		})
	}
}
