//go:build windows

package deploy

import (
	"context"
	"os/exec"
	"syscall"
)

// shellCommand runs command through cmd.exe exactly as typed. Go's default
// argument quoting escapes embedded double quotes with backslashes, which
// cmd.exe does not understand, so `echo x > "C:\a b\f.txt"` would fail with
// "The filename, directory name, or volume label syntax is incorrect".
// `/S /C "<command>"` makes cmd strip only the outer quotes.
func shellCommand(ctx context.Context, command string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "cmd")
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: `cmd /S /C "` + command + `"`}
	return cmd
}
