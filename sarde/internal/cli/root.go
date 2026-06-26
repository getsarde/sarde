package cli

import (
	"github.com/getsarde/sarde/internal/consts"
	"github.com/getsarde/sarde/internal/version"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "sarde",
	Short: "A zero-config static site generator",
	Long:  "Sarde is a zero-config, Go-based static site generator. Drop Markdown files into a content/ folder and get a fully-themed, production-ready static site.",
}

func init() {
	rootCmd.Version = version.Version
	rootCmd.PersistentFlags().StringP("config", "c", consts.FileSiteConfig, "Path to site config file")
	rootCmd.PersistentFlags().String("baseURL", "", "Override site base URL")
	rootCmd.PersistentFlags().BoolP("drafts", "D", false, "Include draft content")
	rootCmd.PersistentFlags().Bool("future", false, "Include future-dated content")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Verbose output")
	rootCmd.PersistentFlags().BoolP("quiet", "q", false, "Suppress non-error output")
}

func Execute() error {
	return rootCmd.Execute()
}

// CollectCLIFlags reads changed flags into a map for config.Resolve.
func CollectCLIFlags(cmd *cobra.Command) map[string]any {
	flags := make(map[string]any)
	if cmd.Flags().Changed("baseURL") {
		flags["site.url"], _ = cmd.Flags().GetString("baseURL")
	}
	if cmd.Flags().Changed("drafts") {
		flags["build.drafts"], _ = cmd.Flags().GetBool("drafts")
	}
	if cmd.Flags().Changed("future") {
		flags["build.future"], _ = cmd.Flags().GetBool("future")
	}
	return flags
}
