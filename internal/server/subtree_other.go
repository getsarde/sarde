//go:build !windows

package server

import (
	"errors"

	"github.com/fsnotify/fsnotify"
)

// subtreeWatchSupported: elsewhere fsnotify's per-folder watches do not
// block renaming the static root, so it is watched like the other roots.
const subtreeWatchSupported = false

type subtreeWatch struct{ done chan struct{} }

func openSubtreeWatch(string, func(fsnotify.Event, <-chan struct{}) bool, func()) (*subtreeWatch, error) {
	return nil, errors.New("subtree watches are Windows only")
}

func (*subtreeWatch) close() {}
