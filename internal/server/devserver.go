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

	ds.rebuilder.SetOnResult(ds.handleRebuildResult)
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
	result := ds.rebuilder.Rebuild(FileChange{Kind: ChangeConfig})
	if result.Error != nil {
		devlog.Error("build", "Initial build failed: %v", result.Error)
		devlog.Warn("build", "Serving stale output (if any). Waiting for file changes...")
		if ds.liveReload {
			msg := ToReloadMessage(FileChange{Kind: ChangeConfig}, result, ds.projectDir)
			ds.hub.SetPendingError(&msg)
		}
	} else {
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

	// Classify the batch to decide how to handle it.
	representative := classifyBatch(changes)

	// All CSS → hot-swap each without rebuilding.
	if representative.Kind == ChangeCSS {
		for _, c := range changes {
			rel, _ := filepath.Rel(ds.projectDir, c.Path)
			ds.hub.Broadcast(ReloadMessage{Type: ReloadCSS, Path: filepath.ToSlash(rel)})
		}
		return
	}

	// Any non-CSS change triggers a rebuild.
	result := ds.rebuilder.Rebuild(representative)
	if result == nil {
		return
	}
	ds.handleRebuildResult(representative, result)
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
		ds.hub.ClearPendingReload()
	} else {
		ds.hub.ClearPendingError()
		ds.hub.ClearPendingReload()
	}
	sent := ds.hub.Broadcast(msg)
	// Only store a pending reload if the broadcast didn't reach any client.
	// This avoids a double-reload: the client that received the broadcast will
	// reload and reconnect, at which point a stale pendingReload would trigger
	// a redundant second reload.
	if result.Success && msg.Type == ReloadFull && sent == 0 {
		ds.hub.SetPendingReload(&msg)
	}

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

// classifyBatch picks the most significant change from a batch to drive rebuild routing.
// Priority: config > template > content > static > css.
// For content changes, all content paths are collected into Paths.
func classifyBatch(changes []FileChange) FileChange {
	priority := map[ChangeKind]int{
		ChangeConfig:   5,
		ChangeTemplate: 4,
		ChangeContent:  3,
		ChangeStatic:   2,
		ChangeCSS:      1,
	}
	best := changes[0]
	for _, c := range changes[1:] {
		if priority[c.Kind] > priority[best.Kind] {
			best = c
		}
		// Keep the earliest detection timestamp for end-to-end timing.
		if !c.DetectedAt.IsZero() && (best.DetectedAt.IsZero() || c.DetectedAt.Before(best.DetectedAt)) {
			best.DetectedAt = c.DetectedAt
		}
	}
	// Collect all content paths so ContentRebuild knows which files changed.
	for _, c := range changes {
		if c.Kind == ChangeContent {
			best.Paths = append(best.Paths, c.Path)
		}
	}
	if len(best.Paths) == 0 {
		best.Paths = []string{best.Path}
	}
	return best
}
