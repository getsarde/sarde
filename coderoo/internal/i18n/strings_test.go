package i18n

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestStringTable_Resolve_BasicLookup(t *testing.T) {
	st := NewStringTable("en")
	st.strings["en"] = map[string]string{
		"nav.next":     "Next",
		"nav.previous": "Previous",
	}
	st.strings["fr"] = map[string]string{
		"nav.next":     "Suivant",
		"nav.previous": "Précédent",
	}

	if got := st.Resolve("fr", "nav.next"); got != "Suivant" {
		t.Errorf("got %q, want %q", got, "Suivant")
	}
	if got := st.Resolve("en", "nav.previous"); got != "Previous" {
		t.Errorf("got %q, want %q", got, "Previous")
	}
}

func TestStringTable_Resolve_FallbackToDefault(t *testing.T) {
	st := NewStringTable("en")
	st.strings["en"] = map[string]string{
		"nav.toc": "On this page",
	}
	st.strings["fr"] = map[string]string{}

	if got := st.Resolve("fr", "nav.toc"); got != "On this page" {
		t.Errorf("got %q, want %q", got, "On this page")
	}
}

func TestStringTable_Resolve_FallbackToKey(t *testing.T) {
	st := NewStringTable("en")
	st.strings["en"] = map[string]string{}

	if got := st.Resolve("en", "missing.key"); got != "missing.key" {
		t.Errorf("got %q, want %q", got, "missing.key")
	}
}

func TestStringTable_Resolve_TemplateExpression(t *testing.T) {
	st := NewStringTable("en")
	st.strings["en"] = map[string]string{
		"nav.reading_time": "{{ .Minutes }} min read",
	}

	data := map[string]any{"Minutes": 5}
	got := st.Resolve("en", "nav.reading_time", data)
	if got != "5 min read" {
		t.Errorf("got %q, want %q", got, "5 min read")
	}
}

func TestStringTable_Resolve_TemplateWithData(t *testing.T) {
	st := NewStringTable("en")
	st.strings["en"] = map[string]string{
		"fallback.notice": "Not available in {{ .Lang }}.",
	}

	data := map[string]string{"Lang": "French"}
	got := st.Resolve("en", "fallback.notice", data)
	if got != "Not available in French." {
		t.Errorf("got %q, want %q", got, "Not available in French.")
	}
}

func TestLoadStrings_FromEmbeddedFS(t *testing.T) {
	fsys := fstest.MapFS{
		"en.yaml": &fstest.MapFile{Data: []byte("nav:\n  next: \"Next\"\n  toc: \"On this page\"\n")},
		"fr.yaml": &fstest.MapFile{Data: []byte("nav:\n  next: \"Suivant\"\n")},
	}

	st, err := LoadStrings(fsys, t.TempDir(), "", "en")
	if err != nil {
		t.Fatal(err)
	}

	if got := st.Resolve("en", "nav.next"); got != "Next" {
		t.Errorf("got %q, want %q", got, "Next")
	}
	if got := st.Resolve("fr", "nav.next"); got != "Suivant" {
		t.Errorf("got %q, want %q", got, "Suivant")
	}
	if got := st.Resolve("fr", "nav.toc"); got != "On this page" {
		t.Errorf("got %q, want %q", got, "On this page")
	}
}

func TestLoadStrings_UserOverridesEmbedded(t *testing.T) {
	fsys := fstest.MapFS{
		"en.yaml": &fstest.MapFile{Data: []byte("nav:\n  next: \"Next\"\n")},
	}

	projectDir := t.TempDir()
	userDir := filepath.Join(projectDir, "i18n")
	if err := os.MkdirAll(userDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(userDir, "en.yaml"), []byte("nav:\n  next: \"Forward\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	st, err := LoadStrings(fsys, projectDir, "", "en")
	if err != nil {
		t.Fatal(err)
	}

	if got := st.Resolve("en", "nav.next"); got != "Forward" {
		t.Errorf("got %q, want %q", got, "Forward")
	}
}

func TestFlatten(t *testing.T) {
	m := map[string]any{
		"nav": map[string]any{
			"next":     "Next",
			"previous": "Previous",
		},
		"fallback": map[string]any{
			"notice": "Not available",
		},
	}
	out := make(map[string]string)
	flatten("", m, out)

	expected := map[string]string{
		"nav.next":        "Next",
		"nav.previous":    "Previous",
		"fallback.notice": "Not available",
	}
	for k, want := range expected {
		if got, ok := out[k]; !ok || got != want {
			t.Errorf("key %q: got %q, want %q", k, got, want)
		}
	}
}
