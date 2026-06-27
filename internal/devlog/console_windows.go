//go:build windows

package devlog

import (
	"os"

	"golang.org/x/sys/windows"
)

func init() {
	enableVirtualTerminal(os.Stderr)
	enableVirtualTerminal(os.Stdout)
}

func enableVirtualTerminal(f *os.File) {
	h := windows.Handle(f.Fd())
	var mode uint32
	if err := windows.GetConsoleMode(h, &mode); err != nil {
		return
	}
	_ = windows.SetConsoleMode(h, mode|windows.ENABLE_VIRTUAL_TERMINAL_PROCESSING)
}
