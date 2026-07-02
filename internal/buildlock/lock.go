// Package buildlock provides an exclusive, crash-safe lock on a build output
// directory so two sarde processes cannot write into the same dist/ at once
// (a second dev server or a build racing a dev server silently corrupts
// fingerprinted assets; see my-docs/handoffs/dev-server-asset-404-handoff.md).
//
// The lock is a kernel advisory lock held on <outputDir>/.sarde.lock for the
// lifetime of the owning session. The file also carries one line of JSON
// metadata (pid, command, version, start time) used purely for diagnostics:
// the OS lock is authoritative, so a crash releases the lock automatically
// and leftover lock files never require manual cleanup.
//
// Limitations: sarde binaries built before this package never acquire the
// lock, so it cannot defend against them. Network filesystems (SMB, NFS)
// have unreliable locking semantics; keep the output dir on local disk.
package buildlock

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/version"
)

// Metadata describes the process holding a lock. Purely diagnostic: it is
// never used to decide whether the lock can be acquired.
type Metadata struct {
	PID     int       `json:"pid"`
	Cmd     string    `json:"cmd"`
	Version string    `json:"version"`
	Started time.Time `json:"started"`
}

// ErrLocked is returned by Acquire when another process holds the lock on
// the output directory. Meta is the zero value when MetaErr is non-nil.
type ErrLocked struct {
	OutputDir string
	LockPath  string
	Meta      Metadata
	MetaErr   error
}

func (e *ErrLocked) Error() string {
	if e.MetaErr != nil {
		return fmt.Sprintf(
			"another sarde process is already writing to output directory %q (lock metadata unreadable); stop it first, or delete %q if you are sure it is not running",
			e.OutputDir, e.LockPath)
	}
	return fmt.Sprintf(
		"another sarde process (pid %d, running %q, started %s) is already writing to output directory %q; stop it first, or delete %q if you are sure it is not running",
		e.Meta.PID, "sarde "+e.Meta.Cmd, e.Meta.Started.Format(time.RFC3339), e.OutputDir, e.LockPath)
}

// IsLocked reports whether err is (or wraps) an *ErrLocked.
func IsLocked(err error) (*ErrLocked, bool) {
	var el *ErrLocked
	if errors.As(err, &el) {
		return el, true
	}
	return nil, false
}

// Lock is a held reference to an output directory's exclusive lock.
type Lock struct {
	key      string
	released bool
}

// entry is the per-output-dir lock state shared by all in-process references.
type entry struct {
	refs int
	file *os.File
}

// registry makes Acquire re-entrant within the process: neither Windows
// LockFileEx nor Unix flock is same-process re-entrant (a second independent
// open of the same path contends), and the desktop app legitimately runs a
// build while a preview holds the lock on the same directory.
var (
	registryMu sync.Mutex
	registry   = map[string]*entry{}
)

// maxMetadataSize caps how much of a foreign lock file is read for diagnostics.
const maxMetadataSize = 4096

// Acquire creates outputDir if needed and takes an exclusive, non-blocking,
// crash-safe lock on it. cmd is a short diagnostic label ("dev", "build",
// "sidecar-preview", "sidecar-build") embedded in the lock file's metadata.
// Re-entrant within the same process: repeated calls for the same output
// directory succeed and reference-count instead of re-locking.
func Acquire(outputDir, cmd string) (*Lock, error) {
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating output directory %q: %w", outputDir, err)
	}
	key, err := canonicalKey(outputDir)
	if err != nil {
		return nil, err
	}

	registryMu.Lock()
	defer registryMu.Unlock()

	if e, ok := registry[key]; ok {
		e.refs++
		return &Lock{key: key}, nil
	}

	lockPath := filepath.Join(outputDir, consts.FileOutputLock)
	if info, statErr := os.Stat(lockPath); statErr == nil && info.IsDir() {
		return nil, fmt.Errorf("cannot create lock file %q: a directory exists at that path; remove it manually", lockPath)
	}

	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, fmt.Errorf("creating lock file %q: %w", lockPath, err)
	}

	if err := lockFile(f); err != nil {
		lerr := &ErrLocked{OutputDir: outputDir, LockPath: lockPath}
		if meta, metaErr := readMetadata(f); metaErr != nil {
			lerr.MetaErr = metaErr
		} else {
			lerr.Meta = meta
		}
		f.Close()
		return nil, lerr
	}

	meta := Metadata{PID: os.Getpid(), Cmd: cmd, Version: version.Version, Started: time.Now()}
	if err := writeMetadata(f, meta); err != nil {
		unlockFile(f)
		f.Close()
		return nil, fmt.Errorf("writing lock file %q: %w", lockPath, err)
	}

	registry[key] = &entry{refs: 1, file: f}
	return &Lock{key: key}, nil
}

// Release drops this reference. The OS lock is released and the file closed
// only when the last in-process reference for the output directory is gone.
// The lock FILE is deliberately left in place: deleting it on release would
// let a contender holding a handle to the removed inode and a newcomer
// creating a fresh file both "acquire" the same directory. Idempotent and
// nil-safe.
func (l *Lock) Release() error {
	if l == nil {
		return nil
	}
	registryMu.Lock()
	defer registryMu.Unlock()

	if l.released {
		return nil
	}
	l.released = true

	e, ok := registry[l.key]
	if !ok {
		return nil
	}
	e.refs--
	if e.refs > 0 {
		return nil
	}
	delete(registry, l.key)
	err := unlockFile(e.file)
	if cerr := e.file.Close(); err == nil {
		err = cerr
	}
	return err
}

// RefCount returns the in-process reference count for outputDir (0 when this
// process does not hold the lock). Intended for tests and diagnostics only.
func RefCount(outputDir string) int {
	key, err := canonicalKey(outputDir)
	if err != nil {
		return 0
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	if e, ok := registry[key]; ok {
		return e.refs
	}
	return 0
}

// canonicalKey normalizes an output dir path for use as a registry key,
// case-folding on the OSes the repo treats as case-insensitive (matches
// outputpath.samePath).
func canonicalKey(outputDir string) (string, error) {
	abs, err := filepath.Abs(outputDir)
	if err != nil {
		return "", fmt.Errorf("resolving output directory %q: %w", outputDir, err)
	}
	key := filepath.Clean(abs)
	if runtime.GOOS == "windows" || runtime.GOOS == "darwin" {
		key = strings.ToLower(key)
	}
	return key, nil
}

// readMetadata reads the diagnostic JSON from the start of a lock file. Safe
// while another process holds the lock: the locked byte range (Windows) is a
// reserved offset far beyond the metadata, and Unix flock is advisory.
func readMetadata(f *os.File) (Metadata, error) {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return Metadata{}, err
	}
	buf := make([]byte, maxMetadataSize)
	n, err := f.Read(buf)
	if err != nil && err != io.EOF {
		return Metadata{}, err
	}
	if n == 0 {
		return Metadata{}, errors.New("empty lock file")
	}
	var m Metadata
	if err := json.Unmarshal(bytes.TrimSpace(buf[:n]), &m); err != nil {
		return Metadata{}, err
	}
	return m, nil
}

func writeMetadata(f *os.File, m Metadata) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	if err := f.Truncate(0); err != nil {
		return err
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	if _, err := f.Write(append(data, '\n')); err != nil {
		return err
	}
	return f.Sync()
}
