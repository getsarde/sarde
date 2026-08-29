package cli

import (
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/theme"
	"github.com/spf13/cobra"
)

var themeCmd = &cobra.Command{
	Use:   "theme",
	Short: "Manage themes",
}

var themeEjectCmd = &cobra.Command{
	Use:   "eject [path...]",
	Short: "Copy embedded theme files into themes/<name>/ for customization",
	Long: `Copy files from the embedded default theme into themes/<name>/ (default: themes/default/).

With no arguments the whole theme is copied. With paths, only those files or
directories are copied and everything else keeps falling back to the embedded
theme. Paths use the ejected layout, for example:

  layouts/_blog/single.html   one template
  layouts/components          every component
  css/blog.css                one stylesheet
  assets                      fonts, vendor scripts, and JS (whole directory only)

theme.yaml is copied too when the destination has none, so the bundled presets
stay selectable. Run "sarde theme eject --list" to see every ejectable path.`,
	Args: cobra.ArbitraryArgs,
	RunE: runThemeEject,
}

var themeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available themes",
	Long:  "List all available themes: the built-in default theme and any themes installed in themes/.",
	RunE:  runThemeList,
}

func init() {
	themeEjectCmd.Flags().Bool("force", false, "Overwrite existing files (or the whole themes/<name>/ directory when no paths are given)")
	themeEjectCmd.Flags().String("name", "default", "Destination theme directory under themes/")
	themeEjectCmd.Flags().Bool("list", false, "List every ejectable path and exit")
	themeCmd.AddCommand(themeEjectCmd)
	themeCmd.AddCommand(themeListCmd)
	themeCmd.AddCommand(themeAddCmd)
	themeCmd.AddCommand(themeRemoveCmd)
	themeCmd.AddCommand(themeInfoCmd)
	rootCmd.AddCommand(themeCmd)
}

func runThemeEject(cmd *cobra.Command, args []string) error {
	all, err := ejectablePaths()
	if err != nil {
		return fmt.Errorf("reading embedded theme: %w", err)
	}

	if list, _ := cmd.Flags().GetBool("list"); list {
		for _, p := range all {
			fmt.Println(p)
		}
		return nil
	}

	name, _ := cmd.Flags().GetString("name")
	if !validThemeName(name) {
		return fmt.Errorf("invalid theme name %q: must be a single directory name", name)
	}
	force, _ := cmd.Flags().GetBool("force")
	quiet, _ := cmd.Flags().GetBool("quiet")

	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}
	destDir := filepath.Join(projectDir, consts.DirThemes, name)
	destRel := consts.DirThemes + "/" + name + "/"

	// No paths: copy the whole theme with directory-level force semantics.
	if len(args) == 0 {
		if _, err := os.Stat(destDir); err == nil {
			if !force {
				return fmt.Errorf("%s already exists; use --force to overwrite", destRel)
			}
			if err := os.RemoveAll(destDir); err != nil {
				return fmt.Errorf("removing existing %s: %w", destRel, err)
			}
		}
		if err := copyEmbeddedFiles(destDir, all); err != nil {
			return fmt.Errorf("ejecting theme: %w", err)
		}
		if err := setThemeSlug(destDir, name); err != nil {
			return err
		}
		if !quiet {
			fmt.Printf("Ejected default theme to %s (%d files)\n", destRel, len(all))
		}
		return nil
	}

	targets, err := resolveEjectTargets(args, all)
	if err != nil {
		return err
	}

	// A theme without theme.yaml is invisible to `theme list` and loses the
	// bundled presets, so ship it whenever the destination has none.
	if _, err := os.Stat(filepath.Join(destDir, consts.FileThemeConfig)); err != nil &&
		!slices.Contains(targets, consts.FileThemeConfig) {
		targets = append(targets, consts.FileThemeConfig)
		sort.Strings(targets)
	}

	if !force {
		var conflicts []string
		for _, rel := range targets {
			if _, err := os.Stat(filepath.Join(destDir, filepath.FromSlash(rel))); err == nil {
				conflicts = append(conflicts, rel)
			}
		}
		if len(conflicts) > 0 {
			return fmt.Errorf("already present in %s (use --force to overwrite):\n  %s",
				destRel, strings.Join(conflicts, "\n  "))
		}
	}

	if err := copyEmbeddedFiles(destDir, targets); err != nil {
		return fmt.Errorf("ejecting theme files: %w", err)
	}
	if slices.Contains(targets, consts.FileThemeConfig) {
		if err := setThemeSlug(destDir, name); err != nil {
			return err
		}
	}

	if !quiet {
		fmt.Printf("Ejected %d file(s) to %s\n", len(targets), destRel)
		for _, rel := range targets {
			fmt.Printf("  %s\n", rel)
		}
	}
	return nil
}

// ejectablePaths returns every file in the embedded theme as a slash-separated
// path in its ejected location (templates under layouts/), sorted.
func ejectablePaths() ([]string, error) {
	var paths []string
	err := fs.WalkDir(embedded.ThemeFS(), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		paths = append(paths, filepath.ToSlash(remapThemePath(p)))
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	return paths, nil
}

// resolveEjectTargets expands user-supplied paths (files or directories, in
// ejected or embedded form) into the sorted, deduplicated list of ejected-form
// file paths they cover.
func resolveEjectTargets(args, all []string) ([]string, error) {
	seen := make(map[string]bool)
	var out []string
	for _, arg := range args {
		rel := normalizeEjectPath(arg)
		if rel == "" || rel == "." || rel == ".." || strings.HasPrefix(rel, "../") {
			return nil, fmt.Errorf("invalid path %q", arg)
		}
		if strings.HasPrefix(rel, consts.DirAssets+"/") {
			return nil, fmt.Errorf("%s: theme assets replace the embedded fonts, vendor scripts, and JS as a whole; eject the full %q directory instead", arg, consts.DirAssets)
		}
		matched := false
		for _, p := range all {
			if p == rel || strings.HasPrefix(p, rel+"/") {
				matched = true
				if !seen[p] {
					seen[p] = true
					out = append(out, p)
				}
			}
		}
		if !matched {
			return nil, fmt.Errorf("no such file in the embedded theme: %s (run \"sarde theme eject --list\")", arg)
		}
	}
	sort.Strings(out)
	return out, nil
}

// normalizeEjectPath cleans a user-supplied path and converts the embedded
// form (_blog/single.html) into the ejected form (layouts/_blog/single.html).
func normalizeEjectPath(p string) string {
	p = strings.TrimSpace(strings.ReplaceAll(p, `\`, "/"))
	p = strings.TrimSuffix(p, "/")
	if p == "" {
		return ""
	}
	p = path.Clean(p)
	first := p
	if i := strings.IndexByte(p, '/'); i >= 0 {
		first = p[:i]
	}
	if slices.Contains(themeLayoutDirs, first) {
		return path.Join(consts.DirLayouts, p)
	}
	return p
}

// embeddedPathFor maps an ejected-form path back to its location in the
// embedded theme FS.
func embeddedPathFor(ejected string) string {
	rest, ok := strings.CutPrefix(ejected, consts.DirLayouts+"/")
	if !ok {
		return ejected
	}
	first := rest
	if i := strings.IndexByte(rest, '/'); i >= 0 {
		first = rest[:i]
	}
	if slices.Contains(themeLayoutDirs, first) {
		return rest
	}
	return ejected
}

// copyEmbeddedFiles writes the given ejected-form paths from the embedded
// theme into destDir, creating directories as needed.
func copyEmbeddedFiles(destDir string, paths []string) error {
	efs := embedded.ThemeFS()
	for _, rel := range paths {
		data, err := fs.ReadFile(efs, embeddedPathFor(rel))
		if err != nil {
			return fmt.Errorf("reading embedded %s: %w", rel, err)
		}
		dest := filepath.Join(destDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return fmt.Errorf("creating directory for %s: %w", rel, err)
		}
		if err := os.WriteFile(dest, data, 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", rel, err)
		}
	}
	return nil
}

var themeSlugLine = regexp.MustCompile(`(?m)^slug:.*$`)

// setThemeSlug rewrites the slug line of destDir/theme.yaml to name, keeping
// the rest of the file (including comments) untouched. "default" is left as is.
func setThemeSlug(destDir, name string) error {
	if name == "default" {
		return nil
	}
	p := filepath.Join(destDir, consts.FileThemeConfig)
	data, err := os.ReadFile(p)
	if err != nil {
		return fmt.Errorf("reading %s: %w", consts.FileThemeConfig, err)
	}
	line := []byte("slug: " + name)
	var updated []byte
	if themeSlugLine.Match(data) {
		updated = themeSlugLine.ReplaceAllLiteral(data, line)
	} else {
		updated = append(append(line, '\n'), data...)
	}
	if err := os.WriteFile(p, updated, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", consts.FileThemeConfig, err)
	}
	return nil
}

func runThemeList(cmd *cobra.Command, args []string) error {
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	configPath, _ := cmd.Flags().GetString("config")
	cfg, err := config.Resolve(config.ResolveOptions{ConfigPath: configPath, Strict: true})
	if err != nil {
		cfg = config.Defaults()
	}
	activeTheme := cfg.Theme.Name

	embeddedTheme, _ := theme.LoadFromFS(embedded.ThemeFS(), ".")
	embeddedDesc := ""
	if embeddedTheme != nil && embeddedTheme.Description != "" {
		embeddedDesc = embeddedTheme.Description
	}

	marker := " "
	if activeTheme == "" || activeTheme == "default" {
		marker = "*"
	}
	fmt.Printf("  %s default (embedded)", marker)
	if embeddedDesc != "" {
		fmt.Printf("  %s", embeddedDesc)
	}
	fmt.Println()

	themesDir := filepath.Join(projectDir, consts.DirThemes)
	entries, err := os.ReadDir(themesDir)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		thm, _ := theme.LoadFromDir(filepath.Join(themesDir, name))
		if thm == nil {
			continue
		}
		marker := " "
		if name == activeTheme {
			marker = "*"
		}
		fmt.Printf("  %s %s", marker, name)
		if thm.Description != "" {
			fmt.Printf("  %s", thm.Description)
		}
		fmt.Println()
	}

	return nil
}

// themeLayoutDirs are the embedded-theme root directories that the template,
// component, partial, and shortcode lookups read from themes/<name>/layouts/.
// Eject relocates each of them under layouts/; css/, assets/, and theme.yaml
// stay at the theme root.
var themeLayoutDirs = []string{
	consts.DirDefault, consts.DirDocs, consts.DirBlog, consts.DirSlides,
	consts.DirPresentation, consts.DirLabs, consts.DirTaxonomy,
	consts.DirComponents, consts.DirPartials, consts.DirShortcodes,
}

// remapThemePath maps an embedded theme path to its location inside an
// ejected theme directory. It matches both the bare directory entry (as
// yielded by fs.WalkDir) and paths beneath it, so no empty directory is
// left behind at the theme root.
func remapThemePath(embeddedPath string) string {
	for _, dir := range themeLayoutDirs {
		if embeddedPath == dir || strings.HasPrefix(embeddedPath, dir+"/") {
			return filepath.Join(consts.DirLayouts, embeddedPath)
		}
	}
	return embeddedPath
}
