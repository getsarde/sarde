package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/content"
	"github.com/getsarde/sarde/internal/directive"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/getsarde/sarde/internal/plugin/external"
	sardetemplate "github.com/getsarde/sarde/internal/template"
	"github.com/spf13/cobra"
)

var directivesCmd = &cobra.Command{
	Use:   "directives [path]",
	Short: "Print the block directive catalog",
	Long: "Print the catalog of ::: block directives Sarde recognizes, grouped by category, with syntax templates and key fields. Used by Sarde Studio to offer a directive picker.\n\n" +
		"With a project path (or when run inside a project), generic directives from the project's directives/ folder are merged in with source \"site\", and directives shipped by plugins under plugins/<slug>/directives/ with source \"plugin:<slug>\". Theme-provided directives are resolved at build time only and are not listed here. This command does not resolve full config, so plugins.disabled and premium license checks are not applied; builds enforce both.",
	Args:         cobra.MaximumNArgs(1),
	SilenceUsage: true,
	RunE:         runDirectives,
}

func init() {
	directivesCmd.Flags().String("format", "pretty", "Output format: pretty, json")
	directivesCmd.Flags().Bool("check", false, "Validate directive definitions only: print warnings and exit 1 if any")
	rootCmd.AddCommand(directivesCmd)
}

// loadProjectDirectives builds a lightweight registry from plugin directive
// folders and the project's directives/ folder (site wins), without full
// config resolution. Templates parse against a stub FuncMap: function names
// must resolve at parse time, but nothing is executed here.
func loadProjectDirectives(projectDir string) (*directive.Registry, []engine.ValidationWarning) {
	var site *engine.SiteContext
	var pageIndex *content.PageIndex
	funcMap := sardetemplate.BuildShortcodeFuncMap(sardetemplate.ShortcodeFuncMapConfig{
		Site:      &site,
		PageIndex: &pageIndex,
	})

	reg := directive.NewRegistry(funcMap)
	var warnings []engine.ValidationWarning
	for _, dir := range external.DirectiveDirs(projectDir) {
		source := "plugin:" + filepath.Base(filepath.Dir(dir))
		warnings = append(warnings, reg.LoadDir(dir, source)...)
	}
	warnings = append(warnings, reg.LoadDir(filepath.Join(projectDir, consts.DirDirectives), "site")...)
	if cat, err := engine.LoadDirectiveCatalog(); err == nil {
		warnings = append(warnings, reg.ValidateAgainstBuiltins(cat)...)
	}
	return reg, warnings
}

func runDirectives(cmd *cobra.Command, args []string) error {
	projectDir := projectDirFromArgs(args)
	reg, warnings := loadProjectDirectives(projectDir)

	if check, _ := cmd.Flags().GetBool("check"); check {
		for _, w := range warnings {
			fmt.Fprintf(os.Stderr, "warning: %s: %s\n", w.File, w.Message)
		}
		if len(warnings) > 0 {
			return fmt.Errorf("%d directive problem(s) found", len(warnings))
		}
		fmt.Printf("All directive definitions are valid (%d loaded).\n", len(reg.Names()))
		return nil
	}

	base, err := engine.LoadDirectiveCatalog()
	if err != nil {
		return err
	}
	cat := directive.MergeCatalog(base, reg)

	for _, w := range warnings {
		fmt.Fprintf(os.Stderr, "warning: %s: %s\n", w.File, w.Message)
	}

	format, _ := cmd.Flags().GetString("format")
	switch format {
	case "json":
		return json.NewEncoder(os.Stdout).Encode(cat)
	case "pretty":
		printDirectiveCatalog(cat)
		return nil
	default:
		return fmt.Errorf("unknown format %q (expected pretty or json)", format)
	}
}

func printDirectiveCatalog(cat *engine.DirectiveCatalog) {
	for _, c := range cat.Categories {
		fmt.Printf("%s\n", c.Label)
		for _, d := range c.Directives {
			fmt.Printf("  %-18s %s\n", d.Name, d.Description)
		}
		fmt.Println()
	}
}
