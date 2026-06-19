package template

import (
	"encoding/json"
	"fmt"
	htmltemplate "html/template"
	"reflect"
	"strings"
)

func fnJsonify(value any) htmltemplate.HTML {
	b, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return htmltemplate.HTML(fmt.Sprintf("<!-- jsonify error: %s -->", err))
	}
	return htmltemplate.HTML(b)
}

func fnDump(value any) htmltemplate.HTML {
	return htmltemplate.HTML(fmt.Sprintf("<pre>%s</pre>", htmltemplate.HTMLEscapeString(fmt.Sprintf("%+v", value))))
}

func fnJoin(list []string, sep string) string {
	return strings.Join(list, sep)
}

func getField(item any, field string) any {
	v := reflect.ValueOf(item)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	if v.Kind() == reflect.Struct {
		f := v.FieldByName(field)
		if f.IsValid() {
			return f.Interface()
		}
	}
	if v.Kind() == reflect.Map {
		f := v.MapIndex(reflect.ValueOf(field))
		if f.IsValid() {
			return f.Interface()
		}
	}
	return nil
}

// wrapCSS wraps raw CSS in a <style> tag.
func wrapCSS(css string) string {
	if css == "" {
		return ""
	}
	return "<style>\n" + css + "</style>"
}
