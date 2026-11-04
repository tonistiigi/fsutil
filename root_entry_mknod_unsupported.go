//go:build !linux && !freebsd && !netbsd && !openbsd && !dragonfly

package fsutil

import "syscall"

// Mknod creates the entry as a device node or FIFO.
func (e *RootEntry) Mknod(mode uint32, dev int) error {
	if _, err := e.Parent(); err != nil {
		return err
	}
	return unsupportedRootOp("mknodat", e.path, syscall.ENOSYS)
}
