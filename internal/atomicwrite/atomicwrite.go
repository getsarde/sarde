// Package atomicwrite provides atomic file write operations via temp-file +
// rename. It lives outside build/asset/plugin to avoid import cycles.
package atomicwrite

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/getsarde/sarde/internal/devlog"
)

const (
	maxRetries   = 5
	baseBackoff  = 10 * time.Millisecond
)

// WriteFile atomically writes data to path via a uniquely-named temp file in
// the same directory, then os.Rename over the target. Parent directories are
// created as needed.
func WriteFile(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("atomicwrite: mkdir %s: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp.*")
	if err != nil {
		return fmt.Errorf("atomicwrite: create temp: %w", err)
	}
	tmpName := tmp.Name()

	_, writeErr := tmp.Write(data)
	if closeErr := tmp.Close(); writeErr == nil {
		writeErr = closeErr
	}
	if writeErr != nil {
		os.Remove(tmpName)
		return fmt.Errorf("atomicwrite: write %s: %w", path, writeErr)
	}

	if err := os.Chmod(tmpName, perm); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("atomicwrite: chmod %s: %w", tmpName, err)
	}

	if err := renameWithRetry(tmpName, path, nil, perm); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}

// CopyFile atomically copies src to dst via streaming (no full-file buffering).
// Parent directories are created as needed.
func CopyFile(src, dst string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("atomicwrite: open source %s: %w", src, err)
	}
	defer in.Close()

	dir := filepath.Dir(dst)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("atomicwrite: mkdir %s: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, filepath.Base(dst)+".tmp.*")
	if err != nil {
		return fmt.Errorf("atomicwrite: create temp: %w", err)
	}
	tmpName := tmp.Name()

	_, copyErr := io.Copy(tmp, in)
	if closeErr := tmp.Close(); copyErr == nil {
		copyErr = closeErr
	}
	if copyErr != nil {
		os.Remove(tmpName)
		return fmt.Errorf("atomicwrite: copy %s -> %s: %w", src, dst, copyErr)
	}

	if err := os.Chmod(tmpName, perm); err != nil {
		os.Remove(tmpName)
		return fmt.Errorf("atomicwrite: chmod %s: %w", tmpName, err)
	}

	if err := renameWithRetry(tmpName, dst, in, perm); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}

// renameWithRetry attempts os.Rename up to maxRetries times with linear
// backoff. On exhausted retries it falls back to a direct (non-atomic) write,
// which is identical to the pre-atomicwrite behavior.
// srcReader, if non-nil, is rewound (Seek) for the copy fallback; otherwise
// tmpPath is read back. perm is applied to the fallback write.
func renameWithRetry(tmpPath, dstPath string, srcReader *os.File, perm os.FileMode) error {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if err := os.Rename(tmpPath, dstPath); err == nil {
			return nil
		} else {
			lastErr = err
		}
		if i < maxRetries-1 {
			time.Sleep(baseBackoff * time.Duration(i+1))
		}
	}

	devlog.Warn("atomicwrite", "rename failed after %d attempts for %s, falling back to direct write: %v", maxRetries, dstPath, lastErr)

	var fallbackErr error
	if srcReader != nil {
		if _, err := srcReader.Seek(0, io.SeekStart); err == nil {
			out, err := os.OpenFile(dstPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, perm)
			if err != nil {
				return fmt.Errorf("atomicwrite: fallback create %s: %w", dstPath, err)
			}
			_, copyErr := io.Copy(out, srcReader)
			if closeErr := out.Close(); copyErr == nil {
				copyErr = closeErr
			}
			fallbackErr = copyErr
		}
	}
	if fallbackErr == nil && srcReader == nil {
		data, err := os.ReadFile(tmpPath)
		if err != nil {
			return fmt.Errorf("atomicwrite: fallback read temp %s: %w", tmpPath, err)
		}
		fallbackErr = os.WriteFile(dstPath, data, perm)
	}

	os.Remove(tmpPath)
	return fallbackErr
}
