package i18n

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"text/template"

	"gopkg.in/yaml.v3"
)

// StringTable holds translation strings for all languages with fallback resolution.
type StringTable struct {
	strings     map[string]map[string]string // lang → dotted-key → value
	defaultLang string
	tmplCache   sync.Map // key → *template.Template
	strict      bool
	misses      sync.Map // "lang\x00key" → true
}

// NewStringTable creates an empty StringTable with the given default language.
func NewStringTable(defaultLang string) *StringTable {
	return &StringTable{
		strings:     make(map[string]map[string]string),
		defaultLang: defaultLang,
	}
}

// LoadStrings loads i18n YAML files from three layers (embedded → theme → user).
// Later layers override earlier layers per-key.
func LoadStrings(embeddedFS fs.FS, projectDir, themeName, defaultLang string) (*StringTable, error) {
	st := NewStringTable(defaultLang)

	// Layer 1: embedded defaults
	if embeddedFS != nil {
		if err := st.loadFromFS(embeddedFS); err != nil {
			return nil, fmt.Errorf("loading embedded i18n: %w", err)
		}
	}

	// Layer 2: theme i18n
	if themeName != "" {
		themeI18nDir := filepath.Join(projectDir, "themes", themeName, "i18n")
		if err := st.loadFromDir(themeI18nDir); err != nil {
			return nil, fmt.Errorf("loading theme i18n: %w", err)
		}
	}

	// Layer 3: user i18n
	userI18nDir := filepath.Join(projectDir, "i18n")
	if err := st.loadFromDir(userI18nDir); err != nil {
		return nil, fmt.Errorf("loading user i18n: %w", err)
	}

	return st, nil
}

// Resolve looks up a translation key for the given language.
// Falls back to the default language, then to the key itself.
// If the value contains Go template syntax, it is executed with the optional data argument.
func (st *StringTable) Resolve(lang, key string, data ...any) string {
	value := st.lookup(lang, key)

	if !strings.Contains(value, "{{") {
		return value
	}

	// Execute as template
	var d any
	if len(data) > 0 {
		d = data[0]
	}

	tmpl, err := st.getOrParseTemplate(key, value)
	if err != nil {
		return value
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, d); err != nil {
		return value
	}
	return buf.String()
}

// SetStrict enables strict mode, which records keys that fall back.
func (st *StringTable) SetStrict(on bool) { st.strict = on }

// Misses returns keys that fell back during resolution, grouped by language.
func (st *StringTable) Misses() map[string][]string {
	result := make(map[string][]string)
	st.misses.Range(func(k, _ any) bool {
		parts := strings.SplitN(k.(string), "\x00", 2)
		if len(parts) == 2 {
			result[parts[0]] = append(result[parts[0]], parts[1])
		}
		return true
	})
	for lang := range result {
		sort.Strings(result[lang])
	}
	return result
}

func (st *StringTable) lookup(lang, key string) string {
	if langStrings, ok := st.strings[lang]; ok {
		if val, ok := langStrings[key]; ok {
			return val
		}
	}
	if lang != st.defaultLang {
		if st.strict {
			st.misses.Store(lang+"\x00"+key, true)
		}
		if defStrings, ok := st.strings[st.defaultLang]; ok {
			if val, ok := defStrings[key]; ok {
				return val
			}
		}
	}
	return key
}

func (st *StringTable) getOrParseTemplate(key, value string) (*template.Template, error) {
	cacheKey := key + "\x00" + value
	if cached, ok := st.tmplCache.Load(cacheKey); ok {
		return cached.(*template.Template), nil
	}
	tmpl, err := template.New(key).Parse(value)
	if err != nil {
		return nil, err
	}
	st.tmplCache.Store(cacheKey, tmpl)
	return tmpl, nil
}

func (st *StringTable) loadFromFS(fsys fs.FS) error {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil // directory may not exist
	}
	for _, e := range entries {
		if e.IsDir() || !isYAML(e.Name()) {
			continue
		}
		lang := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		data, err := fs.ReadFile(fsys, e.Name())
		if err != nil {
			return err
		}
		if err := st.mergeYAML(lang, data); err != nil {
			return fmt.Errorf("parsing %s: %w", e.Name(), err)
		}
	}
	return nil
}

func (st *StringTable) loadFromDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil // directory may not exist
	}
	for _, e := range entries {
		if e.IsDir() || !isYAML(e.Name()) {
			continue
		}
		lang := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return err
		}
		if err := st.mergeYAML(lang, data); err != nil {
			return fmt.Errorf("parsing %s: %w", e.Name(), err)
		}
	}
	return nil
}

func (st *StringTable) mergeYAML(lang string, data []byte) error {
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return err
	}
	if st.strings[lang] == nil {
		st.strings[lang] = make(map[string]string)
	}
	flatten("", raw, st.strings[lang])
	return nil
}

// flatten converts a nested map to dot-notation keys.
func flatten(prefix string, m map[string]any, out map[string]string) {
	for k, v := range m {
		key := k
		if prefix != "" {
			key = prefix + "." + k
		}
		switch val := v.(type) {
		case map[string]any:
			flatten(key, val, out)
		case string:
			out[key] = val
		default:
			out[key] = fmt.Sprintf("%v", val)
		}
	}
}

func isYAML(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".yaml" || ext == ".yml"
}
