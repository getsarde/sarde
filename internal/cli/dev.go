package cli

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/getsarde/sarde/embedded"
	"github.com/getsarde/sarde/internal/build"
	"github.com/getsarde/sarde/internal/buildlock"
	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/devlog"
	"github.com/getsarde/sarde/internal/outputpath"
	"github.com/getsarde/sarde/internal/server"
	"github.com/getsarde/sarde/internal/version"
	"github.com/spf13/cobra"
)

var devCmd = &cobra.Command{
	Use:   "dev",
	Short: "Start development server with live reload",
	Long:  "Build the site and start a local dev server. Watches for changes and reloads the browser automatically.",
	RunE:  runServe,
}

func init() {
	devCmd.Flags().IntP("port", "p", 0, "Server port (default: from config or 4727)")
	devCmd.Flags().String("host", "", "Host to bind to (default: 127.0.0.1, use 0.0.0.0 for LAN access)")
	devCmd.Flags().Bool("no-drafts", false, "Exclude draft content")
	devCmd.Flags().String("base-path", "", "Override URL base path (e.g. /docs/)")
	devCmd.Flags().String("content", "", "Override content directory path")
	devCmd.Flags().Bool("watch-stdin", false, "Exit when stdin closes (sidecar/child-process mode)")
	devCmd.Flags().String("theme-dev", "", "Path to embedded/theme/ source dir for live-reload (framework dev only)")
	devCmd.Flags().Bool("check-syntax", false, "Enable syntax checking for unclosed fenced blocks during rebuilds")
	rootCmd.AddCommand(devCmd)
}

func runServe(cmd *cobra.Command, args []string) error {
	projectDir := projectDirFromArgs(args)

	cfg, themeCfg, err := resolveAll(cmd, projectDir)
	if err != nil {
		return err
	}

	// In serve mode, drafts and expired pages are included by default.
	noDrafts, _ := cmd.Flags().GetBool("no-drafts")
	if noDrafts {
		cfg.Build.Drafts = config.BoolPtr(false)
	} else {
		cfg.Build.Drafts = config.BoolPtr(true)
	}
	cfg.Build.Expired = config.BoolPtr(true)

	// Override base path and content directory from CLI flags.
	applyCommonOverrides(cmd, cfg)

	// Determine host: CLI flag > config > 127.0.0.1.
	host, _ := cmd.Flags().GetString("host")
	if host == "" {
		host = cfg.Server.Host
	}

	// Determine port: CLI flag > config > 4727.
	port, _ := cmd.Flags().GetInt("port")
	if port == 0 {
		port = cfg.Server.Port
	}
	if port == 0 {
		port = consts.DefaultPort
	}

	// Determine output directory.
	outputDir, err := outputpath.ResolveOutputDir(projectDir, cfg.Build.Output)
	if err != nil {
		return err
	}

	// Hold the single-instance output lock for the whole dev session so a
	// second sarde (dev or build) targeting the same dist/ fails fast
	// instead of silently corrupting fingerprinted assets. Acquired before
	// the port bind: without this, a second dev server would just take the
	// next free port and coexist unnoticed.
	lock, err := buildlock.Acquire(outputDir, "dev")
	if err != nil {
		return err
	}
	defer lock.Release()

	// Theme dev mode: read theme assets from disk instead of go:embed.
	themeDevDir, _ := cmd.Flags().GetString("theme-dev")
	if themeDevDir == "" {
		themeDevDir = os.Getenv("SARDE_THEME_DEV")
	}

	var themeFS = embedded.ThemeFS()
	var pluginAssetsDir string
	var themeDevWatchDirs []string
	var pluginDevWatchDirs []string

	if themeDevDir != "" {
		if !filepath.IsAbs(themeDevDir) {
			themeDevDir, _ = filepath.Abs(themeDevDir)
		}
		liveFS, fsErr := embedded.ThemeDirFS(themeDevDir)
		if fsErr != nil {
			return fmt.Errorf("--theme-dev: %w", fsErr)
		}
		themeFS = liveFS
		themeDevWatchDirs = append(themeDevWatchDirs, themeDevDir)
		devlog.Log("sarde", "Theme dev mode: %s", themeDevDir)

		// Infer plugin assets dir from theme dir (embedded/theme → repo root → internal/plugin/clientplugins/assets).
		repoRoot := filepath.Dir(filepath.Dir(themeDevDir))
		candidate := filepath.Join(repoRoot, "internal", "plugin", "clientplugins", "assets")
		if _, statErr := os.Stat(candidate); statErr == nil {
			pluginAssetsDir = candidate
			pluginDevWatchDirs = append(pluginDevWatchDirs, pluginAssetsDir)
			devlog.Log("sarde", "Plugin dev mode: %s", pluginAssetsDir)
		}
	}

	// Builder factory: creates a fresh SiteBuilder on each rebuild.
	builderFactory := func() *build.SiteBuilder {
		// Re-resolve config to pick up sarde.yaml changes.
		latestCfg, latestThemeCfg, err := resolveAll(cmd, projectDir)
		if err != nil {
			devlog.Warn("config", "Config re-resolve failed, using previous config: %v", err)
			latestCfg = cfg
			latestThemeCfg = themeCfg
		}

		// Preserve draft/expired settings from serve mode.
		if noDrafts {
			latestCfg.Build.Drafts = config.BoolPtr(false)
		} else {
			latestCfg.Build.Drafts = config.BoolPtr(true)
		}
		latestCfg.Build.Expired = config.BoolPtr(true)

		// Preserve CLI flag overrides across reloads.
		applyCommonOverrides(cmd, latestCfg)

		checkSyntax, _ := cmd.Flags().GetBool("check-syntax")
		return build.NewSiteBuilder(build.BuildOptions{
			ProjectDir:      projectDir,
			Config:          latestCfg,
			ThemeConfig:     latestThemeCfg,
			EmbeddedFS:      themeFS,
			DevMode:         true,
			CheckSyntax:     checkSyntax,
			PluginAssetsDir: pluginAssetsDir,
		})
	}

	liveReload := config.BoolVal(cfg.Server.LiveReload, true)

	themeName := cfg.Theme.Name
	if themeName == "" {
		themeName = "default"
	}
	contentDir := cfg.Content.Dir
	if contentDir == "" {
		contentDir = "content"
	}
	devlog.Log("sarde", "%s", cfg.Site.Title)
	devlog.Log("sarde", "Theme: %s | Content: %s", themeName, contentDir)
	devlog.Log("sarde", "Live reload: %v", liveReload)
	devlog.Log("sarde", "Environment: development | Version: v%s", version.Version)

	if watchStdin, _ := cmd.Flags().GetBool("watch-stdin"); watchStdin {
		go func() {
			buf := make([]byte, 1)
			_, _ = os.Stdin.Read(buf) // blocks until EOF or 1 byte read
			fmt.Fprintln(os.Stderr, "Parent process gone (stdin closed), shutting down")
			os.Exit(0)
		}()
	}

	verbose, _ := cmd.Flags().GetBool("verbose")
	ds := server.New(server.Options{
		ProjectDir:     projectDir,
		OutputDir:      outputDir,
		Host:           host,
		Port:           port,
		LiveReload:     liveReload,
		Version:        "v" + version.Version,
		BasePath:       cfg.Build.BasePath,
		BuilderFactory: builderFactory,
		ThemeDevDirs:   themeDevWatchDirs,
		PluginDevDirs:  pluginDevWatchDirs,
		Verbose:        verbose,
	})

	// Graceful shutdown on SIGINT/SIGTERM.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		devlog.Log("sarde", "Shutting down...")
		ds.Stop()
	}()

	// Passive update notice before the server blocks; the background refresh
	// has the whole dev session to finish and persist for the next run.
	maybeCheckForUpdate(cmd)

	return ds.Start()
}
