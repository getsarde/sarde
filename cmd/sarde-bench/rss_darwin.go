//go:build darwin

package main

import (
	"os"
	"syscall"
)

// peakRSSMB reports the child's peak resident set size in MB. On Darwin,
// Rusage.Maxrss is reported in bytes (it is kilobytes on Linux).
func peakRSSMB(ps *os.ProcessState) (float64, bool) {
	if ps == nil {
		return 0, false
	}
	ru, ok := ps.SysUsage().(*syscall.Rusage)
	if !ok || ru == nil {
		return 0, false
	}
	return float64(ru.Maxrss) / 1024.0 / 1024.0, true
}
