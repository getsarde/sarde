package asset

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
)

// Bundler handles CSS/JS bundling and minification via esbuild.
type Bundler struct {
	Resolver  *Resolver
	DevMode   bool
	Minify    bool
	OutputDir string // absolute path to output directory
}

// BundledFile represents a single bundled output file.
type BundledFile struct {
	Name         string // output filename (potentially fingerprinted)
	OriginalPath string // original entry point path as referenced
	OutputURL    string // URL path (e.g., "/assets/css/main.a1b2c3d4.css")
	Content      []byte
}

// BundleResult holds the output of a bundle operation.
type BundleResult struct {
	OutputFiles []BundledFile
	Errors      []string
}

// BundleCSS bundles a CSS entry point, resolving @import through the 3-layer resolver.
func (b *Bundler) BundleCSS(entryPoint string) (*BundleResult, error) {
	return b.bundle(entryPoint, api.LoaderCSS)
}

// BundleJS bundles a JS entry point, resolving imports through the 3-layer resolver.
func (b *Bundler) BundleJS(entryPoint string) (*BundleResult, error) {
	return b.bundle(entryPoint, api.LoaderJS)
}

func (b *Bundler) bundle(entryPoint string, loader api.Loader) (*BundleResult, error) {
	// Resolve the entry point through the 3-layer lookup.
	resolvedPath, err := b.Resolver.Resolve(entryPoint)
	if err != nil {
		return nil, fmt.Errorf("resolving entry point %q: %w", entryPoint, err)
	}

	// For embedded assets, we need to read content and use stdin.
	if strings.HasPrefix(resolvedPath, "embedded:") {
		return b.bundleFromContent(entryPoint, loader)
	}

	opts := api.BuildOptions{
		EntryPoints: []string{resolvedPath},
		Bundle:      true,
		Write:       false, // we handle writing ourselves
		Outdir:      "/virtual",
		Plugins:     []api.Plugin{b.resolverPlugin()},
	}

	if !b.DevMode && b.Minify {
		opts.MinifyWhitespace = true
		opts.MinifyIdentifiers = true
		opts.MinifySyntax = true
	}

	if b.DevMode {
		opts.Sourcemap = api.SourceMapInline
	}

	result := api.Build(opts)

	br := &BundleResult{}
	for _, msg := range result.Errors {
		br.Errors = append(br.Errors, msg.Text)
	}
	if len(br.Errors) > 0 {
		return br, fmt.Errorf("bundle errors: %s", strings.Join(br.Errors, "; "))
	}

	for _, f := range result.OutputFiles {
		content := f.Contents
		hash := Fingerprint(content)
		ext := filepath.Ext(entryPoint)
		baseName := strings.TrimSuffix(filepath.Base(entryPoint), ext)

		var outName string
		if b.DevMode {
			outName = baseName + ext
		} else {
			outName = FingerprintedName(baseName+ext, hash)
		}

		// Determine output subdirectory based on file type.
		subDir := "css"
		if ext == ".js" || ext == ".mjs" {
			subDir = "js"
		}

		outputURL := "/assets/" + subDir + "/" + outName

		br.OutputFiles = append(br.OutputFiles, BundledFile{
			Name:         outName,
			OriginalPath: entryPoint,
			OutputURL:    outputURL,
			Content:      content,
		})
	}

	return br, nil
}

// bundleFromContent handles bundling assets that come from the embedded FS.
func (b *Bundler) bundleFromContent(entryPoint string, loader api.Loader) (*BundleResult, error) {
	content, err := b.Resolver.ResolveContent(entryPoint)
	if err != nil {
		return nil, err
	}

	opts := api.BuildOptions{
		Stdin: &api.StdinOptions{
			Contents:   string(content),
			Loader:     loader,
			ResolveDir: filepath.Join(b.Resolver.ProjectDir, "assets"),
			Sourcefile: entryPoint,
		},
		Bundle:  true,
		Write:   false,
		Outdir:  "/virtual",
		Plugins: []api.Plugin{b.resolverPlugin()},
	}

	if !b.DevMode && b.Minify {
		opts.MinifyWhitespace = true
		opts.MinifyIdentifiers = true
		opts.MinifySyntax = true
	}

	result := api.Build(opts)

	br := &BundleResult{}
	for _, msg := range result.Errors {
		br.Errors = append(br.Errors, msg.Text)
	}
	if len(br.Errors) > 0 {
		return br, fmt.Errorf("bundle errors: %s", strings.Join(br.Errors, "; "))
	}

	for _, f := range result.OutputFiles {
		outContent := f.Contents
		hash := Fingerprint(outContent)
		ext := filepath.Ext(entryPoint)
		baseName := strings.TrimSuffix(filepath.Base(entryPoint), ext)

		var outName string
		if b.DevMode {
			outName = baseName + ext
		} else {
			outName = FingerprintedName(baseName+ext, hash)
		}

		subDir := "css"
		if ext == ".js" || ext == ".mjs" {
			subDir = "js"
		}

		outputURL := "/assets/" + subDir + "/" + outName

		br.OutputFiles = append(br.OutputFiles, BundledFile{
			Name:         outName,
			OriginalPath: entryPoint,
			OutputURL:    outputURL,
			Content:      outContent,
		})
	}

	return br, nil
}

// resolverPlugin creates an esbuild plugin that resolves imports through the 3-layer asset lookup.
func (b *Bundler) resolverPlugin() api.Plugin {
	return api.Plugin{
		Name: "sarde-resolver",
		Setup: func(build api.PluginBuild) {
			// Intercept all resolve calls for relative imports.
			build.OnResolve(api.OnResolveOptions{Filter: ".*"}, func(args api.OnResolveArgs) (api.OnResolveResult, error) {
				// Skip absolute paths and already-resolved paths.
				if filepath.IsAbs(args.Path) || strings.HasPrefix(args.Path, "/") {
					return api.OnResolveResult{}, nil
				}

				// Try to resolve through the 3-layer lookup.
				resolved, err := b.Resolver.Resolve(args.Path)
				if err != nil {
					// Let esbuild handle it with its default resolution.
					return api.OnResolveResult{}, nil
				}

				if strings.HasPrefix(resolved, "embedded:") {
					// For embedded assets, read content and provide via stdin-like mechanism.
					return api.OnResolveResult{
						Path:      resolved,
						Namespace: "sarde-embedded",
					}, nil
				}

				return api.OnResolveResult{Path: resolved}, nil
			})

			// Load embedded assets.
			build.OnLoad(api.OnLoadOptions{Filter: ".*", Namespace: "sarde-embedded"}, func(args api.OnLoadArgs) (api.OnLoadResult, error) {
				// Strip "embedded:" prefix.
				assetPath := strings.TrimPrefix(args.Path, "embedded:")
				content, err := b.Resolver.ResolveContent(assetPath)
				if err != nil {
					return api.OnLoadResult{}, err
				}

				ext := filepath.Ext(assetPath)
				var loader api.Loader
				switch ext {
				case ".css":
					loader = api.LoaderCSS
				case ".js", ".mjs":
					loader = api.LoaderJS
				default:
					loader = api.LoaderFile
				}

				contents := string(content)
				return api.OnLoadResult{
					Contents: &contents,
					Loader:   loader,
				}, nil
			})
		},
	}
}

// TransformCSS runs esbuild's Transform API on a raw CSS string.
// When minify is true, whitespace and syntax are minified.
// When minify is false, the input is returned unchanged.
func TransformCSS(content string, minify bool) (string, error) {
	if !minify {
		return content, nil
	}
	result := api.Transform(content, api.TransformOptions{
		Loader:           api.LoaderCSS,
		MinifyWhitespace: true,
		MinifySyntax:     true,
	})
	if len(result.Errors) > 0 {
		return "", fmt.Errorf("esbuild transform: %s", result.Errors[0].Text)
	}
	return string(result.Code), nil
}
