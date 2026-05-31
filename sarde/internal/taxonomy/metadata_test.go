package taxonomy

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadTermMetadata_ValidFile(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	os.MkdirAll(dataDir, 0755)
	os.WriteFile(filepath.Join(dataDir, "tags.yml"), []byte(`
go:
  label: "Go"
  description: "The Go language"
  color: "#00ADD8"
  icon: "gopher"
  hidden: true
  priority: 10
`), 0644)

	meta, err := LoadTermMetadata(dir, "tags")
	if err != nil {
		t.Fatal(err)
	}
	if meta == nil {
		t.Fatal("expected non-nil meta")
	}

	m, ok := meta["go"]
	if !ok {
		t.Fatal("expected 'go' entry")
	}
	if m.Label != "Go" {
		t.Errorf("Label = %q, want %q", m.Label, "Go")
	}
	if m.Description != "The Go language" {
		t.Errorf("Description = %q, want %q", m.Description, "The Go language")
	}
	if m.Color != "#00ADD8" {
		t.Errorf("Color = %q, want %q", m.Color, "#00ADD8")
	}
	if m.Icon != "gopher" {
		t.Errorf("Icon = %q, want %q", m.Icon, "gopher")
	}
	if !m.Hidden {
		t.Error("Hidden = false, want true")
	}
	if m.Priority != 10 {
		t.Errorf("Priority = %d, want 10", m.Priority)
	}
}

func TestLoadTermMetadata_MissingFile(t *testing.T) {
	dir := t.TempDir()

	meta, err := LoadTermMetadata(dir, "tags")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta != nil {
		t.Errorf("expected nil meta for missing file, got %v", meta)
	}
}

func TestLoadTermMetadata_MalformedYAML(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	os.MkdirAll(dataDir, 0755)
	os.WriteFile(filepath.Join(dataDir, "tags.yml"), []byte(`[invalid: yaml: {{`), 0644)

	_, err := LoadTermMetadata(dir, "tags")
	if err == nil {
		t.Fatal("expected error for malformed YAML")
	}
}

func TestLoadTermMetadata_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	os.MkdirAll(dataDir, 0755)
	os.WriteFile(filepath.Join(dataDir, "tags.yml"), []byte(``), 0644)

	meta, err := LoadTermMetadata(dir, "tags")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta != nil {
		t.Errorf("expected nil meta for empty file, got %v", meta)
	}
}

func TestLoadTermMetadataForLang_NoOverlay(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	os.MkdirAll(dataDir, 0755)
	os.WriteFile(filepath.Join(dataDir, "tags.yml"), []byte(`
go:
  label: "Go"
  color: "#00ADD8"
`), 0644)

	meta, err := LoadTermMetadataForLang(dir, "tags", "fr")
	if err != nil {
		t.Fatal(err)
	}
	m := meta["go"]
	if m.Label != "Go" {
		t.Errorf("Label = %q, want %q (base only, no overlay)", m.Label, "Go")
	}
}

func TestLoadTermMetadataForLang_WithOverlay(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	os.MkdirAll(dataDir, 0755)
	os.WriteFile(filepath.Join(dataDir, "tags.yml"), []byte(`
go:
  label: "Go"
  description: "The Go language"
  color: "#00ADD8"
rust:
  label: "Rust"
`), 0644)
	os.WriteFile(filepath.Join(dataDir, "tags.fr.yml"), []byte(`
go:
  label: "Langage Go"
  description: "Le langage Go"
`), 0644)

	meta, err := LoadTermMetadataForLang(dir, "tags", "fr")
	if err != nil {
		t.Fatal(err)
	}

	m := meta["go"]
	if m.Label != "Langage Go" {
		t.Errorf("Label = %q, want %q (overlay wins)", m.Label, "Langage Go")
	}
	if m.Description != "Le langage Go" {
		t.Errorf("Description = %q, want overlay", m.Description)
	}
	if m.Color != "#00ADD8" {
		t.Errorf("Color = %q, want base value (not overridden)", m.Color)
	}

	m2 := meta["rust"]
	if m2.Label != "Rust" {
		t.Errorf("rust Label = %q, want %q (base, no overlay)", m2.Label, "Rust")
	}
}

func TestLoadTermMetadataForLang_EmptyLang(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	os.MkdirAll(dataDir, 0755)
	os.WriteFile(filepath.Join(dataDir, "tags.yml"), []byte(`
go:
  label: "Go"
`), 0644)

	meta, err := LoadTermMetadataForLang(dir, "tags", "")
	if err != nil {
		t.Fatal(err)
	}
	if meta["go"].Label != "Go" {
		t.Errorf("Label = %q, want %q (base only for empty lang)", meta["go"].Label, "Go")
	}
}

func TestLoadTermMetadataForLang_OverlayNewKey(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	os.MkdirAll(dataDir, 0755)
	os.WriteFile(filepath.Join(dataDir, "tags.yml"), []byte(`
go:
  label: "Go"
`), 0644)
	os.WriteFile(filepath.Join(dataDir, "tags.fr.yml"), []byte(`
python:
  label: "Python (FR)"
`), 0644)

	meta, err := LoadTermMetadataForLang(dir, "tags", "fr")
	if err != nil {
		t.Fatal(err)
	}
	if meta["go"].Label != "Go" {
		t.Errorf("go Label = %q, want base", meta["go"].Label)
	}
	if meta["python"].Label != "Python (FR)" {
		t.Errorf("python Label = %q, want overlay-only key", meta["python"].Label)
	}
}

func TestLoadTermMetadata_PartialFields(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	os.MkdirAll(dataDir, 0755)
	os.WriteFile(filepath.Join(dataDir, "tags.yml"), []byte(`
go:
  color: "#00ADD8"
`), 0644)

	meta, err := LoadTermMetadata(dir, "tags")
	if err != nil {
		t.Fatal(err)
	}

	m := meta["go"]
	if m.Color != "#00ADD8" {
		t.Errorf("Color = %q, want %q", m.Color, "#00ADD8")
	}
	if m.Label != "" {
		t.Errorf("Label = %q, want empty", m.Label)
	}
	if m.Hidden {
		t.Error("Hidden = true, want false")
	}
	if m.Priority != 0 {
		t.Errorf("Priority = %d, want 0", m.Priority)
	}
}
