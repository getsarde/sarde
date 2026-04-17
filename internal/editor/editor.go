// Package editor provides safe, auditable file edits for user content.
//
// The package offers two independent facilities:
//
//   - SafeWrite performs an atomic write (temp-file + rename) and, if a file
//     already exists at the target, creates a sibling ".bak" copy so an
//     immediate-prior version is always recoverable even without revision
//     history enabled.
//
//   - CreateRevision / ListRevisions / RestoreRevision / PruneRevisions keep
//     timestamped snapshots under a ".revisions/" directory alongside the
//     edited file.
//
// Both facilities are path-agnostic. Callers are expected to validate that
// paths sit inside the project content root before invoking these functions.
package editor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// SafeWriteOptions controls the behavior of SafeWrite.
type SafeWriteOptions struct {
	// Backup, when true, copies the existing file to "<path>.bak" before
	// overwriting. Ignored when the target does not yet exist.
	Backup bool
	// Revision, when true, creates a timestamped snapshot in the sibling
	// ".revisions/" directory before overwriting. Ignored when the target
	// does not yet exist.
	Revision bool
	// Perm is the mode bits applied when creating a new file.
	Perm os.FileMode
}

// SafeWrite writes data to path atomically. When opts.Backup is set and a
// file already exists at path, the prior contents are copied to "<path>.bak"
// before the new data is written. Parent directories are created as needed.
func SafeWrite(path string, data []byte, opts SafeWriteOptions) error {
	if opts.Perm == 0 {
		opts.Perm = 0o644
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating parent directory: %w", err)
	}

	// If the file exists, snapshot it before overwriting.
	if _, err := os.Stat(path); err == nil {
		if opts.Backup {
			if err := copyFile(path, path+".bak", opts.Perm); err != nil {
				return fmt.Errorf("creating .bak backup: %w", err)
			}
		}
		if opts.Revision {
			if err := CreateRevision(path); err != nil {
				return fmt.Errorf("creating revision: %w", err)
			}
		}
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, opts.Perm); err != nil {
		return fmt.Errorf("writing temp file: %w", err)
	}
	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("renaming temp file: %w", err)
	}
	return nil
}

// ValidateWithinRoot reports an error if relPath would resolve outside root
// after cleaning and symlink-independent joining. It is intentionally
// lexical (no Stat) so callers can validate paths that do not yet exist.
func ValidateWithinRoot(root, relPath string) error {
	if relPath == "" {
		return fmt.Errorf("empty path")
	}
	cleaned := filepath.Clean(filepath.FromSlash(relPath))
	if strings.HasPrefix(cleaned, "..") || strings.Contains(cleaned, string(filepath.Separator)+".."+string(filepath.Separator)) {
		return fmt.Errorf("path traversal not allowed: %s", relPath)
	}
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	absTarget, err := filepath.Abs(filepath.Join(root, cleaned))
	if err != nil {
		return err
	}
	if absTarget != absRoot && !strings.HasPrefix(absTarget, absRoot+string(filepath.Separator)) {
		return fmt.Errorf("path outside root: %s", relPath)
	}
	return nil
}

func copyFile(src, dst string, perm os.FileMode) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, perm)
}
