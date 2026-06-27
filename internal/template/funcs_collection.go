package template

import (
	"fmt"
	"math/rand"
	"reflect"
	"sort"
)

func fnFirst(n int, list any) any {
	v := reflect.ValueOf(list)
	if v.Kind() != reflect.Slice {
		return list
	}
	if n < 0 {
		n = 0
	}
	if n > v.Len() {
		n = v.Len()
	}
	return v.Slice(0, n).Interface()
}

func fnLast(n int, list any) any {
	v := reflect.ValueOf(list)
	if v.Kind() != reflect.Slice {
		return list
	}
	if n < 0 {
		n = 0
	}
	l := v.Len()
	if n > l {
		n = l
	}
	return v.Slice(l-n, l).Interface()
}

func fnAfter(n int, list any) any {
	v := reflect.ValueOf(list)
	if v.Kind() != reflect.Slice {
		return list
	}
	if n < 0 {
		n = 0
	}
	if n > v.Len() {
		n = v.Len()
	}
	return v.Slice(n, v.Len()).Interface()
}

func fnShuffle(list any) any {
	v := reflect.ValueOf(list)
	if v.Kind() != reflect.Slice {
		return list
	}
	l := v.Len()
	result := reflect.MakeSlice(v.Type(), l, l)
	reflect.Copy(result, v)
	rand.Shuffle(l, func(i, j int) {
		vi := result.Index(i).Interface()
		vj := result.Index(j).Interface()
		result.Index(i).Set(reflect.ValueOf(vj))
		result.Index(j).Set(reflect.ValueOf(vi))
	})
	return result.Interface()
}

func fnSort(list any, field string) any {
	v := reflect.ValueOf(list)
	if v.Kind() != reflect.Slice {
		return list
	}
	l := v.Len()
	result := reflect.MakeSlice(v.Type(), l, l)
	reflect.Copy(result, v)

	sort.SliceStable(result.Interface(), func(i, j int) bool {
		a := getField(result.Index(i).Interface(), field)
		b := getField(result.Index(j).Interface(), field)
		return fmt.Sprint(a) < fmt.Sprint(b)
	})
	return result.Interface()
}

func fnWhere(list any, field string, value any) any {
	v := reflect.ValueOf(list)
	if v.Kind() != reflect.Slice {
		return list
	}
	result := reflect.MakeSlice(v.Type(), 0, v.Len())
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i).Interface()
		fv := getField(item, field)
		if fmt.Sprint(fv) == fmt.Sprint(value) {
			result = reflect.Append(result, v.Index(i))
		}
	}
	return result.Interface()
}

func fnGroup(list any, field string) map[string]any {
	v := reflect.ValueOf(list)
	if v.Kind() != reflect.Slice {
		return nil
	}
	groups := make(map[string][]any)
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i).Interface()
		key := fmt.Sprint(getField(item, field))
		groups[key] = append(groups[key], item)
	}
	result := make(map[string]any, len(groups))
	for k, v := range groups {
		result[k] = v
	}
	return result
}

func fnUniq(list any) any {
	v := reflect.ValueOf(list)
	if v.Kind() != reflect.Slice {
		return list
	}
	seen := make(map[string]bool)
	result := reflect.MakeSlice(v.Type(), 0, v.Len())
	for i := 0; i < v.Len(); i++ {
		key := fmt.Sprint(v.Index(i).Interface())
		if !seen[key] {
			seen[key] = true
			result = reflect.Append(result, v.Index(i))
		}
	}
	return result.Interface()
}

func fnIn(list any, value any) bool {
	v := reflect.ValueOf(list)
	if v.Kind() != reflect.Slice {
		return false
	}
	target := fmt.Sprint(value)
	for i := 0; i < v.Len(); i++ {
		if fmt.Sprint(v.Index(i).Interface()) == target {
			return true
		}
	}
	return false
}

func fnSeq(args ...int) []int {
	switch len(args) {
	case 1:
		if args[0] < 0 {
			return nil
		}
		result := make([]int, args[0])
		for i := range result {
			result[i] = i + 1
		}
		return result
	case 2:
		start, end := args[0], args[1]
		if start > end {
			return nil
		}
		result := make([]int, end-start+1)
		for i := range result {
			result[i] = start + i
		}
		return result
	default:
		return nil
	}
}
