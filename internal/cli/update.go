package cli

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/creativeprojects/go-selfupdate"
	"github.com/getsarde/sarde/internal/devlog"
	"github.com/getsarde/sarde/internal/version"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update sarde to the latest version",
	Long:  "Check for and install the latest version of sarde from GitHub Releases.",
	RunE:  runUpdate,
}

func init() {
	updateCmd.Flags().Bool("check", false, "Only check for updates without installing")
	rootCmd.AddCommand(updateCmd)
}

func isHomebrewInstall() bool {
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	for _, prefix := range []string{
		"/opt/homebrew/",
		"/usr/local/Cellar/",
		"/home/linuxbrew/.linuxbrew/",
	} {
		if strings.HasPrefix(exe, prefix) {
			return true
		}
	}
	return false
}

func runUpdate(cmd *cobra.Command, _ []string) error {
	checkOnly, _ := cmd.Flags().GetBool("check")

	if version.Version == "dev" {
		return fmt.Errorf("cannot update a dev build; install a release version first")
	}

	if isHomebrewInstall() {
		fmt.Fprintln(os.Stderr, "Installed via Homebrew. Run `brew upgrade sarde` to update.")
		return nil
	}

	devlog.Log("update", "Checking for updates...")

	source, err := selfupdate.NewGitHubSource(selfupdate.GitHubConfig{})
	if err != nil {
		return fmt.Errorf("initializing update source: %w", err)
	}

	updater, err := selfupdate.NewUpdater(selfupdate.Config{
		Source:    source,
		Filters:  []string{fmt.Sprintf("sarde_%s_%s", runtime.GOOS, runtime.GOARCH)},
		Validator: &selfupdate.ChecksumValidator{UniqueFilename: "checksums.txt"},
	})
	if err != nil {
		return fmt.Errorf("initializing updater: %w", err)
	}

	latest, found, err := updater.DetectLatest(context.Background(), selfupdate.ParseSlug("getsarde/sarde"))
	if err != nil {
		return fmt.Errorf("checking for updates: %w", err)
	}

	if !found {
		devlog.Log("update", "No releases found")
		return nil
	}

	current := version.Version
	if latest.LessOrEqual(current) {
		fmt.Fprintf(os.Stderr, "%s Already up to date (%s)\n",
			devlog.Green("✓"), devlog.Cyan("v"+current))
		return nil
	}

	fmt.Fprintf(os.Stderr, "  Current: %s\n", devlog.Cyan("v"+current))
	fmt.Fprintf(os.Stderr, "  Latest:  %s\n", devlog.Cyan("v"+latest.Version()))

	if checkOnly {
		fmt.Fprintf(os.Stderr, "\n  Run %s to install\n", devlog.Bold("sarde update"))
		return nil
	}

	fmt.Fprintf(os.Stderr, "\n  Update to %s? [y/N] ", devlog.Cyan("v"+latest.Version()))
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer != "y" && answer != "yes" {
		fmt.Fprintln(os.Stderr, "  Update cancelled")
		return nil
	}

	devlog.Log("update", "Downloading %s...", "v"+latest.Version())

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locating current binary: %w", err)
	}

	if err := updater.UpdateTo(context.Background(), latest, exe); err != nil {
		return fmt.Errorf("applying update: %w", err)
	}

	fmt.Fprintf(os.Stderr, "%s Updated to %s\n",
		devlog.Green("✓"), devlog.Cyan("v"+latest.Version()))
	return nil
}
