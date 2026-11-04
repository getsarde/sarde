package build

import (
	"fmt"
	"time"

	"github.com/getsarde/sarde/internal/collection"
	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/i18n"
	"github.com/getsarde/sarde/internal/taxonomy"
	"github.com/getsarde/sarde/internal/version"
)

func (b *SiteBuilder) phaseAssemble(s *buildState) error {
	// Collect all pages.
	var allPages []*engine.Page
	for _, name := range sortedKeys(s.collections) {
		col := s.collections[name]
		allPages = append(allPages, col.Pages...)
	}
	allPages = append(allPages, s.standalones...)

	// Plugin hook: ContentLoaded (serial, plugins can inject virtual pages).
	if err := b.pluginMgr.RunContentLoaded(b.config, s.collections, &allPages); err != nil {
		return err
	}

	// i18n: link real translations, generate fallbacks, link all translations
	if s.isMultiLang {
		langCodes := b.config.I18n.LanguageCodes()
		weights := languageWeights(b.config)

		i18n.LinkTranslations(allPages, weights)

		fallbacks := i18n.GenerateFallbacks(allPages, langCodes, s.defaultLang, i18nFallbackOptions(b.config))
		allPages = append(allPages, fallbacks...)

		collection.RebuildNavTreesWithFallbacks(s.collections, allPages, langCodes, s.contentDir)

		i18n.LinkAllTranslations(allPages, weights)
	}

	// Per-language lists, index pages and lab sidebars, built once fallback
	// copies exist; cleared on a single-language site.
	collection.BuildLanguageViews(s.collections, allPages, b.config.I18n.LanguageCodes(), s.defaultLang)

	collection.LinkVersions(allPages)

	// sidebar.yaml unmatched-key warnings: computed only after every nav tree
	// has been built, including the i18n-fallback rebuild above, so keys that
	// match only in fallback-generated lanes are not falsely flagged.
	s.warnings = append(s.warnings, collection.CollectSidebarOverrideWarnings(s.collections)...)
	s.warnings = append(s.warnings, collection.CollectNavYAMLWarnings(s.collections, s.contentDir)...)

	// Build taxonomies. Multi-language sites get per-language maps; single-
	// language sites build one map with lang="". Permalink resolution on the
	// per-language instances happens after the URLResolver is created (below).
	var taxonomies map[string]*engine.Taxonomy
	var taxByLang map[string]map[string]*engine.Taxonomy

	if s.isMultiLang {
		langCodes := b.config.I18n.LanguageCodes()
		taxByLang = make(map[string]map[string]*engine.Taxonomy, len(langCodes))
		var allTaxWarnings []string
		for _, code := range langCodes {
			langTax, buildWarns := taxonomy.BuildTaxonomies(allPages, b.config.Taxonomies, code)
			allTaxWarnings = append(allTaxWarnings, buildWarns...)
			w, err := taxonomy.EnrichTaxonomies(langTax, b.config.Taxonomies, b.projectDir, code)
			if err != nil {
				return fmt.Errorf("enriching taxonomies for %s: %w", code, err)
			}
			allTaxWarnings = append(allTaxWarnings, w...)
			taxByLang[code] = langTax
		}
		emitTaxonomyWarnings(dedupStrings(allTaxWarnings))
		taxonomies = taxByLang[s.defaultLang]
	} else {
		var buildWarns []string
		taxonomies, buildWarns = taxonomy.BuildTaxonomies(allPages, b.config.Taxonomies, "")
		w, err := taxonomy.EnrichTaxonomies(taxonomies, b.config.Taxonomies, b.projectDir, "")
		if err != nil {
			return fmt.Errorf("enriching taxonomies: %w", err)
		}
		emitTaxonomyWarnings(append(buildWarns, w...))
	}

	// Build SiteContext.
	siteCtx := &engine.SiteContext{
		Title:            b.config.Site.Title,
		BaseURL:          b.config.Site.URL,
		BasePath:         b.config.Build.BasePath,
		SiteID:           computeSiteID(b.config.Site.Title, b.config.Build.BasePath),
		Language:         b.config.Site.Language,
		Generator:        "Sarde v" + version.Version,
		SitemapEnabled:   b.config.SitemapActive(),
		Config:           b.config,
		Collections:      s.collections,
		Taxonomies:       taxonomies,
		TaxonomiesByLang: taxByLang,
		Pages:            allPages,
		BuildTime:        time.Now(),
		EditURL:          b.config.Site.EditURL,
		IconLicenses:     buildIconLicenses(),
	}

	// Resolve all Permalink fields through the URL resolver.
	// RelPermalink stays prefix-free; Permalink gets basePath + lang.
	langSet := make(map[string]bool, len(b.config.I18n.Languages))
	for code := range b.config.I18n.Languages {
		langSet[code] = true
	}
	collMounts, versionIDs := buildResolverRegistries(s.collections)
	b.urlResolver = &engine.URLResolver{
		BasePath:         b.config.Build.BasePath,
		BaseURL:          b.config.Site.URL,
		I18nEnabled:      b.config.I18n.IsMultiLang(),
		DefaultLang:      b.config.I18n.GetDefaultLanguage(),
		Strategy:         b.config.I18n.Strategy,
		Languages:        langSet,
		CollectionMounts: collMounts,
		VersionIDs:       versionIDs,
	}
	urlResolver := b.urlResolver
	b.resolutionKey = urlResolver.CacheKey() + "|escape=" + b.config.LinkValidation.SiteRootEscapePrefix
	resolvePermalinks(urlResolver, allPages)
	addLegacyDateAliases(urlResolver, allPages, b.config.Redirects)

	// Resolve per-language taxonomy permalinks through the URLResolver.
	if taxByLang != nil {
		for code, langTax := range taxByLang {
			for _, tax := range langTax {
				tax.Permalink = urlResolver.URL(tax.Permalink, code, "")
				for _, term := range tax.Terms {
					term.Permalink = urlResolver.URL(term.Permalink, code, "")
				}
			}
		}
		siteCtx.Taxonomies = taxByLang[s.defaultLang]
	}

	// i18n: populate site-level language info
	if s.isMultiLang {
		siteCtx.DefaultLang = s.defaultLang
		for _, code := range b.config.I18n.LanguageCodes() {
			lc := b.config.I18n.Languages[code]
			dir := lc.Dir
			if dir == "" {
				dir = "ltr"
			}
			siteCtx.Languages = append(siteCtx.Languages, engine.Language{
				Code:   code,
				Name:   lc.Name,
				Dir:    dir,
				Weight: lc.Weight,
			})
		}
	}

	s.allPages = allPages
	s.taxonomies = taxonomies
	s.taxByLang = taxByLang
	s.siteCtx = siteCtx
	s.recordTiming("Assembling site")
	return nil
}

func computeSiteID(title, basePath string) string {
	h := uint32(5381)
	for _, b := range []byte(title + basePath) {
		h = h*33 ^ uint32(b)
	}
	return fmt.Sprintf("%08x", h)
}

// i18nFallbackOptions returns the fallback policies GenerateFallbacks applies:
// the site's i18n.fallback and each collection's own override, where a
// versioned collection's versioning.fallback wins over its i18n_fallback. The
// full and the incremental build both call it, so they cannot disagree.
func i18nFallbackOptions(cfg *config.SiteConfig) i18n.FallbackOptions {
	collFallback := make(map[string]string)
	for colName := range cfg.Collections {
		if fb := cfg.CollectionFallbackOverride(colName); fb != "" {
			collFallback[colName] = fb
		}
	}
	return i18n.FallbackOptions{
		SiteFallback:       cfg.I18n.Fallback,
		CollectionFallback: collFallback,
	}
}

// languageWeights maps each configured language code to its weight.
func languageWeights(cfg *config.SiteConfig) map[string]int {
	weights := make(map[string]int, len(cfg.I18n.Languages))
	for code, lc := range cfg.I18n.Languages {
		weights[code] = lc.Weight
	}
	return weights
}
