package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/frostybee/sarde/internal/content"
	"github.com/frostybee/sarde/internal/importer"
	"github.com/spf13/cobra"
)

var importCmd = &cobra.Command{
	Use:   "import",
	Short: "Import content from external sources",
}

var importObsidianCmd = &cobra.Command{
	Use:   "obsidian <vault-path>",
	Short: "Import an Obsidian vault",
	Long:  "Convert an Obsidian vault to Sarde content. Converts wikilinks, callouts, image embeds, and strips comments.",
	Args:  cobra.ExactArgs(1),
	RunE:  runImportObsidian,
}

func init() {
	importObsidianCmd.Flags().StringP("collection", "c", "", "Target collection name (default: vault folder name)")
	importObsidianCmd.Flags().String("content", "content", "Content directory path")
	importCmd.AddCommand(importObsidianCmd)
	rootCmd.AddCommand(importCmd)
}

func runImportObsidian(cmd *cobra.Command, args []string) error {
	vaultPath := args[0]

	// Validate vault path exists.
	info, err := os.Stat(vaultPath)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("vault path %q is not a valid directory", vaultPath)
	}

	collection, _ := cmd.Flags().GetString("collection")
	if collection == "" {
		collection = content.Slugify(filepath.Base(vaultPath))
	}

	contentDir, _ := cmd.Flags().GetString("content")
	if !filepath.IsAbs(contentDir) {
		wd, _ := os.Getwd()
		contentDir = filepath.Join(wd, contentDir)
	}

	quiet, _ := cmd.Flags().GetBool("quiet")

	if !quiet {
		fmt.Printf("Importing Obsidian vault from %s → %s/%s\n", vaultPath, contentDir, collection)
	}

	result, err := importer.ImportObsidian(vaultPath, collection, contentDir)
	if err != nil {
		return fmt.Errorf("import failed: %w", err)
	}

	if !quiet {
		fmt.Printf("Done: %d notes converted, %d images copied, %d links converted",
			result.NotesConverted, result.ImagesCopied, result.LinksConverted)
		if result.ItemsSkipped > 0 {
			fmt.Printf(", %d items skipped", result.ItemsSkipped)
		}
		fmt.Println()
	}

	return nil
}
