# Coderoo

A zero-config, Go-based static site generator that ships as a single binary. Drop Markdown files into a `content/` folder and get a fully-themed, production-ready static site.

## Quick Start

```bash
go run ./cmd/coderoo init my-site
cd my-site
go run ./cmd/coderoo serve
```

## Commands

```bash
coderoo build     # Production build (with minification)
coderoo serve     # Dev server with live reload (port 3000)
coderoo init      # Scaffold a new project
coderoo new       # Create new content
coderoo validate  # Validate content without building
coderoo sidecar   # Start IPC server for desktop app
```

## Development

```bash
make build        # Build binary to dist/
make test         # Run all tests
make bench        # Run performance benchmarks
make vet          # Run static analysis
make clean        # Remove dist/ and .cache/
make build-all    # Cross-compile for linux, macOS, Windows
```

Or without Make:

```bash
go build ./cmd/coderoo                                      # Build binary
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
