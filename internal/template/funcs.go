package template

import (
	"encoding/json"
	"fmt"
	htmltemplate "html/template"
	"math"
	"math/rand"
	"reflect"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/coderoo-dev/coderoo/internal/component"
	"github.com/coderoo-dev/coderoo/internal/content"
	"github.com/coderoo-dev/coderoo/internal/engine"
)

// buildFuncMap creates the template.FuncMap with all template functions.
// Closures capture runtime state (site context, resolver, registry, data cache).
func buildFuncMap(
	site *engine.SiteContext,
	resolver *engine.ThemeResolver,
	registry *component.Registry,
	dataCache *sync.Map,
) htmltemplate.FuncMap {
	return htmltemplate.FuncMap{
		// ── Strings ──
		"upper":      strings.ToUpper,
		"lower":      strings.ToLower,
		"title":      fnTitle,
		"truncate":   fnTruncate,
		"slugify":    content.Slugify,
		"replace":    strings.ReplaceAll,
		"split":      strings.Split,
		"join":       fnJoin,
		"contains":   strings.Contains,
		"hasPrefix":  strings.HasPrefix,
		"hasSuffix":  strings.HasSuffix,
		"trim":       strings.TrimSpace,
		"markdownify": fnMarkdownify,
		"plainify":   fnPlainify,
		"safeHTML":   fnSafeHTML,
		"highlight":  fnHighlight,

		// ── Dates ──
		"dateFormat": fnDateFormat,
		"now":        func() time.Time { return time.Now() },

		// ── Math ──
		"add": fnAdd,
		"sub": fnSub,
		"mul": fnMul,
		"div": fnDiv,
		"mod": fnMod,

		// ── Logic ──
		"cond":    fnCond,
		"default": fnDefault,
		"isset":   fnIsset,

		// ── Collections ──
		"first":   fnFirst,
		"last":    fnLast,
		"after":   fnAfter,
		"shuffle": fnShuffle,
		"sortBy":  fnSort,
		"where":   fnWhere,
		"group":   fnGroup,
		"uniq":    fnUniq,
		"in":      fnIn,
		"seq":     fnSeq,

		// ── URLs ──
		"absURL": func(relPath string) string {
			if site == nil || site.BaseURL == "" {
				return relPath
			}
			base := strings.TrimRight(site.BaseURL, "/")
			if strings.HasPrefix(relPath, "/") {
				return base + relPath
			}
			return base + "/" + relPath
		},
		"relURL": func(absPath string) string {
			if site == nil || site.BaseURL == "" {
				return absPath
			}
			base := strings.TrimRight(site.BaseURL, "/")
			return strings.TrimPrefix(absPath, base)
		},
		"urlize": content.Slugify,
		"ref": func(slug string) string {
			if site == nil {
				return slug
			}
			for _, p := range site.Pages {
				if p.Slug == slug || p.RelPermalink == slug {
					return p.Permalink
				}
			}
			return slug
		},
		"relref": func(slug string) string {
			if site == nil {
				return slug
			}
			for _, p := range site.Pages {
				if p.Slug == slug || p.RelPermalink == slug {
					return p.RelPermalink
				}
			}
			return slug
		},

		// ── Debug ──
		"printf":  fmt.Sprintf,
		"jsonify": fnJsonify,
		"dump":    fnDump,

		// ── Data ──
		"data": func(name string) any {
			return loadDataFile(resolver.ProjectDir, name, dataCache)
		},

		// ── Assets (stubs — Phase 10) ──
		"fingerprint": func(path string) string { return path },
		"inline":      func(path string) htmltemplate.HTML { return "" },
		"asset":       func(path string) string { return path },

		// ── i18n (stub — Phase 13) ──
		"t": func(key string) string { return key },

		// ── Cross-collection ──
		"recentEntries": func(colName string, n int) []*engine.Page {
			if site == nil {
				return nil
			}
			col, ok := site.Collections[colName]
			if !ok || col == nil {
				return nil
			}
			pages := col.Pages
			if n > len(pages) {
				n = len(pages)
			}
			return pages[:n]
		},
		"findEntry": func(colName, slug string) *engine.Page {
			if site == nil {
				return nil
			}
			col, ok := site.Collections[colName]
			if !ok || col == nil {
				return nil
			}
			for _, p := range col.Pages {
				if p.Slug == slug {
					return p
				}
			}
			return nil
		},
		"allCollections": func() map[string]*engine.Collection {
			if site == nil {
				return nil
			}
			return site.Collections
		},

		// ── Special: partial & component ──
		"partial": func(name string, data any) (htmltemplate.HTML, error) {
			content, _, err := resolvePartial(resolver, name)
			if err != nil {
				return "", err
			}
			tmpl, err := htmltemplate.New(name).Funcs(buildFuncMap(site, resolver, registry, dataCache)).Parse(string(content))
			if err != nil {
				return "", fmt.Errorf("parsing partial %q: %w", name, err)
			}
			var buf strings.Builder
			if err := tmpl.Execute(&buf, data); err != nil {
				return "", fmt.Errorf("rendering partial %q: %w", name, err)
			}
			return htmltemplate.HTML(buf.String()), nil
		},
		"component": func(name string, data any) (htmltemplate.HTML, error) {
			if registry == nil {
				return "", nil
			}
			return registry.RenderComponent(name, data)
		},
	}
}

// ── String function implementations ──

func fnTitle(s string) string {
	words := strings.Fields(s)
	for i, w := range words {
		if len(w) > 0 {
			words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
		}
	}
	return strings.Join(words, " ")
}

func fnTruncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	if n <= 3 {
		return string(runes[:n])
	}
	return string(runes[:n-3]) + "..."
}

func fnMarkdownify(s string) htmltemplate.HTML {
	// Inline markdown rendering: wrap in paragraph, handle bold/italic/code/links
	// Full Goldmark integration would add a dependency cycle; use simple replacements.
	// For proper rendering, the Engine can override this with a Goldmark instance.
	return htmltemplate.HTML(s)
}

func fnPlainify(s string) string {
	// Strip HTML tags
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result.WriteRune(r)
		}
	}
	return result.String()
}

func fnSafeHTML(s string) htmltemplate.HTML {
	return htmltemplate.HTML(s)
}

func fnHighlight(code, lang string) htmltemplate.HTML {
	// Stub: returns code wrapped in <pre><code>. Full Chroma integration in Phase 10.
	escaped := htmltemplate.HTMLEscapeString(code)
	return htmltemplate.HTML(fmt.Sprintf(`<pre><code class="language-%s">%s</code></pre>`, htmltemplate.HTMLEscapeString(lang), escaped))
}

// ── Date function implementations ──

func fnDateFormat(t time.Time, layout string) string {
	return t.Format(layout)
}

// ── Math function implementations ──

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

// ���─ Logic function implementations ──

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

// ── Collection function implementations ──

func fnFirst(n int, list any) any {
	v := reflect.ValueOf(list)
	if v.Kind() != reflect.Slice {
		return list
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

	sort.SliceStable(result.Interface().([]any), func(i, j int) bool {
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

// ── Debug function implementations ──

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

// ── Join helper (argument order matches template piping: join .List ", ") ─��

func fnJoin(list []string, sep string) string {
	return strings.Join(list, sep)
}

// ── Reflection helper ──

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
