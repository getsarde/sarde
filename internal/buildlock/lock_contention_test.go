package buildlock

import (
	"bufio"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/getsarde/sarde/internal/consts"
)

// TestLockPrimitive_SecondHandleFailsMetadataReadable exercises the raw OS
// primitives on two independent handles in one process: the second lock
// attempt must fail immediately (this non-reentrancy is why the package
// keeps an in-process registry), and a plain read of the metadata region
// must succeed while the lock is held (proof of the reserved-offset design
// on Windows; flock never blocks reads on Unix).
func TestLockPrimitive_SecondHandleFailsMetadataReadable(t *testing.T) {
	path := filepath.Join(t.TempDir(), consts.FileOutputLock)

	f1, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer f1.Close()
	if _, err := f1.WriteString(`{"pid":123}`); err != nil {
		t.Fatal(err)
	}
	if err := lockFile(f1); err != nil {
		t.Fatalf("first handle lock: %v", err)
	}

	f2, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		t.Fatal(err)
	}
	defer f2.Close()
	if err := lockFile(f2); err == nil {
		unlockFile(f2)
		t.Fatal("second handle lock should fail while first holds it")
	}

	f3, err := os.Open(path)
	if err != nil {
		t.Fatalf("open for read while locked: %v", err)
	}
	data, err := io.ReadAll(f3)
	f3.Close()
	if err != nil {
		t.Fatalf("read metadata while locked: %v", err)
	}
	if !strings.Contains(string(data), "123") {
		t.Errorf("metadata not readable while locked: %q", data)
	}

	if err := unlockFile(f1); err != nil {
		t.Fatalf("unlock: %v", err)
	}
	if err := lockFile(f2); err != nil {
		t.Fatalf("lock after first released: %v", err)
	}
	unlockFile(f2)
}

// TestHelperHoldLock is not a real test: it is re-executed as a child
// process by the cross-process tests below. It acquires the lock on the
// directory named in the environment, reports, and holds until stdin closes.
func TestHelperHoldLock(t *testing.T) {
	if os.Getenv("SARDE_BUILDLOCK_HELPER") != "1" {
		t.Skip("helper process only")
	}
	dir := os.Getenv("SARDE_BUILDLOCK_DIR")
	l, err := Acquire(dir, "helper")
	if err != nil {
		os.Stdout.WriteString("ACQUIRE_FAILED: " + err.Error() + "\n")
		os.Exit(2)
	}
	os.Stdout.WriteString("HELPER_LOCKED\n")
	io.Copy(io.Discard, os.Stdin)
	l.Release()
	os.Exit(0)
}

// startLockHolder re-executes the test binary as a child process that holds
// the lock on dir. Returns the command and its stdin pipe; closing the pipe
// makes the child release gracefully and exit.
func startLockHolder(t *testing.T, dir string) (*exec.Cmd, io.WriteCloser) {
	t.Helper()

	cmd := exec.Command(os.Args[0], "-test.run=^TestHelperHoldLock$", "-test.v")
	cmd.Env = append(os.Environ(), "SARDE_BUILDLOCK_HELPER=1", "SARDE_BUILDLOCK_DIR="+dir)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatalf("starting helper: %v", err)
	}
	t.Cleanup(func() {
		stdin.Close()
		cmd.Process.Kill()
		cmd.Wait()
	})

	lines := make(chan string, 16)
	go func() {
		sc := bufio.NewScanner(stdout)
		for sc.Scan() {
			lines <- sc.Text()
		}
		close(lines)
	}()

	deadline := time.After(30 * time.Second)
	for {
		select {
		case line, ok := <-lines:
			if !ok {
				t.Fatal("helper exited before acquiring the lock")
			}
			if strings.Contains(line, "HELPER_LOCKED") {
				// Keep draining so the child never blocks on a full pipe.
				go func() {
					for range lines {
					}
				}()
				return cmd, stdin
			}
			if strings.Contains(line, "ACQUIRE_FAILED") {
				t.Fatalf("helper could not acquire: %s", line)
			}
		case <-deadline:
			t.Fatal("timeout waiting for helper to lock")
		}
	}
}

// acquireWithRetry polls Acquire until it succeeds or the deadline passes.
// Used after killing the holder: the kernel releases the lock at process
// death, but Wait/handle teardown timing deserves a little slack.
func acquireWithRetry(t *testing.T, dir string, timeout time.Duration) *Lock {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for {
		l, err := Acquire(dir, "test")
		if err == nil {
			return l
		}
		if time.Now().After(deadline) {
			t.Fatalf("Acquire never succeeded: %v", err)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func TestCrossProcess_ContentionAndGracefulRelease(t *testing.T) {
	if testing.Short() {
		t.Skip("re-exec test skipped in -short mode")
	}
	dir := filepath.Join(t.TempDir(), "dist")

	cmd, stdin := startLockHolder(t, dir)

	_, err := Acquire(dir, "test")
	el, ok := IsLocked(err)
	if !ok {
		t.Fatalf("err = %v, want *ErrLocked while helper holds the lock", err)
	}
	if el.Meta.Cmd != "helper" {
		t.Errorf("Meta.Cmd = %q, want helper", el.Meta.Cmd)
	}
	if el.Meta.PID != cmd.Process.Pid {
		t.Errorf("Meta.PID = %d, want helper pid %d", el.Meta.PID, cmd.Process.Pid)
	}

	// Graceful release: closing stdin makes the helper Release and exit.
	stdin.Close()
	cmd.Wait()

	l := acquireWithRetry(t, dir, 5*time.Second)
	l.Release()
}

func TestCrossProcess_HardKillReleasesLock(t *testing.T) {
	if testing.Short() {
		t.Skip("re-exec test skipped in -short mode")
	}
	dir := filepath.Join(t.TempDir(), "dist")

	cmd, _ := startLockHolder(t, dir)

	if _, err := Acquire(dir, "test"); err == nil {
		t.Fatal("Acquire should fail while helper holds the lock")
	}

	// Hard kill: no defers run in the helper, simulating a crash. The kernel
	// must release the lock so no manual cleanup is ever needed.
	if err := cmd.Process.Kill(); err != nil {
		t.Fatalf("kill helper: %v", err)
	}
	cmd.Wait()

	l := acquireWithRetry(t, dir, 5*time.Second)
	l.Release()
}
