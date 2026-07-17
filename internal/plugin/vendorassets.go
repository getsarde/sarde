package plugin

import (
	"io/fs"
	"strings"
)

// WriteFSTree copies every file under root in fsys into the build output,
// rewriting each path from root/... to destPrefix + ... . Used by plugins
// that ship embedded vendor asset trees (KaTeX, Mermaid).
func WriteFSTree(ctx *BuildDoneContext, fsys fs.FS, root, destPrefix string) error {
	return WriteFSTreeFiltered(ctx, fsys, root, destPrefix, nil)
}

// WriteFSTreeFiltered is WriteFSTree with an optional include filter applied
// to each root-relative file path. A nil filter copies everything.
func WriteFSTreeFiltered(ctx *BuildDoneContext, fsys fs.FS, root, destPrefix string, include func(rel string) bool) error {
	return fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel := path
		if root != "." {
			rel = strings.TrimPrefix(path, root+"/")
		}
		if include != nil && !include(rel) {
			return nil
		}
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}
		return ctx.WriteFile(destPrefix+rel, data)
	})
}
