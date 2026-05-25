package cli

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/frostybee/sarde/embedded"
	"github.com/frostybee/sarde/internal/build"
	"github.com/frostybee/sarde/internal/config"
	"github.com/frostybee/sarde/internal/consts"
	"github.com/frostybee/sarde/internal/devlog"
	"github.com/frostybee/sarde/internal/server"
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
	rootCmd.AddCommand(devCmd)
}

func runServe(cmd *cobra.Command, args []string) error {
	projectDir := projectDirFromArgs(args)

	cfg, themeCfg, err := resolveAll(cmd, projectDir)
	if err != nil {
		return err
	}

	// In serve mode, drafts are included by default.
	noDrafts, _ := cmd.Flags().GetBool("no-drafts")
	if noDrafts {
		cfg.Build.Drafts = config.BoolPtr(false)
	} else {
		cfg.Build.Drafts = config.BoolPtr(true)
	}

	// Override base path from CLI flag.
	if basePath, _ := cmd.Flags().GetString("base-path"); basePath != "" {
		cfg.Build.BasePath = basePath
	}

	// Override content directory from CLI flag.
	if contentDir, _ := cmd.Flags().GetString("content"); contentDir != "" {
		if !filepath.IsAbs(contentDir) {
			contentDir, _ = filepath.Abs(contentDir)
		}
		cfg.Content.Dir = contentDir
	}

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
	outputDir, err := build.ResolveOutputDir(projectDir, cfg.Build.Output)
	if err != nil {
		return err
	}

	// Builder factory: creates a fresh SiteBuilder on each rebuild.
	builderFactory := func() *build.SiteBuilder {
		// Re-resolve config to pick up sarde.yaml changes.
		latestCfg, latestThemeCfg, err := resolveAll(cmd, projectDir)
		if err != nil {
			log.Printf("Config re-resolve failed, using previous config: %v", err)
			latestCfg = cfg
			latestThemeCfg = themeCfg
		}

		// Preserve draft setting from serve mode.
		if noDrafts {
			latestCfg.Build.Drafts = config.BoolPtr(false)
		} else {
			latestCfg.Build.Drafts = config.BoolPtr(true)
		}

		// Preserve CLI flag overrides across reloads.
		if basePath, _ := cmd.Flags().GetString("base-path"); basePath != "" {
			latestCfg.Build.BasePath = basePath
		}
		if contentDir, _ := cmd.Flags().GetString("content"); contentDir != "" {
			if !filepath.IsAbs(contentDir) {
				contentDir, _ = filepath.Abs(contentDir)
			}
			latestCfg.Content.Dir = contentDir
		}

		return build.NewSiteBuilder(build.BuildOptions{
			ProjectDir:  projectDir,
			Config:      latestCfg,
			ThemeConfig: latestThemeCfg,
			EmbeddedFS:  embedded.ThemeFS(),
			DevMode:     true,
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
	devlog.Log("sarde", "Environment: development | Version: v%s", Version)

	if watchStdin, _ := cmd.Flags().GetBool("watch-stdin"); watchStdin {
		go func() {
			buf := make([]byte, 1)
			_, _ = os.Stdin.Read(buf) // blocks until EOF or 1 byte read
			fmt.Fprintln(os.Stderr, "Parent process gone (stdin closed), shutting down")
			os.Exit(0)
		}()
	}

	ds := server.New(server.Options{
		ProjectDir:     projectDir,
		OutputDir:      outputDir,
		Host:           host,
		Port:           port,
		LiveReload:     liveReload,
		Version:        "v" + Version,
		BuilderFactory: builderFactory,
	})

	// Graceful shutdown on SIGINT/SIGTERM.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("Shutting down...")
		ds.Stop()
	}()

	return ds.Start()
}
