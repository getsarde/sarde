//go:build windows

package server

import (
	"path/filepath"
	"unsafe"

	"github.com/fsnotify/fsnotify"
	"golang.org/x/sys/windows"
)

// subtreeWatchSupported: on Windows the static root is watched through one
// handle on the folder itself with the subtree flag, instead of fsnotify's
// handle per subfolder. An open handle on a folder inside a directory makes
// renaming that directory fail ("Access is denied"), so the per-folder
// watches blocked renaming public/ while `sarde dev` ran; a handle on the
// directory itself, opened with FILE_SHARE_DELETE, does not.
const subtreeWatchSupported = true

const subtreeNotifyFilter = windows.FILE_NOTIFY_CHANGE_FILE_NAME |
	windows.FILE_NOTIFY_CHANGE_DIR_NAME |
	windows.FILE_NOTIFY_CHANGE_SIZE |
	windows.FILE_NOTIFY_CHANGE_LAST_WRITE |
	windows.FILE_NOTIFY_CHANGE_CREATION

// subtreeWatch reads one directory's changes, subfolders included, until
// close or until the directory goes away.
type subtreeWatch struct {
	dir    string
	handle windows.Handle
	stop   windows.Handle // manual-reset event set by close
	quit   chan struct{}
	done   chan struct{}
}

// openSubtreeWatch watches dir and its subfolders. emit receives each change
// as an fsnotify event and returns false to stop; overflow runs when the
// change buffer overflowed and events were lost.
func openSubtreeWatch(dir string, emit func(fsnotify.Event, <-chan struct{}) bool, overflow func()) (*subtreeWatch, error) {
	name, err := windows.UTF16PtrFromString(dir)
	if err != nil {
		return nil, err
	}
	h, err := windows.CreateFile(name, windows.FILE_LIST_DIRECTORY,
		windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil, windows.OPEN_EXISTING,
		windows.FILE_FLAG_BACKUP_SEMANTICS|windows.FILE_FLAG_OVERLAPPED, 0)
	if err != nil {
		return nil, err
	}
	stop, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		windows.CloseHandle(h)
		return nil, err
	}
	sw := &subtreeWatch{dir: dir, handle: h, stop: stop, quit: make(chan struct{}), done: make(chan struct{})}
	go sw.run(emit, overflow)
	return sw, nil
}

func (sw *subtreeWatch) run(emit func(fsnotify.Event, <-chan struct{}) bool, overflow func()) {
	defer close(sw.done)
	defer windows.CloseHandle(sw.stop)
	defer windows.CloseHandle(sw.handle)

	ready, err := windows.CreateEvent(nil, 1, 0, nil)
	if err != nil {
		return
	}
	defer windows.CloseHandle(ready)
	// Both stay put while a read is pending: the GC does not move heap
	// objects, and the loop waits for every read to finish or be cancelled.
	buf := make([]byte, 64*1024)
	ov := new(windows.Overlapped)
	for {
		*ov = windows.Overlapped{HEvent: ready}
		if err := windows.ReadDirectoryChanges(sw.handle, &buf[0], uint32(len(buf)), true,
			subtreeNotifyFilter, nil, ov, 0); err != nil {
			return
		}
		which, err := windows.WaitForMultipleObjects([]windows.Handle{ready, sw.stop}, false, windows.INFINITE)
		var n uint32
		if err != nil || which != windows.WAIT_OBJECT_0 {
			windows.CancelIoEx(sw.handle, ov)
			windows.GetOverlappedResult(sw.handle, ov, &n, true)
			return
		}
		// Fails once the directory is deleted; the root watch reports that.
		if err := windows.GetOverlappedResult(sw.handle, ov, &n, false); err != nil {
			return
		}
		if n == 0 {
			overflow()
			continue
		}
		for _, ev := range parseNotifications(sw.dir, buf[:n]) {
			if !emit(ev, sw.quit) {
				return
			}
		}
	}
}

// close stops the watch without waiting for it.
func (sw *subtreeWatch) close() {
	select {
	case <-sw.quit:
		return
	default:
	}
	close(sw.quit)
	windows.SetEvent(sw.stop)
}

// parseNotifications turns a FILE_NOTIFY_INFORMATION list into events.
func parseNotifications(dir string, buf []byte) []fsnotify.Event {
	var out []fsnotify.Event
	for off := 0; off+12 <= len(buf); {
		info := (*windows.FileNotifyInformation)(unsafe.Pointer(&buf[off]))
		nameLen := int(info.FileNameLength) / 2
		name := windows.UTF16ToString(unsafe.Slice(&info.FileName, nameLen))
		ev := fsnotify.Event{Name: filepath.Join(dir, name)}
		switch info.Action {
		case windows.FILE_ACTION_ADDED, windows.FILE_ACTION_RENAMED_NEW_NAME:
			ev.Op = fsnotify.Create
		case windows.FILE_ACTION_REMOVED:
			ev.Op = fsnotify.Remove
		case windows.FILE_ACTION_MODIFIED:
			ev.Op = fsnotify.Write
		case windows.FILE_ACTION_RENAMED_OLD_NAME:
			ev.Op = fsnotify.Rename
		}
		if ev.Op != 0 {
			out = append(out, ev)
		}
		if info.NextEntryOffset == 0 {
			break
		}
		off += int(info.NextEntryOffset)
	}
	return out
}
