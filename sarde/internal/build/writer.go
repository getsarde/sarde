// Package build orchestrates the full site build pipeline.
package build

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/frostybee/sarde/internal/engine"
	"golang.org/x/sync/errgroup"
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
	DevMode    bool
	Tracker    *OutputTracker
}

// Write outputs all rendered pages, alias redirects, and static files.
// Returns the number of static files copied.
func (w *Writer) Write(pages []RenderedPage, aliases map[string]string) (int, error) {
	// Pre-create all output directories in a single pass to avoid
	// redundant MkdirAll syscalls during parallel writes.
	dirs := make(map[string]struct{}, len(pages)/4)
	for _, rp := range pages {
		outPath, err := safeOutputPath(w.OutputDir, rp.OutPath)
		if err != nil {
			return 0, err
		}
		dir := filepath.Dir(outPath)
		dirs[dir] = struct{}{}
	}
	for aliasPath := range aliases {
		outPath, err := safeOutputPath(w.OutputDir, PageOutputPath(aliasPath))
		if err != nil {
			return 0, err
		}
		dir := filepath.Dir(outPath)
		dirs[dir] = struct{}{}
	}
	for dir := range dirs {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return 0, fmt.Errorf("creating directory %s: %w", dir, err)
		}
	}

	// Write rendered HTML pages in parallel.
	g, _ := errgroup.WithContext(context.Background())
	g.SetLimit(runtime.NumCPU())
	for _, rp := range pages {
		g.Go(func() error {
			outPath, err := safeOutputPath(w.OutputDir, rp.OutPath)
			if err != nil {
				return err
			}
			if w.Tracker != nil {
				w.Tracker.Track(outPath)
			}
			return os.WriteFile(outPath, rp.HTML, 0o644)
		})
	}
	if err := g.Wait(); err != nil {
		return 0, fmt.Errorf("writing pages: %w", err)
	}

	// Write alias redirects in parallel.
	for aliasPath, target := range aliases {
		g.Go(func() error {
			outPath, err := safeOutputPath(w.OutputDir, PageOutputPath(aliasPath))
			if err != nil {
				return err
			}
			if w.Tracker != nil {
				w.Tracker.Track(outPath)
			}
			return os.WriteFile(outPath, []byte(redirectHTML(target)), 0o644)
		})
	}
	if err := g.Wait(); err != nil {
		return 0, fmt.Errorf("writing aliases: %w", err)
	}

	// Copy static files.
	staticCount, err := w.copyStatic()
	if err != nil {
		return 0, fmt.Errorf("copying static files: %w", err)
	}

	return staticCount, nil
}

// copyStatic copies the ProjectDir/static/ tree to OutputDir/ preserving structure.
// Returns the number of files copied.
func (w *Writer) copyStatic() (int, error) {
	staticDir := filepath.Join(w.ProjectDir, "static")
	info, err := os.Stat(staticDir)
	if err != nil || !info.IsDir() {
		return 0, nil
	}

	type filePair struct{ src, dst string }
	var pairs []filePair
	err = filepath.Walk(staticDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		relPath, _ := filepath.Rel(staticDir, path)
		dst, err := safeOutputPath(w.OutputDir, relPath)
		if err != nil {
			return err
		}
		pairs = append(pairs, filePair{path, dst})
		return nil
	})
	if err != nil {
		return 0, err
	}

	dirs := make(map[string]struct{}, len(pairs)/4)
	for _, p := range pairs {
		dirs[filepath.Dir(p.dst)] = struct{}{}
	}
	for dir := range dirs {
		os.MkdirAll(dir, 0o755)
	}

	g, _ := errgroup.WithContext(context.Background())
	g.SetLimit(runtime.NumCPU())
	for _, p := range pairs {
		g.Go(func() error {
			if w.Tracker != nil {
				w.Tracker.Track(p.dst)
			}
			return copyFile(p.src, p.dst)
		})
	}
	if err := g.Wait(); err != nil {
		return 0, err
	}
	return len(pairs), nil
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

// copyFile copies a single file from src to dst.
// Parent directories must already exist.
func copyFile(src, dst string) error {
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
