package build

import (
	"io/fs"
	"path/filepath"
	"time"

	"github.com/frostybee/kazari"
	"github.com/getsarde/sarde/internal/asset"
	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/content"
	"github.com/getsarde/sarde/internal/content/markdown"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/links"
	"github.com/getsarde/sarde/internal/plugin"
	"github.com/getsarde/sarde/internal/plugin/external"
	"github.com/getsarde/sarde/internal/shortcode"
	sardetemplate "github.com/getsarde/sarde/internal/template"
)

// BuildOptions provides the inputs for creating a SiteBuilder.
type BuildOptions struct {
	ProjectDir      string
	Config          *config.SiteConfig
	ThemeConfig     *engine.ThemeConfig
	EmbeddedFS      fs.FS
	DevMode         bool   // Skip heavy optimizations during serve (image processing, minification, etc.)
	CheckSyntax     bool   // When true, run fenced-block syntax checking during markdown rendering
	PluginAssetsDir string // If set, recompute plugin client bundles from disk (theme-dev mode)
}

// SiteBuilder orchestrates the full six-phase build pipeline.
type SiteBuilder struct {
	projectDir  string
	config      *config.SiteConfig
	themeConfig *engine.ThemeConfig
	embeddedFS  fs.FS
	devMode     bool

	scanner      *content.Scanner
	mdRenderer   *markdown.Renderer
	tmplEngine   *sardetemplate.Engine
	pluginMgr    *plugin.Manager
	rendererPool chan *markdown.Renderer // lazily initialized pool, persisted across rebuilds
	built        bool                    // true after first Build(); gates one-time registrations

	externalPluginDirs          []string                   // templates/ dirs of active external plugins
	externalPluginDirectiveDirs []string                   // directives/ dirs of active external plugins
	externalPluginWarnings      []engine.ValidationWarning // skipped-plugin diagnostics, surfaced each build

	urlResolver   *engine.URLResolver // URL resolver for basePath prefixing
	resolutionKey string              // digest of resolver state; folded into page-cache keys
	rendererKey   string              // auto-fingerprint of Renderer: extension set + config + Kazari CSS
	linkGraph     *links.LinkGraph
	lastCoverage  links.CoverageSummary

	checkOnly         bool                // when true, Build() returns after link validation
	checkReportResult *links.ReportResult // stored for Check() to read after Build() returns
	checkSyntax       bool                // when true, run fenced-block syntax checks during markdown rendering

	// Last-build state for incremental rebuild.
	lastCollections    map[string]*engine.Collection
	lastAllPages       []*engine.Page
	lastSiteCtx        *engine.SiteContext
	lastTaxByLang      map[string]map[string]*engine.Taxonomy
	lastPageIndex      *content.PageIndex
	lastGitIndex       *content.GitLastModIndex // reused by incremental rebuilds; refreshed on full builds
	lastOutputDir      string
	lastAssetPipeline  *asset.Pipeline
	globalCSSURLs      []string
	themeJSURL         string
	themeJSContent     []byte
	themeJSFilename    string
	kazariEngine       *kazari.Engine
	kazariJSURL        string
	kazariJSContent    []byte
	kazariJSFilename   string
	tokenCSSURL        string
	tokenCSSContent    string
	tokenCSSFilename   string
	lastScProcessor    *shortcode.Processor
	lastShortcodesHash string
	lastPageCache      *PageCache
	lastIconRenderKey  string
	lastValidationData map[string]engine.ValidationEntry
}

// NewSiteBuilder creates a SiteBuilder with all dependencies initialized.
func NewSiteBuilder(opts BuildOptions) *SiteBuilder {
	mgr := plugin.NewManager()
	enabled := filterDisabled(opts.Config.Plugins.Enabled, opts.Config.Plugins.Disabled)
	mgr.RegisterBuiltins(enabled, opts.Config.Plugins.Config)
	registerSubpackagePlugins(mgr, enabled, opts.Config.Plugins.Config, opts.PluginAssetsDir)
	extDirs, extDirectiveDirs, extWarnings := external.LoadAll(mgr, opts.ProjectDir, opts.Config, KnownPluginNames(""))
	extWarnings = append(extWarnings, warnUnusedPluginConfig(opts.Config, enabled, opts.ProjectDir)...)

	return &SiteBuilder{
		projectDir:                  opts.ProjectDir,
		config:                      opts.Config,
		themeConfig:                 opts.ThemeConfig,
		embeddedFS:                  opts.EmbeddedFS,
		devMode:                     opts.DevMode,
		checkSyntax:                 opts.CheckSyntax,
		scanner:                     &content.Scanner{},
		tmplEngine:                  sardetemplate.NewEngine(),
		pluginMgr:                   mgr,
		externalPluginDirs:          extDirs,
		externalPluginDirectiveDirs: extDirectiveDirs,
		externalPluginWarnings:      extWarnings,
	}
}

// resolveContentDir returns the absolute path to the content directory,
// respecting the optional Content.Dir override in site config.
func (b *SiteBuilder) resolveContentDir() string {
	contentDir := filepath.Join(b.projectDir, consts.DirContent)
	if b.config.Content.Dir != "" {
		if filepath.IsAbs(b.config.Content.Dir) {
			contentDir = b.config.Content.Dir
		} else {
			contentDir = filepath.Join(b.projectDir, b.config.Content.Dir)
		}
	}
	return contentDir
}

// Build executes the full six-phase pipeline and writes output to disk.
func (b *SiteBuilder) Build() (*engine.BuildResult, error) {
	result, err := b.runBuild()
	if err != nil {
		// A failed build can leave the template engine or resolver partially
		// reinitialized while the last* snapshots still hold the previous
		// build's state; force the next change onto the full-build path.
		b.built = false
	}
	return result, err
}

func (b *SiteBuilder) runBuild() (*engine.BuildResult, error) {
	start := time.Now()
	var timings []engine.PhaseTiming
	phaseStart := time.Now()
	recordTiming := func(phase string) {
		timings = append(timings, engine.PhaseTiming{Phase: phase, Duration: time.Since(phaseStart)})
		phaseStart = time.Now()
	}

	s := &buildState{recordTiming: recordTiming}

	if err := b.phaseInitialize(s); err != nil {
		return nil, err
	}
	if err := b.phaseDiscover(s); err != nil {
		return nil, err
	}
	if err := b.phaseParse(s); err != nil {
		return nil, err
	}
	if err := b.phaseAssemble(s); err != nil {
		return nil, err
	}
	if err := b.phaseAssets(s); err != nil {
		return nil, err
	}
	if s.checkResult != nil {
		s.checkResult.Duration = time.Since(start)
		s.checkResult.PhaseTimings = timings
		return s.checkResult, nil
	}
	if err := b.phaseRender(s); err != nil {
		return nil, err
	}
	result, err := b.phaseWrite(s)
	if err != nil {
		return nil, err
	}
	result.Duration = time.Since(start)
	result.PhaseTimings = timings
	return result, nil
}
