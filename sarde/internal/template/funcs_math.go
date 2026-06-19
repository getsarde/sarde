package template

import (
	"math"
	"reflect"
)

func toFloat64(v any) (float64, bool) {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(rv.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(rv.Uint()), true
	case reflect.Float32, reflect.Float64:
		return rv.Float(), true
	default:
		return 0, false
	}
}

func fnAdd(a, b any) any {
	af, aOk := toFloat64(a)
	bf, bOk := toFloat64(b)
	if !aOk || !bOk {
		return 0
	}
	result := af + bf
	if isInt(a) && isInt(b) {
		return int(result)
	}
	return result
}

func fnSub(a, b any) any {
	af, aOk := toFloat64(a)
	bf, bOk := toFloat64(b)
	if !aOk || !bOk {
		return 0
	}
	result := af - bf
	if isInt(a) && isInt(b) {
		return int(result)
	}
	return result
}

func fnMul(a, b any) any {
	af, aOk := toFloat64(a)
	bf, bOk := toFloat64(b)
	if !aOk || !bOk {
		return 0
	}
	result := af * bf
	if isInt(a) && isInt(b) {
		return int(result)
	}
	return result
}

func fnDiv(a, b any) any {
	af, aOk := toFloat64(a)
	bf, bOk := toFloat64(b)
	if !aOk || !bOk || bf == 0 {
		return 0
	}
	result := af / bf
	if isInt(a) && isInt(b) {
		return int(math.Trunc(result))
	}
	return result
}

func fnMod(a, b any) int {
	af, aOk := toFloat64(a)
	bf, bOk := toFloat64(b)
	if !aOk || !bOk || bf == 0 {
		return 0
	}
	return int(af) % int(bf)
}

func isInt(v any) bool {
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return true
	default:
		return false
	}
}
