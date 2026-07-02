package plugin

import (
	"io/fs"
	"strings"
)

// WriteFSTree copies every file under root in fsys into the build output,
// rewriting each path from root/... to destPrefix + ... . Used by plugins
// that ship embedded vendor asset trees (KaTeX, Mermaid).
func WriteFSTree(ctx *BuildDoneContext, fsys fs.FS, root, destPrefix string) error {
	return fs.WalkDir(fsys, root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}
		rel := strings.TrimPrefix(path, root+"/")
		return ctx.WriteFile(destPrefix+rel, data)
	})
}
