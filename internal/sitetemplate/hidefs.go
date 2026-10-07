package sitetemplate

import "io/fs"

// hiddenFS hides a few root-level entries of an fs.FS. Open covers fs.Stat
// and fs.ReadFile; ReadDir covers fs.WalkDir and os.CopyFS, which prefer
// ReadDirFS. Nested entries with the same names are untouched.
type hiddenFS struct {
	fs.FS
	hidden map[string]bool
}

// HideRootFiles returns fsys without the named root-level entries. The CLI
// uses it to keep a template's README.md and LICENSE, which describe the
// template on GitHub, out of the new site.
func HideRootFiles(fsys fs.FS, names ...string) fs.FS {
	h := &hiddenFS{FS: fsys, hidden: make(map[string]bool, len(names))}
	for _, n := range names {
		h.hidden[n] = true
	}
	return h
}

func (h *hiddenFS) Open(name string) (fs.File, error) {
	if h.hidden[name] {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	f, err := h.FS.Open(name)
	if err != nil || name != "." {
		return f, err
	}
	// The root directory read through its file handle must agree with
	// ReadDir, or fs.WalkDir and os.CopyFS could still see hidden entries.
	if d, ok := f.(fs.ReadDirFile); ok {
		return &hiddenDir{ReadDirFile: d, hidden: h.hidden}, nil
	}
	return f, nil
}

type hiddenDir struct {
	fs.ReadDirFile
	hidden map[string]bool
}

func (d *hiddenDir) ReadDir(n int) ([]fs.DirEntry, error) {
	var kept []fs.DirEntry
	for {
		entries, err := d.ReadDirFile.ReadDir(n)
		for _, e := range entries {
			if !d.hidden[e.Name()] {
				kept = append(kept, e)
			}
		}
		// With a positive n, keep reading past a batch made only of hidden
		// entries, so an empty result means the end of the directory.
		if err != nil || n <= 0 || len(kept) > 0 || len(entries) == 0 {
			return kept, err
		}
	}
}

func (h *hiddenFS) ReadDir(name string) ([]fs.DirEntry, error) {
	entries, err := fs.ReadDir(h.FS, name)
	if err != nil || name != "." {
		return entries, err
	}
	kept := entries[:0]
	for _, e := range entries {
		if !h.hidden[e.Name()] {
			kept = append(kept, e)
		}
	}
	return kept, nil
}
