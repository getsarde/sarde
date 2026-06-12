package project

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/frostybee/sarde/embedded"
	"github.com/frostybee/sarde/internal/build"
	"github.com/frostybee/sarde/internal/config"
	"github.com/frostybee/sarde/internal/consts"
	"github.com/frostybee/sarde/internal/content"
	"github.com/frostybee/sarde/internal/content/markdown"
	"github.com/frostybee/sarde/internal/editor"
	"github.com/frostybee/sarde/internal/engine"
	"github.com/frostybee/sarde/internal/theme"
	"gopkg.in/yaml.v3"
)

// PreviewServer is the interface for the dev server to avoid circular imports.
type PreviewServer interface {
	Start() error
	Stop() error
	// Ready reports the actual bound port: sent once after the listener
	// binds, then the channel is closed. Closed without a value if the
	// server fails or is stopped before binding.
	Ready() <-chan int
}

// PreviewFactory creates a PreviewServer. Set by the caller to break the import cycle.
type PreviewFactory func(projectDir, outputDir string, port int, liveReload bool, builderFactory func() *build.SiteBuilder) PreviewServer

// builderDeps is an immutable snapshot of the inputs for constructing a
// SiteBuilder. Published atomically so builds can run without holding pm.mu.
// Never nil once a project has been opened; CloseProject deliberately leaves
// the last-known-good snapshot so a straggler watcher rebuild that fires
// after DevServer.Stop() builds stale-but-valid deps instead of panicking
// on a nil config.
type builderDeps struct {
	projectDir string
	config     *config.SiteConfig
	themeCfg   *engine.ThemeConfig
}

// ProjectManager is the unified API for content CRUD, config management, and build lifecycle.
// It bridges the desktop app (Tauri) and CLI to the site engine.
type ProjectManager struct {
	mu             sync.RWMutex
	projectDir     string
	state          ProjectState
	config         *config.SiteConfig
	themeCfg       *engine.ThemeConfig
	deps           atomic.Pointer[builderDeps]
	embeddedFS     fs.FS
	eventHub       *EventHub
	devServer      PreviewServer
	previewFactory PreviewFactory
	mdRenderer     *markdown.Renderer
}

// NewProjectManager creates a new ProjectManager.
func NewProjectManager(hub *EventHub, efs fs.FS, pf PreviewFactory) *ProjectManager {
	return &ProjectManager{
		state:          StateClosed,
		embeddedFS:     efs,
		eventHub:       hub,
		previewFactory: pf,
		mdRenderer:     markdown.NewRenderer(),
	}
}

// ---------------------------------------------------------------------------
// Lifecycle
// ---------------------------------------------------------------------------

// OpenProject loads an existing project from the given directory.
func (pm *ProjectManager) OpenProject(dir string) (*ProjectInfo, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("resolving path: %w", err)
	}

	// Validate that content/ directory exists.
	contentDir := filepath.Join(absDir, consts.DirContent)
	if _, err := os.Stat(contentDir); os.IsNotExist(err) {
		return nil, fmt.Errorf("not a valid project: content/ directory not found in %s", absDir)
	}

	// Resolve config.
	cfg, themeCfg, err := pm.resolveConfig(absDir)
	if err != nil {
		return nil, err
	}

	pm.setProjectConfig(absDir, cfg, themeCfg)
	pm.state = StateOpen

	info := pm.buildProjectInfo()
	pm.eventHub.Broadcast(Event{Type: "project:opened", Data: info})
	return info, nil
}

// CloseProject closes the current project and stops any running preview.
func (pm *ProjectManager) CloseProject() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.state == StatePreviewing {
		pm.stopPreviewLocked()
	}

	// pm.deps is intentionally NOT cleared: a watcher timer rebuild can fire
	// after DevServer.Stop() returns (Stop joins only the fsnotify loop, not
	// armed timers), and its builder factory must never observe nil deps.
	// Build/Validate can't reach the stale snapshot — they gate on pm.state.
	pm.projectDir = ""
	pm.config = nil
	pm.themeCfg = nil
	pm.state = StateClosed

	pm.eventHub.Broadcast(Event{Type: "project:closed"})
	return nil
}

// CreateProject scaffolds a new project and opens it.
func (pm *ProjectManager) CreateProject(dir string, opts CreateOpts) (*ProjectInfo, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("resolving path: %w", err)
	}

	// Check if site already exists.
	if _, err := os.Stat(filepath.Join(absDir, consts.FileSiteConfig)); err == nil {
		return nil, fmt.Errorf("sarde.yaml already exists in %s", absDir)
	}

	title := opts.Title
	if title == "" {
		title = "My Site"
	}

	// Scaffold directories.
	for _, d := range []string{consts.DirContent, "static"} {
		if err := os.MkdirAll(filepath.Join(absDir, d), 0o755); err != nil {
			return nil, fmt.Errorf("creating directory: %w", err)
		}
	}

	// Write sarde.yaml.
	siteYAML := fmt.Sprintf("site:\n  title: %q\n  url: \"http://localhost:%d\"\n", title, consts.DefaultPort)
	if err := os.WriteFile(filepath.Join(absDir, consts.FileSiteConfig), []byte(siteYAML), 0o644); err != nil {
		return nil, err
	}

	// Write content/_index.md.
	indexMD := "---\ntitle: Welcome\n---\n\n# Welcome to your new site\n\nEdit this page at `content/_index.md`, then run `sarde serve` to see your changes.\n"
	os.MkdirAll(filepath.Join(absDir, consts.DirContent), 0o755)
	if err := os.WriteFile(filepath.Join(absDir, "content", "_index.md"), []byte(indexMD), 0o644); err != nil {
		return nil, err
	}

	// Write .gitignore.
	if err := os.WriteFile(filepath.Join(absDir, ".gitignore"), []byte("dist/\n.cache/\n"), 0o644); err != nil {
		return nil, err
	}

	return pm.OpenProject(absDir)
}

// State returns the current project state.
func (pm *ProjectManager) State() ProjectState {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.state
}

// ProjectDir returns the root directory of the currently open project.
func (pm *ProjectManager) ProjectDir() string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.projectDir
}

// ContentDir returns the absolute path to the content directory.
func (pm *ProjectManager) ContentDir() string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.contentDir()
}

// GetSchema returns the frontmatter schema for a collection, or nil if none exists.
func (pm *ProjectManager) GetSchema(collection string) (*engine.FrontmatterSchema, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if pm.state == StateClosed {
		return nil, fmt.Errorf("no project open")
	}

	colDir := filepath.Join(pm.contentDir(), collection)
	return content.LoadSchema(colDir)
}

// ---------------------------------------------------------------------------
// Build
// ---------------------------------------------------------------------------

// Build runs a full site build and returns the result.
func (pm *ProjectManager) Build() (*engine.BuildResult, error) {
	pm.mu.Lock()
	if pm.state == StateClosed {
		pm.mu.Unlock()
		return nil, fmt.Errorf("no project open")
	}
	pm.state = StateBuilding
	d := pm.deps.Load()
	pm.mu.Unlock()

	pm.eventHub.Broadcast(Event{Type: "build:started"})

	builder := pm.newBuilder(d)
	result, err := builder.Build()

	// Only transition out of StateBuilding if we're still building. A
	// concurrent CloseProject() during the build sets StateClosed; don't
	// resurrect the project to Open/Previewing after it was closed.
	pm.mu.Lock()
	if pm.state == StateBuilding {
		if pm.devServer != nil {
			pm.state = StatePreviewing
		} else {
			pm.state = StateOpen
		}
	}
	pm.mu.Unlock()

	if err != nil {
		pm.eventHub.Broadcast(Event{Type: "build:error", Data: map[string]any{"error": err.Error()}})
		return nil, err
	}

	pm.eventHub.Broadcast(Event{Type: "build:complete", Data: map[string]any{
		"pageCount": result.PageCount,
		"duration":  result.Duration.String(),
	}})
	return result, nil
}

// Validate runs phases 1-4 without rendering or writing.
func (pm *ProjectManager) Validate() (*build.ValidateResult, error) {
	pm.mu.RLock()
	if pm.state == StateClosed {
		pm.mu.RUnlock()
		return nil, fmt.Errorf("no project open")
	}
	d := pm.deps.Load()
	pm.mu.RUnlock()

	builder := pm.newBuilder(d)
	return builder.Validate()
}

// StartPreview starts the dev server and returns the actual bound port.
func (pm *ProjectManager) StartPreview(port int) (int, error) {
	ds, err := pm.installPreview(port)
	if err != nil {
		return 0, err
	}

	startErr := make(chan error, 1)
	go func() {
		err := ds.Start()
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			pm.mu.Lock()
			// Only tear down if this server is still the active one; a
			// Stop + restart may have installed a new devServer since.
			if pm.devServer == ds {
				pm.devServer = nil
				pm.state = StateOpen
			}
			pm.eventHub.Broadcast(Event{Type: "preview:error", Data: map[string]any{"error": err.Error()}})
			pm.mu.Unlock()
		}
		startErr <- err // sent after teardown so the waiter observes cleaned-up state
	}()

	// Wait for the bind LOCK-FREE: the initial build runs inside Start and
	// must not block other ProjectManager operations.
	actualPort, ok := <-ds.Ready()
	if !ok {
		// Start returned before binding; its error is authoritative.
		err := <-startErr
		if err == nil || errors.Is(err, http.ErrServerClosed) {
			return 0, fmt.Errorf("preview stopped before it became ready")
		}
		return 0, err
	}

	pm.mu.Lock()
	defer pm.mu.Unlock()
	if pm.devServer != ds {
		// StopPreview/CloseProject won the race after binding; its
		// preview:stopped event already fired — don't announce a dead server.
		return 0, fmt.Errorf("preview was stopped while starting")
	}
	pm.eventHub.Broadcast(Event{Type: "preview:started", Data: map[string]any{"port": actualPort}})
	return actualPort, nil
}

// installPreview validates state, constructs the dev server via the factory,
// and registers it as the active preview — all under pm.mu. The blocking
// wait for the bound port happens in StartPreview, outside the lock.
func (pm *ProjectManager) installPreview(port int) (PreviewServer, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.state == StateClosed {
		return nil, fmt.Errorf("no project open")
	}
	if pm.devServer != nil {
		return nil, fmt.Errorf("preview already running")
	}

	if port == 0 {
		port = pm.config.Server.Port
	}
	if port == 0 {
		port = consts.DefaultPort
	}

	outputDir, err := build.ResolveOutputDir(pm.projectDir, pm.config.Build.Output)
	if err != nil {
		return nil, err
	}

	if pm.previewFactory == nil {
		return nil, fmt.Errorf("preview not available (no preview factory configured)")
	}

	ds := pm.previewFactory(pm.projectDir, outputDir, port, config.BoolVal(pm.config.Server.LiveReload, true), func() *build.SiteBuilder {
		// Freshness barrier: UpdateSettings writes sarde.yaml and publishes
		// the re-resolved snapshot inside one pm.mu critical section. The
		// watcher's config-change rebuild can fire (debounce) before that
		// section ends; loading under RLock orders this rebuild after the
		// publish so it builds with the settings that triggered it. No
		// deadlock: Stop() never joins the watcher timer goroutine, so a
		// writer holding pm.mu always completes.
		pm.mu.RLock()
		d := pm.deps.Load()
		pm.mu.RUnlock()
		return pm.newBuilder(d)
	})

	pm.devServer = ds
	pm.state = StatePreviewing
	return ds, nil
}

// StopPreview stops the dev server.
func (pm *ProjectManager) StopPreview() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.devServer == nil {
		return fmt.Errorf("preview not running")
	}

	return pm.stopPreviewLocked()
}

func (pm *ProjectManager) stopPreviewLocked() error {
	err := pm.devServer.Stop()
	pm.devServer = nil
	pm.state = StateOpen
	pm.eventHub.Broadcast(Event{Type: "preview:stopped"})
	return err
}

// ---------------------------------------------------------------------------
// Content CRUD
// ---------------------------------------------------------------------------

// ListContent returns a list of content files in the given collection.
func (pm *ProjectManager) ListContent(collection string) ([]ContentSummary, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if pm.state == StateClosed {
		return nil, fmt.Errorf("no project open")
	}

	contentDir := pm.contentDir()
	scanner := &content.Scanner{}
	files, err := scanner.DiscoverFiles(contentDir)
	if err != nil {
		return nil, err
	}

	var summaries []ContentSummary
	for _, cf := range files {
		if collection != "" && cf.CollectionName != collection {
			continue
		}
		// Parse frontmatter for summary fields.
		fm, body, err := readContentFile(contentDir, cf.RelPath)
		if err != nil {
			continue
		}
		title, draft, date, weight, _, _ := extractSummary(fm, body)
		summaries = append(summaries, ContentSummary{
			Path:   cf.RelPath,
			Title:  title,
			Draft:  draft,
			Date:   date,
			Weight: weight,
		})
	}

	return summaries, nil
}

// ReadContent reads a single content file.
func (pm *ProjectManager) ReadContent(relPath string) (*ContentFile, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if pm.state == StateClosed {
		return nil, fmt.Errorf("no project open")
	}

	contentDir := pm.contentDir()
	if err := validateContentPath(contentDir, relPath); err != nil {
		return nil, err
	}

	fm, body, err := readContentFile(contentDir, relPath)
	if err != nil {
		return nil, err
	}

	title, draft, date, _, wc, rt := extractSummary(fm, body)

	// Determine collection from path.
	collection := ""
	parts := strings.SplitN(filepath.ToSlash(relPath), "/", 2)
	if len(parts) > 1 {
		collection = parts[0]
	}

	return &ContentFile{
		Path:        relPath,
		Title:       title,
		Collection:  collection,
		Frontmatter: fm,
		Body:        body,
		Draft:       draft,
		Date:        date,
		WordCount:   wc,
		ReadingTime: rt,
	}, nil
}

// CreateContent creates a new content file in the given collection.
func (pm *ProjectManager) CreateContent(collection, title string) (*ContentFile, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.state == StateClosed {
		return nil, fmt.Errorf("no project open")
	}

	slug := slugify(title)
	if slug == "" {
		return nil, fmt.Errorf("cannot generate slug from title %q", title)
	}

	relPath := filepath.Join(collection, slug+".md")
	contentDir := pm.contentDir()

	if err := validateContentPath(contentDir, relPath); err != nil {
		return nil, err
	}

	absPath := filepath.Join(contentDir, relPath)
	if _, err := os.Stat(absPath); err == nil {
		return nil, fmt.Errorf("file already exists: %s", relPath)
	}

	fm := scaffoldFrontmatter(pm.projectDir, collection, title)
	if err := writeContentFile(contentDir, relPath, fm, "\n"); err != nil {
		return nil, err
	}

	pm.eventHub.Broadcast(Event{Type: "file:created", Data: map[string]any{"path": filepath.ToSlash(relPath)}})

	// Return the created file.
	return &ContentFile{
		Path:        filepath.ToSlash(relPath),
		Title:       title,
		Collection:  collection,
		Frontmatter: fm,
		Body:        "\n",
		Draft:       true,
		Date:        time.Now(),
	}, nil
}

// SaveContent writes frontmatter and body to an existing content file.
func (pm *ProjectManager) SaveContent(relPath string, fm map[string]any, body string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.state == StateClosed {
		return fmt.Errorf("no project open")
	}

	contentDir := pm.contentDir()
	if err := validateContentPath(contentDir, relPath); err != nil {
		return err
	}

	if err := writeContentFile(contentDir, relPath, fm, body); err != nil {
		return err
	}

	pm.eventHub.Broadcast(Event{Type: "file:changed", Data: map[string]any{"path": relPath}})
	return nil
}

// ListRevisions returns the revision history for a content file, newest first.
func (pm *ProjectManager) ListRevisions(relPath string) ([]RevisionSummary, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if pm.state == StateClosed {
		return nil, fmt.Errorf("no project open")
	}
	contentDir := pm.contentDir()
	if err := validateContentPath(contentDir, relPath); err != nil {
		return nil, err
	}
	absPath := filepath.Join(contentDir, filepath.FromSlash(relPath))
	revs := editor.ListRevisions(absPath)

	out := make([]RevisionSummary, 0, len(revs))
	for _, r := range revs {
		out = append(out, RevisionSummary{
			ID:        filepath.Base(r.Path),
			Timestamp: r.Timestamp,
			Size:      r.Size,
		})
	}
	return out, nil
}

// RestoreRevision overwrites the content file with the named revision's contents.
// The current version is snapshotted first so the restore is itself reversible.
func (pm *ProjectManager) RestoreRevision(relPath, revisionID string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.state == StateClosed {
		return fmt.Errorf("no project open")
	}
	contentDir := pm.contentDir()
	if err := validateContentPath(contentDir, relPath); err != nil {
		return err
	}
	// Revision ID must be a plain filename — reject any path segment.
	if strings.ContainsAny(revisionID, "/\\") || strings.Contains(revisionID, "..") {
		return fmt.Errorf("invalid revision id")
	}
	absPath := filepath.Join(contentDir, filepath.FromSlash(relPath))
	revPath := filepath.Join(filepath.Dir(absPath), ".revisions", revisionID)
	if _, err := os.Stat(revPath); err != nil {
		return fmt.Errorf("revision not found")
	}
	if err := editor.RestoreRevision(absPath, revPath); err != nil {
		return err
	}
	pm.eventHub.Broadcast(Event{Type: "file:changed", Data: map[string]any{"path": relPath}})
	return nil
}

// DeleteContent removes a content file.
func (pm *ProjectManager) DeleteContent(relPath string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.state == StateClosed {
		return fmt.Errorf("no project open")
	}

	contentDir := pm.contentDir()
	if err := validateContentPath(contentDir, relPath); err != nil {
		return err
	}

	absPath := filepath.Join(contentDir, filepath.FromSlash(relPath))
	if err := os.Remove(absPath); err != nil {
		return err
	}

	pm.eventHub.Broadcast(Event{Type: "file:deleted", Data: map[string]any{"path": relPath}})
	return nil
}

// RenameContent moves a content file to a new path.
func (pm *ProjectManager) RenameContent(oldPath, newPath string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.state == StateClosed {
		return fmt.Errorf("no project open")
	}

	contentDir := pm.contentDir()
	if err := validateContentPath(contentDir, oldPath); err != nil {
		return err
	}
	if err := validateContentPath(contentDir, newPath); err != nil {
		return err
	}

	absOld := filepath.Join(contentDir, filepath.FromSlash(oldPath))
	absNew := filepath.Join(contentDir, filepath.FromSlash(newPath))

	if err := os.MkdirAll(filepath.Dir(absNew), 0o755); err != nil {
		return err
	}
	if err := os.Rename(absOld, absNew); err != nil {
		return err
	}

	pm.eventHub.Broadcast(Event{Type: "file:renamed", Data: map[string]any{"oldPath": oldPath, "newPath": newPath}})
	return nil
}

// ---------------------------------------------------------------------------
// Config
// ---------------------------------------------------------------------------

// GetConfig returns the current site configuration.
func (pm *ProjectManager) GetConfig() *config.SiteConfig {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.config
}

// UpdateSettings applies partial config updates and re-resolves.
func (pm *ProjectManager) UpdateSettings(input SettingsInput) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if pm.state == StateClosed {
		return fmt.Errorf("no project open")
	}

	// Read current sarde.yaml.
	siteYAMLPath := filepath.Join(pm.projectDir, consts.FileSiteConfig)
	data, err := os.ReadFile(siteYAMLPath)
	if err != nil {
		return fmt.Errorf("reading sarde.yaml: %w", err)
	}

	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil {
		raw = make(map[string]any)
	}

	// Ensure site section exists.
	siteSection, _ := raw["site"].(map[string]any)
	if siteSection == nil {
		siteSection = make(map[string]any)
		raw["site"] = siteSection
	}

	// Apply non-nil fields.
	if input.Title != nil {
		siteSection["title"] = *input.Title
	}
	if input.URL != nil {
		siteSection["url"] = *input.URL
	}
	if input.Language != nil {
		siteSection["language"] = *input.Language
	}
	if input.Description != nil {
		siteSection["description"] = *input.Description
	}

	// Write back.
	out, err := yaml.Marshal(raw)
	if err != nil {
		return fmt.Errorf("marshaling sarde.yaml: %w", err)
	}
	if err := os.WriteFile(siteYAMLPath, out, 0o644); err != nil {
		return err
	}

	// Re-resolve config.
	cfg, themeCfg, err := pm.resolveConfig(pm.projectDir)
	if err != nil {
		return err
	}
	pm.setProjectConfig(pm.projectDir, cfg, themeCfg)

	pm.eventHub.Broadcast(Event{Type: "config:changed"})
	return nil
}

// GetCollections returns metadata about all collections.
func (pm *ProjectManager) GetCollections() ([]CollectionInfo, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	if pm.state == StateClosed {
		return nil, fmt.Errorf("no project open")
	}

	return pm.getCollectionsLocked()
}

// getCollectionsLocked is the lock-free version for internal use when lock is already held.
// Uses a lightweight directory scan instead of the full DiscoverFiles pipeline.
func (pm *ProjectManager) getCollectionsLocked() ([]CollectionInfo, error) {
	contentDir := pm.contentDir()
	entries, err := os.ReadDir(contentDir)
	if err != nil {
		return nil, err
	}

	var collections []CollectionInfo
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") || strings.HasPrefix(e.Name(), "_") {
			continue
		}
		name := e.Name()
		count := countMarkdownFiles(filepath.Join(contentDir, name))
		collections = append(collections, CollectionInfo{
			Name:      name,
			Title:     content.FilenameToTitle(name),
			PageCount: count,
		})
	}
	return collections, nil
}

func countMarkdownFiles(dir string) int {
	count := 0
	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".md") {
			count++
		}
		return nil
	})
	return count
}

// ---------------------------------------------------------------------------
// Rendering
// ---------------------------------------------------------------------------

// RenderMarkdown converts markdown to HTML with heading extraction.
func (pm *ProjectManager) RenderMarkdown(md string) (*RenderResult, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	result, err := pm.mdRenderer.Render(md)
	if err != nil {
		return nil, err
	}

	wc := len(strings.Fields(md))
	rt := 0
	if wc > 0 {
		rt = int(float64(wc)/200.0 + 0.5)
		if rt < 1 {
			rt = 1
		}
	}

	return &RenderResult{
		HTML:        result.HTML,
		Headings:    result.Headings,
		WordCount:   wc,
		ReadingTime: rt,
	}, nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

func (pm *ProjectManager) contentDir() string {
	dir := consts.DirContent
	if pm.config != nil && pm.config.Content.Dir != "" {
		dir = pm.config.Content.Dir
	}
	return filepath.Join(pm.projectDir, dir)
}

// setProjectConfig sets the authoritative project fields and publishes the
// matching builder snapshot. Caller must hold pm.mu (write).
func (pm *ProjectManager) setProjectConfig(dir string, cfg *config.SiteConfig, themeCfg *engine.ThemeConfig) {
	pm.projectDir = dir
	pm.config = cfg
	pm.themeCfg = themeCfg
	pm.deps.Store(&builderDeps{projectDir: dir, config: cfg, themeCfg: themeCfg})
}

// newBuilder constructs a SiteBuilder from a deps snapshot. Lock-free; safe
// from any goroutine. d must come from pm.deps.Load(), which is non-nil
// whenever a builder is reachable (Build/Validate gate on pm.state; the
// preview factory closure only exists after a successful StartPreview).
func (pm *ProjectManager) newBuilder(d *builderDeps) *build.SiteBuilder {
	return build.NewSiteBuilder(build.BuildOptions{
		ProjectDir:  d.projectDir,
		Config:      d.config,
		ThemeConfig: d.themeCfg,
		EmbeddedFS:  pm.embeddedFS,
	})
}

func (pm *ProjectManager) resolveConfig(projectDir string) (*config.SiteConfig, *engine.ThemeConfig, error) {
	configPath := filepath.Join(projectDir, consts.FileSiteConfig)
	cfg, err := config.Resolve(config.ResolveOptions{
		ConfigPath: configPath,
		EnvPrefix:  "SARDE",
		Strict:     true,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("resolving config: %w", err)
	}

	// Load theme.
	var thm *theme.Theme
	if cfg.Theme.Name != "" && cfg.Theme.Name != "default" {
		thm, _ = theme.LoadFromDir(filepath.Join(projectDir, "themes", cfg.Theme.Name))
	}
	if thm == nil {
		thm, _ = theme.LoadFromFS(embedded.ThemeFS(), ".")
	}

	// Fold theme shortcut fields into overrides before token resolution.
	foldThemeShortcuts(cfg)

	// Validate token names in overrides.
	known := theme.KnownTokens()
	if err := theme.ValidateOverrides("theme.overrides", cfg.Theme.Overrides, known); err != nil {
		return nil, nil, err
	}
	if err := theme.ValidateOverrides("theme.dark_overrides", cfg.Theme.DarkOverrides, known); err != nil {
		return nil, nil, err
	}

	// Resolve tokens.
	lightTokens := theme.ResolveTokens(theme.DefaultTokens(), thm, cfg.Theme.Preset, cfg.Theme.Overrides)
	lightTokens = theme.DeriveTokens(lightTokens)
	darkTokens := theme.ResolveDarkTokens(theme.DefaultDarkTokens(), thm, cfg.Theme.Preset, darkOverrides(cfg))
	styleTag := theme.GenerateStyleTag(lightTokens, darkTokens)

	name := "Default"
	slug := "default"
	if thm != nil {
		if thm.Name != "" {
			name = thm.Name
		}
		if thm.Slug != "" {
			slug = thm.Slug
		}
	}

	themeCfg := &engine.ThemeConfig{
		Name:        name,
		Slug:        slug,
		Tokens:      lightTokens,
		DarkTokens:  darkTokens,
		DarkEnabled: config.BoolVal(cfg.Theme.Dark, true),
		StyleTag:    styleTag,
	}

	return cfg, themeCfg, nil
}

func foldThemeShortcuts(cfg *config.SiteConfig) {
	accentVal := cfg.Theme.AccentColor
	if accentVal == "" {
		accentVal = cfg.Theme.PrimaryColor
	}
	shortcuts := map[string]string{
		"accent":    accentVal,
		"font-sans": cfg.Theme.FontFamily,
		"font-mono": cfg.Theme.FontMono,
	}
	if cfg.Theme.Overrides == nil {
		cfg.Theme.Overrides = make(map[string]string)
	}
	for token, val := range shortcuts {
		if val != "" {
			if _, exists := cfg.Theme.Overrides[token]; !exists {
				cfg.Theme.Overrides[token] = val
			}
		}
	}
}

func darkOverrides(cfg *config.SiteConfig) map[string]string {
	if len(cfg.Theme.DarkOverrides) > 0 {
		return cfg.Theme.DarkOverrides
	}
	return cfg.Theme.Overrides
}

func (pm *ProjectManager) buildProjectInfo() *ProjectInfo {
	info := &ProjectInfo{
		Dir:   pm.projectDir,
		State: pm.state.String(),
		Title: pm.config.Site.Title,
	}

	// Scan collections (best effort). Use lock-free internal method
	// since the caller already holds the lock.
	if cols, err := pm.getCollectionsLocked(); err == nil {
		info.Collections = cols
	}

	return info
}
