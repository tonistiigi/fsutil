//go:build !linux && !darwin && !freebsd && !netbsd && !openbsd && !dragonfly

package fsutil

import (
	"os"
	"syscall"
	"time"
)

func (e *RootEntry) OpenFileNoFollow(flag int, perm os.FileMode) (*os.File, error) {
	if _, err := e.Parent(); err != nil {
		return nil, err
	}
	return nil, unsupportedRootOp("openat", e.path, syscall.ENOSYS)
}

func (e *RootEntry) Mkdir(mode os.FileMode) error {
	if _, err := e.Parent(); err != nil {
		return err
	}
	return e.root.Mkdir(e.path, mode)
}

func (e *RootEntry) Symlink(target string) error {
	if _, err := e.Parent(); err != nil {
		return err
	}
	return e.root.Symlink(target, e.path)
}

func (e *RootEntry) Lchown(uid, gid int) error {
	if _, err := e.Parent(); err != nil {
		return err
	}
	return e.root.Lchown(e.path, uid, gid)
}

func (e *RootEntry) ChmodNoFollow(mode os.FileMode) error {
	if _, err := e.Parent(); err != nil {
		return err
	}
	return unsupportedRootOp("fchmodat", e.path, syscall.ENOSYS)
}

func (e *RootEntry) ChtimesNoFollow(atime, mtime time.Time) error {
	if _, err := e.Parent(); err != nil {
		return err
	}
	return unsupportedRootOp("utimensat", e.path, syscall.ENOSYS)
}
