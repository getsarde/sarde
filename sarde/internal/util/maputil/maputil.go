package maputil

// GetStringOr extracts a string from a map or returns a default.
func GetStringOr(m map[string]any, key, def string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return def
}

// GetIntOr extracts an int from a map or returns a default.
func GetIntOr(m map[string]any, key string, def int) int {
	if v, ok := m[key]; ok {
		if i, ok := ToInt(v); ok {
			return i
		}
	}
	return def
}

// GetBoolOr extracts a bool from a map or returns a default.
func GetBoolOr(m map[string]any, key string, def bool) bool {
	if v, ok := m[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return def
}

// GetStringSlice extracts a string slice from a map.
func GetStringSlice(m map[string]any, key string) []string {
	if v, ok := m[key]; ok {
		if arr, ok := v.([]any); ok {
			result := make([]string, 0, len(arr))
			for _, item := range arr {
				if s, ok := item.(string); ok {
					result = append(result, s)
				}
			}
			return result
		}
	}
	return nil
}

// ToInt converts an any value to int, handling int, int64, and float64.
func ToInt(v any) (int, bool) {
	switch n := v.(type) {
	case int:
		return n, true
	case int64:
		return int(n), true
	case float64:
		return int(n), true
	}
	return 0, false
}
