package build

import (
	"fmt"
	"io/fs"
	"path/filepath"
)

// WriteEmbeddedAssets walks the `assets/` subtree inside the given embedded
// theme filesystem and writes every file to outputDir/assets/. It is a no-op
// when the `assets/` directory is absent.
func WriteEmbeddedAssets(embeddedFS fs.FS, outputDir string) error {
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
		destPath := filepath.Join(outputDir, filepath.FromSlash(path))
		return writeFile(destPath, data)
	})
}
