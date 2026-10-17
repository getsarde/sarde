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

func writeStringFile(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadLayers_KeepsEachLayerApart(t *testing.T) {
	fsys := fstest.MapFS{
		"en.yaml": &fstest.MapFile{Data: []byte("nav:\n  next: \"Next\"\n  toc: \"On this page\"\n")},
	}
	projectDir := t.TempDir()
	writeStringFile(t, filepath.Join(projectDir, "themes", "acme", "i18n"), "fr.yaml", "nav:\n  next: Suivant\n")
	writeStringFile(t, filepath.Join(projectDir, "i18n"), "en.yml", "nav:\n  next: Forward\nedit: Edit\n")

	layers, problems := LoadLayers(fsys, projectDir, "acme")
	if len(problems) != 0 {
		t.Fatalf("problems: %v", problems)
	}
	if len(layers) != 3 {
		t.Fatalf("got %d layers, want 3", len(layers))
	}
	want := []struct{ source, dir, lang, key, value string }{
		{SourceBuiltin, "", "en", "nav.toc", "On this page"},
		{SourceTheme, "themes/acme/i18n", "fr", "nav.next", "Suivant"},
		{SourceProject, "i18n", "en", "nav.next", "Forward"},
	}
	for i, w := range want {
		l := layers[i]
		if l.Source != w.source || l.Dir != w.dir {
			t.Errorf("layer %d: got %s %q, want %s %q", i, l.Source, l.Dir, w.source, w.dir)
		}
		if got := l.Strings[w.lang][w.key]; got != w.value {
			t.Errorf("layer %d %s %s: got %q, want %q", i, w.lang, w.key, got, w.value)
		}
	}
	if _, ok := layers[2].Strings["en"]["nav.toc"]; ok {
		t.Error("the project layer must not hold the built-in nav.toc")
	}
}

func TestLoadLayers_ReportsABadFileAndKeepsTheRest(t *testing.T) {
	projectDir := t.TempDir()
	dir := filepath.Join(projectDir, "i18n")
	writeStringFile(t, dir, "de.yaml", "nav: [unclosed\n")
	writeStringFile(t, dir, "fr.yaml", "nav:\n  next: Suivant\n")

	layers, problems := LoadLayers(nil, projectDir, "")
	if len(problems) != 1 || problems[0].File != "i18n/de.yaml" || problems[0].Source != SourceProject {
		t.Fatalf("problems: %+v", problems)
	}
	if got := layers[0].Strings["fr"]["nav.next"]; got != "Suivant" {
		t.Errorf("fr nav.next: got %q", got)
	}

	// The build still refuses the bad file.
	if _, err := LoadStrings(nil, projectDir, "", "en"); err == nil {
		t.Error("LoadStrings: want an error for the bad file")
	}
}
