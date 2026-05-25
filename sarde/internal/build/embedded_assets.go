package build

import (
	"fmt"
	"io/fs"
)

// WriteEmbeddedCSS writes the concatenated embedded CSS bundle to an external
// file at outputDir/assets/css/sarde.css. Returns the root-relative URL.
func WriteEmbeddedCSS(outputDir string, css string, tracker *OutputTracker) error {
	if css == "" {
		return nil
	}
	destPath, err := safeOutputPath(outputDir, "assets/css/sarde.css")
	if err != nil {
		return err
	}
	if tracker != nil {
		tracker.Track(destPath)
	}
	return writeFile(destPath, []byte(css))
}

// WriteEmbeddedAssets walks the `assets/` subtree inside the given embedded
// theme filesystem and writes every file to outputDir/assets/. It is a no-op
// when the `assets/` directory is absent.
func WriteEmbeddedAssets(embeddedFS fs.FS, outputDir string, tracker *OutputTracker) error {
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
