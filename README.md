# Sarde

A zero-config, Go-based static site generator that ships as a single binary. Drop Markdown files into a `content/` folder and get a fully-themed, production-ready static site.

## Installation

**Homebrew (macOS/Linux):**

```bash
brew tap getsarde/sarde
brew install sarde
```

**Shell script (macOS/Linux):**

```bash
curl -sSfL https://raw.githubusercontent.com/getsarde/sarde/main/install.sh | sh
```

**Binary download:**

Grab the latest release from [GitHub Releases](https://github.com/getsarde/sarde/releases).

**From source:**

```bash
go install github.com/getsarde/sarde/cmd/sarde@latest
```

## Quick Start

```bash
cd sarde
go run ./cmd/sarde new site my-site
cd my-site
go run ../cmd/sarde dev
```

## Commands

```bash
sarde build                        # Production build (with minification)
sarde dev                          # Dev server with live reload (port 4727)
sarde new site [path]              # Scaffold a new project
sarde new <collection> <title>     # Create new content file
sarde new course <name>            # Scaffold a new course
sarde new lesson <course> <name>   # Add an auto-numbered lesson to a course
sarde validate                     # Validate content without building
sarde render                       # Render markdown from stdin to JSON
sarde deploy                       # Deploy the built site
sarde import obsidian <vault>      # Import an Obsidian vault
sarde sidecar                      # Start IPC server for desktop app
sarde version                      # Print version info
```

## Building

All build commands must be run from the `sarde/` directory (where `go.mod` lives):

```bash
cd sarde

# Build the binary (output: sarde or sarde.exe)
go build -o ../dist/sarde ./cmd/sarde

# Build with a version tag
go build -ldflags "-X github.com/getsarde/sarde/internal/version.Version=1.0.0" -o ../dist/sarde ./cmd/sarde

# Cross-compile examples
GOOS=linux   GOARCH=amd64 go build -o ../dist/sarde-linux   ./cmd/sarde
GOOS=darwin  GOARCH=arm64 go build -o ../dist/sarde-macos   ./cmd/sarde
GOOS=windows GOARCH=amd64 go build -o ../dist/sarde.exe     ./cmd/sarde
```

## Development

```bash
go test ./...                                               # Run all tests
go test -bench=. -benchmem -timeout 300s ./internal/build/  # Run benchmarks
go vet ./...                                                # Static analysis
```

## Architecture

Three-layer design:

1. **Interface Layer** -- CLI (Cobra) and Desktop App (Tauri+Svelte) both call into ProjectManager
2. **Site Engine** -- Six-phase pipeline: Initialize, Discover, Parse, Assemble, Render, Write
3. **Plugin System** -- Four lifecycle hooks: ConfigSetup, ContentLoaded, BeforeRender, BuildDone

## Features

- **Zero config** -- directory structure IS site structure
- **Collections** -- auto-detected from directory names (blog, docs, courses, etc.)
- **24 Goldmark extensions** -- code blocks, tabs, callouts, cards, math, mermaid, and more
- **Theme system** -- CSS design tokens, 4-layer token resolution, dark mode
- **Navigation** -- sidebar nav trees, breadcrumbs, prev/next, global nav
- **Asset pipeline** -- image processing (resize, WebP, LQIP), CSS/JS bundling via esbuild, fingerprinting
- **Plugins** -- sitemap, robots.txt, RSS, SEO meta tags, reading time, search index, link checker
- **i18n** -- opt-in internationalization with language directories, translation strings, fallback pages
- **Dev server** -- live reload via WebSocket, error overlay with file/line/frame context
- **Performance** -- parallel rendering, markdown render cache, HTML minification
- **IPC server** -- HTTP JSON API for desktop app integration

## Tech Stack

- **Core**: Go 1.25+, Goldmark, Cobra, Chroma v2, fsnotify, esbuild (Go API), go:embed
- **Desktop**: Tauri v2 (Rust) + Svelte 5 + CodeMirror 6
- **Output**: Pure HTML/CSS + ~1KB inline JS
- **Minification**: tdewolff/minify (HTML), esbuild (CSS/JS)
