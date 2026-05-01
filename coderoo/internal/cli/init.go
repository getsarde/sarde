package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/coderoo-dev/coderoo/internal/consts"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init [path]",
	Short: "Create a new Coderoo site",
	Long:  "Scaffold a new Coderoo site with starter files, example content, and a discoverable config.",
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

	if _, err := os.Stat(filepath.Join(absDir, consts.FileSiteConfig)); err == nil {
		return fmt.Errorf("%s already exists in %s", consts.FileSiteConfig, absDir)
	}

	dirs := []string{
		filepath.Join(absDir, consts.DirContent),
		filepath.Join(absDir, consts.DirContent, "blog"),
		filepath.Join(absDir, consts.DirContent, "docs"),
		filepath.Join(absDir, "static"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("creating directory %s: %w", d, err)
		}
	}

	files := map[string]string{
		consts.FileSiteConfig: siteYAMLContent,
		filepath.Join(consts.DirContent, "_index.md"):                    indexMDContent,
		filepath.Join(consts.DirContent, "blog", "hello-world.md"):      blogPostContent,
		filepath.Join(consts.DirContent, "docs", "getting-started.md"):  docsPageContent,
		filepath.Join("static", ".gitkeep"):                              "",
		".gitignore":                                                     "dist/\n.cache/\n",
	}

	for relPath, content := range files {
		fullPath := filepath.Join(absDir, relPath)
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("writing %s: %w", relPath, err)
		}
	}

	quiet, _ := cmd.Flags().GetBool("quiet")
	if !quiet {
		fmt.Printf("Created new site at %s\n", absDir)
		fmt.Println("  Run 'coderoo serve' to start the dev server.")
	}

	return nil
}

const siteYAMLContent = `site:
  title: "My Site"
  description: "A site built with Coderoo"
  url: "http://localhost:4727"

theme:
  name: "default"
  # preset: "ocean"        # Try: ocean, forest, rose, clean, minimal, docs, academic
  # overrides:
  #   primary: "#6366f1"

# collections:
#   blog:
#     title: "Blog"
#     sort: "date"
#     feed: true
#   docs:
#     title: "Documentation"
#     sort: "weight"
`

const indexMDContent = `---
title: Welcome
---

# Welcome to your new site

Edit this page at ` + "`content/_index.md`" + `, then run ` + "`coderoo serve`" + ` to see your changes.
`

const blogPostContent = `---
title: Hello World
date: 2024-01-01
tags: [getting-started]
---

This is your first blog post. Edit it or create new posts in ` + "`content/blog/`" + `.
`

const docsPageContent = `---
title: Getting Started
weight: 1
---

Welcome to the documentation. Add more pages to ` + "`content/docs/`" + ` and they will appear in the sidebar automatically.

## Next Steps

- Edit ` + "`site.yaml`" + ` to customize your site
- Run ` + "`coderoo serve`" + ` to start the dev server
- Run ` + "`coderoo build`" + ` to generate the static site
`
