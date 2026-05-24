package shortcode

import (
	"fmt"
	htmltemplate "html/template"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/frostybee/sarde/internal/engine"
)

// mockRenderer wraps inner markdown in <p> tags for testing.
type mockRenderer struct{}

func (m *mockRenderer) Render(markdown string) (engine.RenderResult, error) {
	trimmed := strings.TrimSpace(markdown)
	html := fmt.Sprintf("<p>%s</p>", trimmed)
	return engine.RenderResult{HTML: html}, nil
}

func newTestRegistry(t *testing.T, shortcodes map[string]string) *Registry {
	t.Helper()
	fm := htmltemplate.FuncMap{
		"or": func(vals ...any) any {
			for _, v := range vals {
				if v != nil && v != "" {
					return v
				}
			}
			return ""
		},
	}
	r := &Registry{
		templates: make(map[string]*htmltemplate.Template),
		sources:   make(map[string][]byte),
		funcMap:   fm,
	}
	for name, tmpl := range shortcodes {
		if err := r.Register(name, []byte(tmpl)); err != nil {
			t.Fatalf("registering shortcode %q: %v", name, err)
		}
	}
	return r
}

func TestProcessor_SelfClosing(t *testing.T) {
	reg := newTestRegistry(t, map[string]string{
		"hr": `<hr class="divider">`,
	})
	proc := NewProcessor(reg)

	result, warnings := proc.Process(`before {{< hr />}} after`, &engine.Page{}, &engine.SiteContext{}, nil)

	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
	if !strings.Contains(result, `<hr class="divider">`) {
		t.Errorf("expected rendered shortcode, got %q", result)
	}
	if strings.Contains(result, "{{<") {
		t.Error("raw shortcode syntax should not remain")
	}
}

func TestProcessor_SelfClosingWithParams(t *testing.T) {
	reg := newTestRegistry(t, map[string]string{
		"youtube": `<iframe src="https://youtube.com/embed/{{ .Params.id }}"></iframe>`,
	})
	proc := NewProcessor(reg)

	result, warnings := proc.Process(`{{< youtube id="dQw4w9WgXcQ" />}}`, &engine.Page{}, &engine.SiteContext{}, nil)

	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
	if !strings.Contains(result, "dQw4w9WgXcQ") {
		t.Errorf("expected youtube ID in output, got %q", result)
	}
}

func TestProcessor_WithInner(t *testing.T) {
	reg := newTestRegistry(t, map[string]string{
		"note": `<div class="note">{{ .Inner }}</div>`,
	})
	proc := NewProcessor(reg)
	md := &mockRenderer{}

	result, warnings := proc.Process(`{{< note >}}hello world{{< /note >}}`, &engine.Page{}, &engine.SiteContext{}, md)

	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
	if !strings.Contains(result, `<div class="note">`) {
		t.Errorf("expected note wrapper, got %q", result)
	}
	if !strings.Contains(result, "<p>hello world</p>") {
		t.Errorf("expected inner content rendered as markdown, got %q", result)
	}
}

func TestProcessor_InnerWithMarkdown(t *testing.T) {
	reg := newTestRegistry(t, map[string]string{
		"box": `<div class="box">{{ .Inner }}</div>`,
	})
	proc := NewProcessor(reg)
	md := &mockRenderer{}

	result, _ := proc.Process(`{{< box >}}**bold** text{{< /box >}}`, &engine.Page{}, &engine.SiteContext{}, md)

	if !strings.Contains(result, "**bold** text") {
		t.Errorf("expected inner markdown content passed through renderer, got %q", result)
	}
}

func TestProcessor_UnknownShortcode(t *testing.T) {
	reg := newTestRegistry(t, map[string]string{})
	proc := NewProcessor(reg)

	input := `before {{< unknown />}} after`
	result, warnings := proc.Process(input, &engine.Page{FilePath: "test.md"}, &engine.SiteContext{}, nil)

	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(warnings))
	}
	if !strings.Contains(warnings[0].Message, "unknown shortcode") {
		t.Errorf("expected 'unknown shortcode' warning, got %q", warnings[0].Message)
	}
	if !strings.Contains(result, "{{< unknown />}}") {
		t.Errorf("raw text should be preserved for unknown shortcodes, got %q", result)
	}
}

func TestProcessor_MismatchedClosing(t *testing.T) {
	reg := newTestRegistry(t, map[string]string{
		"alert": `<div>{{ .Inner }}</div>`,
	})
	proc := NewProcessor(reg)

	input := `{{< alert >}}content without closing`
	result, warnings := proc.Process(input, &engine.Page{FilePath: "test.md"}, &engine.SiteContext{}, nil)

	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %d: %v", len(warnings), warnings)
	}
	if !strings.Contains(warnings[0].Message, "unclosed") {
		t.Errorf("expected 'unclosed' warning, got %q", warnings[0].Message)
	}
	if !strings.Contains(result, "{{< alert") {
		t.Errorf("raw text should be preserved for unclosed shortcodes, got %q", result)
	}
}

func TestProcessor_NestedSameName(t *testing.T) {
	reg := newTestRegistry(t, map[string]string{
		"box": `[BOX:{{ .Inner }}]`,
	})
	proc := NewProcessor(reg)
	md := &mockRenderer{}

	input := `{{< box >}}outer {{< box >}}inner{{< /box >}} end{{< /box >}}`
	result, warnings := proc.Process(input, &engine.Page{}, &engine.SiteContext{}, md)

	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
	if !strings.Contains(result, "[BOX:") {
		t.Errorf("expected rendered nested boxes, got %q", result)
	}
}

func TestProcessor_InsideCodeBlock(t *testing.T) {
	reg := newTestRegistry(t, map[string]string{
		"alert": `<div class="alert"></div>`,
	})
	proc := NewProcessor(reg)

	input := "```\n{{< alert />}}\n```"
	result, warnings := proc.Process(input, &engine.Page{}, &engine.SiteContext{}, nil)

	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
	if !strings.Contains(result, "{{< alert />}}") {
		t.Errorf("shortcode inside code block should NOT be expanded, got %q", result)
	}
}

func TestProcessor_EmptyRegistry(t *testing.T) {
	reg := newTestRegistry(t, map[string]string{})
	proc := NewProcessor(reg)

	input := "plain markdown content"
	result, warnings := proc.Process(input, &engine.Page{}, &engine.SiteContext{}, nil)

	if result != input {
		t.Errorf("with empty registry, input should pass through unchanged, got %q", result)
	}
	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
}

func TestProcessor_MultipleShortcodes(t *testing.T) {
	reg := newTestRegistry(t, map[string]string{
		"a": `[A]`,
		"b": `[B]`,
	})
	proc := NewProcessor(reg)

	result, warnings := proc.Process(`{{< a />}} middle {{< b />}}`, &engine.Page{}, &engine.SiteContext{}, nil)

	if len(warnings) != 0 {
		t.Errorf("unexpected warnings: %v", warnings)
	}
	if !strings.Contains(result, "[A] middle [B]") {
		t.Errorf("expected both shortcodes expanded, got %q", result)
	}
}

func TestNewRegistry_LoadsEmbeddedShortcodes(t *testing.T) {
	fs := fstest.MapFS{
		"shortcodes/alert.html": &fstest.MapFile{
			Data: []byte(`<div class="alert">{{ .Inner }}</div>`),
		},
		"shortcodes/note.html": &fstest.MapFile{
			Data: []byte(`<aside>{{ .Inner }}</aside>`),
		},
	}

	fm := htmltemplate.FuncMap{}
	reg, err := NewRegistry(fs, fm)
	if err != nil {
		t.Fatalf("NewRegistry error: %v", err)
	}

	if !reg.Has("alert") {
		t.Error("expected 'alert' shortcode to be loaded")
	}
	if !reg.Has("note") {
		t.Error("expected 'note' shortcode to be loaded")
	}
	if reg.Has("nonexistent") {
		t.Error("'nonexistent' should not be registered")
	}
}

func TestRegistry_TemplateHash(t *testing.T) {
	fm := htmltemplate.FuncMap{}
	reg := &Registry{
		templates: make(map[string]*htmltemplate.Template),
		sources:   make(map[string][]byte),
		funcMap:   fm,
	}

	hash1 := reg.TemplateHash()
	if hash1 != "" {
		t.Errorf("empty registry should return empty hash, got %q", hash1)
	}

	reg.Register("alert", []byte(`<div>{{ .Inner }}</div>`))
	hash2 := reg.TemplateHash()
	if hash2 == "" {
		t.Error("non-empty registry should return non-empty hash")
	}

	reg.Register("alert", []byte(`<span>{{ .Inner }}</span>`))
	hash3 := reg.TemplateHash()
	if hash3 == hash2 {
		t.Error("hash should change when template source changes")
	}
}

func TestProcessor_TemplateExecutionError(t *testing.T) {
	reg := newTestRegistry(t, map[string]string{
		"bad": `{{ .NonExistentMethod }}`,
	})
	proc := NewProcessor(reg)

	_, warnings := proc.Process(`{{< bad />}}`, &engine.Page{FilePath: "test.md"}, &engine.SiteContext{}, nil)

	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning for execution error, got %d", len(warnings))
	}
	if !strings.Contains(warnings[0].Message, "execution error") {
		t.Errorf("expected execution error warning, got %q", warnings[0].Message)
	}
}
