// Package build orchestrates the full site build pipeline.
package build

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/sync/errgroup"

	"github.com/getsarde/sarde/internal/atomicwrite"
	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/outputpath"
	"github.com/getsarde/sarde/internal/workers"
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
	Tracker    *OutputTracker
}

// resolvedPage holds a pre-resolved output path alongside the rendered HTML,
// so path computation happens once (serial) and goroutines only do I/O.
type resolvedPage struct {
	outPath string
	html    []byte
}

// resolvedAlias holds a pre-resolved alias redirect.
type resolvedAlias struct {
	outPath string
	html    []byte
}

// Write outputs all rendered pages, alias redirects, and static files.
// Returns the number of static files copied.
func (w *Writer) Write(pages []RenderedPage, aliases map[string]string) (int, error) {
	absOutputDir, err := filepath.Abs(w.OutputDir)
	if err != nil {
		return 0, fmt.Errorf("resolving output dir: %w", err)
	}

	// Pre-resolve all output paths in a single pass. Each path is computed
	// once and reused for directory collection, tracking, and the write itself.
	resolved := make([]resolvedPage, 0, len(pages))
	dirs := make(map[string]struct{}, len(pages)/4)

	for _, rp := range pages {
		outPath, err := outputpath.SafeJoinWithRoot(absOutputDir, rp.OutPath)
		if err != nil {
			return 0, err
		}
		dirs[filepath.Dir(outPath)] = struct{}{}
		resolved = append(resolved, resolvedPage{outPath: outPath, html: rp.HTML})
	}

	resolvedAliases := make([]resolvedAlias, 0, len(aliases))
	for aliasPath, target := range aliases {
		outPath, err := outputpath.SafeJoinWithRoot(absOutputDir, PageOutputPath(aliasPath))
		if err != nil {
			return 0, err
		}
		dirs[filepath.Dir(outPath)] = struct{}{}
		resolvedAliases = append(resolvedAliases, resolvedAlias{
			outPath: outPath,
			html:    []byte(redirectHTML(target)),
		})
	}

	// Pre-create all output directories in parallel.
	dirSlice := make([]string, 0, len(dirs))
	for d := range dirs {
		dirSlice = append(dirSlice, d)
	}
	mg := new(errgroup.Group)
	mg.SetLimit(workers.IOLimit(len(dirSlice)))
	for _, dir := range dirSlice {
		mg.Go(func() error {
			return os.MkdirAll(dir, 0o755)
		})
	}
	if err := mg.Wait(); err != nil {
		return 0, fmt.Errorf("creating directories: %w", err)
	}

	// Write rendered HTML pages in parallel.
	// Track paths via an indexed slice (lock-free) instead of per-goroutine
	// mutex calls.
	tracked := make([]string, len(resolved)+len(resolvedAliases))

	g := new(errgroup.Group)
	g.SetLimit(workers.IOLimit(len(resolved)))
	for i, rp := range resolved {
		i, rp := i, rp
		g.Go(func() error {
			tracked[i] = rp.outPath
			return atomicwrite.WriteFile(rp.outPath, rp.html, 0o644)
		})
	}
	if err := g.Wait(); err != nil {
		return 0, fmt.Errorf("writing pages: %w", err)
	}

	// Write alias redirects in parallel.
	ag := new(errgroup.Group)
	ag.SetLimit(workers.IOLimit(len(resolvedAliases)))
	base := len(resolved)
	for i, ra := range resolvedAliases {
		i, ra := i, ra
		ag.Go(func() error {
			tracked[base+i] = ra.outPath
			return atomicwrite.WriteFile(ra.outPath, ra.html, 0o644)
		})
	}
	if err := ag.Wait(); err != nil {
		return 0, fmt.Errorf("writing aliases: %w", err)
	}

	// Batch-register all written paths with the tracker (single lock acquisition).
	if w.Tracker != nil {
		w.Tracker.TrackBatch(tracked)
	}

	// Copy public files.
	publicCount, err := w.copyPublic()
	if err != nil {
		return 0, fmt.Errorf("copying public files: %w", err)
	}

	return publicCount, nil
}

// copyPublic copies the ProjectDir/public/ tree to OutputDir/ preserving structure.
func (w *Writer) copyPublic() (int, error) {
	publicDir := filepath.Join(w.ProjectDir, consts.DirPublic)
	info, err := os.Stat(publicDir)
	if err != nil || !info.IsDir() {
		return 0, nil
	}

	type filePair struct{ src, dst string }
	var pairs []filePair
	err = filepath.Walk(publicDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		relPath, _ := filepath.Rel(publicDir, path)
		dst, err := outputpath.SafeJoin(w.OutputDir, relPath)
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

	tracked := make([]string, len(pairs))
	g := new(errgroup.Group)
	g.SetLimit(workers.IOLimit(len(pairs)))
	for i, p := range pairs {
		i, p := i, p
		g.Go(func() error {
			tracked[i] = p.dst
			return copyFile(p.src, p.dst)
		})
	}
	if err := g.Wait(); err != nil {
		return 0, err
	}
	if w.Tracker != nil {
		w.Tracker.TrackBatch(tracked)
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

func writeOutputFile(outputDir, relPath string, data []byte) (string, error) {
	path, err := outputpath.SafeJoin(outputDir, relPath)
	if err != nil {
		return "", err
	}
	return path, writeFile(path, data)
}

// writeFile creates parent directories and writes data to path atomically.
func writeFile(path string, data []byte) error {
	return atomicwrite.WriteFile(path, data, 0o644)
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// copyFile atomically copies a single file from src to dst.
// Parent directories are created as needed.
func copyFile(src, dst string) error {
	return atomicwrite.CopyFile(src, dst, 0o644)
}
