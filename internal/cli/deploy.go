package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/getsarde/sarde/internal/build"
	"github.com/getsarde/sarde/internal/buildlock"
	"github.com/getsarde/sarde/internal/config"
	"github.com/getsarde/sarde/internal/deploy"
	"github.com/getsarde/sarde/internal/outputpath"
)

var deployCmd = &cobra.Command{
	Use:   "deploy [dir]",
	Short: "Deploy the built site",
	Long: `Deploy the site output directory to a hosting provider. Run 'sarde build' first,
or pass --build to build and deploy in one run.

Provider tokens come from environment variables, never from sarde.yaml:
NETLIFY_AUTH_TOKEN, VERCEL_TOKEN, CLOUDFLARE_API_TOKEN. GitHub Pages uses the
git credentials already configured for the repository.

--check verifies the credentials and access to the configured site or project
without uploading anything.

The deploy holds the output directory's lock, so it fails while 'sarde dev' or
another build is writing to the same directory.`,
	SilenceUsage: true,
	RunE:         runDeploy,
}

// deployOptionsHook lets tests point API deployers at a fake server.
var deployOptionsHook func(*deploy.Options)

// deployReporter is what the command needs beyond deploy.Reporter.
type deployReporter interface {
	deploy.Reporter
	Start(provider string)
	Result(res *deploy.Result, elapsed time.Duration)
	Check(res *deploy.CheckResult)
}

func init() {
	deployCmd.Flags().String("provider", "", "Override deploy provider (github, netlify, cloudflare, vercel, custom)")
	deployCmd.Flags().StringP("output", "o", "", "Override output directory (default: dist)")
	deployCmd.Flags().String("format", "pretty", "Output format: pretty, json")
	deployCmd.Flags().Bool("check", false, "Verify credentials and target access without deploying")
	deployCmd.Flags().Bool("build", false, "Build the site first, then deploy it")
	rootCmd.AddCommand(deployCmd)
}

func runDeploy(cmd *cobra.Command, args []string) error {
	format, _ := cmd.Flags().GetString("format")
	if format != "pretty" && format != "json" {
		return fmt.Errorf("unknown format %q (expected pretty or json)", format)
	}
	check, _ := cmd.Flags().GetBool("check")
	buildFirst, _ := cmd.Flags().GetBool("build")
	if check && buildFirst {
		return fmt.Errorf("--build and --check cannot be combined")
	}

	var rep deployReporter
	if format == "json" {
		rep = newJSONDeployReporter(cmd.OutOrStdout())
	} else {
		quiet, _ := cmd.Flags().GetBool("quiet")
		tty := false
		if f, ok := cmd.OutOrStdout().(*os.File); ok {
			tty = term.IsTerminal(int(f.Fd()))
		}
		rep = newPrettyDeployReporter(cmd.OutOrStdout(), quiet, tty)
	}

	ctx, stop := signal.NotifyContext(cmdContext(cmd), os.Interrupt, syscall.SIGTERM)
	defer stop()

	err := runDeployWith(ctx, cmd, args, deployRun{check: check, build: buildFirst, format: format}, rep)
	if err != nil && format == "json" {
		kind := "deploy_failed"
		var bf *deployBuildError
		switch {
		case check:
			kind = "deploy_check_failed"
		case errors.As(err, &bf):
			kind = "build_failed"
		}
		writeJSONError(cmd.OutOrStdout(), kind, deploy.ErrorCode(err), err)
	}
	return err
}

func cmdContext(cmd *cobra.Command) context.Context {
	if ctx := cmd.Context(); ctx != nil {
		return ctx
	}
	return context.Background()
}

// deployRun is what the flags ask of one deploy.
type deployRun struct {
	check  bool   // verify access only
	build  bool   // build first (--build)
	format string // "pretty" or "json"
}

// deployBuildError marks a failure of the build that --build runs first, so
// the JSON envelope names it build_failed rather than deploy_failed.
type deployBuildError struct{ err error }

func (e *deployBuildError) Error() string { return e.err.Error() }
func (e *deployBuildError) Unwrap() error { return e.err }

func runDeployWith(ctx context.Context, cmd *cobra.Command, args []string, run deployRun, rep deployReporter) error {
	check := run.check
	projectDir := projectDirFromArgs(args)

	configPath, _ := cmd.Flags().GetString("config")
	if !filepath.IsAbs(configPath) {
		configPath = filepath.Join(projectDir, configPath)
	}
	cfg, err := config.Resolve(config.ResolveOptions{
		ConfigPath:   configPath,
		CLIFlags:     CollectCLIFlags(cmd),
		EnvPrefix:    "SARDE",
		Strict:       true,
		KnownPlugins: build.KnownPluginNames(projectDir),
	})
	if err != nil {
		return fmt.Errorf("resolving config: %w", err)
	}

	deployCfg := cfg.Deploy
	if provider, _ := cmd.Flags().GetString("provider"); provider != "" {
		deployCfg.Provider = provider
	}

	opts := deploy.Options{ProjectDir: projectDir}
	if deployOptionsHook != nil {
		deployOptionsHook(&opts)
	}
	deployer, err := deploy.NewDeployer(deployCfg, opts)
	if err != nil {
		return err
	}

	if check {
		checker, ok := deployer.(deploy.Checker)
		if !ok {
			return fmt.Errorf("--check is not supported for provider %q", deployCfg.Provider)
		}
		res, err := checker.Check(ctx, rep)
		if err != nil {
			return err
		}
		rep.Check(res)
		return nil
	}

	outputDir := cfg.Build.Output
	if output, _ := cmd.Flags().GetString("output"); output != "" {
		outputDir = output
	}
	outputDir, err = outputpath.ResolveOutputDir(projectDir, outputDir)
	if err != nil {
		return err
	}

	// Hold the output dir for the whole deploy, so a build or dev server
	// cannot rewrite it mid-upload (dev writes drafts into it). Under --build
	// the build runs inside this lock; Acquire is re-entrant in one process.
	lock, err := buildlock.Acquire(outputDir, "deploy")
	if err != nil {
		return err
	}
	defer lock.Release()

	if run.build {
		// JSON mode keeps stdout to the deploy's event stream: the build
		// reports nothing there, and its failure surfaces as build_failed.
		out := cmd.OutOrStdout()
		if run.format == "json" {
			out = io.Discard
		}
		err := runBuildTo(cmd, args, buildRun{format: run.format, out: out, provider: deployCfg.Provider})
		if err != nil {
			return &deployBuildError{err: err}
		}
	} else if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		return fmt.Errorf("output directory %q does not exist; run 'sarde build' first, or pass --build", outputDir)
	}

	rep.Start(deployer.Name())
	start := time.Now()
	res, err := deployer.Deploy(ctx, outputDir, rep)
	if err != nil {
		return fmt.Errorf("deploy failed: %w", err)
	}
	rep.Result(res, time.Since(start))
	return nil
}
