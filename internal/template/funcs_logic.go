package template

import "reflect"

func fnCond(condition bool, trueVal, falseVal any) any {
	if condition {
		return trueVal
	}
	return falseVal
}

func fnDefault(value, fallback any) any {
	if value == nil {
		return fallback
	}
	rv := reflect.ValueOf(value)
	if rv.IsZero() {
		return fallback
	}
	return value
}

func fnIsset(m map[string]any, key string) bool {
	_, ok := m[key]
	return ok
}
