package asset

import (
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"
)

func TestResolver_UserAsset(t *testing.T) {
	tmpDir := t.TempDir()
	assetsDir := filepath.Join(tmpDir, "assets", "css")
	os.MkdirAll(assetsDir, 0o755)
	os.WriteFile(filepath.Join(assetsDir, "main.css"), []byte("body{}"), 0o644)

	r := &Resolver{ProjectDir: tmpDir}

	path, err := r.Resolve("css/main.css")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if path != filepath.Join(assetsDir, "main.css") {
		t.Errorf("got %q, want user asset path", path)
	}
}

func TestResolver_ThemeAsset(t *testing.T) {
	tmpDir := t.TempDir()
	themeDir := filepath.Join(tmpDir, "themes", "mytheme", "assets", "css")
	os.MkdirAll(themeDir, 0o755)
	os.WriteFile(filepath.Join(themeDir, "theme.css"), []byte("h1{}"), 0o644)

	r := &Resolver{ProjectDir: tmpDir, ThemeName: "mytheme"}

	path, err := r.Resolve("css/theme.css")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if path != filepath.Join(themeDir, "theme.css") {
		t.Errorf("got %q, want theme asset path", path)
	}
}

func TestResolver_EmbeddedAsset(t *testing.T) {
	efs := fstest.MapFS{
		"assets/js/app.js": {Data: []byte("console.log('hi')")},
	}

	r := &Resolver{ProjectDir: t.TempDir(), EmbeddedFS: efs}

	path, err := r.Resolve("js/app.js")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	if path != "embedded:js/app.js" {
		t.Errorf("got %q, want embedded: prefix", path)
	}
}

func TestResolver_NotFound(t *testing.T) {
	r := &Resolver{ProjectDir: t.TempDir()}

	_, err := r.Resolve("nonexistent.css")
	if err == nil {
		t.Error("expected error for missing asset")
	}
}

func TestResolver_UserOverridesTheme(t *testing.T) {
	tmpDir := t.TempDir()
	// Create both user and theme versions.
	userDir := filepath.Join(tmpDir, "assets", "css")
	themeDir := filepath.Join(tmpDir, "themes", "mytheme", "assets", "css")
	os.MkdirAll(userDir, 0o755)
	os.MkdirAll(themeDir, 0o755)
	os.WriteFile(filepath.Join(userDir, "main.css"), []byte("user"), 0o644)
	os.WriteFile(filepath.Join(themeDir, "main.css"), []byte("theme"), 0o644)

	r := &Resolver{ProjectDir: tmpDir, ThemeName: "mytheme"}

	path, err := r.Resolve("css/main.css")
	if err != nil {
		t.Fatalf("Resolve failed: %v", err)
	}
	// User layer should win.
	if path != filepath.Join(userDir, "main.css") {
		t.Errorf("user asset should override theme, got %q", path)
	}
}

func TestResolveContent_ReadsFile(t *testing.T) {
	tmpDir := t.TempDir()
	assetsDir := filepath.Join(tmpDir, "assets")
	os.MkdirAll(assetsDir, 0o755)
	os.WriteFile(filepath.Join(assetsDir, "style.css"), []byte("body{color:red}"), 0o644)

	r := &Resolver{ProjectDir: tmpDir}

	data, err := r.ResolveContent("style.css")
	if err != nil {
		t.Fatalf("ResolveContent failed: %v", err)
	}
	if string(data) != "body{color:red}" {
		t.Errorf("got %q, want css content", string(data))
	}
}

func TestResolveContent_Embedded(t *testing.T) {
	efs := fstest.MapFS{
		"assets/config.json": {Data: []byte(`{"key":"val"}`)},
	}

	r := &Resolver{ProjectDir: t.TempDir(), EmbeddedFS: efs}

	data, err := r.ResolveContent("config.json")
	if err != nil {
		t.Fatalf("ResolveContent failed: %v", err)
	}
	if string(data) != `{"key":"val"}` {
		t.Errorf("got %q", string(data))
	}
}

func TestMediaTypeFromExt(t *testing.T) {
	tests := []struct {
		ext  string
		want string
	}{
		{".jpg", "image/jpeg"},
		{".jpeg", "image/jpeg"},
		{".png", "image/png"},
		{".gif", "image/gif"},
		{".svg", "image/svg+xml"},
		{".webp", "image/webp"},
		{".css", "text/css"},
		{".js", "application/javascript"},
		{".woff2", "font/woff2"},
		{".pdf", "application/pdf"},
		{".mp4", "video/mp4"},
		{".unknown", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			got := MediaTypeFromExt(tt.ext)
			if got != tt.want {
				t.Errorf("MediaTypeFromExt(%q) = %q, want %q", tt.ext, got, tt.want)
			}
		})
	}
}

func TestIsImage(t *testing.T) {
	tests := []struct {
		name  string
		image bool
	}{
		{"photo.jpg", true},
		{"icon.png", true},
		{"logo.svg", true},
		{"hero.webp", true},
		{"style.css", false},
		{"app.js", false},
		{"doc.pdf", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsImage(tt.name)
			if got != tt.image {
				t.Errorf("IsImage(%q) = %v, want %v", tt.name, got, tt.image)
			}
		})
	}
}
