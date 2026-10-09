//go:build linux

package fsutil

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
	"golang.org/x/sys/unix"
)

// LSetxattr sets an xattr on the entry without following a final symlink.
func (e *RootEntry) LSetxattr(key string, value []byte, flags int) error {
	parent, err := e.Parent()
	if err != nil {
		return err
	}
	path := fmt.Sprintf("/proc/self/fd/%d/%s", parent.Fd(), e.base)
	if err := unix.Lsetxattr(path, key, value, flags); err != nil {
		return errors.WithStack(&os.PathError{Op: "lsetxattr", Path: e.path, Err: err})
	}
	return nil
}
