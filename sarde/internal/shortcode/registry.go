package shortcode

import (
	"crypto/sha256"
	"fmt"
	htmltemplate "html/template"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/frostybee/sarde/internal/consts"
)

// Registry manages shortcode templates with three-layer override support.
// Shortcodes are user-defined content components invoked from markdown
// via {{< name >}} syntax. The registry is immutable after construction
// and safe for concurrent read access.
type Registry struct {
	templates map[string]*htmltemplate.Template
	sources   map[string][]byte
	funcMap   htmltemplate.FuncMap
}

// NewRegistry creates a Registry and loads built-in shortcodes from the embedded FS.
func NewRegistry(embeddedFS fs.FS, funcMap htmltemplate.FuncMap) (*Registry, error) {
	r := &Registry{
		templates: make(map[string]*htmltemplate.Template),
		sources:   make(map[string][]byte),
		funcMap:   funcMap,
	}

	if embeddedFS != nil {
		if err := r.loadFromFS(embeddedFS, consts.DirShortcodes); err != nil {
			return nil, fmt.Errorf("loading embedded shortcodes: %w", err)
		}
	}

	return r, nil
}

// Register parses raw template bytes and registers the result under name.
func (r *Registry) Register(name string, content []byte) error {
	tmpl, err := htmltemplate.New(name).Funcs(r.funcMap).Parse(string(content))
	if err != nil {
		return fmt.Errorf("parsing shortcode %q: %w", name, err)
	}
	r.templates[name] = tmpl
	r.sources[name] = content
	return nil
}

// Resolve returns the template for a named shortcode, or nil if not registered.
func (r *Registry) Resolve(name string) *htmltemplate.Template {
	return r.templates[name]
}

// Has returns true if the registry contains a shortcode with the given name.
func (r *Registry) Has(name string) bool {
	_, ok := r.templates[name]
	return ok
}

// Names returns all registered shortcode names.
func (r *Registry) Names() []string {
	names := make([]string, 0, len(r.templates))
	for name := range r.templates {
		names = append(names, name)
	}
	return names
}

// LoadOverridesFromDir loads shortcode overrides from a filesystem directory.
// Each .html file overrides the shortcode named after the file (without extension).
func (r *Registry) LoadOverridesFromDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".html") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".html")
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return fmt.Errorf("reading shortcode override %q: %w", name, err)
		}
		if err := r.Register(name, data); err != nil {
			return err
		}
	}
	return nil
}

// TemplateHash returns a sha256 hex digest over all registered template sources,
// sorted by name. Used for cache invalidation in dev mode.
func (r *Registry) TemplateHash() string {
	if len(r.sources) == 0 {
		return ""
	}
	names := make([]string, 0, len(r.sources))
	for k := range r.sources {
		names = append(names, k)
	}
	sort.Strings(names)

	h := sha256.New()
	for _, name := range names {
		h.Write([]byte(name))
		h.Write(r.sources[name])
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func (r *Registry) loadFromFS(efs fs.FS, dir string) error {
	entries, err := fs.ReadDir(efs, dir)
	if err != nil {
		return nil
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".html") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".html")
		data, err := fs.ReadFile(efs, path.Join(dir, e.Name()))
		if err != nil {
			return fmt.Errorf("reading embedded shortcode %q: %w", name, err)
		}
		if err := r.Register(name, data); err != nil {
			return err
		}
	}
	return nil
}
