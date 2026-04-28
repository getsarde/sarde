package cli

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/coderoo-dev/coderoo/embedded"
	"github.com/coderoo-dev/coderoo/internal/build"
	"github.com/coderoo-dev/coderoo/internal/config"
	"github.com/coderoo-dev/coderoo/internal/server"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start development server with live reload",
	Long:  "Build the site and start a local dev server. Watches for changes and reloads the browser automatically.",
	RunE:  runServe,
}

func init() {
	serveCmd.Flags().IntP("port", "p", 0, "Server port (default: from config or 4727)")
	serveCmd.Flags().Bool("no-drafts", false, "Exclude draft content")
	serveCmd.Flags().String("base-path", "", "Override URL base path (e.g. /docs/)")
	serveCmd.Flags().String("content", "", "Override content directory path")
	serveCmd.Flags().Bool("watch-stdin", false, "Exit when stdin closes (sidecar/child-process mode)")
	rootCmd.AddCommand(serveCmd)
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

	// Determine port: CLI flag > config > 4727.
	port, _ := cmd.Flags().GetInt("port")
	if port == 0 {
		port = cfg.Server.Port
	}
	if port == 0 {
		port = 4727
	}

	// Determine output directory.
	outputDir := cfg.Build.Output
	if outputDir == "" {
		outputDir = "dist"
	}
	if !filepath.IsAbs(outputDir) {
		outputDir = filepath.Join(projectDir, outputDir)
	}

	// Builder factory: creates a fresh SiteBuilder on each rebuild.
	builderFactory := func() *build.SiteBuilder {
		// Re-resolve config to pick up site.yaml changes.
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

	if verbose, _ := cmd.Flags().GetBool("verbose"); verbose {
		log.Printf("Config: %s", cfg.Site.Title)
		log.Printf("Theme: %s", cfg.Theme.Name)
		log.Printf("Content dir: %s", cfg.Content.Dir)
		log.Printf("Base path: %q", cfg.Build.BasePath)
		log.Printf("Live reload: %v", liveReload)
	}

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
		Port:           port,
		LiveReload:     liveReload,
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
