package build

import (
	"fmt"
	"io/fs"
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
