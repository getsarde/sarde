package deploy

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

// CustomDeployer runs a user-provided shell command with DIST_DIR set to the
// absolute output directory.
type CustomDeployer struct {
	Command string
	opts    Options
}

func (d *CustomDeployer) Name() string { return "custom" }

func (d *CustomDeployer) Deploy(ctx context.Context, distDir string, rep Reporter) (*Result, error) {
	rep = orNop(rep)
	absDir, err := filepath.Abs(plainDir(distDir))
	if err != nil {
		absDir = plainDir(distDir)
	}

	cmd := shellCommand(ctx, d.Command)
	if d.opts.ProjectDir != "" {
		cmd.Dir = plainDir(d.opts.ProjectDir)
	}
	cmd.Env = append(os.Environ(), "DIST_DIR="+absDir)

	rep.Step(StepUpload, "Running the deploy command")
	if err := runForwarding(cmd, rep); err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, fmt.Errorf("deploy command failed: %w", err)
	}
	return &Result{Provider: d.Name()}, nil
}
