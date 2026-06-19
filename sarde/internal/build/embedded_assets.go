package build

import (
	"bytes"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"

	"github.com/evanw/esbuild/pkg/api"
	"github.com/frostybee/sarde/internal/asset"
)

// WriteEmbeddedCSS writes the embedded CSS bundle to outputDir/assets/css/<filename>.
func WriteEmbeddedCSS(outputDir string, css string, filename string, tracker *OutputTracker) error {
	if css == "" {
		return nil
	}
	destPath, err := safeOutputPath(outputDir, "assets/css/"+filename)
	if err != nil {
		return err
	}
	if tracker != nil {
		tracker.Track(destPath)
	}
	return writeFile(destPath, []byte(css))
}

// WriteEmbeddedAssets walks the `assets/` subtree inside the given embedded
// theme filesystem and writes every file to outputDir/assets/.
// Files under any of the skipPrefixes are skipped (used to exclude bundled JS).
// It is a no-op when the `assets/` directory is absent.
func WriteEmbeddedAssets(embeddedFS fs.FS, outputDir string, tracker *OutputTracker, skipPrefixes []string) error {
	if embeddedFS == nil {
		return nil
	}
	root := "assets"
	if _, err := fs.Stat(embeddedFS, root); err != nil {
		return nil
	}
	return fs.WalkDir(embeddedFS, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		for _, prefix := range skipPrefixes {
			if strings.HasPrefix(path, prefix) {
				return nil
			}
		}
		data, err := fs.ReadFile(embeddedFS, path)
		if err != nil {
			return fmt.Errorf("reading embedded asset %s: %w", path, err)
		}
		destPath, err := safeOutputPath(outputDir, path)
		if err != nil {
			return err
		}
		if tracker != nil {
			tracker.Track(destPath)
		}
		return writeFile(destPath, data)
	})
}

// BundleEmbeddedJS concatenates all JS files under assets/js/ in the embedded
// theme FS into a single bundle, wrapping each in an IIFE. In production mode
// the bundle is minified via esbuild. Returns the bundled content and a
// filename (fingerprinted in production, plain in dev).
func BundleEmbeddedJS(embeddedFS fs.FS, devMode bool) ([]byte, string, error) {
	if embeddedFS == nil {
		return nil, "", nil
	}
	jsDir := "assets/js"
	if _, err := fs.Stat(embeddedFS, jsDir); err != nil {
		return nil, "", nil
	}

	var files []string
	err := fs.WalkDir(embeddedFS, jsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(path) == ".js" {
			if devMode && filepath.Base(path) == "prefetch.js" {
				return nil
			}
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, "", fmt.Errorf("walking embedded JS: %w", err)
	}
	if len(files) == 0 {
		return nil, "", nil
	}
	sort.Strings(files)

	var buf bytes.Buffer
	for _, f := range files {
		data, err := fs.ReadFile(embeddedFS, f)
		if err != nil {
			return nil, "", fmt.Errorf("reading %s: %w", f, err)
		}
		name := filepath.Base(f)
		buf.WriteString("/* " + name + " */\n;(function(){\n")
		buf.Write(data)
		buf.WriteString("\n})();\n")
	}

	content := buf.Bytes()
	if !devMode {
		content = minifyJS(content)
	}

	hash := asset.Fingerprint(content)
	var filename string
	if devMode {
		filename = "sarde.js"
	} else {
		filename = asset.FingerprintedName("sarde.js", hash)
	}

	return content, filename, nil
}

func minifyJS(data []byte) []byte {
	result := api.Transform(string(data), api.TransformOptions{
		Loader:            api.LoaderJS,
		MinifyWhitespace:  true,
		MinifySyntax:      true,
		MinifyIdentifiers: true,
	})
	if len(result.Errors) > 0 {
		return data
	}
	return result.Code
}
