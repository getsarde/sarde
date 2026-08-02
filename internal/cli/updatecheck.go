package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/creativeprojects/go-selfupdate"
	"github.com/getsarde/sarde/internal/devlog"
	"github.com/getsarde/sarde/internal/version"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

const (
	// updateCheckInterval is the minimum time between passive release
	// lookups. The check runs at most once per interval across all commands.
	updateCheckInterval = 24 * time.Hour
	// passiveDetectTimeout bounds the background lookup itself.
	passiveDetectTimeout = 3 * time.Second
	// passiveNotifyWait is how long the end of a build waits for an
	// in-flight lookup started at the beginning of the run. Sized to cover
	// a cold GitHub lookup (~400ms) even when the build itself is instant.
	passiveNotifyWait = 500 * time.Millisecond
)

// updateCheckState is the on-disk cache at ~/.sarde/update-check.json.
// LastChecked records the last lookup attempt (successful or not), so a slow
// or offline network costs at most one bounded wait per interval.
type updateCheckState struct {
	LastChecked   time.Time `json:"last_checked"`
	LatestVersion string    `json:"latest_version"`
	Notified      bool      `json:"notified"`
}

// isStale reports whether the cached lookup is older than the check interval.
func (s *updateCheckState) isStale(now time.Time) bool {
	return now.Sub(s.LastChecked) > updateCheckInterval
}

// updateCachePath returns ~/.sarde/update-check.json, following the same
// home-directory convention as the license store.
func updateCachePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".sarde", "update-check.json"), nil
}

func loadUpdateCache(path string) (*updateCheckState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s updateCheckState
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func saveUpdateCache(path string, s *updateCheckState) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// shouldSkipUpdateCheck gates the passive check. It skips dev builds (no
// release baseline), CI and explicitly opted-out environments, quiet mode,
// and non-interactive stderr, so automation never sees the notice or the
// network call.
func shouldSkipUpdateCheck(ver string, getenv func(string) string, quiet, isTTY bool) bool {
	if ver == "dev" {
		return true
	}
	if getenv("CI") != "" || getenv("SARDE_NO_UPDATE_CHECK") != "" {
		return true
	}
	if quiet || !isTTY {
		return true
	}
	return false
}

// updateNotice returns the one-line notice when the cached latest version is
// strictly newer than the running version. Unparsable versions produce no
// notice rather than a false alarm.
func updateNotice(current, latest string) (string, bool) {
	if latest == "" {
		return "", false
	}
	cur, err := semver.NewVersion(current)
	if err != nil {
		return "", false
	}
	lat, err := semver.NewVersion(latest)
	if err != nil {
		return "", false
	}
	if !lat.GreaterThan(cur) {
		return "", false
	}
	return fmt.Sprintf("%s A new version of sarde is available: %s (run %s)",
		devlog.Cyan("↑"), devlog.Cyan("v"+latest), devlog.Bold("sarde update")), true
}

// fetchLatestVersion looks up the newest release version. Seamed as a
// function variable so tests can drive the begin/finish flow offline.
var fetchLatestVersion = func() (string, bool) {
	source, err := selfupdate.NewGitHubSource(selfupdate.GitHubConfig{})
	if err != nil {
		return "", false
	}
	updater, err := selfupdate.NewUpdater(selfupdate.Config{
		Source:  source,
		Filters: []string{fmt.Sprintf("sarde_%s_%s", runtime.GOOS, runtime.GOARCH)},
	})
	if err != nil {
		return "", false
	}
	ctx, cancel := context.WithTimeout(context.Background(), passiveDetectTimeout)
	defer cancel()
	latest, found, err := updater.DetectLatest(ctx, selfupdate.ParseSlug(updateRepoSlug))
	if err != nil || !found {
		return "", false
	}
	return latest.Version(), true
}

type fetchResult struct {
	version string
	ok      bool
}

// pendingUpdateCheck carries a passive check across a build run: begun before
// the build so the network lookup overlaps with the build work, finished
// after the summary. The goroutine never touches the cache file; it reports
// over the channel and the foreground does the single write, so there is
// exactly one writer per process.
type pendingUpdateCheck struct {
	path   string
	state  updateCheckState
	result chan fetchResult // nil when the cache was fresh and no lookup started
}

// beginUpdateCheck gates the passive check and, when the cache is stale,
// starts the release lookup in the background. Returns nil when the check is
// skipped; finishAndNotify is nil-receiver safe so call sites stay
// unconditional.
func beginUpdateCheck(cmd *cobra.Command) *pendingUpdateCheck {
	quiet, _ := cmd.Flags().GetBool("quiet")
	isTTY := term.IsTerminal(int(os.Stderr.Fd()))
	if shouldSkipUpdateCheck(version.Version, os.Getenv, quiet, isTTY) {
		return nil
	}
	path, err := updateCachePath()
	if err != nil {
		return nil
	}
	state, err := loadUpdateCache(path)
	if err != nil || state == nil {
		state = &updateCheckState{}
	}
	return newPendingUpdateCheck(path, *state, time.Now())
}

// newPendingUpdateCheck is the post-gate half of beginUpdateCheck, split out
// so tests can drive it with an explicit path, state, and clock.
func newPendingUpdateCheck(path string, state updateCheckState, now time.Time) *pendingUpdateCheck {
	p := &pendingUpdateCheck{path: path, state: state}
	if state.isStale(now) {
		p.result = make(chan fetchResult, 1)
		go func() {
			v, ok := fetchLatestVersion()
			p.result <- fetchResult{version: v, ok: ok}
		}()
	}
	return p
}

// finishAndNotify completes the passive check at the end of a build: it
// waits briefly for an in-flight lookup, prints the notice when one is due,
// and persists the cache.
func (p *pendingUpdateCheck) finishAndNotify() {
	if p == nil {
		return
	}
	p.finish(os.Stderr, passiveNotifyWait, time.Now())
}

func (p *pendingUpdateCheck) finish(w io.Writer, maxWait time.Duration, now time.Time) {
	changed := false
	if p.result != nil {
		select {
		case res := <-p.result:
			if res.ok {
				if res.version != p.state.LatestVersion {
					p.state.Notified = false
				}
				p.state.LatestVersion = res.version
			}
			p.state.LastChecked = now
		case <-time.After(maxWait):
			// Lookup still in flight; its result is lost at process exit.
			// Record the attempt anyway so a slow network pays this wait at
			// most once per interval instead of on every build.
			p.state.LastChecked = now
		}
		changed = true
	}
	if msg, ok := updateNotice(version.Version, p.state.LatestVersion); ok && !p.state.Notified {
		fmt.Fprintln(w, msg)
		p.state.Notified = true
		changed = true
	}
	if changed {
		_ = saveUpdateCache(p.path, &p.state)
	}
}

// maybeCheckForUpdate is the dev-server variant of the passive check. The
// process is long-lived, so the notice prints instantly from the previous
// run's cache and a detached goroutine refreshes the cache for next time.
// The foreground finishes all cache reads and writes before the goroutine
// starts, so the two never touch the file concurrently.
func maybeCheckForUpdate(cmd *cobra.Command) {
	quiet, _ := cmd.Flags().GetBool("quiet")
	isTTY := term.IsTerminal(int(os.Stderr.Fd()))
	if shouldSkipUpdateCheck(version.Version, os.Getenv, quiet, isTTY) {
		return
	}
	path, err := updateCachePath()
	if err != nil {
		return
	}
	state, err := loadUpdateCache(path)
	if err != nil || state == nil {
		state = &updateCheckState{}
	}

	if msg, ok := updateNotice(version.Version, state.LatestVersion); ok && !state.Notified {
		fmt.Fprintln(os.Stderr, msg)
		state.Notified = true
		if err := saveUpdateCache(path, state); err != nil {
			return
		}
	}

	if !state.isStale(time.Now()) {
		return
	}
	go refreshUpdateCache(path, *state)
}

// refreshUpdateCache looks up the latest release and rewrites the cache.
// Errors are swallowed: a failed passive check must never surface to the
// user. A version the user was already notified about stays marked; a newer
// one resets the flag so it gets announced once.
func refreshUpdateCache(path string, prev updateCheckState) {
	v, ok := fetchLatestVersion()
	if !ok {
		return
	}
	state := &updateCheckState{
		LastChecked:   time.Now(),
		LatestVersion: v,
	}
	if prev.LatestVersion == v {
		state.Notified = prev.Notified
	}
	_ = saveUpdateCache(path, state)
}
