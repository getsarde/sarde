package server

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/coderoo-dev/coderoo/embedded"
	"github.com/coderoo-dev/coderoo/internal/build"
)

// Options configures the dev server.
type Options struct {
	ProjectDir     string
	OutputDir      string
	Port           int
	LiveReload     bool
	BuilderFactory func() *build.SiteBuilder
}

// DevServer runs the development HTTP server with live reload.
type DevServer struct {
	projectDir string
	outputDir  string
	port       int
	liveReload bool
	hub        *Hub
	watcher    *Watcher
	rebuilder  *Rebuilder
	server     *http.Server
}

// New creates a new DevServer.
func New(opts Options) *DevServer {
	ds := &DevServer{
		projectDir: opts.ProjectDir,
		outputDir:  opts.OutputDir,
		port:       opts.Port,
		liveReload: opts.LiveReload,
		hub:        NewHub(),
		rebuilder:  NewRebuilder(opts.BuilderFactory, opts.ProjectDir),
	}

	ds.watcher = NewWatcher(opts.ProjectDir, 50*time.Millisecond, ds.onFileChange)
	return ds
}

// Start runs the initial build, starts the file watcher, and serves HTTP.
// It blocks until the server is stopped.
func (ds *DevServer) Start() error {
	// Initial build.
	result := ds.rebuilder.Rebuild()
	if result.Error != nil {
		log.Printf("Initial build failed: %v", result.Error)
		log.Println("Serving stale output (if any). Waiting for file changes...")
	} else {
		log.Printf("Built %d pages in %s", result.PageCount, result.Duration)
	}

	// Start file watcher.
	if err := ds.watcher.Start(); err != nil {
		return fmt.Errorf("starting file watcher: %w", err)
	}

	// Build HTTP handler.
	mux := http.NewServeMux()
	if ds.liveReload {
		mux.HandleFunc("/ws", ds.hub.HandleWS)
	}

	var handler http.Handler = ds.fileHandler()
	if ds.liveReload {
		handler = ds.injectScript(handler)
	}
	mux.Handle("/", handler)

	ds.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", ds.port),
		Handler: mux,
	}

	log.Printf("Dev server running at http://localhost:%d", ds.port)
	log.Printf("Watching: content/, layouts/, assets/, data/")

	return ds.server.ListenAndServe()
}

// Stop gracefully shuts down the dev server.
func (ds *DevServer) Stop() error {
	ds.watcher.Stop()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return ds.server.Shutdown(ctx)
}

func (ds *DevServer) onFileChange(change FileChange) {
	rel, _ := filepath.Rel(ds.projectDir, change.Path)
	rel = filepath.ToSlash(rel)
	log.Printf("Changed [%s]: %s", change.Kind, rel)

	// CSS changes in static/ don't need a rebuild — files are served from disk.
	if change.Kind == ChangeCSS {
		ds.hub.Broadcast(ReloadMessage{Type: ReloadCSS, Path: rel})
		return
	}

	result := ds.rebuilder.Rebuild()
	if result.Error != nil {
		log.Printf("Rebuild failed: %v", result.Error)
	} else {
		log.Printf("Rebuilt %d pages in %s", result.PageCount, result.Duration)
	}

	msg := ToReloadMessage(change, result, ds.projectDir)
	ds.hub.Broadcast(msg)

	// Send a follow-up warning if there are warnings after a successful rebuild.
	if result.Success && len(result.Warnings) > 0 {
		var parts []string
		for _, w := range result.Warnings {
			parts = append(parts, fmt.Sprintf("%s (%s)", w.File, w.Field))
		}
		ds.hub.Broadcast(ReloadMessage{
			Type:  ReloadWarning,
			Error: fmt.Sprintf("%d warnings: %s", len(result.Warnings), strings.Join(parts, ", ")),
		})
	}
}

// fileHandler returns an HTTP handler that serves static files with clean URL support.
func (ds *DevServer) fileHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		urlPath := r.URL.Path

		// Try the exact file first.
		filePath := filepath.Join(ds.outputDir, filepath.FromSlash(urlPath))

		// Clean URL: /foo/ → foo/index.html
		if strings.HasSuffix(urlPath, "/") {
			candidate := filepath.Join(filePath, "index.html")
			if _, err := os.Stat(candidate); err == nil {
				http.ServeFile(w, r, candidate)
				return
			}
		}

		// Clean URL: /foo → foo/index.html (no extension)
		if filepath.Ext(urlPath) == "" && !strings.HasSuffix(urlPath, "/") {
			candidate := filepath.Join(filePath, "index.html")
			if _, err := os.Stat(candidate); err == nil {
				http.ServeFile(w, r, candidate)
				return
			}
		}

		// Exact file exists.
		if _, err := os.Stat(filePath); err == nil {
			http.ServeFile(w, r, filePath)
			return
		}

		// 404 fallback.
		notFound := filepath.Join(ds.outputDir, "404.html")
		if _, err := os.Stat(notFound); err == nil {
			w.WriteHeader(http.StatusNotFound)
			data, _ := os.ReadFile(notFound)
			w.Write(data)
			return
		}

		http.NotFound(w, r)
	})
}

// injectScript wraps an HTTP handler to inject the live reload script into HTML responses.
func (ds *DevServer) injectScript(next http.Handler) http.Handler {
	script := []byte("\n<script>" + string(embedded.LiveReloadJS) + "</script>\n")

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip injection for WebSocket, static assets.
		if r.URL.Path == "/ws" {
			next.ServeHTTP(w, r)
			return
		}
		ext := strings.ToLower(filepath.Ext(r.URL.Path))
		if ext != "" && ext != ".html" && ext != ".htm" {
			next.ServeHTTP(w, r)
			return
		}

		// Buffer the response.
		buf := &bufferedResponseWriter{
			header: make(http.Header),
			body:   &bytes.Buffer{},
		}
		next.ServeHTTP(buf, r)

		// Copy headers.
		for k, v := range buf.header {
			w.Header()[k] = v
		}

		body := buf.body.Bytes()

		// Inject script before </body> if present.
		if idx := bytes.LastIndex(body, []byte("</body>")); idx != -1 {
			w.Header().Del("Content-Length")
			w.WriteHeader(buf.statusCode)
			w.Write(body[:idx])
			w.Write(script)
			w.Write(body[idx:])
		} else {
			w.WriteHeader(buf.statusCode)
			w.Write(body)
		}
	})
}

// bufferedResponseWriter captures an HTTP response in memory.
type bufferedResponseWriter struct {
	header     http.Header
	body       *bytes.Buffer
	statusCode int
}

func (b *bufferedResponseWriter) Header() http.Header {
	return b.header
}

func (b *bufferedResponseWriter) Write(data []byte) (int, error) {
	return b.body.Write(data)
}

func (b *bufferedResponseWriter) WriteHeader(statusCode int) {
	b.statusCode = statusCode
}

// ReadFrom implements io.ReaderFrom to support io.Copy from http.ServeFile.
func (b *bufferedResponseWriter) ReadFrom(r io.Reader) (int64, error) {
	return io.Copy(b.body, r)
}
