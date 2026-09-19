package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"

	"github.com/getsarde/sarde/internal/collection"
	"github.com/spf13/cobra"
)

var schemaCmd = &cobra.Command{
	Use:   "schema <collection> [project-dir]",
	Short: "Show a collection's resolved frontmatter schema",
	Long: "Prints the fully resolved frontmatter schema for one collection: every catalog field " +
		"applicable to its effective layout and inferred type, merged with the collection's custom " +
		"frontmatter_schema from content/<collection>/config.yaml. Each field is labeled with its " +
		"source (\"catalog\" or \"custom\"). Powers Sarde Studio's schema editor prefill.",
	Args:         cobra.RangeArgs(1, 2),
	SilenceUsage: true,
	RunE:         runSchema,
}

func init() {
	schemaCmd.Flags().String("format", "pretty", "Output format: pretty, json")
	rootCmd.AddCommand(schemaCmd)
}

func runSchema(cmd *cobra.Command, args []string) error {
	format, _ := cmd.Flags().GetString("format")
	if format != "json" && format != "pretty" {
		return fmt.Errorf("unknown format %q (expected pretty or json)", format)
	}

	name := args[0]
	projectDir := projectDirFromArgs(args[1:])

	resolved, err := collection.ResolveSchema(projectDir, name)
	if err != nil {
		if format == "json" {
			return printJSONError(err)
		}
		return err
	}

	if format == "json" {
		return json.NewEncoder(os.Stdout).Encode(resolved)
	}
	printResolvedSchema(resolved)
	return nil
}

func printResolvedSchema(rs *collection.ResolvedSchema) {
	fmt.Printf("%s  (type %s, layout %s)\n", rs.Collection, rs.Type, rs.Layout)
	names := make([]string, 0, len(rs.Fields))
	for n := range rs.Fields {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, source := range []string{collection.SchemaSourceCustom, collection.SchemaSourceCatalog} {
		for _, n := range names {
			f := rs.Fields[n]
			if f.Source != source {
				continue
			}
			fmt.Printf("  %-24s %-10s [%s]\n", n, f.Type, f.Source)
			printChildFields(f.Fields, "    ")
		}
	}
}

func printChildFields(fields map[string]collection.ResolvedField, indent string) {
	names := make([]string, 0, len(fields))
	for n := range fields {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		fmt.Printf("%s%-22s %s\n", indent, n, fields[n].Type)
	}
}
