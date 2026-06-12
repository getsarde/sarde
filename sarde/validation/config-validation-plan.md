# Config Validation Layer for Sarde SSG

## Implementation Plan

### Overview

Design a centralized config validation layer that catches invalid sarde.yaml values (typos, wrong enums, out-of-range numbers) and reports clear, actionable errors. The validation runs on the merged config (after all 5 cascade layers) and integrates into both sarde build and sarde validate.

---

## Architecture

### New Files

| File | Purpose |
|------|---------|
| internal/config/validate.go | Core validation logic: Validate(*SiteConfig) []ConfigError plus per-concern validators |
| internal/config/validate_test.go | Table-driven tests for every validation rule |

### Modified Files

| File | Change |
|------|--------|
| internal/config/loader.go | Add LoadFileStrict() using yaml.NewDecoder + KnownFields(true) |
| internal/config/merge.go | Call Validate() at end of Resolve(), add StrictFields to ResolveOptions |
| internal/cli/build.go | Set StrictFields: true in resolveAll() |

---

## Part 1: The ConfigError Type

Define in internal/config/validate.go. A dedicated type (not reusing engine.ValidationWarning) because config errors have a config path (not a file path) and are structurally different.

Fields:
- Path string -- dot-separated config path, e.g. collections.docs.layout
- Value string -- the invalid value found
- Message string -- human-readable explanation
- Level string -- error or warn

The Error() method format: server.port: must be >= 1 (got "-5")

Helper functions:
- FormatConfigErrors(errs []ConfigError) string -- multi-line CLI output
- HasErrors(errs []ConfigError) bool -- true if any error-level finding exists

---

## Part 2: Unknown Field Detection via LoadFileStrict

Add LoadFileStrict(path string) (*SiteConfig, error) to internal/config/loader.go.

Uses yaml.NewDecoder(bytes.NewReader(data)) with dec.KnownFields(true) to reject unknown YAML fields. Catches typos like tab: false (instead of tabs: false).

Apply ONLY to user-authored sarde.yaml (layer 3). Trusted internal sources (embedded defaults layer 1, theme.yaml layer 2) continue using the permissive LoadFile.

Add StrictFields bool to ResolveOptions. When true, layer 3 uses LoadFileStrict.

### Unknown-field detection: hard error by default

- Unknown fields are almost always typos. Silent acceptance is the worst outcome.
- yaml.v3 produces clear error messages with line/column numbers.
- Escape hatch: StrictFields boolean in ResolveOptions (defaulting to true in CLI).

### Caveats

- Custom UnmarshalYAML methods (Logo, TaxonomyConfig, LastUpdatedStrategy) use alias-type decoding -- inner fields ARE still checked.
- Map fields (map[string]string, map[string]any) naturally accept any key -- no false positives for redirects, permalinks, theme.overrides, plugins.config.
- PluginSettings.Config (map[string]map[string]any) accepts arbitrary plugin config -- intentional, no false positives.

---

## Part 3: Centralized Validate() Function

```go
func Validate(cfg *SiteConfig) []ConfigError {
    var errs []ConfigError
    errs = append(errs, validateEnums(cfg)...)
    errs = append(errs, validateRanges(cfg)...)
    errs = append(errs, validateInterdependencies(cfg)...)
    errs = append(errs, validateCollections(cfg)...)
    errs = append(errs, validateVersioningAll(cfg)...)
    return errs
}
```

### 3a. Enum Validation (table-driven)

A slice of enumRule structs, each with path, value func(*SiteConfig) string, and valid []string.

Static enum rules (14 total):
- build.last_updated: git, mtime, false, off
- link_validation.on_broken: error, warn, ignore
- link_validation.on_broken_anchor: error, warn, ignore
- link_validation.report: pretty, json, github-actions
- link_validation.on_relative_links: error, warn, ignore
- link_validation.on_local_links: error, warn, ignore
- link_validation.on_unverified_internal: error, warn, ignore
- link_validation.external.on_broken: error, warn, ignore
- link_validation.external.method: head-then-get, head, get
- icons.render: inline, sprite
- deploy.provider: github, netlify, cloudflare, vercel, custom
- deploy.redirect_format: html, netlify, vercel, all
- i18n.strategy: prefix-except-default
- i18n.fallback: default, omit

Dynamic enums (via validateDynamicEnums):
- i18n.languages[code].dir: ltr, rtl
- collections[name].layout: default, docs, splash, wide, full, centered, split, presentation
- collections[name].i18n_fallback: default, omit
- collections[name].versioning.versions[i].banner: none, unmaintained, unreleased
- collections[name].versioning.versions[i].redirect: same-page, root
- taxonomies[name].undefined_tags: warn, error, ignore, create

All enum validators skip empty strings (unset fields).

### 3b. Range Validation (table-driven)

A slice of rangeRule structs with path, value func, min, max (0 = no upper bound).

Rules (9 total):
- server.port: 1-65535
- toc.min_level: 1-6
- toc.max_level: 1-6
- markdown.toc.min_heading_level: 1-6
- markdown.toc.max_heading_level: 1-6
- images.quality: 1-100
- images.max_width: >= 1
- link_validation.external.concurrency: >= 1
- content.summary_length: >= 1

All range validators skip zero values (unset fields).

### 3c. Interdependency Validation

- toc.min_level <= toc.max_level
- markdown.toc.min_heading_level <= markdown.toc.max_heading_level

### 3d. Collection-Scoped Validation

Per-collection checks:
- sidebar.max_depth >= 0
- toc.depth between 1 and 6

### 3e. Versioning Validation

Wraps existing ValidateVersioning() -- converts its error return into []ConfigError for consistency.

---

## Part 4: Integration into Resolve()

At the end of Resolve() in merge.go, after normalization:

```go
if errs := Validate(cfg); len(errs) > 0 {
    if HasErrors(errs) {
        return nil, fmt.Errorf("%s", FormatConfigErrors(errs))
    }
    // Warnings only -- log but continue.
    for _, e := range errs {
        devlog.Warn("config", "%s", e.Error())
    }
}
```

normalizeI18n already does some enum validation. Keep both initially for zero-risk migration; remove duplicates from normalizeI18n in a follow-up.

---

## Part 5: CLI Integration

resolveAll() in build.go is the single gateway for all CLI commands. Set StrictFields: true there:

```go
cfg, err := config.Resolve(config.ResolveOptions{
    ConfigPath:   configPath,
    CLIFlags:     CollectCLIFlags(cmd),
    EnvPrefix:    "SARDE",
    StrictFields: true,
})
```

All commands (build, dev, validate, check-links) automatically get config validation. No per-command changes needed.

Example error output:
```
Error: resolving config: config: 2 error(s):
  ERROR server.port: must be >= 1 (got "-5")
  ERROR collections.docs.layout: must be one of: default, docs, splash, wide, full, centered, split, presentation (got "banana")
```

---

## Part 6: Test Plan

### validate_test.go

Table-driven tests:
1. TestValidateEnums_ValidConfig -- defaults produce zero errors
2. TestValidateEnums_InvalidValues -- each enum field with bad value
3. TestValidateRanges_InvalidValues -- port too low/high, toc levels out of range, quality > 100
4. TestValidateInterdependencies_MinGtMax -- min_level > max_level
5. TestValidate_ZeroValuesSkipped -- empty SiteConfig produces no errors
6. TestHasErrors -- nil, warnings-only, errors

### Loader tests (extend config_test.go)

1. TestLoadFileStrict_UnknownField -- typo like "titl" is rejected
2. TestLoadFileStrict_ValidConfig -- valid file loads fine
3. TestLoadFileStrict_MissingFile -- returns (nil, nil)

---

## Part 7: Migration of Existing Validation

| Current location | What it validates | Action |
|---|---|---|
| normalizeI18n() in merge.go | strategy, fallback enums | Keep (also sets defaults). Add parallel rules in validateEnums. Remove in follow-up. |
| ValidateVersioning() in config.go | version ID uniqueness | Keep. Wrap in validateVersioningAll(). Remove registry.go call in follow-up. |
| ValidateLayout() in engine/types.go | 8 layout types | Use same list in validateDynamicEnums. Do NOT import engine (circular dep). Duplicate as string slice. |

### Circular Dependency Avoidance

internal/config must NOT import internal/engine. Layout values are duplicated as a []string in validate.go with comment: // Keep in sync with engine.validLayouts.

---

## Implementation Sequence

1. **Create validate.go + validate_test.go** -- ConfigError type, Validate(), all sub-validators, comprehensive table-driven tests. Run in isolation, no integration yet.
2. **Add LoadFileStrict to loader.go** -- bytes import, KnownFields(true) decoder, tests for unknown fields.
3. **Wire into Resolve()** -- StrictFields in ResolveOptions, switch layer 3 loading, call Validate() after normalization. Run existing test suite for regression check.
4. **Set StrictFields: true in CLI** -- Update resolveAll() in build.go. Test with real project configs.
5. **Cleanup follow-up** -- Remove redundant i18n enum checks from normalizeI18n. Remove redundant ValidateVersioning call from registry.go. Add --no-strict-fields flag if needed.

---

## Potential Challenges

1. **Custom UnmarshalYAML and KnownFields**: Types with custom unmarshalers (Logo, TaxonomyConfig, LastUpdatedStrategy) use alias-type decoding (type plain Logo). The plain alias preserves struct tags, so KnownFields(true) still works for inner fields.

2. **Map fields accepting arbitrary keys**: Fields like redirects, permalinks, theme.overrides, plugins.config are map types. yaml.v3 correctly allows any keys for maps even with KnownFields(true). No false positives.

3. **Zero-value skipping**: The merge layer uses zero-value semantics (mergeStr skips empty, mergeInt skips 0). Validation must also skip zero values. All validators check for zero/empty before validating.

4. **Negative port numbers**: yaml.Unmarshal decodes port: -5 as int -5. The range check catches this.

5. **Boolean fields as *bool**: No range/enum validation needed for booleans. Unknown-field detection handles typos like tab: false (instead of tabs: false).

---

## Critical Files for Implementation

- internal/config/validate.go (new)
- internal/config/validate_test.go (new)
- internal/config/loader.go (add LoadFileStrict)
- internal/config/merge.go (wire Validate into Resolve, add StrictFields)
- internal/cli/build.go (set StrictFields: true in resolveAll)
