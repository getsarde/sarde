package component

import (
	"bytes"
	"fmt"
	htmltemplate "html/template"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/frostybee/sarde/internal/consts"
)

// Registry manages component templates with override support.
// Components are engine-owned UI slots with safe defaults that
// can be overridden by themes and users.
type Registry struct {
	slots   map[string]*htmltemplate.Template
	funcMap htmltemplate.FuncMap
}

// NewRegistry creates a Registry and loads default component templates from the embedded FS.
// funcMap is shared with the template engine so components can use the same functions.
func NewRegistry(embeddedFS fs.FS, funcMap htmltemplate.FuncMap) (*Registry, error) {
	r := &Registry{
		slots:   make(map[string]*htmltemplate.Template),
		funcMap: funcMap,
	}

	if embeddedFS != nil {
		if err := r.loadFromFS(embeddedFS, consts.DirComponents); err != nil {
			return nil, fmt.Errorf("loading embedded components: %w", err)
		}
	}

	return r, nil
}

// Register parses raw template bytes and registers the result for a named slot.
func (r *Registry) Register(name string, content []byte) error {
	tmpl, err := htmltemplate.New(name).Funcs(r.funcMap).Parse(string(content))
	if err != nil {
		return fmt.Errorf("parsing component %q: %w", name, err)
	}
	r.slots[name] = tmpl
	return nil
}

// RegisterTemplate directly registers a pre-parsed template for a named slot.
func (r *Registry) RegisterTemplate(name string, tmpl *htmltemplate.Template) {
	r.slots[name] = tmpl
}

// Resolve returns the template for a named component slot, or nil if not registered.
func (r *Registry) Resolve(name string) *htmltemplate.Template {
	return r.slots[name]
}

// RenderComponent executes a component template and returns the rendered HTML.
// Returns empty HTML (no error) for unknown slots to allow graceful degradation.
func (r *Registry) RenderComponent(name string, data any) (htmltemplate.HTML, error) {
	tmpl := r.slots[name]
	if tmpl == nil {
		return "", nil
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("rendering component %q: %w", name, err)
	}
	return htmltemplate.HTML(buf.String()), nil
}

// RenderComponentWithFuncs executes a component with a per-render FuncMap.
func (r *Registry) RenderComponentWithFuncs(name string, data any, funcs htmltemplate.FuncMap) (htmltemplate.HTML, error) {
	tmpl := r.slots[name]
	if tmpl == nil {
		return "", nil
	}
	clone, err := tmpl.Clone()
	if err != nil {
		return "", fmt.Errorf("cloning component %q: %w", name, err)
	}
	clone.Funcs(funcs)

	var buf bytes.Buffer
	if err := clone.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("rendering component %q: %w", name, err)
	}
	return htmltemplate.HTML(buf.String()), nil
}

// LoadOverridesFromDir loads component overrides from a filesystem directory.
// Each .html file in the directory overrides the component named after the file
// (without extension). For example, Header.html overrides the "Header" component.
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
			return fmt.Errorf("reading component override %q: %w", name, err)
		}
		if err := r.Register(name, data); err != nil {
			return err
		}
	}
	return nil
}

// loadFromFS loads all .html files from a directory in an fs.FS.
func (r *Registry) loadFromFS(efs fs.FS, dir string) error {
	entries, err := fs.ReadDir(efs, dir)
	if err != nil {
		return nil // directory may not exist
	}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".html") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".html")
		data, err := fs.ReadFile(efs, path.Join(dir, e.Name()))
		if err != nil {
			return fmt.Errorf("reading embedded component %q: %w", name, err)
		}
		if err := r.Register(name, data); err != nil {
			return err
		}
	}
	return nil
}

// SlotNames returns the names of all registered component slots.
func (r *Registry) SlotNames() []string {
	names := make([]string, 0, len(r.slots))
	for name := range r.slots {
		names = append(names, name)
	}
	return names
}
