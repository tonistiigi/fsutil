//go:build freebsd

package fsutil

import (
	"os"

	"github.com/pkg/errors"
	"golang.org/x/sys/unix"
)

// Mknod creates the entry as a device node or FIFO.
func (e *RootEntry) Mknod(mode uint32, dev int) error {
	parent, err := e.Parent()
	if err != nil {
		return err
	}
	if err := unix.Mknodat(int(parent.Fd()), e.base, mode, uint64(dev)); err != nil {
		return errors.WithStack(&os.PathError{Op: "mknodat", Path: e.path, Err: err})
	}
	return nil
}
