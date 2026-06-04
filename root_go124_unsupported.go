//go:build !go1.25 && !linux && !darwin && !freebsd && !netbsd && !openbsd && !dragonfly && !windows

package fsutil

import "syscall"

func (r *root) Lchown(name string, uid, gid int) error {
	return unsupportedRootOp("lchown", name, syscall.ENOSYS)
}

func (r *root) Link(oldname, newname string) error {
	return unsupportedRootOp("link", newname, syscall.ENOSYS)
}

func (r *root) Readlink(name string) (string, error) {
	return "", unsupportedRootOp("readlink", name, syscall.ENOSYS)
}

func (r *root) Rename(oldname, newname string) error {
	return unsupportedRootOp("rename", oldname, syscall.ENOSYS)
}

func (r *root) Symlink(oldname, newname string) error {
	return unsupportedRootOp("symlink", newname, syscall.ENOSYS)
}
