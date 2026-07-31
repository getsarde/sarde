//go:build windows

package main

import "os"

// peakRSSMB is not implemented on Windows. Reading a child's peak working
// set retroactively requires polling the live process handle with
// GetProcessMemoryInfo via golang.org/x/sys/windows, which is outside this
// tool's stdlib-only budget. CI runs on ubuntu-latest where the Linux
// implementation is accurate; on Windows dev machines wall time and phase
// timings still work and RSS reports as n/a.
func peakRSSMB(ps *os.ProcessState) (float64, bool) {
	return 0, false
}
