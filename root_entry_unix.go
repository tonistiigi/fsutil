//go:build linux || darwin || freebsd || netbsd || openbsd || dragonfly

package fsutil

import (
	"os"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/sys/unix"
)

func (r *root) openRootEntry(name string) (*RootEntry, error) {
	path, _, _, err := cleanRootEntryPath(name)
	if err != nil {
		return nil, err
	}
	parent, base, closeParent, err := r.openRootParent(path)
	if err != nil {
		return nil, err
	}
	return &RootEntry{
		root:        r,
		path:        path,
		base:        base,
		parent:      parent,
		closeParent: closeParent,
	}, nil
}

// OpenFileNoFollow opens the entry without following a final symlink.
func (e *RootEntry) OpenFileNoFollow(flag int, perm os.FileMode) (*os.File, error) {
	if perm&0o777 != perm {
		return nil, errors.WithStack(&os.PathError{Op: "openat", Path: e.path, Err: errors.New("unsupported file mode")})
	}
	parent, err := e.Parent()
	if err != nil {
		return nil, err
	}
	fd, err := unix.Openat(int(parent.Fd()), e.base, flag|unix.O_NOFOLLOW|unix.O_CLOEXEC, uint32(perm))
	if err != nil {
		return nil, errors.WithStack(&os.PathError{Op: "openat", Path: e.path, Err: err})
	}
	return os.NewFile(uintptr(fd), e.path), nil
}

// Mkdir creates the entry as a directory.
func (e *RootEntry) Mkdir(mode os.FileMode) error {
	if mode&0o777 != mode {
		return errors.WithStack(&os.PathError{Op: "mkdirat", Path: e.path, Err: errors.New("unsupported file mode")})
	}
	parent, err := e.Parent()
	if err != nil {
		return err
	}
	if err := unix.Mkdirat(int(parent.Fd()), e.base, uint32(mode)); err != nil {
		return errors.WithStack(&os.PathError{Op: "mkdirat", Path: e.path, Err: err})
	}
	return nil
}

// Symlink creates the entry as a symlink to target.
func (e *RootEntry) Symlink(target string) error {
	parent, err := e.Parent()
	if err != nil {
		return err
	}
	if err := unix.Symlinkat(target, int(parent.Fd()), e.base); err != nil {
		return errors.WithStack(&os.PathError{Op: "symlinkat", Path: e.path, Err: err})
	}
	return nil
}

// Lchown changes ownership of the entry without following a final symlink.
func (e *RootEntry) Lchown(uid, gid int) error {
	parent, err := e.Parent()
	if err != nil {
		return err
	}
	if err := unix.Fchownat(int(parent.Fd()), e.base, uid, gid, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return errors.WithStack(&os.PathError{Op: "fchownat", Path: e.path, Err: err})
	}
	return nil
}

// ChmodNoFollow changes mode without following a final symlink.
func (e *RootEntry) ChmodNoFollow(mode os.FileMode) error {
	parent, err := e.Parent()
	if err != nil {
		return err
	}
	if err := unix.Fchmodat(int(parent.Fd()), e.base, rootEntryPerm(mode), unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return errors.WithStack(&os.PathError{Op: "fchmodat", Path: e.path, Err: err})
	}
	return nil
}

// ChtimesNoFollow changes access and modification times without following a
// final symlink.
func (e *RootEntry) ChtimesNoFollow(atime, mtime time.Time) error {
	parent, err := e.Parent()
	if err != nil {
		return err
	}
	times := []unix.Timespec{
		unix.NsecToTimespec(atime.UnixNano()),
		unix.NsecToTimespec(mtime.UnixNano()),
	}
	if err := unix.UtimesNanoAt(int(parent.Fd()), e.base, times, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return errors.WithStack(&os.PathError{Op: "utimensat", Path: e.path, Err: err})
	}
	return nil
}

func rootEntryPerm(mode os.FileMode) uint32 {
	perm := uint32(mode.Perm())
	if mode&os.ModeSetuid != 0 {
		perm |= unix.S_ISUID
	}
	if mode&os.ModeSetgid != 0 {
		perm |= unix.S_ISGID
	}
	if mode&os.ModeSticky != 0 {
		perm |= unix.S_ISVTX
	}
	return perm
}
