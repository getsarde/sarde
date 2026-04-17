package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init [path]",
	Short: "Create a new Coderoo site",
	Long:  "Scaffold a new Coderoo site with minimal starter files.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runInit,
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	targetDir := "."
	if len(args) > 0 {
		targetDir = args[0]
	}

	absDir, err := filepath.Abs(targetDir)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	// Check if site already exists.
	if _, err := os.Stat(filepath.Join(absDir, "site.yaml")); err == nil {
		return fmt.Errorf("site.yaml already exists in %s", absDir)
	}

	// Create directories.
	dirs := []string{
		filepath.Join(absDir, "content"),
		filepath.Join(absDir, "static"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("creating directory %s: %w", d, err)
		}
	}

	// Write site.yaml.
	siteYAML := `site:
  title: "My Site"
  url: "http://localhost:4727"
`
	if err := os.WriteFile(filepath.Join(absDir, "site.yaml"), []byte(siteYAML), 0o644); err != nil {
		return fmt.Errorf("writing site.yaml: %w", err)
	}

	// Write content/_index.md.
	indexMD := `---
title: Welcome
---

# Welcome to your new site

Edit this page at ` + "`content/_index.md`" + `, then run ` + "`coderoo serve`" + ` to see your changes.
`
	if err := os.WriteFile(filepath.Join(absDir, "content", "_index.md"), []byte(indexMD), 0o644); err != nil {
		return fmt.Errorf("writing _index.md: %w", err)
	}

	// Write static/.gitkeep so git tracks the otherwise-empty directory.
	if err := os.WriteFile(filepath.Join(absDir, "static", ".gitkeep"), []byte(""), 0o644); err != nil {
		return fmt.Errorf("writing static/.gitkeep: %w", err)
	}

	// Write .gitignore.
	gitignore := `dist/
.cache/
`
	if err := os.WriteFile(filepath.Join(absDir, ".gitignore"), []byte(gitignore), 0o644); err != nil {
		return fmt.Errorf("writing .gitignore: %w", err)
	}

	quiet, _ := cmd.Flags().GetBool("quiet")
	if !quiet {
		fmt.Printf("Created new site at %s\n", absDir)
		fmt.Println("  Run 'coderoo serve' to start the dev server.")
	}

	return nil
}
