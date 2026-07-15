package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/consts"
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
		filepath.Join(absDir, consts.DirPublic),
		filepath.Join(absDir, consts.DirPublic, "images"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("creating directory %s: %w", d, err)
		}
	}

	files := map[string]string{
		consts.FileSiteConfig:  siteYAMLContent,
		"kazari.config.yaml":   kazariConfigContent,
		filepath.Join(consts.DirContent, "_index.md"):                    indexMDContent,
		filepath.Join(consts.DirContent, "blog", "_index.md"):           blogIndexContent,
		filepath.Join(consts.DirContent, "blog", "hello-world.md"):      blogPostContent,
		filepath.Join(consts.DirContent, "docs", "_index.md"):           docsIndexContent,
		filepath.Join(consts.DirContent, "docs", "getting-started.md"):  docsPageContent,
		filepath.Join(consts.DirPublic, "images", "hero-light.svg"):         string(embedded.ScaffoldHeroLight),
		filepath.Join(consts.DirPublic, "images", "hero-dark.svg"):       string(embedded.ScaffoldHeroDark),
		filepath.Join(consts.DirPublic, ".gitkeep"):                      "",
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

const kazariConfigContent = `# Kazari code block configuration.
# Docs: https://frostybee.github.io/kazari

# --- Themes ---
themes:
  light: github-light
  dark: github-dark

darkMode:
  kind: selector
  selector: ".dark"

# --- Toolbar ---
copyButton: true
fullscreenButton: true
wrapButton: true
themeToggleButton: false
languageBadge: true
languageIconMode: none
fileIcons: true

# --- Block Defaults ---
lineNumbers: false

defaults:
  wrap: false
  preserveIndent: true
  hangingIndent: 0
  lineNumbers: false
  frame: auto

# --- Frame Detection ---
frameDetection: true
fileNameExtraction: true
terminalDotStyle: colored
terminalCommentStripping: true

# --- Collapsible ---
# collapsible:
#   lineThreshold: 15
#   previewLines: 5
#   defaultCollapsed: true
#   preserveIndent: true
#   style: github

# --- Language Defaults ---
languageDefaults:
  "bash, sh, zsh":
    frame: terminal

# --- Language Aliases ---
languageAliases:
  js: javascript
  ts: typescript
  py: python

# --- Styling ---
themedScrollbars: true
themedSelection: false
styleReset: true
cascadeLayer: kazari
themeCSSRoot: ":root"
# styleOverrides:
#   radius: "0.5rem"
#   font-family: "'JetBrains Mono Variable', monospace"
#   font-size: "0.875rem"
#   line-height: "1.6"
#   shadow: "0 2px 8px rgba(0,0,0,0.15)"
#   border: "1px solid transparent"
#   code-padding-block: "1rem"
#   code-padding-inline: "1.35rem"

# --- Other ---
tabWidth: 2
minContrast: 5.5
minify: true
dataLineCount: true
contentExclusion: false
mermaidPassThrough: true
links: false
locale: en-US
# uiStrings:
#   copy.label: "Copy"
`
