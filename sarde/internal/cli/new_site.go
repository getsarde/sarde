package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/frostybee/sarde/embedded"
	"github.com/frostybee/sarde/internal/consts"
	"github.com/spf13/cobra"
)

var newSiteCmd = &cobra.Command{
	Use:   "site [path]",
	Short: "Create a new Sarde site",
	Long:  "Scaffold a new Sarde site with starter files, example content, and a discoverable config.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runNewSite,
}

func runNewSite(cmd *cobra.Command, args []string) error {
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
		filepath.Join(absDir, "static", "images"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("creating directory %s: %w", d, err)
		}
	}

	files := map[string]string{
		consts.FileSiteConfig: siteYAMLContent,
		filepath.Join(consts.DirContent, "_index.md"):                    indexMDContent,
		filepath.Join(consts.DirContent, "blog", "_index.md"):           blogIndexContent,
		filepath.Join(consts.DirContent, "blog", "hello-world.md"):      blogPostContent,
		filepath.Join(consts.DirContent, "docs", "_index.md"):           docsIndexContent,
		filepath.Join(consts.DirContent, "docs", "getting-started.md"):  docsPageContent,
		filepath.Join("static", "images", "hero-light.svg"):                string(embedded.ScaffoldHeroLight),
		filepath.Join("static", "images", "hero-dark.svg"):               string(embedded.ScaffoldHeroDark),
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
		fmt.Println("  Run 'sarde dev' to start the dev server.")
	}

	return nil
}

const siteYAMLContent = `site:
  title: "My Site"
  description: "A site built with Sarde"
  url: "http://localhost:4727"

theme:
  name: "default"
  # preset: "ocean"        # Try: ocean, forest, rose, clean, minimal, docs, academic
  # overrides:
  #   primary: "#6366f1"

# collections:
#   blog:
#     sort: "date"
#     feed: true
#   docs:
#     sort: "order"

homepage:
  hero:
    title: "Welcome to My Site"
    subtitle: "A modern site powered by Sarde. Edit sarde.yaml to make it yours."
    background: gradient
    cta:
      label: "Get Started"
      url: "/docs/getting-started"
    secondary_cta:
      label: "Read the Blog"
      url: "/blog/hello-world"
    image:
      light: /images/hero-light.svg
      dark: /images/hero-dark.svg
      alt: "Hero illustration"
`

const indexMDContent = `---
title: Welcome
---
`

const blogIndexContent = `---
title: Blog
---
`

const docsIndexContent = `---
title: Documentation
---
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
sidebar:
  order: 1
---

Welcome to the documentation. Add more pages to ` + "`content/docs/`" + ` and they will appear in the sidebar automatically.

## Next Steps

- Edit ` + "`sarde.yaml`" + ` to customize your site
- Run ` + "`sarde dev`" + ` to start the dev server
- Run ` + "`sarde build`" + ` to generate the static site
`
