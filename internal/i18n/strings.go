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

	"github.com/getsarde/sarde/internal/consts"
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

// Layer sources, in load order: later layers override earlier ones per key.
const (
	SourceBuiltin = "builtin"
	SourceTheme   = "theme"
	SourceProject = "project"
)

// Layer is one source of translation strings.
type Layer struct {
	Source string `json:"source"`
	// Dir is the layer's folder, relative to the project and slash-separated.
	// Empty for the built-in strings.
	Dir string `json:"dir,omitempty"`
	// Strings maps a language code to its dotted keys and values.
	Strings map[string]map[string]string `json:"strings"`
}

// FileProblem is a string file that could not be read or parsed.
type FileProblem struct {
	Source  string `json:"source"`
	File    string `json:"file"`
	Message string `json:"message"`
	err     error
}

// LoadLayers reads the three string layers (embedded, theme, project)
// without merging them. A file that cannot be read or parsed is reported
// and skipped; a missing directory is an empty layer.
func LoadLayers(embeddedFS fs.FS, projectDir, themeName string) ([]Layer, []FileProblem) {
	var layers []Layer
	var problems []FileProblem
	add := func(source, dir string, fsys fs.FS) {
		strs, probs := readLayer(source, dir, fsys)
		layers = append(layers, Layer{Source: source, Dir: dir, Strings: strs})
		problems = append(problems, probs...)
	}

	// Layer 1: embedded defaults
	if embeddedFS != nil {
		add(SourceBuiltin, "", embeddedFS)
	}

	// Layer 2: theme i18n
	if themeName != "" {
		add(SourceTheme, consts.DirThemes+"/"+themeName+"/"+consts.DirI18n,
			os.DirFS(filepath.Join(projectDir, consts.DirThemes, themeName, consts.DirI18n)))
	}

	// Layer 3: user i18n
	add(SourceProject, consts.DirI18n, os.DirFS(filepath.Join(projectDir, consts.DirI18n)))

	return layers, problems
}

// LoadStrings loads i18n YAML files from three layers (embedded → theme → user).
// Later layers override earlier layers per-key.
func LoadStrings(embeddedFS fs.FS, projectDir, themeName, defaultLang string) (*StringTable, error) {
	layers, problems := LoadLayers(embeddedFS, projectDir, themeName)
	if len(problems) > 0 {
		p := problems[0]
		label := map[string]string{
			SourceBuiltin: "embedded",
			SourceTheme:   "theme",
			SourceProject: "user",
		}[p.Source]
		return nil, fmt.Errorf("loading %s i18n: %w", label, p.err)
	}

	st := NewStringTable(defaultLang)
	for _, layer := range layers {
		for lang, values := range layer.Strings {
			if st.strings[lang] == nil {
				st.strings[lang] = make(map[string]string)
			}
			for k, v := range values {
				st.strings[lang][k] = v
			}
		}
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

// readLayer reads every YAML file at the root of fsys, in name order, into
// lang → dotted key → value. Files of the same language merge, later wins.
func readLayer(source, dir string, fsys fs.FS) (map[string]map[string]string, []FileProblem) {
	out := make(map[string]map[string]string)
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return out, nil // directory may not exist
	}
	var problems []FileProblem
	for _, e := range entries {
		if e.IsDir() || !isYAML(e.Name()) {
			continue
		}
		file := e.Name()
		if dir != "" {
			file = dir + "/" + file
		}
		lang := strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
		data, err := fs.ReadFile(fsys, e.Name())
		if err == nil {
			var raw map[string]any
			if err = yaml.Unmarshal(data, &raw); err != nil {
				err = fmt.Errorf("parsing %s: %w", e.Name(), err)
			} else {
				if out[lang] == nil {
					out[lang] = make(map[string]string)
				}
				flatten("", raw, out[lang])
				continue
			}
		}
		problems = append(problems, FileProblem{Source: source, File: file, Message: err.Error(), err: err})
	}
	return out, problems
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
