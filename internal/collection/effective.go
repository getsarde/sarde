package collection

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/engine"
)

// FieldSource identifies where an effective collection config value came
// from: the zero-config, directory-name-based inference in InferCollection,
// or an explicit override in the user's sarde.yaml.
type FieldSource string

const (
	SourceInferred  FieldSource = "inferred"
	SourceSardeYAML FieldSource = "sarde_yaml"
)

// FieldValue pairs an effective (merged) value with its provenance.
type FieldValue struct {
	Value  any         `json:"value"`
	Source FieldSource `json:"source"`
}

// EffectiveSidebar mirrors the subset of engine.SidebarConfig that Sarde
// Studio's collection settings expose. CollapseLevel and sidebar.yaml path
// overrides are intentionally out of scope (see BuildEffectiveConfig).
type EffectiveSidebar struct {
	Collapsible        FieldValue `json:"collapsible"`
	CollapsedByDefault FieldValue `json:"collapsedByDefault"`
	MaxDepth           FieldValue `json:"maxDepth"`
	Search             FieldValue `json:"search"`
}

// EffectiveTOC mirrors the subset of engine.TOCConfig Studio exposes.
// MinLevel has no sarde.yaml field (always inference-only), so it is omitted.
type EffectiveTOC struct {
	Enabled         FieldValue `json:"enabled"`
	ScrollHighlight FieldValue `json:"scrollHighlight"`
	Depth           FieldValue `json:"depth"` // engine.TOCConfig.MaxLevel
}

// EffectivePrevNext mirrors engine.PrevNextConfig.
type EffectivePrevNext struct {
	Enabled FieldValue `json:"enabled"`
	Labels  FieldValue `json:"labels"` // value: [2]string
}

// EffectiveCollection is one collection's fully resolved config plus
// per-field provenance. Scope is limited to fields that flow through
// InferCollection/MergeCollectionConfig.
type EffectiveCollection struct {
	Name         string             `json:"name"`
	InferredType string             `json:"inferredType"` // "blog" | "docs" | "default"
	SortBy       FieldValue         `json:"sortBy"`
	SortOrder    FieldValue         `json:"sortOrder"`
	Layout       FieldValue         `json:"layout"`
	Permalink    FieldValue         `json:"permalink"`
	Paginate     FieldValue         `json:"paginate"`
	Feed         FieldValue         `json:"feed"`
	Tabs         FieldValue         `json:"tabs"` // value: bool | null (null = auto-detect)
	Sidebar      *EffectiveSidebar  `json:"sidebar"`
	TOC          *EffectiveTOC      `json:"toc"`
	PrevNext     *EffectivePrevNext `json:"prevNext"`
}

// BuildEffectiveConfig computes the merged per-collection config and each
// field's provenance for every collection in projectDir. It re-resolves
// sarde.yaml fresh on every call so callers always see the file's current
// state — the Studio bridge must not cache this either.
//
// Excluded from the report entirely: Enabled/Path/URLPrefix/I18nFallback
// (InferCollection never sets them, so there is no inferred-vs-explicit
// story), Versioning (always explicit opt-in, never inferred), TOC.MinLevel
// and Sidebar.CollapseLevel (no sarde.yaml field / not edited in Studio),
// and sidebar.yaml path overrides (a different file entirely).
//
// PROVENANCE CAVEAT: cfg.Collections and cfg.Permalinks (from
// config.Resolve) are used both as MergeCollectionConfig's input AND as the
// "was this explicitly set in sarde.yaml" signal. That is valid only
// because nothing in today's 5-layer cascade other than sarde.yaml (layer
// 3) populates SiteConfig.Collections/Permalinks: Defaults() sets neither,
// ResolveOptions.ThemeDir is never set by any caller in this repo, and CLI
// flags/env (layers 4-5) never touch collections. If a future change
// starts merging theme.yaml collection settings, these "sarde_yaml" labels
// would silently cover theme.yaml too — switch provenance to a separate,
// isolated config.LoadFile(configPath) call if that happens.
func BuildEffectiveConfig(projectDir string) ([]EffectiveCollection, error) {
	configPath := filepath.Join(projectDir, consts.FileSiteConfig)

	cfg, err := config.Resolve(config.ResolveOptions{
		ConfigPath: configPath,
		EnvPrefix:  "SARDE",
	})
	if err != nil {
		return nil, err
	}

	names := enumerateCollectionNames(projectDir, cfg)

	result := make([]EffectiveCollection, 0, len(names))
	for _, name := range names {
		inferred := InferCollection(name)
		raw := cfg.Collections[name] // nil if not set in sarde.yaml
		merged := MergeCollectionConfig(inferred, raw)

		rawPermalinkPattern := ""
		if cfg.Permalinks != nil {
			rawPermalinkPattern = cfg.Permalinks[name]
		}
		if merged.Permalink == "" && rawPermalinkPattern != "" {
			permalinked := *merged
			permalinked.Permalink = rawPermalinkPattern
			merged = &permalinked
		}

		result = append(result, describeCollection(name, merged, raw, rawPermalinkPattern))
	}

	return result, nil
}

// enumerateCollectionNames returns every collection name: subdirectories of
// content/ (same dot-/underscore-prefix filter as the project scanner)
// unioned with keys of sarde.yaml's collections map, so a
// configured-but-not-yet-created collection still appears in the output.
func enumerateCollectionNames(projectDir string, cfg *config.SiteConfig) []string {
	seen := make(map[string]bool)
	var names []string

	contentDir := filepath.Join(projectDir, consts.DirContent)
	if entries, err := os.ReadDir(contentDir); err == nil {
		for _, e := range entries {
			if !e.IsDir() || strings.HasPrefix(e.Name(), ".") || strings.HasPrefix(e.Name(), "_") {
				continue
			}
			if !seen[e.Name()] {
				seen[e.Name()] = true
				names = append(names, e.Name())
			}
		}
	}
	for name := range cfg.Collections {
		if !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}

	sort.Strings(names)
	return names
}

// inferredBucket reports which name-based inference bucket a collection
// falls into. Reads blogNames/docsNames directly (infer.go, same package)
// so it can never drift from InferCollection.
func inferredBucket(name string) string {
	switch {
	case blogNames[name]:
		return "blog"
	case docsNames[name]:
		return "docs"
	case labsNames[name]:
		return "labs"
	case slidesNames[name]:
		return "slides"
	default:
		return "default"
	}
}

// describeCollection builds one collection's report. merged is the
// effective (post-inference, post-merge) config; raw is the collection's
// entry straight from sarde.yaml (nil if absent), used only to decide each
// field's source. Every predicate below intentionally mirrors
// MergeCollectionConfig's non-zero-wins condition for the same field —
// keep them in sync if MergeCollectionConfig ever changes.
func describeCollection(name string, merged *engine.CollectionConfig, raw *config.CollectionSiteConfig, rawPermalinkPattern string) EffectiveCollection {
	ec := EffectiveCollection{
		Name:         name,
		InferredType: inferredBucket(name),
	}

	sortBySrc, sortOrderSrc := SourceInferred, SourceInferred
	if raw != nil && raw.Sort != "" {
		by, order := parseSortString(raw.Sort)
		if by != "" {
			sortBySrc = SourceSardeYAML
		}
		if order != "" {
			sortOrderSrc = SourceSardeYAML
		}
	}
	ec.SortBy = FieldValue{merged.SortBy, sortBySrc}
	ec.SortOrder = FieldValue{merged.SortOrder, sortOrderSrc}

	ec.Layout = FieldValue{string(merged.Layout), boolSrc(raw != nil && raw.Layout != "")}

	// Permalink has no inference story (InferCollection never sets it);
	// "inferred" here really means "no override — the engine's built-in
	// default pattern applies". Studio renders this with different badge
	// copy ("Default") than the name-bucket fields.
	permalinkSet := (raw != nil && raw.Permalink != "") || rawPermalinkPattern != ""
	ec.Permalink = FieldValue{merged.Permalink, boolSrc(permalinkSet)}

	ec.Paginate = FieldValue{merged.Paginate, boolSrc(raw != nil && raw.Paginate != 0)}
	ec.Feed = FieldValue{merged.Feed, boolSrc(raw != nil && raw.Feed != nil)}

	var tabsVal any
	if merged.Tabs != nil {
		tabsVal = *merged.Tabs
	}
	ec.Tabs = FieldValue{tabsVal, boolSrc(raw != nil && raw.Tabs != nil)}

	if merged.Sidebar != nil {
		var rs *config.CollectionSidebarConfig
		if raw != nil {
			rs = raw.Sidebar
		}
		ec.Sidebar = &EffectiveSidebar{
			Collapsible:        FieldValue{merged.Sidebar.Collapsible, boolSrc(rs != nil && rs.Collapsible != nil)},
			CollapsedByDefault: FieldValue{merged.Sidebar.CollapsedByDefault, boolSrc(rs != nil && rs.CollapsedByDefault != nil)},
			MaxDepth:           FieldValue{merged.Sidebar.MaxDepth, boolSrc(rs != nil && rs.MaxDepth != 0)},
			Search:             FieldValue{merged.Sidebar.Search, boolSrc(rs != nil && rs.Search != nil)},
		}
	}

	if merged.TOC != nil {
		var rt *config.CollectionTOCConfig
		if raw != nil {
			rt = raw.TOC
		}
		ec.TOC = &EffectiveTOC{
			Enabled:         FieldValue{merged.TOC.Enabled, boolSrc(rt != nil && rt.Enabled != nil)},
			ScrollHighlight: FieldValue{merged.TOC.ScrollHighlight, boolSrc(rt != nil && rt.ScrollHighlight != nil)},
			Depth:           FieldValue{merged.TOC.MaxLevel, boolSrc(rt != nil && rt.Depth != 0)},
		}
	}

	if merged.PrevNext != nil {
		var rp *config.CollectionPrevNextConfig
		if raw != nil {
			rp = raw.PrevNext
		}
		ec.PrevNext = &EffectivePrevNext{
			Enabled: FieldValue{merged.PrevNext.Enabled, boolSrc(rp != nil && rp.Enabled != nil)},
			Labels:  FieldValue{[]string{merged.PrevNext.Labels[0], merged.PrevNext.Labels[1]}, boolSrc(rp != nil && len(rp.Labels) == 2)},
		}
	}

	return ec
}

func boolSrc(isSet bool) FieldSource {
	if isSet {
		return SourceSardeYAML
	}
	return SourceInferred
}
