//go:build unix

package buildlock

import (
	"os"

	"golang.org/x/sys/unix"
)

// flock is advisory and whole-file: it never blocks plain reads, so the
// metadata at offset 0 stays readable by contenders without the reserved
// offset trick Windows needs.
func lockFile(f *os.File) error {
	return unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
}

func unlockFile(f *os.File) error {
	return unix.Flock(int(f.Fd()), unix.LOCK_UN)
}
