package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/frostybee/sarde/internal/build"
	"github.com/frostybee/sarde/internal/config"
	"github.com/frostybee/sarde/internal/deploy"
	"github.com/spf13/cobra"
)

var deployCmd = &cobra.Command{
	Use:   "deploy",
	Short: "Deploy the built site",
	Long:  "Deploy the site output directory to a hosting provider. Run 'sarde build' first.",
	RunE:  runDeploy,
}

func init() {
	deployCmd.Flags().String("provider", "", "Override deploy provider (github, netlify, cloudflare, vercel, custom)")
	deployCmd.Flags().StringP("output", "o", "", "Override output directory (default: dist)")
	rootCmd.AddCommand(deployCmd)
}

func runDeploy(cmd *cobra.Command, args []string) error {
	projectDir := projectDirFromArgs(args)

	// Resolve config.
	configPath, _ := cmd.Flags().GetString("config")
	if !filepath.IsAbs(configPath) {
		configPath = filepath.Join(projectDir, configPath)
	}
	cfg, err := config.Resolve(config.ResolveOptions{
		ConfigPath: configPath,
		CLIFlags:   CollectCLIFlags(cmd),
		EnvPrefix:  "SARDE",
	})
	if err != nil {
		return fmt.Errorf("resolving config: %w", err)
	}

	// Override from CLI flags.
	deployCfg := cfg.Deploy
	if provider, _ := cmd.Flags().GetString("provider"); provider != "" {
		deployCfg.Provider = provider
	}

	// Resolve output directory.
	outputDir := cfg.Build.Output
	if output, _ := cmd.Flags().GetString("output"); output != "" {
		outputDir = output
	}
	outputDir, err = build.ResolveOutputDir(projectDir, outputDir)
	if err != nil {
		return err
	}

	// Verify output exists.
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		return fmt.Errorf("output directory %q does not exist; run 'sarde build' first", outputDir)
	}

	// Create and run deployer.
	deployer, err := deploy.NewDeployer(deployCfg)
	if err != nil {
		return err
	}

	quiet, _ := cmd.Flags().GetBool("quiet")
	if !quiet {
		fmt.Printf("Deploying with %s...\n", deployer.Name())
	}

	if err := deployer.Deploy(outputDir); err != nil {
		return fmt.Errorf("deploy failed: %w", err)
	}

	if !quiet {
		fmt.Println("Deploy complete.")
	}

	return nil
}
