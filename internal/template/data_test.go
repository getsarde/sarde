package template

import (
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestLoadDataFile_YAML(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	os.MkdirAll(dataDir, 0o755)
	os.WriteFile(filepath.Join(dataDir, "authors.yaml"), []byte("- name: Alice\n- name: Bob\n"), 0o644)

	var cache sync.Map
	result := loadDataFile(dir, "authors", &cache)
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	list, ok := result.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", result)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 items, got %d", len(list))
	}
}

func TestLoadDataFile_JSON(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	os.MkdirAll(dataDir, 0o755)
	os.WriteFile(filepath.Join(dataDir, "config.json"), []byte(`{"key": "value"}`), 0o644)

	var cache sync.Map
	result := loadDataFile(dir, "config", &cache)
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	m, ok := result.(map[string]any)
	if !ok {
		t.Fatalf("expected map, got %T", result)
	}
	if m["key"] != "value" {
		t.Errorf("got %v", m["key"])
	}
}

func TestLoadDataFile_TOML(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	os.MkdirAll(dataDir, 0o755)
	os.WriteFile(filepath.Join(dataDir, "settings.toml"), []byte("[section]\nkey = \"value\"\n"), 0o644)

	var cache sync.Map
	result := loadDataFile(dir, "settings", &cache)
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

func TestLoadDataFile_NotFound(t *testing.T) {
	dir := t.TempDir()
	var cache sync.Map
	result := loadDataFile(dir, "nonexistent", &cache)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}

func TestLoadDataFile_Caching(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	os.MkdirAll(dataDir, 0o755)
	os.WriteFile(filepath.Join(dataDir, "test.yaml"), []byte("value: 42\n"), 0o644)

	var cache sync.Map

	// First load
	r1 := loadDataFile(dir, "test", &cache)
	if r1 == nil {
		t.Fatal("expected non-nil")
	}

	// Delete the file
	os.Remove(filepath.Join(dataDir, "test.yaml"))

	// Second load should use cache
	r2 := loadDataFile(dir, "test", &cache)
	if r2 == nil {
		t.Fatal("expected cached non-nil")
	}
}

func TestLoadDataFile_YMLExtension(t *testing.T) {
	dir := t.TempDir()
	dataDir := filepath.Join(dir, "data")
	os.MkdirAll(dataDir, 0o755)
	os.WriteFile(filepath.Join(dataDir, "test.yml"), []byte("key: value\n"), 0o644)

	var cache sync.Map
	result := loadDataFile(dir, "test", &cache)
	if result == nil {
		t.Fatal("expected non-nil for .yml")
	}
}
