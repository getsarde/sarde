package collection

import (
	"html/template"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/content"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/workers"
)

func groupByCollection(files []content.ContentFile) map[string][]content.ContentFile {
	result := make(map[string][]content.ContentFile)
	for _, f := range files {
		result[f.CollectionName] = append(result[f.CollectionName], f)
	}
	return result
}

func buildPages(
	files []content.ContentFile,
	contentDir string,
	collCfg *engine.CollectionConfig,
	schema *engine.FrontmatterSchema,
	summaryLength int,
	lastUpdatedStrategy string,
	taxCfg map[string]config.TaxonomyConfig,
) ([]*engine.Page, []engine.ValidationWarning, error) {
	return buildPagesWithOptions(files, contentDir, collCfg, schema, summaryLength, lastUpdatedStrategy, BuildOptions{}, taxCfg)
}

func buildPagesWithOptions(
	files []content.ContentFile,
	contentDir string,
	collCfg *engine.CollectionConfig,
	schema *engine.FrontmatterSchema,
	summaryLength int,
	lastUpdatedStrategy string,
	opts BuildOptions,
	taxCfg map[string]config.TaxonomyConfig,
) ([]*engine.Page, []engine.ValidationWarning, error) {
	type result struct {
		page     *engine.Page
		warnings []engine.ValidationWarning
	}
	results := make([]result, len(files))
	err := workers.ParallelFor(files, opts.Parallel, opts.WorkerCount, func(i int, cf content.ContentFile) error {
		page, pageWarnings, err := buildPage(cf, contentDir, collCfg, schema, summaryLength, lastUpdatedStrategy, taxCfg)
		if err != nil {
			return err
		}
		results[i] = result{page: page, warnings: pageWarnings}
		return nil
	})
	if err != nil {
		return nil, nil, err
	}

	pages := make([]*engine.Page, 0, len(files))
	var warnings []engine.ValidationWarning
	for _, r := range results {
		pages = append(pages, r.page)
		warnings = append(warnings, r.warnings...)
	}
	return pages, warnings, nil
}

func buildPage(
	cf content.ContentFile,
	contentDir string,
	collCfg *engine.CollectionConfig,
	schema *engine.FrontmatterSchema,
	summaryLength int,
	lastUpdatedStrategy string,
	taxCfg map[string]config.TaxonomyConfig,
) (*engine.Page, []engine.ValidationWarning, error) {
	inferrer := &content.Inferrer{LastUpdatedStrategy: lastUpdatedStrategy}
	transformer := &content.Transformer{SummaryLength: summaryLength}
	validator := &content.Validator{}

	var warnings []engine.ValidationWarning

	f, err := os.Open(cf.FilePath)
	if err != nil {
		return nil, nil, err
	}
	fi, _ := f.Stat()
	raw, err := io.ReadAll(f)
	f.Close()
	if err != nil {
		return nil, nil, err
	}

	// Parse frontmatter: single pass produces both untyped map and typed struct.
	fmMap, fm, body, fmLines, err := content.ParseAll(raw)
	if err != nil {
		return nil, nil, err
	}

	contentDigest := rawDigest(raw)
	frontmatterDigest := fmDigest(fmMap)

	// Detect unknown frontmatter keys (always runs)
	if unknownWarnings := content.DetectUnknownFields(fmMap, schema, taxCfg, cf.RelPath); len(unknownWarnings) > 0 {
		warnings = append(warnings, unknownWarnings...)
	}

	// Apply schema defaults
	if schema != nil {
		fmMap = content.ApplyDefaults(fmMap, schema)
	}

	// Validate against schema
	if schema != nil {
		w := validator.Validate(fmMap, schema)
		for i := range w {
			w[i].File = cf.RelPath
		}
		warnings = append(warnings, w...)
	}

	// Build Page
	page := &engine.Page{
		PageIdentity: engine.PageIdentity{
			Title:       fm.Title,
			Slug:        fm.Slug,
			Date:        fm.Date,
			Updated:     fm.Updated,
			PublishDate: fm.PublishDate,
			ExpiryDate:  fm.ExpiryDate,
			Kind:        cf.Kind,
			FilePath:    cf.FilePath,
		},
		PageContent: engine.PageContent{
			RawContent:        body,
			ContentDigest:     contentDigest,
			FrontmatterDigest: frontmatterDigest,
			FrontmatterLines:  fmLines,
		},
		PageMeta: engine.PageMeta{
			Draft:       fm.Draft,
			Description: fm.Description,
			Image:       fm.Image,
		},
		PageTaxonomy: engine.PageTaxonomy{
			Tags:       fm.Tags,
			Categories: fm.Categories,
			Aliases:    fm.Aliases,
		},
		Sidebar: engine.PageSidebar{
			Order:  fm.Sidebar.Order,
			Label:  fm.Sidebar.Label,
			Hidden: fm.Sidebar.Hidden,
			Attrs:  fm.Sidebar.Attrs,
			Badge:  fm.Sidebar.Badge,
			Icon:   fm.Sidebar.Icon,
		},
		ShowTags: fm.ShowTags,
		TOC: engine.PageTOC{
			Enabled:  fm.TOC.Enabled,
			MinLevel: fm.TOC.MinLevel,
			MaxLevel: fm.TOC.MaxLevel,
		},
		PageI18n: engine.PageI18n{
			Lang:        cf.Lang,
			LangRelPath: cf.LangRelPath,
		},
		PageVersioning: engine.PageVersioning{
			Version:        cf.Version,
			VersionRelPath: cf.VersionRelPath,
		},
		Params: fm.Params,
	}
	if rel, err := filepath.Rel(contentDir, cf.FilePath); err == nil {
		page.RelPath = filepath.ToSlash(rel)
	}

	mapFrontmatterToParams(page, fm, fmMap, taxCfg)

	// Pre-populate Date from the already-opened FileInfo to avoid a
	// redundant os.Stat inside Infer.
	if page.Date.IsZero() && fi != nil {
		page.Date = fi.ModTime()
	}

	// Infer defaults (title, date, slug, weight)
	if err := inferrer.Infer(page, cf.FilePath); err != nil {
		return nil, nil, err
	}

	if pageWarnings := content.ValidatePageFields(page, fm); len(pageWarnings) > 0 {
		for i := range pageWarnings {
			pageWarnings[i].File = cf.RelPath
		}
		warnings = append(warnings, pageWarnings...)
	}

	// Compute permalink: use pattern if configured, else directory-based
	isIndex := filepath.Base(cf.FilePath) == "_index.md" || filepath.Base(cf.FilePath) == "index.md"
	if collCfg != nil && collCfg.Permalink != "" && !isIndex {
		vars := content.PermalinkVars{
			Slug:       page.Slug,
			Year:       page.Date.Format("2006"),
			Month:      page.Date.Format("01"),
			Day:        page.Date.Format("02"),
			Section:    extractSection(cf.FilePath, contentDir),
			Collection: cf.CollectionName,
			Title:      content.Slugify(page.Title),
		}
		page.RelPermalink = content.ComputePatternPermalink(collCfg.Permalink, vars)
	} else {
		// For versioned pages, compute RelPermalink from the version-free
		// relative path so that RelPermalink is version-free but
		// collection-bearing (e.g. "/docs/guides/auth/" not "/docs/v1/guides/auth/").
		relPathForPermalink := cf.LangRelPath
		if cf.Version != "" {
			relPathForPermalink = content.VersionFreeRelPath(&cf)
		}
		if relPathForPermalink != "" {
			page.RelPermalink = content.ComputePermalinkFromRelPath(relPathForPermalink)
		} else {
			page.RelPermalink = content.ComputePermalink(contentDir, cf.FilePath)
		}
	}
	page.Permalink = page.RelPermalink

	// Transform (word count, reading time, summary)
	if err := transformer.Transform(page); err != nil {
		return nil, nil, err
	}

	// Copy bundle assets
	if cf.IsBundle {
		for _, asset := range cf.BundleAssets {
			page.Resources = append(page.Resources, engine.Resource{
				Name: filepath.Base(asset),
			})
		}
	}

	return page, warnings, nil
}

// mapFrontmatterToParams transfers optional frontmatter fields to page.Params
// and page.Summary. It reads from the typed struct for all known fields and
// falls back to fmMap for fields not present on Frontmatter (e.g. "featured").
func mapFrontmatterToParams(page *engine.Page, fm *engine.Frontmatter, fmMap map[string]interface{}, taxCfg map[string]config.TaxonomyConfig) {
	if page.Params == nil {
		page.Params = make(map[string]any)
	}

	if fm.Transparent {
		page.Params["transparent"] = true
	}
	if fm.Render != nil && !*fm.Render {
		page.Params["render"] = false
	}
	if b, ok := fmMap["featured"].(bool); ok && b {
		page.Params["featured"] = true
	}
	if fm.Template != "" {
		page.Params["template"] = fm.Template
	}
	if fm.Summary != "" {
		page.Summary = template.HTML(fm.Summary)
	}
	if fm.Prev != nil {
		page.Params["prev"] = fm.Prev
	}
	if fm.Next != nil {
		page.Params["next"] = fm.Next
	}
	if fm.Pagefind != nil {
		page.Params["pagefind"] = *fm.Pagefind
	}
	if fm.Layout != "" {
		page.Params["layout"] = fm.Layout
	}
	if fm.Type != "" {
		page.Params["type"] = fm.Type
	}
	if fm.Hero != nil {
		fm.Hero.SanitizeAttrs()
		page.Params["hero"] = fm.Hero
	}
	if fm.EditURL != nil {
		if fm.EditURL.CustomURL != "" {
			page.Params["edit_url"] = fm.EditURL.CustomURL
		} else if fm.EditURL.Disabled {
			page.Params["edit_url"] = false
		}
	}
	if len(fm.Head) > 0 {
		var filtered []engine.HeadTag
		for _, h := range fm.Head {
			if engine.AllowedHeadTags[h.Tag] {
				filtered = append(filtered, h)
			}
		}
		if len(filtered) > 0 {
			page.Params["head"] = filtered
		}
	}
	if fm.Banner != nil && fm.Banner.Content != "" {
		page.Params["banner"] = fm.Banner
	}
	if fm.ShowUpdated != nil {
		page.Params["show_updated"] = *fm.ShowUpdated
	}
	if fm.Icon != "" {
		page.Params["icon"] = fm.Icon
	}
	if len(fm.LearningObjectives) > 0 {
		objs := make([]any, len(fm.LearningObjectives))
		for i, o := range fm.LearningObjectives {
			objs[i] = o
		}
		page.Params["learning_objectives"] = objs
	}
	if len(fm.Cascade) > 0 {
		page.Params[consts.CascadeKey] = fm.Cascade
	}

	for key := range taxCfg {
		if key == "tags" || key == "categories" {
			continue
		}
		raw, ok := fmMap[key]
		if !ok {
			continue
		}
		vals := toStringSlice(raw)
		if len(vals) == 0 {
			continue
		}
		if page.Extra == nil {
			page.Extra = make(map[string][]string)
		}
		page.Extra[key] = vals
	}
}

func toStringSlice(v interface{}) []string {
	switch t := v.(type) {
	case []string:
		return t
	case []interface{}:
		out := make([]string, 0, len(t))
		for _, item := range t {
			if s, ok := item.(string); ok && s != "" {
				out = append(out, s)
			}
		}
		return out
	case string:
		if t != "" {
			return []string{t}
		}
	}
	return nil
}

// extractSection returns the immediate parent directory name relative to the collection root.
func extractSection(filePath, contentDir string) string {
	rel, _ := filepath.Rel(contentDir, filePath)
	rel = filepath.ToSlash(rel)
	parts := strings.Split(rel, "/")
	// parts[0] is collection, parts[1..n-1] are sections, parts[n-1] is filename
	if len(parts) > 2 {
		return parts[len(parts)-2]
	}
	return ""
}

// extractFeatured returns the subset of pages whose frontmatter Params has
// `featured: true`.
func extractFeatured(pages []*engine.Page) []*engine.Page {
	var out []*engine.Page
	for _, p := range pages {
		if p.Params == nil {
			continue
		}
		if v, ok := p.Params["featured"]; ok {
			if b, _ := v.(bool); b {
				out = append(out, p)
			}
		}
	}
	return out
}
