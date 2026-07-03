package server

import (
	"context"
	"fmt"
	"mime"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"golang.org/x/term"

	"github.com/getsarde/sarde/internal/build"
	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/devlog"
	"github.com/getsarde/sarde/internal/outputpath"
)

func init() {
	mime.AddExtensionType(".css", "text/css; charset=utf-8")
	mime.AddExtensionType(".js", "text/javascript; charset=utf-8")
	mime.AddExtensionType(".mjs", "text/javascript; charset=utf-8")
	mime.AddExtensionType(".json", "application/json")
	mime.AddExtensionType(".svg", "image/svg+xml")
	mime.AddExtensionType(".woff2", "font/woff2")
	mime.AddExtensionType(".woff", "font/woff")
}

// Options configures the dev server.
type Options struct {
	ProjectDir     string
	OutputDir      string
	Host           string
	Port           int
	LiveReload     bool
	Version        string
	BasePath       string // normalized: "/docs/" or "/"
	BuilderFactory func() *build.SiteBuilder
	ThemeDevDirs   []string // external dirs to watch as ChangeTemplate (for --theme-dev)
}

// DevServer runs the development HTTP server with live reload.
type DevServer struct {
	projectDir string
	outputDir  string
	host       string
	port       int
	liveReload bool
	version    string
	basePath   string // normalized: "/docs/" or "/"
	hub        *Hub
	watcher    *Watcher
	rebuilder  *Rebuilder
	server     *http.Server // constructed in New(), immutable afterwards

	ready     chan int // actual bound port; sent once then closed (closed empty on pre-bind failure)
	readyOnce sync.Once

	stopMu  sync.Mutex
	stopped bool
}

// New creates a new DevServer.
func New(opts Options) *DevServer {
	host := opts.Host
	if host == "" {
		host = consts.DefaultHost
	}
	version := opts.Version
	if version == "" {
		version = "vdev"
	}
	basePath := opts.BasePath
	if basePath == "" {
		basePath = "/"
	}
	ds := &DevServer{
		projectDir: opts.ProjectDir,
		outputDir:  opts.OutputDir,
		host:       host,
		port:       opts.Port,
		liveReload: opts.LiveReload,
		version:    version,
		basePath:   basePath,
		hub:        NewHub(),
		rebuilder:  NewRebuilder(opts.BuilderFactory, opts.ProjectDir),
	}

	ds.watcher = NewWatcher(opts.ProjectDir, opts.OutputDir, 150*time.Millisecond, ds.onFileChange)
	for _, dir := range opts.ThemeDevDirs {
		ds.watcher.AddExternalDir(dir, ChangeTemplate)
	}
	ds.ready = make(chan int, 1)
	// Constructing the server here (single write before any goroutine exists)
	// avoids the Start/Stop data race on ds.server.
	ds.server = &http.Server{Handler: devRequestLogger(ds.buildHandler())}
	return ds
}

// buildHandler assembles the dev HTTP handler (file serving, live-reload
// script injection, base-path routing). All inputs are set in New().
func (ds *DevServer) buildHandler() http.Handler {
	mux := http.NewServeMux()
	if ds.liveReload {
		mux.HandleFunc("/ws", ds.hub.HandleWS)
	}

	var handler http.Handler = ds.fileHandler()
	if ds.liveReload {
		handler = ds.injectScript(handler)
	}
	if ds.basePath != "/" {
		bp := strings.TrimRight(ds.basePath, "/")
		mux.Handle(bp+"/", http.StripPrefix(bp, handler))
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/" {
				http.Redirect(w, r, bp+"/", http.StatusFound)
				return
			}
			http.NotFound(w, r)
		})
	} else {
		mux.Handle("/", handler)
	}
	return mux
}

// Ready reports the actual bound port: the port is sent once after the
// listener binds, then the channel is closed. If Start fails or the server
// is stopped before binding, the channel is closed without a value.
func (ds *DevServer) Ready() <-chan int { return ds.ready }

func (ds *DevServer) signalReady(port int) {
	ds.readyOnce.Do(func() {
		if port > 0 {
			ds.ready <- port // cap 1, never blocks
		}
		close(ds.ready)
	})
}

func (ds *DevServer) isStopped() bool {
	ds.stopMu.Lock()
	defer ds.stopMu.Unlock()
	return ds.stopped
}

// bind acquires the TCP listener, retrying up to 10 consecutive ports on
// EADDRINUSE, and returns the listener plus the actually-bound port.
func (ds *DevServer) bind() (net.Listener, int, error) {
	basePort := ds.port
	for i := 0; i < 10; i++ {
		tryAddr := fmt.Sprintf("%s:%d", ds.host, basePort+i)
		ln, listenErr := net.Listen("tcp", tryAddr)
		if listenErr == nil {
			return ln, ln.Addr().(*net.TCPAddr).Port, nil
		}
		if i == 9 {
			return nil, 0, fmt.Errorf("could not bind to any port in range %d–%d: %w", basePort, basePort+9, listenErr)
		}
		devlog.Warn("server", "Port %d in use, trying %d...", basePort+i, basePort+i+1)
	}
	return nil, 0, fmt.Errorf("could not bind to any port starting at %d", basePort)
}

// Start binds the listener, runs the initial build, starts the file watcher,
// and serves HTTP. It blocks until the server is stopped. The listener is
// bound up front so Ready() reports the actual port immediately; connections
// arriving before the initial build finishes queue in the accept backlog.
func (ds *DevServer) Start() error {
	defer ds.signalReady(0) // closes ready on every failure/early-return path

	ln, actualPort, err := ds.bind()
	if err != nil {
		return err
	}
	defer ln.Close() // releases the port on post-bind failure paths; harmless double-close after Shutdown

	if ds.isStopped() { // Stop raced before/at bind
		return http.ErrServerClosed
	}
	ds.signalReady(actualPort)

	// Initial build (treated as a config change to force full init).
	initial := FileChange{Kind: ChangeConfig}
	_, result := ds.rebuilder.Rebuild(initial)
	if result.Error != nil {
		devlog.Error("build", "Initial build failed: %v", result.Error)
		devlog.Warn("build", "Serving stale output (if any). Waiting for file changes...")
		if ds.liveReload {
			msg := ToReloadMessage(initial, result, ds.projectDir)
			ds.hub.SetPendingError(&msg)
		}
	} else {
		ds.hub.BumpBuildID()
		devlog.Log("build", "Built %d pages in %s", result.PageCount, result.Duration)
	}

	// Start file watcher.
	if err := ds.watcher.Start(); err != nil {
		return fmt.Errorf("starting file watcher: %w", err)
	}
	if ds.isStopped() {
		// Stop raced during the build: its watcher.Stop() saw started==false
		// and was a no-op, so shut the watcher down here.
		ds.watcher.Stop()
		return http.ErrServerClosed
	}

	localURL := fmt.Sprintf("http://localhost:%d", actualPort)
	if ds.basePath != "/" {
		localURL += strings.TrimRight(ds.basePath, "/")
	}
	devlog.Banner(ds.version, localURL, ds.host, result.Duration)
	devlog.Log("watch", "Watching: content/, layouts/, assets/, data/")

	// Emit JSON ready signal on stdout for the Tauri desktop app to detect.
	// This deliberately stays AFTER the initial build — the desktop app
	// navigates to the URL when it sees this line.
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		fmt.Fprintf(os.Stdout, "{\"ready\":true,\"port\":%d}\n", actualPort)
	}

	return ds.server.Serve(ln)
}

// Stop gracefully shuts down the dev server. Safe to call before, during, or
// after Start; the server is single-use afterwards.
func (ds *DevServer) Stop() error {
	ds.stopMu.Lock()
	ds.stopped = true
	ds.stopMu.Unlock()

	if ds.watcher != nil {
		ds.watcher.Stop()
	}
	if ds.server == nil {
		// Zero-value DevServer (tests construct bare literals); nothing to shut down.
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// Shutdown before Serve is safe: it marks the server in-shutdown, so a
	// later Serve returns http.ErrServerClosed immediately.
	return ds.server.Shutdown(ctx)
}

func (ds *DevServer) onFileChange(changes []FileChange) {
	// Log all changes in the batch.
	for _, c := range changes {
		rel, _ := filepath.Rel(ds.projectDir, c.Path)
		rel = filepath.ToSlash(rel)
		devlog.Log("watch", "Changed [%s]: %s", c.Kind, rel)
	}

	// Merge the batch into a single change to decide how to handle it.
	representative := mergeChanges(changes)

	// All CSS → hot-swap each without rebuilding. The hot-swap path skips the
	// rebuild that copies static/ into the output dir, and fileHandler serves
	// only from the output dir, so sync each changed file there first or the
	// browser's re-fetch would read the stale copy.
	if representative.Kind == ChangeCSS {
		for _, c := range changes {
			if err := ds.syncStaticFile(c.Path); err != nil {
				devlog.Warn("watch", "CSS hot-swap: could not sync %s to output dir: %v", c.Path, err)
			}
			rel, _ := filepath.Rel(ds.projectDir, c.Path)
			ds.hub.Broadcast(ReloadMessage{Type: ReloadCSS, Path: filepath.ToSlash(rel)})
		}
		return
	}

	// Any non-CSS change triggers a rebuild. A nil result means the change was
	// merged into an in-flight rebuild, whose caller broadcasts the final
	// result; the returned change is the one the result actually belongs to.
	executed, result := ds.rebuilder.Rebuild(representative)
	if result == nil {
		return
	}
	ds.handleRebuildResult(executed, result)
}

// syncStaticFile copies a changed static/ file to its mirrored path in the
// output dir, matching the mapping Writer.copyStatic uses during full builds
// (static/<rel> → <outputDir>/<rel>).
func (ds *DevServer) syncStaticFile(path string) error {
	staticDir := filepath.Join(ds.projectDir, consts.DirStatic)
	rel, err := filepath.Rel(staticDir, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("file is not under %s", consts.DirStatic)
	}
	dst, err := outputpath.SafeJoin(ds.outputDir, rel)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

func (ds *DevServer) handleRebuildResult(change FileChange, result *RebuildResult) {
	if result.Error != nil {
		devlog.Error("build", "Rebuild failed: %v", result.Error)
	} else {
		devlog.Log("build", "Rebuilt %d pages in %s", result.PageCount, result.Duration)
	}

	msg := ToReloadMessage(change, result, ds.projectDir)
	if result.Error != nil {
		ds.hub.SetPendingError(&msg)
	} else {
		ds.hub.ClearPendingError()
	}
	// Stamp successful reloads with a fresh build ID before broadcasting.
	// Clients that miss this broadcast (disconnected, dead connection) catch
	// up via the ReloadSync announcement when they reconnect.
	if result.Success && msg.Type == ReloadFull {
		msg.BuildID = ds.hub.BumpBuildID()
	}
	ds.hub.Broadcast(msg)

	if result.Success && msg.Type == ReloadFull {
		var items []WarningItem
		for _, w := range result.Warnings {
			if w.Field != "syntax" {
				continue
			}
			items = append(items, WarningItem{
				File:    w.File,
				Line:    0,
				Message: w.Message,
			})
		}
		if len(items) > 0 {
			ds.hub.Broadcast(ReloadMessage{
				Type:     ReloadWarning,
				Warnings: items,
			})
		} else {
			ds.hub.Broadcast(ReloadMessage{Type: ReloadWarning})
		}
	}
}

// changePriority ranks change kinds for batch classification and pending-merge
// decisions: config > template > content > static > css.
var changePriority = map[ChangeKind]int{
	ChangeConfig:   5,
	ChangeTemplate: 4,
	ChangeContent:  3,
	ChangeStatic:   2,
	ChangeCSS:      1,
}

// mergeChanges reduces a batch of changes to a single change that drives
// rebuild routing without losing work:
//   - config or template present: that kind wins (a fresh builder plus a full
//     build covers everything else in the batch)
//   - content mixed with static/css: escalated to ChangeStatic (full build on
//     the reused builder; the incremental content path would silently drop the
//     static/css files, which only a full build copies to the output dir)
//   - content only: incremental rebuild with the union of all content paths
//   - static/css only: highest-priority kind
//
// Paths carries the deduped union of all content paths in the batch, and
// DetectedAt the earliest detection timestamp for end-to-end timing.
func mergeChanges(changes []FileChange) FileChange {
	best := changes[0]
	for _, c := range changes[1:] {
		if changePriority[c.Kind] > changePriority[best.Kind] {
			best = c
		}
	}

	seen := make(map[string]struct{})
	var contentPaths []string
	addPath := func(p string) {
		if p == "" {
			return
		}
		if _, ok := seen[p]; ok {
			return
		}
		seen[p] = struct{}{}
		contentPaths = append(contentPaths, p)
	}
	hasNonContent := false
	for _, c := range changes {
		if c.Kind == ChangeContent {
			if len(c.Paths) > 0 {
				for _, p := range c.Paths {
					addPath(p)
				}
			} else {
				addPath(c.Path)
			}
		} else {
			hasNonContent = true
		}
	}

	if best.Kind == ChangeContent && hasNonContent {
		best.Kind = ChangeStatic
	}

	best.Paths = contentPaths
	if len(best.Paths) == 0 {
		best.Paths = []string{best.Path}
	}

	best.DetectedAt = time.Time{}
	for _, c := range changes {
		if !c.DetectedAt.IsZero() && (best.DetectedAt.IsZero() || c.DetectedAt.Before(best.DetectedAt)) {
			best.DetectedAt = c.DetectedAt
		}
	}
	return best
}
