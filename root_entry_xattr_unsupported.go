//go:build !linux && !darwin && !freebsd && !netbsd

package fsutil

import "syscall"

// LSetxattr sets an xattr on the entry without following a final symlink.
func (e *RootEntry) LSetxattr(key string, value []byte, flags int) error {
	if _, err := e.Parent(); err != nil {
		return err
	}
	return unsupportedRootOp("lsetxattr", e.path, syscall.ENOSYS)
}
