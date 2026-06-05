package server

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/frostybee/sarde/internal/devlog"

	"github.com/frostybee/sarde/internal/consts"
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
	Path  string
	Paths []string // all changed file paths in the batch (populated for content changes)
	Kind  ChangeKind
}

// externalWatch represents a directory outside the project tree that should
// be watched, with all changes classified as the given kind.
type externalWatch struct {
	dir  string
	kind ChangeKind
}

// Watcher monitors project directories for changes and triggers a callback.
type Watcher struct {
	projectDir   string
	outputDir    string // build output dir relative to projectDir (slash form); changes under it are ignored
	onChange     func([]FileChange)
	debounce     time.Duration
	watcher      *fsnotify.Watcher
	externalDirs []externalWatch

	mu      sync.Mutex
	timer   *time.Timer
	pending []FileChange
	done    chan struct{}
	started bool
}

// NewWatcher creates a file watcher for the given project directory.
// outputDir is the build output directory (absolute or relative); changes
// under it are ignored so a build's own writes don't trigger a rebuild loop.
func NewWatcher(projectDir, outputDir string, debounce time.Duration, onChange func([]FileChange)) *Watcher {
	rel := ""
	if outputDir != "" {
		if r, err := filepath.Rel(projectDir, outputDir); err == nil {
			rel = filepath.ToSlash(r)
		}
	}
	return &Watcher{
		projectDir: projectDir,
		outputDir:  rel,
		onChange:   onChange,
		debounce:   debounce,
		done:       make(chan struct{}),
	}
}

// AddExternalDir registers a directory outside the project tree to watch.
// All changes under it are classified with the given kind. Must be called
// before Start().
func (w *Watcher) AddExternalDir(dir string, kind ChangeKind) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		abs = dir
	}
	w.externalDirs = append(w.externalDirs, externalWatch{dir: abs, kind: kind})
}

// Start begins watching project directories for changes.
func (w *Watcher) Start() error {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	w.watcher = fsw

	// Add recursive watches on key directories.
	watchDirs := []string{consts.DirContent, consts.DirLayouts, consts.DirAssets, consts.DirData, consts.DirStatic, consts.DirThemes}
	for _, dir := range watchDirs {
		abs := filepath.Join(w.projectDir, dir)
		if info, err := os.Stat(abs); err == nil && info.IsDir() {
			w.addRecursive(abs)
		}
	}

	// Watch individual config files.
	configFiles := []string{consts.FileSiteConfig, consts.FileThemeConfig, consts.FileNavConfig}
	for _, f := range configFiles {
		abs := filepath.Join(w.projectDir, f)
		if _, err := os.Stat(abs); err == nil {
			fsw.Add(abs)
		}
	}

	// Watch external directories (e.g. theme-dev source tree).
	for _, ext := range w.externalDirs {
		if info, err := os.Stat(ext.dir); err == nil && info.IsDir() {
			w.addRecursive(ext.dir)
		}
	}

	w.mu.Lock()
	w.started = true
	w.mu.Unlock()
	go w.loop()
	return nil
}

// Stop stops the file watcher.
func (w *Watcher) Stop() {
	if w.watcher != nil {
		w.watcher.Close()
	}
	// Only wait for loop() to exit if it was actually started; otherwise
	// done is never closed and this would deadlock (e.g. after a failed Start).
	w.mu.Lock()
	started := w.started
	w.mu.Unlock()
	if started {
		<-w.done
	}
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
			devlog.Error("watch", "%v", err)
		}
	}
}

func (w *Watcher) debounceChange(change FileChange) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Dedup by path: if this path is already pending, replace with the latest event.
	found := false
	for i, p := range w.pending {
		if p.Path == change.Path {
			w.pending[i] = change
			found = true
			break
		}
	}
	if !found {
		w.pending = append(w.pending, change)
	}

	if w.timer != nil {
		w.timer.Stop()
	}
	w.timer = time.AfterFunc(w.debounce, w.firePending)
}

// firePending drains the pending batch under the lock and calls onChange.
func (w *Watcher) firePending() {
	w.mu.Lock()
	batch := w.pending
	w.pending = nil
	w.mu.Unlock()

	if len(batch) > 0 {
		w.onChange(batch)
	}
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

	// Ignore the configured build output directory so the build's own writes
	// don't trigger a rebuild loop (handles non-default build.output values).
	if w.outputDir != "" && w.outputDir != "." &&
		(rel == w.outputDir || strings.HasPrefix(rel, w.outputDir+"/")) {
		return true
	}

	// Ignore default output and cache directories.
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
	// Skip the configured output dir if it happens to nest under a watched root.
	if w.outputDir != "" && name == filepath.Base(w.outputDir) {
		return true
	}
	switch name {
	case ".git", ".cache", "dist", "public", "node_modules", ".svn", ".hg":
		return true
	}
	return strings.HasPrefix(name, ".")
}

func (w *Watcher) classifyChange(path string) ChangeKind {
	// Check external watched directories first (e.g. --theme-dev source tree).
	absPath, _ := filepath.Abs(path)
	for _, ext := range w.externalDirs {
		if isUnderDir(absPath, ext.dir) {
			return ext.kind
		}
	}

	rel, err := filepath.Rel(w.projectDir, path)
	if err != nil {
		return ChangeStatic
	}
	rel = filepath.ToSlash(rel)

	// Config files.
	base := filepath.Base(rel)
	if base == consts.FileSiteConfig || base == consts.FileThemeConfig || base == consts.FileNavConfig {
		return ChangeConfig
	}

	// CSS files in static/ can be hot-swapped directly. CSS under assets/
	// may be bundled/fingerprinted, so it must go through a rebuild.
	fext := strings.ToLower(filepath.Ext(path))
	if fext == ".css" && strings.HasPrefix(rel, "static/") {
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

func isUnderDir(path, dir string) bool {
	path = filepath.Clean(path)
	dir = filepath.Clean(dir)
	return strings.HasPrefix(strings.ToLower(path), strings.ToLower(dir)+string(filepath.Separator)) || strings.EqualFold(path, dir)
}

func isRelevantOp(op fsnotify.Op) bool {
	return op&(fsnotify.Write|fsnotify.Create|fsnotify.Remove|fsnotify.Rename) != 0
}
