package asset

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFingerprint_Deterministic(t *testing.T) {
	data := []byte("hello world")
	h1 := Fingerprint(data)
	h2 := Fingerprint(data)

	if h1 != h2 {
		t.Errorf("same data produced different hashes: %q vs %q", h1, h2)
	}

	// Should be 8 hex characters.
	if len(h1) != 8 {
		t.Errorf("hash length = %d, want 8", len(h1))
	}
}

func TestFingerprint_DifferentData(t *testing.T) {
	h1 := Fingerprint([]byte("hello"))
	h2 := Fingerprint([]byte("world"))

	if h1 == h2 {
		t.Error("different data produced same hash")
	}
}

func TestFingerprintFile(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.txt")
	os.WriteFile(path, []byte("test content"), 0o644)

	hash, err := FingerprintFile(path)
	if err != nil {
		t.Fatalf("FingerprintFile failed: %v", err)
	}
	if len(hash) != 8 {
		t.Errorf("hash length = %d, want 8", len(hash))
	}

	// Should match direct computation.
	data, _ := os.ReadFile(path)
	expected := Fingerprint(data)
	if hash != expected {
		t.Errorf("FingerprintFile = %q, Fingerprint = %q", hash, expected)
	}
}

func TestFingerprintedName(t *testing.T) {
	tests := []struct {
		name string
		hash string
		want string
	}{
		{"main.css", "a1b2c3d4", "main.a1b2c3d4.css"},
		{"app.min.js", "deadbeef", "app.min.deadbeef.js"},
		{"style.css", "12345678", "style.12345678.css"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FingerprintedName(tt.name, tt.hash)
			if got != tt.want {
				t.Errorf("FingerprintedName(%q, %q) = %q, want %q", tt.name, tt.hash, got, tt.want)
			}
		})
	}
}
