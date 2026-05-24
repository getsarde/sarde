package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// Version is set to "dev" by default and overridden at build time via:
//
//	go build -ldflags "-X github.com/frostybee/sarde/internal/cli.Version=1.2.3"
var Version = "dev"

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Long:  "Print the sarde version, Go runtime version, and OS/architecture.",
	Run:   runVersion,
}

func init() {
	rootCmd.AddCommand(versionCmd)
}

func runVersion(_ *cobra.Command, _ []string) {
	fmt.Printf("sarde %s\n", Version)
	fmt.Printf("Go: %s\n", runtime.Version())
	fmt.Printf("OS/Arch: %s/%s\n", runtime.GOOS, runtime.GOARCH)
}
