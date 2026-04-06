package deploy

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// CustomDeployer executes a user-provided shell command for deployment.
type CustomDeployer struct {
	Command string
}

func (d *CustomDeployer) Name() string { return "custom" }

func (d *CustomDeployer) Deploy(distDir string) error {
	absDir, err := filepath.Abs(distDir)
	if err != nil {
		absDir = distDir
	}

	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", d.Command)
	} else {
		cmd = exec.Command("sh", "-c", d.Command)
	}
	cmd.Env = append(os.Environ(), "DIST_DIR="+absDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("deploy command failed: %w", err)
	}
	return nil
}
