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

	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/content"
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

func init() {
	i18nAddLanguageCmd.Flags().String("name", "", "Display name for the language")
	i18nAddLanguageCmd.Flags().Int("weight", 0, "Sort weight (lower sorts first)")
	i18nAddLanguageCmd.Flags().String("dir", "ltr", "Text direction: ltr or rtl")

	i18nRemoveLanguageCmd.Flags().Bool("delete-content", false, "Also delete content/<code>/")

	i18nCmd.AddCommand(i18nAddLanguageCmd)
	i18nCmd.AddCommand(i18nRemoveLanguageCmd)
	i18nCmd.AddCommand(i18nScaffoldCmd)
	i18nCmd.AddCommand(i18nStatusCmd)
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

func runI18nScaffold(cmd *cobra.Command, args []string) error {
	projectDir := projectDirFromArgs(args)
	code := strings.ToLower(strings.TrimSpace(args[1]))
	if !i18nCodePattern.MatchString(code) {
		return printJSONError(fmt.Errorf("invalid language code %q", code))
	}

	siteYAMLPath := filepath.Join(projectDir, consts.FileSiteConfig)
	raw, _ := i18nReadRawYAML(siteYAMLPath)

	defaultLang := "en"
	if i18nSection, ok := raw["i18n"].(map[string]any); ok {
		if dl, ok := i18nSection["default_language"].(string); ok && dl != "" {
			defaultLang = dl
		}
	}

	contentDir := filepath.Join(projectDir, consts.DirContent)

	// Discover collection names from the default-language content tree, if
	// present; otherwise fall back to the content root (pre-i18n layout).
	scanDir := contentDir
	if fi, err := os.Stat(filepath.Join(contentDir, defaultLang)); err == nil && fi.IsDir() {
		scanDir = filepath.Join(contentDir, defaultLang)
	}

	entries, err := os.ReadDir(scanDir)
	if err != nil {
		return printJSONError(fmt.Errorf("reading content directory: %w", err))
	}

	targetRoot := filepath.Join(contentDir, code)
	scaffolded := 0
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") || strings.HasPrefix(e.Name(), "_") {
			continue
		}
		colDir := filepath.Join(targetRoot, e.Name())
		if err := os.MkdirAll(colDir, 0o755); err != nil {
			return printJSONError(fmt.Errorf("creating collection directory: %w", err))
		}
		indexPath := filepath.Join(colDir, "_index.md")
		if _, err := os.Stat(indexPath); os.IsNotExist(err) {
			title := content.FilenameToTitle(e.Name())
			stub := fmt.Sprintf("---\ntitle: %q\n---\n", title)
			if err := os.WriteFile(indexPath, []byte(stub), 0o644); err != nil {
				return printJSONError(fmt.Errorf("writing index stub: %w", err))
			}
		}
		scaffolded++
	}

	// Seed i18n/<code>.yaml from the default language's file, or create empty.
	i18nDir := filepath.Join(projectDir, "i18n")
	if err := os.MkdirAll(i18nDir, 0o755); err != nil {
		return printJSONError(fmt.Errorf("creating i18n directory: %w", err))
	}
	targetFile := filepath.Join(i18nDir, code+".yaml")
	if _, err := os.Stat(targetFile); os.IsNotExist(err) {
		defaultFile := filepath.Join(i18nDir, defaultLang+".yaml")
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

func runI18nStatus(cmd *cobra.Command, args []string) error {
	projectDir := projectDirFromArgs(args)

	siteYAMLPath := filepath.Join(projectDir, consts.FileSiteConfig)
	raw, _ := i18nReadRawYAML(siteYAMLPath)

	defaultLang := "en"
	var languages []string
	if i18nSection, ok := raw["i18n"].(map[string]any); ok {
		if dl, ok := i18nSection["default_language"].(string); ok && dl != "" {
			defaultLang = dl
		}
		if langsMap, ok := i18nSection["languages"].(map[string]any); ok {
			for code := range langsMap {
				languages = append(languages, code)
			}
		}
	}
	sort.Strings(languages)

	allLangs := []string{defaultLang}
	for _, l := range languages {
		if l != defaultLang {
			allLangs = append(allLangs, l)
		}
	}

	contentDir := filepath.Join(projectDir, consts.DirContent)
	defaultDir := contentDir
	if fi, err := os.Stat(filepath.Join(contentDir, defaultLang)); err == nil && fi.IsDir() {
		defaultDir = filepath.Join(contentDir, defaultLang)
	}

	entries, err := os.ReadDir(defaultDir)
	if err != nil {
		return printJSONError(fmt.Errorf("reading content directory: %w", err))
	}

	status := make(map[string]map[string]map[string]int)
	for _, e := range entries {
		if !e.IsDir() || strings.HasPrefix(e.Name(), ".") || strings.HasPrefix(e.Name(), "_") {
			continue
		}
		colName := e.Name()
		total := countMarkdownFilesCLI(filepath.Join(defaultDir, colName))

		perLang := make(map[string]map[string]int, len(allLangs))
		perLang[defaultLang] = map[string]int{"total": total, "translated": total}

		for _, lang := range allLangs {
			if lang == defaultLang {
				continue
			}
			translated := countMarkdownFilesCLI(filepath.Join(contentDir, lang, colName))
			perLang[lang] = map[string]int{"total": total, "translated": translated}
		}
		status[colName] = perLang
	}

	printJSONResult(map[string]any{"ok": true, "status": status})
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

func countMarkdownFilesCLI(dir string) int {
	count := 0
	filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".md") {
			count++
		}
		return nil
	})
	return count
}
