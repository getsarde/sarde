// Package build orchestrates the full site build pipeline.
package build

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/frostybee/sarde/internal/engine"
)

// RenderedPage holds a rendered page and its output path.
type RenderedPage struct {
	Page    *engine.Page
	HTML    []byte
	OutPath string // relative to output dir (e.g., "docs/intro/index.html")
}

// Writer handles the Write phase: outputting HTML, aliases, and static files.
type Writer struct {
	OutputDir  string
	ProjectDir string
	Clean      bool
}

// Write outputs all rendered pages, alias redirects, and static files.
func (w *Writer) Write(pages []RenderedPage, aliases map[string]string) error {
	// Clean output directory if configured.
	if w.Clean {
		os.RemoveAll(w.OutputDir)
	}

	// Write rendered HTML pages.
	for _, rp := range pages {
		outPath := filepath.Join(w.OutputDir, filepath.FromSlash(rp.OutPath))
		if err := writeFile(outPath, rp.HTML); err != nil {
			return fmt.Errorf("writing %s: %w", rp.OutPath, err)
		}
	}

	// Write alias redirects.
	for aliasPath, target := range aliases {
		outPath := filepath.Join(w.OutputDir, filepath.FromSlash(PageOutputPath(aliasPath)))
		html := redirectHTML(target)
		if err := writeFile(outPath, []byte(html)); err != nil {
			return fmt.Errorf("writing alias %s: %w", aliasPath, err)
		}
	}

	// Copy static files.
	if err := w.copyStatic(); err != nil {
		return fmt.Errorf("copying static files: %w", err)
	}

	return nil
}

// copyStatic copies the ProjectDir/static/ tree to OutputDir/ preserving structure.
func (w *Writer) copyStatic() error {
	staticDir := filepath.Join(w.ProjectDir, "static")
	info, err := os.Stat(staticDir)
	if err != nil || !info.IsDir() {
		return nil // no static dir — zero-config behavior
	}

	return filepath.Walk(staticDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(staticDir, path)
		destPath := filepath.Join(w.OutputDir, relPath)

		return copyFile(path, destPath)
	})
}

// PageOutputPath converts a RelPermalink to an output file path.
// "/" → "index.html"
// "/docs/intro/" → "docs/intro/index.html"
// "/404.html" → "404.html"
func PageOutputPath(relPermalink string) string {
	if relPermalink == "/" {
		return "index.html"
	}

	p := strings.TrimPrefix(relPermalink, "/")

	// Already has a file extension (e.g., "404.html").
	if filepath.Ext(p) != "" {
		return p
	}

	p = strings.TrimSuffix(p, "/")
	return p + "/index.html"
}

// redirectHTML generates a minimal redirect page.
func redirectHTML(target string) string {
	safe := strings.ReplaceAll(target, `"`, "%22")
	safe = strings.ReplaceAll(safe, `<`, "%3C")
	safe = strings.ReplaceAll(safe, `>`, "%3E")
	return fmt.Sprintf(
		`<!DOCTYPE html><html><head><meta charset="utf-8"><meta http-equiv="refresh" content="0; url=%s"><link rel="canonical" href="%s"></head><body>Redirecting to <a href="%s">%s</a></body></html>`,
		safe, safe, safe, safe,
	)
}

// writeFile creates parent directories and writes data to path.
func writeFile(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// copyFile copies a single file from src to dst, creating parent dirs.
func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}

	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}
