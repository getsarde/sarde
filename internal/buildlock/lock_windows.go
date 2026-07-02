//go:build windows

package buildlock

import (
	"os"

	"golang.org/x/sys/windows"
)

// reservedLockOffset is the byte locked instead of byte 0. LockFileEx is a
// MANDATORY byte-range lock on Windows: locking byte 0, where the metadata
// lives, would make the metadata unreadable by the contending process that
// wants to print the diagnostic message. Locking a range beyond EOF is legal
// and still contends, so metadata reads stay free while exclusion holds.
const reservedLockOffset = 1 << 30

func lockFile(f *os.File) error {
	ol := &windows.Overlapped{Offset: reservedLockOffset}
	return windows.LockFileEx(
		windows.Handle(f.Fd()),
		windows.LOCKFILE_EXCLUSIVE_LOCK|windows.LOCKFILE_FAIL_IMMEDIATELY,
		0, 1, 0, ol,
	)
}

func unlockFile(f *os.File) error {
	ol := &windows.Overlapped{Offset: reservedLockOffset}
	return windows.UnlockFileEx(windows.Handle(f.Fd()), 0, 1, 0, ol)
}
