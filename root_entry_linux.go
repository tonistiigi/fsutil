package fsutil

import (
	"os"
	"strconv"

	"github.com/pkg/errors"
	"golang.org/x/sys/unix"
)

const rootEntryParentFlags = unix.O_PATH | unix.O_DIRECTORY

// ChmodNoFollow changes mode without following a final symlink.
func (e *RootEntry) ChmodNoFollow(mode os.FileMode) error {
	parent, err := e.Parent()
	if err != nil {
		return err
	}
	perm := rootEntryPerm(mode)
	if err := unix.Fchmodat(int(parent.Fd()), e.base, perm, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		if errors.Is(err, unix.EOPNOTSUPP) {
			return errors.WithStack(e.chmodFallback(parent, perm))
		}
		return errors.WithStack(&os.PathError{Op: "fchmodat", Path: e.path, Err: err})
	}
	return nil
}

func (e *RootEntry) chmodFallback(parent *os.File, perm uint32) error {
	// Pin the inode without requiring read permission or opening a FIFO/device
	// for I/O. Older kernels cannot fchmod an O_PATH descriptor, so use procfs.
	fd, err := unix.Openat(int(parent.Fd()), e.base, unix.O_PATH|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return &os.PathError{Op: "openat", Path: e.path, Err: err}
	}
	defer unix.Close(fd)
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		return &os.PathError{Op: "fstat", Path: e.path, Err: err}
	}
	if st.Mode&unix.S_IFMT == unix.S_IFLNK {
		return &os.PathError{Op: "chmod", Path: e.path, Err: unix.EOPNOTSUPP}
	}
	if err := unix.Chmod("/proc/self/fd/"+strconv.Itoa(fd), perm); err != nil {
		if errors.Is(err, unix.ENOENT) {
			err = errors.Wrap(err, "chmod fallback requires /proc/self/fd")
		}
		return &os.PathError{Op: "chmod", Path: e.path, Err: err}
	}
	return nil
}
