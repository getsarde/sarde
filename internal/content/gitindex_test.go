package content

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// initTestRepo creates a git repository with local identity config so commits
// work regardless of the machine's global git setup.
func initTestRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	gitRun(t, dir, "init", "-q")
	gitRun(t, dir, "config", "user.email", "test@example.com")
	gitRun(t, dir, "config", "user.name", "Test")
	gitRun(t, dir, "config", "commit.gpgsign", "false")
	return dir
}

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// commitAt commits everything staged with a fixed author and committer date.
func commitAt(t *testing.T, dir, message string, when time.Time) {
	t.Helper()
	stamp := when.Format(time.RFC3339)
	cmd := exec.Command("git", "commit", "-q", "-m", message, "--date", stamp)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_COMMITTER_DATE="+stamp)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
}

func TestGitIndex_ResolvesCommitTimeNotMtime(t *testing.T) {
	repo := initTestRepo(t)
	want := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)

	p := writeFileTest(t, repo, "page.md", "hello")
	gitRun(t, repo, "add", "page.md")
	commitAt(t, repo, "add page", want)

	// Touch the file well after the commit. The commit time must still win,
	// which is the whole point of the git strategy.
	later := time.Now().Add(48 * time.Hour)
	if err := os.Chtimes(p, later, later); err != nil {
		t.Fatal(err)
	}

	idx, err := BuildGitLastModIndex(repo, []string{p})
	if err != nil {
		t.Fatalf("BuildGitLastModIndex: %v", err)
	}
	if !idx.Available() {
		t.Fatal("index should be available inside a repo with commits")
	}

	got, ok := idx.Lookup(p)
	if !ok {
		t.Fatal("committed file missing from index")
	}
	if !got.Equal(want) {
		t.Errorf("Lookup = %v, want %v", got.UTC(), want)
	}

	resolved := GetLastUpdated(p, "git", idx)
	if resolved == nil || !resolved.Equal(want) {
		t.Errorf("GetLastUpdated = %v, want %v", resolved, want)
	}
}

// The per-file path used to run git in the process working directory, so a
// build started from outside the repo reported the file as outside the
// repository and silently fell back to mtime.
func TestGitIndex_WorksWhenProcessCWDIsOutsideRepo(t *testing.T) {
	repo := initTestRepo(t)
	want := time.Date(2026, 1, 2, 9, 30, 0, 0, time.UTC)

	p := writeFileTest(t, repo, "page.md", "hello")
	gitRun(t, repo, "add", "page.md")
	commitAt(t, repo, "add page", want)

	t.Chdir(t.TempDir()) // anywhere that is not the repo

	idx, err := BuildGitLastModIndex(repo, []string{p})
	if err != nil {
		t.Fatalf("BuildGitLastModIndex: %v", err)
	}
	if got, ok := idx.Lookup(p); !ok || !got.Equal(want) {
		t.Errorf("Lookup = %v (ok=%v), want %v", got, ok, want)
	}

	// The single-file path must survive the same conditions.
	if got, ok := gitLastCommitTime(p); !ok || !got.Equal(want) {
		t.Errorf("gitLastCommitTime = %v (ok=%v), want %v", got, ok, want)
	}
}

func TestGitIndex_MostRecentCommitWinsPerPath(t *testing.T) {
	repo := initTestRepo(t)
	first := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	second := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	a := writeFileTest(t, repo, "a.md", "one")
	b := writeFileTest(t, repo, "b.md", "one")
	gitRun(t, repo, "add", ".")
	commitAt(t, repo, "first", first)

	writeFileTest(t, repo, "a.md", "two")
	gitRun(t, repo, "add", "a.md")
	commitAt(t, repo, "second", second)

	idx, err := BuildGitLastModIndex(repo, []string{a, b})
	if err != nil {
		t.Fatalf("BuildGitLastModIndex: %v", err)
	}

	if got, _ := idx.Lookup(a); !got.Equal(second) {
		t.Errorf("a.md = %v, want most recent %v", got.UTC(), second)
	}
	if got, _ := idx.Lookup(b); !got.Equal(first) {
		t.Errorf("b.md = %v, want %v", got.UTC(), first)
	}
}

// Renames resolve at the rename commit: with --name-status the destination is
// the line's last field, and even --name-only listed the destination, so this
// guards behavior the index has always had against a parser regression.
func TestGitIndex_RenamedFileResolvesAtRenameCommit(t *testing.T) {
	repo := initTestRepo(t)
	t1 := time.Date(2026, 1, 5, 8, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 4, 20, 16, 45, 0, 0, time.UTC)

	writeFileTest(t, repo, "old.md", "same body, moved later")
	gitRun(t, repo, "add", "old.md")
	commitAt(t, repo, "add old", t1)

	gitRun(t, repo, "mv", "old.md", "new.md")
	commitAt(t, repo, "rename to new", t2)

	newPath := filepath.Join(repo, "new.md")
	idx, err := BuildGitLastModIndex(repo, []string{newPath})
	if err != nil {
		t.Fatalf("BuildGitLastModIndex: %v", err)
	}
	if got, ok := idx.Lookup(newPath); !ok || !got.Equal(t2) {
		t.Errorf("renamed file = %v (ok=%v), want rename-commit time %v", got.UTC(), ok, t2)
	}
}

// A rename that also edits content (lower similarity, still detected or split
// into A + D) must likewise resolve at the commit that produced the new path.
func TestGitIndex_RenameWithEditResolvesAtRenameCommit(t *testing.T) {
	repo := initTestRepo(t)
	t1 := time.Date(2026, 2, 1, 10, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 5, 10, 11, 30, 0, 0, time.UTC)

	writeFileTest(t, repo, "old.md", "line one\nline two\nline three\nline four\n")
	gitRun(t, repo, "add", "old.md")
	commitAt(t, repo, "add old", t1)

	gitRun(t, repo, "mv", "old.md", "new.md")
	writeFileTest(t, repo, "new.md", "line one\nline two\nline three\nchanged\n")
	gitRun(t, repo, "add", "new.md")
	commitAt(t, repo, "rename and edit", t2)

	newPath := filepath.Join(repo, "new.md")
	idx, err := BuildGitLastModIndex(repo, []string{newPath})
	if err != nil {
		t.Fatalf("BuildGitLastModIndex: %v", err)
	}
	if got, ok := idx.Lookup(newPath); !ok || !got.Equal(t2) {
		t.Errorf("renamed+edited file = %v (ok=%v), want %v", got.UTC(), ok, t2)
	}
}

// A file whose newest git event is a deletion, but which still exists on disk
// (kept untracked via git rm --cached), must miss the index and fall back to
// mtime. Recording the deletion time, or letting an older commit claim the
// path, would both be wrong.
func TestGitIndex_DeletedButOnDiskFallsBackToMtime(t *testing.T) {
	repo := initTestRepo(t)
	t1 := time.Date(2026, 1, 10, 9, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 3, 3, 13, 0, 0, 0, time.UTC)

	p := writeFileTest(t, repo, "page.md", "hello")
	gitRun(t, repo, "add", "page.md")
	commitAt(t, repo, "add page", t1)

	gitRun(t, repo, "rm", "--cached", "-q", "page.md")
	commitAt(t, repo, "untrack page", t2)

	// A distinct mtime proves the fallback source.
	stamp := time.Date(2026, 6, 6, 6, 0, 0, 0, time.UTC)
	if err := os.Chtimes(p, stamp, stamp); err != nil {
		t.Fatal(err)
	}

	idx, err := BuildGitLastModIndex(repo, []string{p})
	if err != nil {
		t.Fatalf("BuildGitLastModIndex: %v", err)
	}
	if got, ok := idx.Lookup(p); ok {
		t.Errorf("deleted-in-git file resolved to %v, want a miss", got.UTC())
	}

	resolved := GetLastUpdated(p, "git", idx)
	if resolved == nil {
		t.Fatal("expected mtime fallback, got nil")
	}
	if !resolved.Equal(stamp) {
		t.Errorf("fallback = %v, want mtime %v (not commit times %v / %v)",
			resolved.UTC(), stamp, t1, t2)
	}
}

func TestGitIndex_UntrackedFileFallsBackToMtime(t *testing.T) {
	repo := initTestRepo(t)
	tracked := writeFileTest(t, repo, "tracked.md", "hello")
	gitRun(t, repo, "add", "tracked.md")
	commitAt(t, repo, "add", time.Date(2026, 2, 2, 0, 0, 0, 0, time.UTC))

	untracked := writeFileTest(t, repo, "untracked.md", "hello")

	idx, err := BuildGitLastModIndex(repo, []string{tracked, untracked})
	if err != nil {
		t.Fatalf("BuildGitLastModIndex: %v", err)
	}
	if _, ok := idx.Lookup(untracked); ok {
		t.Error("untracked file should miss the index")
	}

	// Missing from the index means mtime, never nil.
	got := GetLastUpdated(untracked, "git", idx)
	if got == nil {
		t.Fatal("expected mtime fallback for untracked file")
	}
	info, err := os.Stat(untracked)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Equal(info.ModTime()) {
		t.Errorf("fallback = %v, want mtime %v", got, info.ModTime())
	}
}

func TestGitIndex_NotARepository(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	p := writeFileTest(t, dir, "page.md", "hello")

	idx, err := BuildGitLastModIndex(dir, []string{p})
	if err == nil {
		t.Fatal("expected an error outside a repository")
	}
	if idx.Available() {
		t.Error("index must not report available outside a repository")
	}
	// An unavailable index must not be retried per file.
	if got := GetLastUpdated(p, "git", idx); got == nil {
		t.Error("expected mtime fallback when the index is unavailable")
	}
}

func TestGitIndex_DetectsShallowClone(t *testing.T) {
	origin := initTestRepo(t)
	for i, when := range []time.Time{
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC),
	} {
		writeFileTest(t, origin, "page.md", string(rune('a'+i)))
		gitRun(t, origin, "add", "page.md")
		commitAt(t, origin, "commit", when)
	}

	clone := filepath.Join(t.TempDir(), "shallow")
	cmd := exec.Command("git", "clone", "-q", "--depth", "1", "file://"+filepath.ToSlash(origin), clone)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("shallow clone unsupported here: %v\n%s", err, out)
	}

	idx, err := BuildGitLastModIndex(clone, []string{filepath.Join(clone, "page.md")})
	if err != nil {
		t.Fatalf("BuildGitLastModIndex: %v", err)
	}
	if !idx.Shallow() {
		t.Error("expected the clone to be reported as shallow")
	}
}

func TestGitIndex_NilAndUnavailableAreDistinct(t *testing.T) {
	var nilIdx *GitLastModIndex
	if nilIdx.Available() {
		t.Error("nil index must not be available")
	}
	if nilIdx.Shallow() {
		t.Error("nil index must not report shallow")
	}
	if _, ok := nilIdx.Lookup("anything"); ok {
		t.Error("nil index must not resolve lookups")
	}
	if nilIdx.Stale() {
		t.Error("nil index must not report stale")
	}
}
