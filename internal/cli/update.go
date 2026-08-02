package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/creativeprojects/go-selfupdate"
	"github.com/getsarde/sarde/internal/devlog"
	"github.com/getsarde/sarde/internal/updatesign"
	"github.com/getsarde/sarde/internal/version"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update sarde to the latest version",
	Long:  "Check for and install the latest version of sarde from GitHub Releases.",
	RunE:  runUpdate,
}

func init() {
	updateCmd.Flags().Bool("check", false, "Only check for updates without installing")
	updateCmd.Flags().BoolP("yes", "y", false, "Skip the confirmation prompt")
	rootCmd.AddCommand(updateCmd)
}

const (
	// detectTimeout bounds the GitHub release lookup.
	detectTimeout = 30 * time.Second
	// applyTimeout bounds the download plus binary replacement.
	applyTimeout = 5 * time.Minute
	// releaseNotesMaxLines caps the release notes shown before the prompt.
	releaseNotesMaxLines = 10
	// updateRepoSlug is the GitHub repository updates are fetched from.
	updateRepoSlug = "getsarde/sarde"
)

// updateRelease is the subset of release information the update flow needs,
// decoupled from selfupdate.Release (whose version field is unexported and
// cannot be fabricated in tests).
type updateRelease struct {
	version string
	notes   string
	url     string
	raw     *selfupdate.Release
}

// selfUpdater seams the network-facing updater so the command flow can be
// tested offline with a fake.
type selfUpdater interface {
	detectLatest(ctx context.Context) (*updateRelease, bool, error)
	updateTo(ctx context.Context, rel *updateRelease, exePath string) error
}

type realUpdater struct{ up *selfupdate.Updater }

func (r realUpdater) detectLatest(ctx context.Context) (*updateRelease, bool, error) {
	latest, found, err := r.up.DetectLatest(ctx, selfupdate.ParseSlug(updateRepoSlug))
	if err != nil || !found {
		return nil, found, err
	}
	return &updateRelease{
		version: latest.Version(),
		notes:   latest.ReleaseNotes,
		url:     latest.URL,
		raw:     latest,
	}, true, nil
}

func (r realUpdater) updateTo(ctx context.Context, rel *updateRelease, exePath string) error {
	return r.up.UpdateTo(ctx, rel.raw, exePath)
}

// newSelfUpdater constructs the real GitHub-backed updater. Tests swap this
// function variable for a fake.
var newSelfUpdater = func() (selfUpdater, error) {
	source, err := selfupdate.NewGitHubSource(selfupdate.GitHubConfig{})
	if err != nil {
		return nil, fmt.Errorf("initializing update source: %w", err)
	}
	up, err := selfupdate.NewUpdater(selfupdate.Config{
		Source:    source,
		Filters:   []string{fmt.Sprintf("sarde_%s_%s", runtime.GOOS, runtime.GOARCH)},
		Validator: newReleaseValidator(),
	})
	if err != nil {
		return nil, fmt.Errorf("initializing updater: %w", err)
	}
	return realUpdater{up}, nil
}

// releaseIsNewer reports whether latest is strictly newer than current.
// An unparsable current version never blocks an update; an unparsable latest
// version never triggers one.
func releaseIsNewer(current, latest string) bool {
	cur, err := semver.NewVersion(current)
	if err != nil {
		return true
	}
	lat, err := semver.NewVersion(latest)
	if err != nil {
		return false
	}
	return lat.GreaterThan(cur)
}

// packageManager identifies how the running binary was installed, when a
// package manager owns it. Self-replacing a managed binary breaks the
// manager's bookkeeping, so update defers to the manager instead.
type packageManager int

const (
	pmNone packageManager = iota
	pmHomebrew
	pmScoop
	pmChocolatey
	pmWinget
	pmSystem
)

// detectPackageManager classifies the (symlink-resolved) executable path.
// Homebrew, Scoop, and Chocolatey use well-known roots; winget detection is
// best-effort. Paths under /usr/bin or /usr/lib are assumed to belong to a
// system package manager, since manually installed static binaries
// conventionally live in /usr/local/bin or under the home directory.
func detectPackageManager(exePath string) packageManager {
	for _, prefix := range []string{
		"/opt/homebrew/",
		"/usr/local/Cellar/",
		"/home/linuxbrew/.linuxbrew/",
	} {
		if strings.HasPrefix(exePath, prefix) {
			return pmHomebrew
		}
	}
	// Windows paths are case-insensitive; normalize before matching.
	lower := strings.ToLower(filepath.ToSlash(exePath))
	if strings.Contains(lower, "/scoop/apps/") {
		return pmScoop
	}
	if strings.HasPrefix(lower, "c:/programdata/chocolatey/") {
		return pmChocolatey
	}
	if strings.Contains(lower, "/microsoft/winget/packages/") {
		return pmWinget
	}
	if strings.HasPrefix(exePath, "/usr/bin/") || strings.HasPrefix(exePath, "/usr/lib/") {
		return pmSystem
	}
	return pmNone
}

// upgradeHint returns the message shown instead of self-updating when a
// package manager owns the binary. Empty for pmNone.
func (pm packageManager) upgradeHint() string {
	switch pm {
	case pmHomebrew:
		return "Installed via Homebrew. Run `brew upgrade sarde` to update."
	case pmScoop:
		return "Installed via Scoop. Run `scoop update sarde` to update."
	case pmChocolatey:
		return "Installed via Chocolatey. Run `choco upgrade sarde` to update."
	case pmWinget:
		return "Installed via winget. Run `winget upgrade sarde` to update."
	case pmSystem:
		return "Installed by a system package manager. Update it through your package manager."
	}
	return ""
}

// resolvedExecutable returns the current executable path with symlinks
// resolved. Homebrew on Intel macs links /usr/local/bin/sarde into the Cellar,
// and os.Executable can return the symlink itself, so the raw path would miss
// prefix checks. On resolution failure the unresolved path is returned rather
// than failing the command.
func resolvedExecutable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if resolved, rerr := filepath.EvalSymlinks(exe); rerr == nil {
		exe = resolved
	}
	return exe, nil
}

// checksumsFilename is the GoReleaser checksum artifact covering every
// release asset; its detached ed25519 signature lives in the sibling .sig
// asset produced by tools/sarde-release-sign.
const checksumsFilename = "checksums.txt"

// ed25519SigValidator verifies a file against its detached ed25519 signature
// using the trusted release keys embedded in internal/updatesign.
type ed25519SigValidator struct{}

func (ed25519SigValidator) Validate(_ string, release, signature []byte) error {
	return updatesign.Verify(release, signature)
}

func (ed25519SigValidator) GetValidationAssetName(releaseFilename string) string {
	return releaseFilename + ".sig"
}

// newReleaseValidator builds the full validation chain: each asset is checked
// against checksums.txt, and checksums.txt itself is checked against its
// ed25519 signature. Fail-closed: a release without a signature asset fails
// validation. Mirrors the library's own NewChecksumWithECDSAValidator
// composition.
func newReleaseValidator() selfupdate.Validator {
	return new(selfupdate.PatternValidator).
		Add(checksumsFilename, ed25519SigValidator{}).
		SkipValidation("*.sig").
		Add("*", &selfupdate.ChecksumValidator{UniqueFilename: checksumsFilename})
}

// truncateNotes trims release notes to at most maxLines lines, appending an
// ellipsis marker when content was cut. Truncation happens on line boundaries
// so markdown links and multi-byte characters are never split.
func truncateNotes(notes string, maxLines int) string {
	notes = strings.TrimRight(notes, "\n\r \t")
	if notes == "" || maxLines <= 0 {
		return ""
	}
	lines := strings.Split(notes, "\n")
	if len(lines) <= maxLines {
		return notes
	}
	return strings.Join(lines[:maxLines], "\n") + "\n  ..."
}

// permissionHint maps a permission-denied error from replacing the binary to
// actionable advice. Returns false for every other error.
func permissionHint(err error) (string, bool) {
	if err == nil || !errors.Is(err, os.ErrPermission) {
		return "", false
	}
	return "Permission denied while replacing the binary. Check who owns it " +
		"(e.g. `ls -l $(which sarde)`) and rerun with elevated privileges " +
		"(e.g. `sudo sarde update`) or reinstall to a user-writable location.", true
}

// confirmUpdate decides whether the update should proceed. yes bypasses the
// prompt entirely. When stdin is not a terminal the prompt cannot be answered,
// so instead of treating EOF as a silent refusal the caller gets an error
// telling them to pass --yes.
func confirmUpdate(r io.Reader, w io.Writer, targetVersion string, yes, isTTY bool) (bool, error) {
	if yes {
		return true, nil
	}
	if !isTTY {
		return false, fmt.Errorf("stdin is not a terminal; pass --yes to update non-interactively")
	}
	fmt.Fprintf(w, "\n  Update to %s? [y/N] ", devlog.Cyan("v"+targetVersion))
	reader := bufio.NewReader(r)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes", nil
}

func runUpdate(cmd *cobra.Command, _ []string) error {
	checkOnly, _ := cmd.Flags().GetBool("check")
	assumeYes, _ := cmd.Flags().GetBool("yes")

	if version.Version == "dev" {
		return fmt.Errorf("cannot update a dev build; install a release version first")
	}

	exe, err := resolvedExecutable()
	if err != nil {
		return fmt.Errorf("locating current binary: %w", err)
	}

	if pm := detectPackageManager(exe); pm != pmNone {
		fmt.Fprintln(os.Stderr, pm.upgradeHint())
		return nil
	}

	devlog.Log("update", "Checking for updates...")

	updater, err := newSelfUpdater()
	if err != nil {
		return err
	}

	detectCtx, cancelDetect := context.WithTimeout(context.Background(), detectTimeout)
	defer cancelDetect()
	latest, found, err := updater.detectLatest(detectCtx)
	if err != nil {
		return fmt.Errorf("checking for updates: %w", err)
	}

	if !found {
		devlog.Log("update", "No releases found")
		return nil
	}

	current := version.Version
	if !releaseIsNewer(current, latest.version) {
		fmt.Fprintf(os.Stderr, "%s Already up to date (%s)\n",
			devlog.Green("✓"), devlog.Cyan("v"+current))
		return nil
	}

	fmt.Fprintf(os.Stderr, "  Current: %s\n", devlog.Cyan("v"+current))
	fmt.Fprintf(os.Stderr, "  Latest:  %s\n", devlog.Cyan("v"+latest.version))

	if notes := truncateNotes(latest.notes, releaseNotesMaxLines); notes != "" {
		fmt.Fprintf(os.Stderr, "\n%s\n", notes)
	}
	if latest.url != "" {
		fmt.Fprintf(os.Stderr, "\n  Release page: %s\n", devlog.Dim(latest.url))
	}

	if checkOnly {
		fmt.Fprintf(os.Stderr, "\n  Run %s to install\n", devlog.Bold("sarde update"))
		return nil
	}

	proceed, err := confirmUpdate(os.Stdin, os.Stderr, latest.version, assumeYes,
		term.IsTerminal(int(os.Stdin.Fd())))
	if err != nil {
		return err
	}
	if !proceed {
		fmt.Fprintln(os.Stderr, "  Update cancelled")
		return nil
	}

	devlog.Log("update", "Downloading %s...", "v"+latest.version)

	applyCtx, cancelApply := context.WithTimeout(context.Background(), applyTimeout)
	defer cancelApply()
	if err := updater.updateTo(applyCtx, latest, exe); err != nil {
		if hint, ok := permissionHint(err); ok {
			fmt.Fprintf(os.Stderr, "\n  %s\n", hint)
		}
		return fmt.Errorf("applying update: %w", err)
	}

	fmt.Fprintf(os.Stderr, "%s Updated to %s\n",
		devlog.Green("✓"), devlog.Cyan("v"+latest.version))
	return nil
}
