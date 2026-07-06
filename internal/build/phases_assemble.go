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
		weights := make(map[string]int)
		for code, lc := range b.config.I18n.Languages {
			weights[code] = lc.Weight
		}

		i18n.LinkTranslations(allPages, weights)

		collFallback := make(map[string]string)
		for colName, colCfg := range b.config.Collections {
			if colCfg == nil {
				continue
			}
			if colCfg.I18nFallback != "" {
				collFallback[colName] = colCfg.I18nFallback
			}
			// Versioning fallback overrides i18n fallback for versioned collections.
			if colCfg.Versioning != nil && config.BoolVal(colCfg.Versioning.Enabled, false) && colCfg.Versioning.Fallback != "" {
				collFallback[colName] = colCfg.Versioning.Fallback
			}
		}
		fbOpts := i18n.FallbackOptions{
			SiteFallback:       b.config.I18n.Fallback,
			CollectionFallback: collFallback,
		}

		fallbacks := i18n.GenerateFallbacks(allPages, langCodes, s.defaultLang, fbOpts)
		allPages = append(allPages, fallbacks...)

		collection.RebuildNavTreesWithFallbacks(s.collections, allPages, langCodes)

		i18n.LinkAllTranslations(allPages, weights)
	}

	collection.LinkVersions(allPages)

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
		SitemapEnabled:   b.config.Build.Sitemap == nil || *b.config.Build.Sitemap,
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
