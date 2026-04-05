package cli

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/coderoo-dev/coderoo/embedded"
	"github.com/coderoo-dev/coderoo/internal/build"
	"github.com/coderoo-dev/coderoo/internal/project"
	"github.com/coderoo-dev/coderoo/internal/server"
	"github.com/spf13/cobra"
)

var sidecarCmd = &cobra.Command{
	Use:   "sidecar",
	Short: "Start IPC server for desktop app",
	Long:  "Start the IPC API server that the desktop app (Tauri) communicates with over HTTP/WebSocket.",
	RunE:  runSidecar,
}

func init() {
	sidecarCmd.Flags().IntP("port", "p", 0, "Server port (0 = auto-assign)")
	rootCmd.AddCommand(sidecarCmd)
}

func runSidecar(cmd *cobra.Command, args []string) error {
	port, _ := cmd.Flags().GetInt("port")

	hub := project.NewEventHub()

	// PreviewFactory bridges the project package to the server package,
	// breaking the import cycle (project doesn't import server).
	previewFactory := func(projectDir, outputDir string, previewPort int, liveReload bool, builderFactory func() *build.SiteBuilder) project.PreviewServer {
		return server.New(server.Options{
			ProjectDir:     projectDir,
			OutputDir:      outputDir,
			Port:           previewPort,
			LiveReload:     liveReload,
			BuilderFactory: builderFactory,
		})
	}

	pm := project.NewProjectManager(hub, embedded.ThemeFS(), previewFactory)

	apiServer := server.NewAPIServer(pm, hub)
	actualPort, err := apiServer.Start(port)
	if err != nil {
		return fmt.Errorf("starting API server: %w", err)
	}

	// Print startup JSON for Tauri to read.
	startup := map[string]any{
		"ready": true,
		"port":  actualPort,
	}
	json.NewEncoder(os.Stdout).Encode(startup)

	log.Printf("IPC server running on port %d", actualPort)

	// Wait for shutdown signal.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Println("Shutting down...")
	pm.CloseProject()
	return apiServer.Stop()
}
