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
	serveCmd.Flags().IntP("port", "p", 0, "Server port (default: from config or 3000)")
	serveCmd.Flags().Bool("no-drafts", false, "Exclude draft content")
	rootCmd.AddCommand(serveCmd)
}

func runServe(cmd *cobra.Command, args []string) error {
	projectDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

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

	// Determine port: CLI flag > config > 3000.
	port, _ := cmd.Flags().GetInt("port")
	if port == 0 {
		port = cfg.Server.Port
	}
	if port == 0 {
		port = 3000
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

		return build.NewSiteBuilder(build.BuildOptions{
			ProjectDir:  projectDir,
			Config:      latestCfg,
			ThemeConfig: latestThemeCfg,
			EmbeddedFS:  embedded.ThemeFS(),
			DevMode:     true,
		})
	}

	liveReload := config.BoolVal(cfg.Server.LiveReload, true)

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
