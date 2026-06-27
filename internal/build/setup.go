package build

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/collection"
	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/devlog"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/i18n"
	"github.com/getsarde/sarde/internal/outputpath"
	"github.com/getsarde/sarde/internal/plugin"
	"github.com/getsarde/sarde/internal/plugin/announcements"
	"github.com/getsarde/sarde/internal/plugin/clientplugins"
	"github.com/getsarde/sarde/internal/plugin/katex"
	"github.com/getsarde/sarde/internal/plugin/mermaid"
	"github.com/getsarde/sarde/internal/plugin/socialcards"
	"github.com/getsarde/sarde/internal/workers"
)

// registerSubpackagePlugins wires plugins whose assets live in their own
// subpackage (and therefore can't be referenced from internal/plugin/registry.go
// without creating an import cycle).
func registerSubpackagePlugins(mgr *plugin.Manager, enabled []string, configs map[string]map[string]any) {
	if err := clientplugins.Initialize(); err != nil {
		devlog.Warn("build", "clientplugins init: %v", err)
	}
	for _, name := range enabled {
		switch name {
		case "katex":
			mgr.Register(katex.New(configs[name]))
		case "mermaid":
			mgr.Register(mermaid.New(configs[name]))
		case "announcements":
			// Registered in Build() after stringTable is available (needs i18n).
		case "social_cards":
			mgr.Register(socialcards.New(configs[name]))
		}
	}
	clientplugins.RegisterAll(mgr, enabled, configs)
}

// appendVersionedLatestPages appends duplicate RenderedPage entries for
// collections with PublishLatestAtVersionURL enabled. These duplicates emit
// the latest-version content at the explicit versioned URL (e.g.
// /docs/v2/guides/auth/) alongside the alias URL (/docs/guides/auth/).
func appendVersionedLatestPages(rendered []RenderedPage, collections map[string]*engine.Collection, r *engine.URLResolver) []RenderedPage {
	for _, col := range collections {
		vc := col.Config.Versioning
		if vc == nil || !vc.Enabled || !vc.PublishLatestAtVersionURL || vc.LastVersion == "" {
			continue
		}
		for i := range rendered {
			rp := &rendered[i]
			if rp.Page == nil || rp.Page.Collection != col {
				continue
			}
			if rp.Page.Version != vc.LastVersion {
				continue
			}
			versionedOutPath := PageOutputPath(r.OutputRelPath(rp.Page.RelPermalink, rp.Page.Lang, vc.LastVersion))
			rendered = append(rendered, RenderedPage{
				Page:    rp.Page,
				HTML:    rp.HTML,
				OutPath: versionedOutPath,
			})
		}
	}
	return rendered
}

// buildScannerVersionIDs constructs the collection→versionIDs map from config
// for the scanner's version detection pass.
func buildScannerVersionIDs(collections map[string]*config.CollectionSiteConfig) map[string]map[string]bool {
	if len(collections) == 0 {
		return nil
	}
	result := make(map[string]map[string]bool)
	for name, colCfg := range collections {
		if colCfg == nil || colCfg.Versioning == nil || !config.BoolVal(colCfg.Versioning.Enabled, false) {
			continue
		}
		ids := make(map[string]bool, len(colCfg.Versioning.Versions))
		for _, v := range colCfg.Versioning.Versions {
			if v.ID != "" {
				ids[v.ID] = true
			}
		}
		if len(ids) > 0 {
			result[name] = ids
		}
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

// buildResolverRegistries extracts collection mounts and a union of version IDs
// from all built collections for the URL resolver.
func buildResolverRegistries(collections map[string]*engine.Collection) ([]string, map[string]bool) {
	mounts := make([]string, 0, len(collections))
	versionIDs := make(map[string]bool)
	for name, col := range collections {
		mounts = append(mounts, "/"+name)
		if col.Config != nil && col.Config.Versioning != nil && col.Config.Versioning.Enabled {
			for _, vd := range col.Config.Versioning.Versions {
				versionIDs[vd.ID] = true
			}
		}
	}
	return mounts, versionIDs
}

// subpackagePluginNames lists Go subpackage plugins registered via
// registerSubpackagePlugins (not in plugin.BuiltinNames()).
var subpackagePluginNames = []string{"katex", "mermaid", "announcements", "social_cards"}

// KnownPluginNames returns the union of all valid plugin names from the
// Go-side registry, subpackage plugins, and client-side manifest.
func KnownPluginNames() []string {
	_ = clientplugins.Initialize()
	names := plugin.BuiltinNames()
	names = append(names, subpackagePluginNames...)
	names = append(names, clientplugins.PluginSlugs()...)
	return names
}

func (b *SiteBuilder) phaseInitialize(s *buildState) error {
	s.contentDir = b.resolveContentDir()

	outputDir, err := outputpath.ResolveOutputDir(b.projectDir, b.config.Build.Output)
	if err != nil {
		return err
	}
	s.outputDir = outputDir
	s.parallel = config.BoolVal(b.config.Build.Parallel, true)
	s.workerCount = workers.Count()

	s.isMultiLang = b.config.I18n.IsMultiLang()
	s.defaultLang = b.config.I18n.GetDefaultLanguage()
	stringTable, err := i18n.LoadStrings(embedded.I18nFS(), b.projectDir, b.config.Theme.Name, s.defaultLang)
	if err != nil {
		return fmt.Errorf("loading i18n strings: %w", err)
	}
	if b.config.I18n.Strict {
		stringTable.SetStrict(true)
	}
	s.stringTable = stringTable

	if s.isMultiLang {
		langCodes := make(map[string]bool)
		for code := range b.config.I18n.Languages {
			langCodes[code] = true
		}
		b.scanner.Languages = langCodes
		b.scanner.DefaultLang = s.defaultLang
	}
	b.scanner.VersionIDs = buildScannerVersionIDs(b.config.Collections)

	if !b.built {
		for _, name := range b.config.Plugins.Enabled {
			if name == "announcements" {
				b.pluginMgr.Register(announcements.New(
					b.config.Plugins.Config[name],
					stringTable,
					b.tmplEngine.CurrentLangPtr(),
				))
				break
			}
		}
		if err := b.pluginMgr.RunConfigSetup(b.config); err != nil {
			return err
		}
		b.loadIconSources()
	}
	return nil
}

func (b *SiteBuilder) phaseDiscover(s *buildState) error {
	files, err := b.scanner.DiscoverFiles(s.contentDir)
	if err != nil {
		return fmt.Errorf("discovering content: %w", err)
	}
	s.files = files
	s.recordTiming("Discovering content")
	return nil
}

func (b *SiteBuilder) phaseParse(s *buildState) error {
	parseOpts := collection.BuildOptions{Parallel: s.parallel, WorkerCount: s.workerCount}
	collections, warnings, err := collection.BuildCollectionsWithOptions(s.files, b.config, s.contentDir, parseOpts)
	if err != nil {
		return fmt.Errorf("building collections: %w", err)
	}
	standalones, err := collection.BuildStandalonePagesWithOptions(s.files, s.contentDir, b.config.Content.SummaryLength, string(b.config.Build.LastUpdated), parseOpts)
	if err != nil {
		return fmt.Errorf("building standalone pages: %w", err)
	}
	s.collections = collections
	s.standalones = standalones
	s.warnings = warnings
	s.recordTiming("Parsing content")
	return nil
}

func detectFavicon(projectDir string) string {
	for _, name := range []string{"favicon.svg", "favicon.ico", "favicon.png"} {
		p := filepath.Join(projectDir, "static", name)
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return "/" + name
		}
	}
	return ""
}

func faviconMIME(path string) string {
	switch {
	case strings.HasSuffix(path, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(path, ".ico"):
		return "image/x-icon"
	case strings.HasSuffix(path, ".png"):
		return "image/png"
	default:
		return ""
	}
}
