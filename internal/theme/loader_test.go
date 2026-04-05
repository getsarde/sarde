package theme

import (
	"testing"
	"testing/fstest"
)

func TestLoadFromFS_Valid(t *testing.T) {
	efs := fstest.MapFS{
		"theme.yaml": {Data: []byte(`
name: Test Theme
slug: test
version: "1.0.0"
tokens:
  primary: "#3b82f6"
  bg: "#ffffff"
dark_tokens:
  bg: "#0f172a"
presets:
  ocean:
    name: Ocean
    tokens:
      primary: "#0ea5e9"
`)},
	}

	theme, err := LoadFromFS(efs, ".")
	if err != nil {
		t.Fatal(err)
	}
	if theme == nil {
		t.Fatal("expected non-nil theme")
	}
	if theme.Name != "Test Theme" {
		t.Errorf("name: got %q", theme.Name)
	}
	if theme.Tokens["primary"] != "#3b82f6" {
		t.Errorf("primary: got %q", theme.Tokens["primary"])
	}
	if theme.DarkTokens["bg"] != "#0f172a" {
		t.Errorf("dark bg: got %q", theme.DarkTokens["bg"])
	}
	if theme.Presets["ocean"].Tokens["primary"] != "#0ea5e9" {
		t.Errorf("ocean preset primary: got %q", theme.Presets["ocean"].Tokens["primary"])
	}
}

func TestLoadFromFS_NotFound(t *testing.T) {
	efs := fstest.MapFS{}

	theme, err := LoadFromFS(efs, ".")
	if err != nil {
		t.Fatal(err)
	}
	if theme != nil {
		t.Error("expected nil theme for missing file")
	}
}

func TestLoadFromFS_MalformedYAML(t *testing.T) {
	efs := fstest.MapFS{
		"theme.yaml": {Data: []byte(`{{{invalid yaml`)},
	}

	_, err := LoadFromFS(efs, ".")
	if err == nil {
		t.Error("expected error for malformed YAML")
	}
}

func TestLoadFromDir_NotFound(t *testing.T) {
	theme, err := LoadFromDir(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if theme != nil {
		t.Error("expected nil theme for missing dir")
	}
}

func TestDefaultTokens(t *testing.T) {
	tokens := DefaultTokens()
	if tokens["primary"] == "" {
		t.Error("expected primary token")
	}
	if tokens["bg"] == "" {
		t.Error("expected bg token")
	}
}

func TestDefaultDarkTokens(t *testing.T) {
	tokens := DefaultDarkTokens()
	if tokens["bg"] == "" {
		t.Error("expected dark bg token")
	}
}
