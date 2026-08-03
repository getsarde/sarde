package external

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/license"
	"github.com/getsarde/sarde/internal/plugin"
	"github.com/getsarde/sarde/internal/plugin/cfgutil"
)

// dirTemplates is the plugin subdirectory holding template contributions
// (partials/, components/, shortcodes/).
const dirTemplates = "templates"

// dirDirectives is the plugin subdirectory holding generic directive
// definitions (<name>.yaml + <name>.html, optional <name>.css).
const dirDirectives = "directives"

// LoadAll discovers, validates, and registers every external plugin found
// under {projectDir}/plugins. External plugins are enabled by presence on
// disk; slugs listed in plugins.disabled are skipped. reserved holds plugin
// names already taken by built-in, subpackage, and client-side plugins.
//
// Each plugin is handled independently: a malformed manifest, a reserved
// slug, or a failed premium license check produces a warning and skips only
// that plugin. Returns the templates/ directories of active plugins (for the
// template overlay chain), their directives/ directories (for the generic
// directive overlay chain), and the accumulated warnings.
func LoadAll(mgr *plugin.Manager, projectDir string, cfg *config.SiteConfig, reserved []string) (templateDirs, directiveDirs []string, warnings []engine.ValidationWarning) {
	disabled := make(map[string]bool, len(cfg.Plugins.Disabled))
	for _, name := range cfg.Plugins.Disabled {
		disabled[name] = true
	}
	reservedSet := make(map[string]bool, len(reserved))
	for _, name := range reserved {
		reservedSet[name] = true
	}

	for _, dir := range DiscoverDirs(projectDir) {
		slug := filepath.Base(dir)
		manifestRef := consts.DirPlugins + "/" + slug + "/" + consts.FilePluginManifest
		if disabled[slug] {
			continue
		}

		m, err := LoadManifest(dir)
		if err != nil {
			warnings = append(warnings, pluginWarning(manifestRef, err.Error()))
			continue
		}
		if err := m.Validate(slug); err != nil {
			warnings = append(warnings, pluginWarning(manifestRef, err.Error()))
			continue
		}
		if reservedSet[slug] {
			warnings = append(warnings, pluginWarning(manifestRef,
				fmt.Sprintf("slug %q conflicts with a built-in plugin name; rename the plugin directory and slug", slug)))
			continue
		}
		if m.Premium {
			if err := license.VerifyFor(projectDir, slug, m.Version); err != nil {
				msg := fmt.Sprintf("premium plugin %q disabled: %v. Install a license with 'sarde license install'", slug, err)
				if m.PurchaseURL != "" {
					msg += " (purchase at " + m.PurchaseURL + ")"
				}
				warnings = append(warnings, pluginWarning(manifestRef, msg))
				continue
			}
		}

		defaults, err := LoadDefaults(dir)
		if err != nil {
			warnings = append(warnings, pluginWarning(manifestRef, err.Error()))
			continue
		}

		mgr.Register(newExternalPlugin(m, dir, mergeConfig(defaults, cfg.Plugins.Config[slug])))

		tplDir := filepath.Join(dir, dirTemplates)
		if info, err := os.Stat(tplDir); err == nil && info.IsDir() {
			templateDirs = append(templateDirs, tplDir)
		}
		if dirDir, ok := directivesDirOf(dir); ok {
			directiveDirs = append(directiveDirs, dirDir)
		}
	}
	warnings = append(warnings, templateCollisionWarnings(templateDirs)...)
	warnings = append(warnings, directiveCollisionWarnings(directiveDirs)...)
	return templateDirs, directiveDirs, warnings
}

// templateCollisionWarnings flags template filenames claimed by more than one
// plugin. Registration order is the sorted slug order, so the last-listed
// plugin wins; the warning makes that shadowing visible. Plugin templates
// shadowing embedded defaults is intentional overriding and is not flagged.
func templateCollisionWarnings(templateDirs []string) []engine.ValidationWarning {
	kinds := []string{consts.DirPartials, consts.DirComponents, consts.DirShortcodes}
	claimed := make(map[string][]string) // "kind/name.html" -> plugin slugs, in registration order
	for _, tplDir := range templateDirs {
		slug := filepath.Base(filepath.Dir(tplDir))
		for _, kind := range kinds {
			entries, err := os.ReadDir(filepath.Join(tplDir, kind))
			if err != nil {
				continue
			}
			for _, e := range entries {
				if e.IsDir() || !strings.HasSuffix(e.Name(), ".html") {
					continue
				}
				key := kind + "/" + e.Name()
				claimed[key] = append(claimed[key], slug)
			}
		}
	}

	keys := make([]string, 0, len(claimed))
	for key, slugs := range claimed {
		if len(slugs) > 1 {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	var warnings []engine.ValidationWarning
	for _, key := range keys {
		slugs := claimed[key]
		warnings = append(warnings, pluginWarning(consts.DirPlugins,
			fmt.Sprintf("template %s is provided by multiple plugins (%s); %s wins",
				key, strings.Join(slugs, ", "), slugs[len(slugs)-1])))
	}
	return warnings
}

// DirectiveDirs returns the directives/ directories of every plugin present
// under {projectDir}/plugins, regardless of enablement or licensing. It is
// the lightweight discovery used by CLI tools that do not resolve full
// config; LoadAll applies the disabled-list and license gating for builds.
func DirectiveDirs(projectDir string) []string {
	var dirs []string
	for _, dir := range DiscoverDirs(projectDir) {
		if dirDir, ok := directivesDirOf(dir); ok {
			dirs = append(dirs, dirDir)
		}
	}
	return dirs
}

// directivesDirOf reports the directives/ subdirectory of one plugin dir, if
// present.
func directivesDirOf(pluginDir string) (string, bool) {
	d := filepath.Join(pluginDir, dirDirectives)
	if info, err := os.Stat(d); err == nil && info.IsDir() {
		return d, true
	}
	return "", false
}

// directiveCollisionWarnings flags directive names claimed by more than one
// plugin. Directive dirs are loaded in the sorted slug order, so the
// last-listed plugin wins; the warning makes that shadowing visible. Site or
// theme directives shadowing plugin ones is intentional overriding and is not
// flagged.
func directiveCollisionWarnings(directiveDirs []string) []engine.ValidationWarning {
	claimed := make(map[string][]string) // "name.yaml" -> plugin slugs, in load order
	for _, dirDir := range directiveDirs {
		slug := filepath.Base(filepath.Dir(dirDir))
		entries, err := os.ReadDir(dirDir)
		if err != nil {
			continue
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
				continue
			}
			claimed[e.Name()] = append(claimed[e.Name()], slug)
		}
	}

	keys := make([]string, 0, len(claimed))
	for key, slugs := range claimed {
		if len(slugs) > 1 {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	var warnings []engine.ValidationWarning
	for _, key := range keys {
		slugs := claimed[key]
		warnings = append(warnings, pluginWarning(consts.DirPlugins,
			fmt.Sprintf("directive %s is provided by multiple plugins (%s); %s wins",
				key, strings.Join(slugs, ", "), slugs[len(slugs)-1])))
	}
	return warnings
}

// newExternalPlugin synthesizes the hook functions for one manifest:
// BeforeRender appends asset URLs when the inject condition matches,
// BuildDone copies the assets/ tree to dist. URLs are root-relative;
// resolveRouteAssets applies basePath afterward.
func newExternalPlugin(m *Manifest, dir string, cfg map[string]any) *plugin.Plugin {
	prefix := m.EffectivePrefix()
	urlPrefix := "/" + prefix
	always := cfgutil.Bool(cfg, "always", false)
	inject := m.Inject

	return &plugin.Plugin{
		Name: m.Slug,
		Hooks: plugin.PluginHooks{
			BeforeRender: func(ctx *plugin.BeforeRenderContext) error {
				if !inject.HasAssets() {
					return nil
				}
				if !always && !injectMatches(&inject, ctx.Page, ctx.RouteData) {
					return nil
				}
				rd := ctx.RouteData
				for _, s := range inject.Styles {
					rd.Styles = cfgutil.AppendUnique(rd.Styles, urlPrefix+s)
				}
				for _, s := range inject.Scripts {
					rd.Scripts = cfgutil.AppendUnique(rd.Scripts, urlPrefix+s)
				}
				for _, s := range inject.ModuleScripts {
					rd.ModuleScripts = cfgutil.AppendUnique(rd.ModuleScripts, urlPrefix+s)
				}
				return nil
			},
			BuildDone: func(ctx *plugin.BuildDoneContext) error {
				if ctx.Incremental {
					return nil
				}
				assetsDir := filepath.Join(dir, consts.DirAssets)
				if info, err := os.Stat(assetsDir); err != nil || !info.IsDir() {
					return nil
				}
				return plugin.WriteFSTreeFiltered(ctx, os.DirFS(assetsDir), ".", prefix, m.IncludeFilter())
			},
		},
	}
}

func pluginWarning(file, msg string) engine.ValidationWarning {
	return engine.ValidationWarning{File: file, Field: "plugin", Message: msg, Level: "warning"}
}

// mergeConfig overlays user config on top of plugin-shipped defaults.
// Deprecated camelCase spellings of snake_case default keys are accepted and
// re-keyed to the canonical name first.
func mergeConfig(defaults, userCfg map[string]any) map[string]any {
	resolved, _ := cfgutil.ResolveAliases(defaults, userCfg)
	merged := make(map[string]any, len(defaults)+len(resolved))
	for k, v := range defaults {
		merged[k] = v
	}
	for k, v := range resolved {
		merged[k] = v
	}
	return merged
}
