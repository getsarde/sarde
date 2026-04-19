package server

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// ChangeKind classifies a file system change.
type ChangeKind string

const (
	ChangeContent  ChangeKind = "content"
	ChangeTemplate ChangeKind = "template"
	ChangeCSS      ChangeKind = "css"
	ChangeConfig   ChangeKind = "config"
	ChangeStatic   ChangeKind = "static"
)

// FileChange represents a detected filesystem change.
type FileChange struct {
	Path string
	Kind ChangeKind
}

// Watcher monitors project directories for changes and triggers a callback.
type Watcher struct {
	projectDir string
	onChange   func(FileChange)
	debounce   time.Duration
	watcher    *fsnotify.Watcher

	mu      sync.Mutex
	timer   *time.Timer
	pending *FileChange
	done    chan struct{}
}

// NewWatcher creates a file watcher for the given project directory.
func NewWatcher(projectDir string, debounce time.Duration, onChange func(FileChange)) *Watcher {
	return &Watcher{
		projectDir: projectDir,
		onChange:   onChange,
		debounce:   debounce,
		done:       make(chan struct{}),
	}
}

// Start begins watching project directories for changes.
func (w *Watcher) Start() error {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	w.watcher = fsw

	// Add recursive watches on key directories.
	watchDirs := []string{"content", "layouts", "assets", "data", "static", "themes"}
	for _, dir := range watchDirs {
		abs := filepath.Join(w.projectDir, dir)
		if info, err := os.Stat(abs); err == nil && info.IsDir() {
			w.addRecursive(abs)
		}
	}

	// Watch individual config files.
	configFiles := []string{"site.yaml", "theme.yaml", "nav.yaml"}
	for _, f := range configFiles {
		abs := filepath.Join(w.projectDir, f)
		if _, err := os.Stat(abs); err == nil {
			fsw.Add(abs)
		}
	}

	go w.loop()
	return nil
}

// Stop stops the file watcher.
func (w *Watcher) Stop() {
	if w.watcher != nil {
		w.watcher.Close()
	}
	<-w.done
}

func (w *Watcher) loop() {
	defer close(w.done)

	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}
			if !isRelevantOp(event.Op) {
				continue
			}
			if w.shouldIgnore(event.Name) {
				continue
			}

			// If a new directory is created, watch it recursively.
			if event.Op&fsnotify.Create != 0 {
				if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
					w.addRecursive(event.Name)
				}
			}

			change := FileChange{
				Path: event.Name,
				Kind: w.classifyChange(event.Name),
			}
			w.debounceChange(change)

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Watcher error: %v", err)
		}
	}
}

func (w *Watcher) debounceChange(change FileChange) {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.pending = &change

	if w.timer != nil {
		w.timer.Stop()
	}
	w.timer = time.AfterFunc(w.debounce, func() {
		w.mu.Lock()
		pending := w.pending
		w.pending = nil
		w.mu.Unlock()

		if pending != nil {
			w.onChange(*pending)
		}
	})
}

func (w *Watcher) addRecursive(root string) {
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			if w.shouldIgnoreDir(info.Name()) {
				return filepath.SkipDir
			}
			w.watcher.Add(path)
		}
		return nil
	})
}

func (w *Watcher) shouldIgnore(path string) bool {
	// Normalize to forward slashes for consistent matching.
	rel, err := filepath.Rel(w.projectDir, path)
	if err != nil {
		rel = path
	}
	rel = filepath.ToSlash(rel)

	// Ignore output and cache directories.
	if strings.HasPrefix(rel, "dist/") || strings.HasPrefix(rel, "public/") ||
		strings.HasPrefix(rel, ".cache/") || strings.HasPrefix(rel, ".git/") {
		return true
	}

	base := filepath.Base(path)
	// Ignore hidden files.
	if strings.HasPrefix(base, ".") {
		return true
	}
	// Ignore temp files.
	if strings.HasSuffix(base, "~") || strings.HasSuffix(base, ".swp") ||
		strings.HasSuffix(base, ".tmp") || strings.HasPrefix(base, "~") {
		return true
	}

	return false
}

func (w *Watcher) shouldIgnoreDir(name string) bool {
	switch name {
	case ".git", ".cache", "dist", "public", "node_modules", ".svn", ".hg":
		return true
	}
	return strings.HasPrefix(name, ".")
}

func (w *Watcher) classifyChange(path string) ChangeKind {
	rel, err := filepath.Rel(w.projectDir, path)
	if err != nil {
		return ChangeStatic
	}
	rel = filepath.ToSlash(rel)

	// Config files.
	base := filepath.Base(rel)
	if base == "site.yaml" || base == "theme.yaml" || base == "nav.yaml" {
		return ChangeConfig
	}

	// CSS files in static/ or assets/.
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".css" && (strings.HasPrefix(rel, "static/") || strings.HasPrefix(rel, "assets/")) {
		return ChangeCSS
	}

	// Templates and themes.
	if strings.HasPrefix(rel, "layouts/") || strings.HasPrefix(rel, "themes/") {
		return ChangeTemplate
	}

	// Content.
	if strings.HasPrefix(rel, "content/") {
		return ChangeContent
	}

	return ChangeStatic
}

func isRelevantOp(op fsnotify.Op) bool {
	return op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0
}
