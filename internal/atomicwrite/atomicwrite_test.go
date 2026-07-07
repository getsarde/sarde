package atomicwrite

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
)

func TestWriteFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "test.html")
	data := []byte("<html>hello</html>")

	if err := WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(data) {
		t.Errorf("content = %q, want %q", got, data)
	}

	if runtime.GOOS != "windows" {
		info, _ := os.Stat(path)
		if perm := info.Mode().Perm(); perm != 0o644 {
			t.Errorf("perm = %o, want 644", perm)
		}
	}

	// No temp files left behind.
	entries, _ := os.ReadDir(filepath.Dir(path))
	for _, e := range entries {
		if e.Name() != "test.html" {
			t.Errorf("leftover file: %s", e.Name())
		}
	}
}

func TestWriteFile_Overwrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "page.html")

	if err := WriteFile(path, []byte("old content"), 0o644); err != nil {
		t.Fatalf("first write: %v", err)
	}

	newData := []byte("new content")
	if err := WriteFile(path, newData, 0o644); err != nil {
		t.Fatalf("second write: %v", err)
	}

	got, _ := os.ReadFile(path)
	if string(got) != string(newData) {
		t.Errorf("content = %q, want %q", got, newData)
	}
}

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "out", "dst.txt")

	srcData := []byte("source file content")
	if err := os.WriteFile(src, srcData, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := CopyFile(src, dst, 0o644); err != nil {
		t.Fatalf("CopyFile: %v", err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(srcData) {
		t.Errorf("content = %q, want %q", got, srcData)
	}

	entries, _ := os.ReadDir(filepath.Dir(dst))
	for _, e := range entries {
		if e.Name() != "dst.txt" {
			t.Errorf("leftover file: %s", e.Name())
		}
	}
}

func TestCopyFile_MissingSrc(t *testing.T) {
	dir := t.TempDir()
	err := CopyFile(filepath.Join(dir, "missing"), filepath.Join(dir, "dst"), 0o644)
	if err == nil {
		t.Fatal("expected error for missing source")
	}
}

func TestWriteFile_Concurrent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "shared.html")
	const n = 20

	var wg sync.WaitGroup
	errs := make([]error, n)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			data := make([]byte, 1024)
			for j := range data {
				data[j] = byte('A' + idx%26)
			}
			errs[idx] = WriteFile(path, data, 0o644)
		}(i)
	}
	wg.Wait()

	for i, err := range errs {
		if err != nil {
			t.Errorf("goroutine %d: %v", i, err)
		}
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("final read: %v", err)
	}
	if len(got) != 1024 {
		t.Fatalf("file length = %d, want 1024", len(got))
	}
	// Every byte should be the same character (one writer won).
	ch := got[0]
	for i, b := range got {
		if b != ch {
			t.Fatalf("byte %d = %c, want %c (file is a mix of writes)", i, b, ch)
		}
	}

	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if e.Name() != "shared.html" {
			t.Errorf("leftover temp file: %s", e.Name())
		}
	}
}
