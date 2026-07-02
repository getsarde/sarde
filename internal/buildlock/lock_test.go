package buildlock

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/getsarde/sarde/internal/consts"
)

func TestAcquireRelease_RoundTrip(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "dist")

	l, err := Acquire(dir, "test")
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}

	lockPath := filepath.Join(dir, consts.FileOutputLock)
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatalf("lock file not created: %v", err)
	}
	if !strings.Contains(string(data), fmt.Sprintf(`"pid":%d`, os.Getpid())) {
		t.Errorf("metadata missing own pid: %s", data)
	}
	if !strings.Contains(string(data), `"cmd":"test"`) {
		t.Errorf("metadata missing cmd: %s", data)
	}
	if got := RefCount(dir); got != 1 {
		t.Errorf("RefCount = %d, want 1", got)
	}

	if err := l.Release(); err != nil {
		t.Fatalf("Release: %v", err)
	}
	if got := RefCount(dir); got != 0 {
		t.Errorf("RefCount after release = %d, want 0", got)
	}

	// The lock file stays behind by design and must be re-acquirable.
	if _, err := os.Stat(lockPath); err != nil {
		t.Errorf("lock file should remain after release: %v", err)
	}
	l2, err := Acquire(dir, "test")
	if err != nil {
		t.Fatalf("re-Acquire after release: %v", err)
	}
	l2.Release()
}

func TestAcquire_CreatesMissingOutputDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "nested", "deeper", "dist")

	l, err := Acquire(dir, "test")
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	defer l.Release()

	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		t.Errorf("output dir not created: %v", err)
	}
}

func TestAcquire_ReentrantRefCounting(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "dist")

	l1, err := Acquire(dir, "test")
	if err != nil {
		t.Fatalf("first Acquire: %v", err)
	}
	l2, err := Acquire(dir, "test")
	if err != nil {
		t.Fatalf("re-entrant Acquire: %v", err)
	}
	if got := RefCount(dir); got != 2 {
		t.Errorf("RefCount = %d, want 2", got)
	}

	if err := l2.Release(); err != nil {
		t.Fatalf("Release l2: %v", err)
	}
	if got := RefCount(dir); got != 1 {
		t.Errorf("RefCount after one release = %d, want 1", got)
	}

	if err := l1.Release(); err != nil {
		t.Fatalf("Release l1: %v", err)
	}
	if got := RefCount(dir); got != 0 {
		t.Errorf("RefCount after both releases = %d, want 0", got)
	}
}

func TestRelease_IdempotentAndNilSafe(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "dist")

	l1, err := Acquire(dir, "test")
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}
	l2, err := Acquire(dir, "test")
	if err != nil {
		t.Fatalf("Acquire: %v", err)
	}

	// Double-release of the same reference must not decrement twice.
	if err := l2.Release(); err != nil {
		t.Fatalf("Release: %v", err)
	}
	if err := l2.Release(); err != nil {
		t.Fatalf("second Release: %v", err)
	}
	if got := RefCount(dir); got != 1 {
		t.Errorf("RefCount after double release of one ref = %d, want 1", got)
	}
	l1.Release()

	var nilLock *Lock
	if err := nilLock.Release(); err != nil {
		t.Errorf("nil Release: %v", err)
	}
}

func TestAcquire_LockPathIsDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "dist")
	if err := os.MkdirAll(filepath.Join(dir, consts.FileOutputLock), 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := Acquire(dir, "test")
	if err == nil {
		t.Fatal("Acquire should fail when the lock path is a directory")
	}
	if !strings.Contains(err.Error(), "directory exists at that path") {
		t.Errorf("err = %v, want directory-at-path message", err)
	}
}

func TestAcquire_ContentionReportsHolderMetadata(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "dist")
	os.MkdirAll(dir, 0o755)
	lockPath := filepath.Join(dir, consts.FileOutputLock)

	// Simulate a foreign holder: bypass the registry, take the raw OS lock
	// on an independent handle with valid metadata at offset 0.
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	f.WriteString(`{"pid":4242,"cmd":"dev","version":"test","started":"2026-01-01T00:00:00Z"}` + "\n")
	if err := lockFile(f); err != nil {
		t.Fatalf("raw lock: %v", err)
	}
	defer unlockFile(f)

	_, err = Acquire(dir, "test")
	el, ok := IsLocked(err)
	if !ok {
		t.Fatalf("err = %v, want *ErrLocked", err)
	}
	if el.Meta.PID != 4242 || el.Meta.Cmd != "dev" {
		t.Errorf("Meta = %+v, want pid 4242 / cmd dev", el.Meta)
	}
	msg := el.Error()
	// The message quotes paths with %q, which escapes Windows backslashes.
	if !strings.Contains(msg, "4242") || !strings.Contains(msg, "sarde dev") || !strings.Contains(msg, fmt.Sprintf("%q", lockPath)) {
		t.Errorf("message missing diagnostics: %s", msg)
	}
}

func TestAcquire_ContentionWithCorruptMetadataDegrades(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "dist")
	os.MkdirAll(dir, 0o755)
	lockPath := filepath.Join(dir, consts.FileOutputLock)

	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	f.WriteString("not-json{{{")
	if err := lockFile(f); err != nil {
		t.Fatalf("raw lock: %v", err)
	}
	defer unlockFile(f)

	_, err = Acquire(dir, "test")
	el, ok := IsLocked(err)
	if !ok {
		t.Fatalf("err = %v, want *ErrLocked", err)
	}
	if el.MetaErr == nil {
		t.Error("MetaErr should be set for corrupt metadata")
	}
	msg := el.Error()
	if !strings.Contains(msg, "metadata unreadable") {
		t.Errorf("message should degrade gracefully: %s", msg)
	}
	if strings.Contains(msg, "not-json") {
		t.Errorf("raw metadata leaked into message: %s", msg)
	}
}

func TestIsLocked_UnwrapsWrappedErrors(t *testing.T) {
	inner := &ErrLocked{OutputDir: "x", LockPath: "y"}
	wrapped := fmt.Errorf("starting dev server: %w", inner)

	el, ok := IsLocked(wrapped)
	if !ok || el != inner {
		t.Errorf("IsLocked failed to unwrap: %v, %v", el, ok)
	}
	if _, ok := IsLocked(fmt.Errorf("plain error")); ok {
		t.Error("IsLocked matched a non-lock error")
	}
}
