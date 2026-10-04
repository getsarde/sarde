package cli

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/build"
	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/devlog"
	"github.com/getsarde/sarde/internal/engine"
	"github.com/spf13/cobra"
)

// configPathFor returns the --config path resolved against projectDir.
func configPathFor(cmd *cobra.Command, projectDir string) string {
	configPath, _ := cmd.Flags().GetString("config")
	if !filepath.IsAbs(configPath) {
		configPath = filepath.Join(projectDir, configPath)
	}
	return configPath
}

// requireSite returns a *notSiteError when projectDir is not a Sarde site: no
// config file and no content/ directory. Zero-config sites have no sarde.yaml,
// so either one is enough; an explicit --content skips the check. Called before
// config resolution so the error is not buried under config warnings.
func requireSite(cmd *cobra.Command, projectDir string) error {
	if contentDir, _ := cmd.Flags().GetString("content"); contentDir != "" {
		return nil
	}
	// Anything but "not found" (a permission error, say) is left for config
	// resolution to report.
	if _, err := os.Stat(configPathFor(cmd, projectDir)); !errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if info, err := os.Stat(filepath.Join(projectDir, consts.DirContent)); err == nil && info.IsDir() {
		return nil
	}
	// Execute prints this error in color; keep Cobra from printing a second,
	// plain copy.
	cmd.SilenceErrors = true
	return &notSiteError{
		dir:     projectDir,
		subdirs: sitesInSubdirs(projectDir, 3),
		cmdPath: cmd.CommandPath(),
	}
}

// sitesInSubdirs returns the names of up to limit immediate subfolders of dir
// that hold a sarde.yaml, for the "found a site in a subfolder" hint.
func sitesInSubdirs(dir string, limit int) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(dir, e.Name(), consts.FileSiteConfig)); err == nil {
			names = append(names, e.Name())
			if len(names) == limit {
				break
			}
		}
	}
	return names
}

// notSiteError reports that a command ran outside a Sarde site. Error() is
// plain text, for the JSON error envelope and logs; Pretty() is the colored
// form Execute prints to the terminal.
type notSiteError struct {
	dir     string   // folder the command ran in
	subdirs []string // immediate subfolders holding a sarde.yaml, at most 3
	cmdPath string   // e.g. "sarde dev", used in the hints
}

func (e *notSiteError) Error() string { return e.render(false) }

// Pretty returns the message with an "Error:" prefix, colored when the
// terminal supports it.
func (e *notSiteError) Pretty() string {
	return devlog.Red("Error:") + " " + e.render(true) + "\n"
}

func (e *notSiteError) render(color bool) string {
	plain := func(s string) string { return s }
	bold, dim, cyan := plain, plain, plain
	if color {
		bold, dim, cyan = devlog.Bold, devlog.Dim, devlog.Cyan
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s: %s\n", bold("not a Sarde site"), dim(e.dir))
	fmt.Fprintf(&b, "  This folder has no %s and no %s/ folder.\n", consts.FileSiteConfig, consts.DirContent)
	if len(e.subdirs) > 0 {
		quoted := make([]string, len(e.subdirs))
		for i, name := range e.subdirs {
			quoted[i] = `"` + name + `"`
		}
		found := "a site in the subfolder " + quoted[0]
		if len(quoted) > 1 {
			found = "sites in the subfolders " + strings.Join(quoted, ", ")
		}
		fmt.Fprintf(&b, "  Found %s. Run:\n", found)
		fmt.Fprintf(&b, "    %s\n", cyan("cd "+quoteIfSpaced(e.subdirs[0])))
		fmt.Fprintf(&b, "    %s", cyan(e.cmdPath))
	} else {
		fmt.Fprintf(&b, "  Run the command from your site's folder, or pass its path:\n")
		fmt.Fprintf(&b, "    %s\n", cyan(e.cmdPath+" <site-dir>"))
		fmt.Fprintf(&b, "  To create a new site:\n")
		fmt.Fprintf(&b, "    %s", cyan("sarde new site <name>"))
	}
	return b.String()
}

// quoteIfSpaced wraps a path in double quotes when it contains whitespace, so
// a suggested shell command can be pasted as is.
func quoteIfSpaced(path string) string {
	if strings.ContainsAny(path, " \t") {
		return `"` + path + `"`
	}
	return path
}

// applyCommonOverrides applies the --base-path and --content flag overrides
// shared by the build, check-links, and dev commands to the resolved config.
func applyCommonOverrides(cmd *cobra.Command, cfg *config.SiteConfig) {
	if basePath, _ := cmd.Flags().GetString("base-path"); basePath != "" {
		cfg.Build.BasePath = config.NormalizeBasePath(basePath)
	}
	if contentDir, _ := cmd.Flags().GetString("content"); contentDir != "" {
		if !filepath.IsAbs(contentDir) {
			contentDir, _ = filepath.Abs(contentDir)
		}
		cfg.Content.Dir = contentDir
	}
}

// newSiteBuilder constructs a SiteBuilder with the standard embedded theme,
// the shape shared by the build, check-links, and validate commands.
func newSiteBuilder(projectDir string, cfg *config.SiteConfig, themeCfg *engine.ThemeConfig) *build.SiteBuilder {
	return build.NewSiteBuilder(build.BuildOptions{
		ProjectDir:  projectDir,
		Config:      cfg,
		ThemeConfig: themeCfg,
		EmbeddedFS:  embedded.ThemeFS(),
	})
}
