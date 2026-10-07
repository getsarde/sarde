package devlog

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// capture redirects devlog output to a buffer with the given progress
// capability and restores the previous state when the test ends. Tests in
// this file share package state, so none of them run in parallel.
func capture(t *testing.T, progressCapable bool) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	mu.Lock()
	prevOut, prevOK, prevProgress := out, progressOK, progress
	out, progressOK, progress = &buf, progressCapable, ""
	mu.Unlock()
	prevQuiet := quiet.Load()
	t.Cleanup(func() {
		mu.Lock()
		out, progressOK, progress = prevOut, prevOK, prevProgress
		mu.Unlock()
		quiet.Store(prevQuiet)
	})
	return &buf
}

func TestProgress_NonTerminalWritesNothing(t *testing.T) {
	buf := capture(t, false)
	SetProgress("build", "Rendering %d/%d", 1, 2)
	ClearProgress()
	if buf.Len() != 0 {
		t.Fatalf("expected no output without a terminal, got %q", buf.String())
	}
}

func TestClearProgress_NoActiveLineWritesNothing(t *testing.T) {
	buf := capture(t, true)
	ClearProgress()
	if buf.Len() != 0 {
		t.Fatalf("expected no output, got %q", buf.String())
	}
}

func TestLog_RedrawsActiveProgress(t *testing.T) {
	buf := capture(t, true)
	SetProgress("build", "working")
	buf.Reset()

	Log("links", "hello")
	got := buf.String()
	if !strings.HasPrefix(got, "\r\033[K") {
		t.Errorf("expected the progress line to be cleared first, got %q", got)
	}
	if !strings.Contains(got, "hello\n") {
		t.Errorf("expected the log line, got %q", got)
	}
	if !strings.HasSuffix(got, "working") {
		t.Errorf("expected the progress line to be redrawn last, got %q", got)
	}

	buf.Reset()
	ClearProgress()
	if buf.String() != "\r\033[K" {
		t.Errorf("ClearProgress = %q, want a single clear sequence", buf.String())
	}
}

func TestQuiet_SilencesLogAndProgressNotWarn(t *testing.T) {
	buf := capture(t, true)
	SetQuiet(true)

	Log("links", "info")
	SetProgress("build", "working")
	if buf.Len() != 0 {
		t.Fatalf("expected quiet Log and SetProgress to write nothing, got %q", buf.String())
	}

	Warn("git", "careful")
	Error("build", "broken")
	got := buf.String()
	if !strings.Contains(got, "careful") || !strings.Contains(got, "broken") {
		t.Errorf("expected Warn and Error to print in quiet mode, got %q", got)
	}
}

func TestPrint_AddsTrailingNewline(t *testing.T) {
	buf := capture(t, false)
	Print("a\nb")
	Print("")
	if buf.String() != "a\nb\n" {
		t.Errorf("Print = %q, want %q", buf.String(), "a\nb\n")
	}
}

func TestDisableProgress_ClearsAndStaysOff(t *testing.T) {
	buf := capture(t, true)
	SetProgress("build", "working")
	DisableProgress()
	buf.Reset()
	SetProgress("build", "again")
	if buf.Len() != 0 {
		t.Errorf("expected no progress after DisableProgress, got %q", buf.String())
	}
}

func TestWarnCount_ConcurrentIncrements(t *testing.T) {
	capture(t, false)
	before := WarnCount()
	var wg sync.WaitGroup
	for range 100 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			Warn("test", "w")
		}()
	}
	wg.Wait()
	if got := WarnCount() - before; got != 100 {
		t.Errorf("WarnCount delta = %d, want 100", got)
	}
}

func TestDisplayPath(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	inside := filepath.Join(wd, "content", "docs", "a.md")
	if got := DisplayPath(inside); got != "content/docs/a.md" {
		t.Errorf("inside working dir: got %q", got)
	}
	outside := filepath.Join(filepath.Dir(wd), "elsewhere", "b.md")
	if got := DisplayPath(outside); got != outside {
		t.Errorf("outside working dir: got %q, want it unchanged", got)
	}
	if got := DisplayPath("docs/c.md"); got != "docs/c.md" {
		t.Errorf("relative path: got %q", got)
	}
	if got := DisplayPath("sarde.yaml: plugins.config.x"); got != "sarde.yaml: plugins.config.x" {
		t.Errorf("config reference: got %q", got)
	}
}
