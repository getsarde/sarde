package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/content"
	"github.com/getsarde/sarde/internal/i18n"
	"github.com/spf13/cobra"
)

var i18nCodePattern = regexp.MustCompile(`^[a-z]{2,3}(-[a-z0-9]+)*$`)

var i18nCmd = &cobra.Command{
	Use:           "i18n",
	Short:         "Manage internationalization (languages and translations)",
	SilenceUsage:  true,
	SilenceErrors: true,
}

var i18nAddLanguageCmd = &cobra.Command{
	Use:           "add-language <project-dir> <code>",
	Short:         "Add a language to the project",
	Args:          cobra.ExactArgs(2),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runI18nAddLanguage,
}

var i18nRemoveLanguageCmd = &cobra.Command{
	Use:           "remove-language <project-dir> <code>",
	Short:         "Remove a language from the project",
	Args:          cobra.ExactArgs(2),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runI18nRemoveLanguage,
}

var i18nScaffoldCmd = &cobra.Command{
	Use:           "scaffold <project-dir> <code>",
	Short:         "Scaffold content directories and translation file for a language",
	Args:          cobra.ExactArgs(2),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runI18nScaffold,
}

var i18nStatusCmd = &cobra.Command{
	Use:           "status <project-dir>",
	Short:         "Show translation coverage across all languages and collections",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runI18nStatus,
}

var i18nStringsCmd = &cobra.Command{
	Use:           "strings <project-dir>",
	Short:         "List the translation strings of every layer (built-in, theme, project)",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runI18nStrings,
}

func init() {
	i18nAddLanguageCmd.Flags().String("name", "", "Display name for the language")
	i18nAddLanguageCmd.Flags().Int("weight", 0, "Sort weight (lower sorts first)")
	i18nAddLanguageCmd.Flags().String("dir", "ltr", "Text direction: ltr or rtl")

	i18nRemoveLanguageCmd.Flags().Bool("delete-content", false, "Also delete content/<code>/")

	i18nCmd.AddCommand(i18nAddLanguageCmd)
	i18nCmd.AddCommand(i18nRemoveLanguageCmd)
	i18nCmd.AddCommand(i18nScaffoldCmd)
	i18nCmd.AddCommand(i18nStatusCmd)
	i18nCmd.AddCommand(i18nStringsCmd)
	rootCmd.AddCommand(i18nCmd)
}

func runI18nAddLanguage(cmd *cobra.Command, args []string) error {
	projectDir := projectDirFromArgs(args)
	code := strings.ToLower(strings.TrimSpace(args[1]))
	if !i18nCodePattern.MatchString(code) {
		return printJSONError(fmt.Errorf("invalid language code %q", code))
	}

	name, _ := cmd.Flags().GetString("name")
	weight, _ := cmd.Flags().GetInt("weight")
	dir, _ := cmd.Flags().GetString("dir")

	siteYAMLPath := filepath.Join(projectDir, consts.FileSiteConfig)
	raw, err := i18nReadRawYAML(siteYAMLPath)
	if err != nil {
		return printJSONError(err)
	}

	i18nSection := i18nEnsureMapSection(raw, "i18n")
	languages := i18nEnsureMapSection(i18nSection, "languages")
	wasEmpty := len(languages) == 0

	entry := map[string]any{}
	if name != "" {
		entry["name"] = name
	}
	if weight != 0 {
		entry["weight"] = weight
	}
	if dir != "" && dir != "ltr" {
		entry["dir"] = dir
	}
	languages[code] = entry

	defaultLang, _ := i18nSection["default_language"].(string)
	if wasEmpty && defaultLang == "" {
		defaultLang = "en"
		if siteSection, ok := raw["site"].(map[string]any); ok {
			if lang, ok := siteSection["language"].(string); ok && lang != "" {
				defaultLang = lang
			}
		}
		i18nSection["default_language"] = defaultLang
	}

	if err := i18nWriteRawYAML(siteYAMLPath, raw); err != nil {
		return printJSONError(err)
	}

	printJSONResult(map[string]any{
		"ok":              true,
		"code":            code,
		"defaultLanguage": defaultLang,
	})
	return nil
}

func runI18nRemoveLanguage(cmd *cobra.Command, args []string) error {
	projectDir := projectDirFromArgs(args)
	code := strings.ToLower(strings.TrimSpace(args[1]))
	deleteContent, _ := cmd.Flags().GetBool("delete-content")

	siteYAMLPath := filepath.Join(projectDir, consts.FileSiteConfig)
	raw, err := i18nReadRawYAML(siteYAMLPath)
	if err != nil {
		return printJSONError(err)
	}

	i18nSection := i18nEnsureMapSection(raw, "i18n")
	if defaultLang, _ := i18nSection["default_language"].(string); defaultLang == code {
		return printJSONError(fmt.Errorf("cannot remove default language %q", code))
	}

	languages := i18nEnsureMapSection(i18nSection, "languages")
	if _, ok := languages[code]; !ok {
		return printJSONError(fmt.Errorf("language %q is not configured", code))
	}
	delete(languages, code)

	if err := i18nWriteRawYAML(siteYAMLPath, raw); err != nil {
		return printJSONError(err)
	}

	if deleteContent {
		contentDir := filepath.Join(projectDir, consts.DirContent)
		if err := os.RemoveAll(filepath.Join(contentDir, code)); err != nil {
			return printJSONError(fmt.Errorf("removing content directory: %w", err))
		}
	}

	printJSONResult(map[string]any{"ok": true, "code": code})
	return nil
}

// i18nSite is what the scaffold and status commands read from a project: the
// resolved config, its content directory, and the language codes.
type i18nSite struct {
	contentDir  string
	defaultLang string
	languages   map[string]bool // every configured code, the default included
}

func loadI18nSite(projectDir string) (*i18nSite, error) {
	cfg, err := config.Resolve(config.ResolveOptions{
		ConfigPath: filepath.Join(projectDir, consts.FileSiteConfig),
		EnvPrefix:  "SARDE",
	})
	if err != nil {
		return nil, err
	}
	contentDir := filepath.Join(projectDir, consts.DirContent)
	if dir := cfg.Content.Dir; dir != "" {
		if filepath.IsAbs(dir) {
			contentDir = dir
		} else {
			contentDir = filepath.Join(projectDir, dir)
		}
	}
	site := &i18nSite{
		contentDir:  contentDir,
		defaultLang: cfg.I18n.GetDefaultLanguage(),
		languages:   map[string]bool{},
	}
	for _, code := range cfg.I18n.LanguageCodes() {
		site.languages[code] = true
	}
	site.languages[site.defaultLang] = true
	return site, nil
}

// collections lists the default language's collections the way the build
// sees them: every content folder that is not ignored and is not another
// language's folder. A folder named after the default language is not a
// language folder to the build, which reads it as a collection, so it is
// listed (and the caller warns).
func (s *i18nSite) collections() ([]string, error) {
	entries, err := os.ReadDir(s.contentDir)
	if err != nil {
		return nil, fmt.Errorf("reading content directory: %w", err)
	}
	var cols []string
	for _, e := range entries {
		name := e.Name()
		if !e.IsDir() || content.IsIgnoredDirName(name) {
			continue
		}
		if s.languages[name] && name != s.defaultLang {
			continue
		}
		cols = append(cols, name)
	}
	sort.Strings(cols)
	return cols, nil
}

// defaultLangFolderWarning is printed to stderr when content/<default>/
// exists: the default language has no folder, so the build reads it as a
// collection named after the language and never pairs its pages.
func (s *i18nSite) defaultLangFolderWarning(cmd *cobra.Command) {
	if fi, err := os.Stat(filepath.Join(s.contentDir, s.defaultLang)); err == nil && fi.IsDir() {
		fmt.Fprintf(cmd.ErrOrStderr(),
			"warning: the default language has no folder; content/%s/ is read as a collection named %q. Move its pages up to content/ to pair them with their translations.\n",
			s.defaultLang, s.defaultLang)
	}
}

func runI18nScaffold(cmd *cobra.Command, args []string) error {
	projectDir := projectDirFromArgs(args)
	code := strings.ToLower(strings.TrimSpace(args[1]))
	if !i18nCodePattern.MatchString(code) {
		return printJSONError(fmt.Errorf("invalid language code %q", code))
	}

	site, err := loadI18nSite(projectDir)
	if err != nil {
		return printJSONError(err)
	}
	if code == site.defaultLang {
		return printJSONError(fmt.Errorf("%q is the default language, which has no folder of its own", code))
	}
	site.defaultLangFolderWarning(cmd)
	cols, err := site.collections()
	if err != nil {
		return printJSONError(err)
	}

	targetRoot := filepath.Join(site.contentDir, code)
	scaffolded := 0
	for _, col := range cols {
		colDir := filepath.Join(targetRoot, col)
		if err := os.MkdirAll(colDir, 0o755); err != nil {
			return printJSONError(fmt.Errorf("creating collection directory: %w", err))
		}
		indexPath := filepath.Join(colDir, "_index.md")
		if _, err := os.Stat(indexPath); os.IsNotExist(err) {
			title := content.FilenameToTitle(col)
			stub := fmt.Sprintf("---\ntitle: %q\n---\n", title)
			if err := os.WriteFile(indexPath, []byte(stub), 0o644); err != nil {
				return printJSONError(fmt.Errorf("writing index stub: %w", err))
			}
		}
		scaffolded++
	}

	// Seed i18n/<code>.yaml from the default language's file, or create empty.
	i18nDir := filepath.Join(projectDir, consts.DirI18n)
	if err := os.MkdirAll(i18nDir, 0o755); err != nil {
		return printJSONError(fmt.Errorf("creating i18n directory: %w", err))
	}
	targetFile := filepath.Join(i18nDir, code+".yaml")
	if _, err := os.Stat(targetFile); os.IsNotExist(err) {
		defaultFile := filepath.Join(i18nDir, site.defaultLang+".yaml")
		data, rerr := os.ReadFile(defaultFile)
		if rerr != nil {
			data = []byte{}
		}
		if werr := os.WriteFile(targetFile, data, 0o644); werr != nil {
			return printJSONError(fmt.Errorf("writing i18n file: %w", werr))
		}
	}

	printJSONResult(map[string]any{
		"ok":                    true,
		"code":                  code,
		"collectionsScaffolded": scaffolded,
	})
	return nil
}

// runI18nStatus counts, per collection and language, the default-language
// pages and how many of them have a translation. Pages pair by their path
// without the language folder, as the build pairs them, so a translated
// page with no default-language source does not count.
func runI18nStatus(cmd *cobra.Command, args []string) error {
	projectDir := projectDirFromArgs(args)
	site, err := loadI18nSite(projectDir)
	if err != nil {
		return printJSONError(err)
	}
	site.defaultLangFolderWarning(cmd)
	cols, err := site.collections()
	if err != nil {
		return printJSONError(err)
	}

	scanner := &content.Scanner{Languages: site.languages, DefaultLang: site.defaultLang}
	files, err := scanner.DiscoverFiles(site.contentDir)
	if err != nil {
		return printJSONError(fmt.Errorf("reading content directory: %w", err))
	}
	// Default-language pages per collection, keyed by their pairing path.
	sources := make(map[string]map[string]bool, len(cols))
	// Translated pairing paths per language.
	translations := make(map[string]map[string]bool)
	for _, f := range files {
		if f.Lang == site.defaultLang {
			if sources[f.CollectionName] == nil {
				sources[f.CollectionName] = map[string]bool{}
			}
			sources[f.CollectionName][f.LangRelPath] = true
			continue
		}
		if translations[f.Lang] == nil {
			translations[f.Lang] = map[string]bool{}
		}
		translations[f.Lang][f.LangRelPath] = true
	}

	status := make(map[string]map[string]map[string]int, len(cols))
	for _, col := range cols {
		total := len(sources[col])
		perLang := map[string]map[string]int{
			site.defaultLang: {"total": total, "translated": total},
		}
		for lang := range site.languages {
			if lang == site.defaultLang {
				continue
			}
			translated := 0
			for rel := range sources[col] {
				if translations[lang][rel] {
					translated++
				}
			}
			perLang[lang] = map[string]int{"total": total, "translated": translated}
		}
		status[col] = perLang
	}

	printJSONResult(map[string]any{"ok": true, "status": status})
	return nil
}

// runI18nStrings prints the string layers unmerged, in load order, so an
// editor can show where each value comes from. A file that fails to parse is
// listed under "problems" instead of failing the command.
func runI18nStrings(cmd *cobra.Command, args []string) error {
	projectDir := projectDirFromArgs(args)

	cfg, err := config.Resolve(config.ResolveOptions{
		ConfigPath: filepath.Join(projectDir, consts.FileSiteConfig),
		EnvPrefix:  "SARDE",
	})
	if err != nil {
		return printJSONError(err)
	}

	layers, problems := i18n.LoadLayers(embedded.I18nFS(), projectDir, cfg.Theme.Name)
	if problems == nil {
		problems = []i18n.FileProblem{}
	}
	languages := cfg.I18n.LanguageCodes()
	defaultLang := cfg.I18n.GetDefaultLanguage()
	if len(languages) == 0 {
		languages = []string{defaultLang}
	}

	printJSONResult(map[string]any{
		"ok":              true,
		"defaultLanguage": defaultLang,
		"languages":       languages,
		"layers":          layers,
		"problems":        problems,
	})
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// printJSONResult marshals v and writes it to stdout as the command's
// structured success output.
func printJSONResult(v any) {
	data, err := json.Marshal(v)
	if err != nil {
		fmt.Printf("{\"error\":%q}\n", err.Error())
		return
	}
	fmt.Println(string(data))
}

// printJSONError writes a structured {"error": "..."} object to stdout (so
// the Tauri sidecar bridge can extract a message regardless of stderr
// formatting) and returns the error so the process exits non-zero.
func printJSONError(err error) error {
	fmt.Printf("{\"error\":%q}\n", err.Error())
	return err
}

// i18nReadRawYAML reads and unmarshals a YAML file into a generic map,
// treating a missing/unparseable file as an empty document.
func i18nReadRawYAML(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return make(map[string]any), nil
	}
	var raw map[string]any
	if err := yaml.Unmarshal(data, &raw); err != nil || raw == nil {
		raw = make(map[string]any)
	}
	return raw, nil
}

// i18nWriteRawYAML marshals and writes a generic map back to a YAML file.
func i18nWriteRawYAML(path string, raw map[string]any) error {
	out, err := yaml.Marshal(raw)
	if err != nil {
		return fmt.Errorf("marshaling %s: %w", filepath.Base(path), err)
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", filepath.Base(path), err)
	}
	return nil
}

// i18nEnsureMapSection returns raw[key] as a map[string]any, creating and
// inserting one if absent or of an unexpected type.
func i18nEnsureMapSection(raw map[string]any, key string) map[string]any {
	section, ok := raw[key].(map[string]any)
	if !ok {
		section = make(map[string]any)
		raw[key] = section
	}
	return section
}
