//go:build darwin || freebsd || netbsd

package fsutil

import (
	"os"

	"github.com/pkg/errors"
	"golang.org/x/sys/unix"
)

// LSetxattr sets an xattr on the entry without following a final symlink.
func (e *RootEntry) LSetxattr(key string, value []byte, flags int) error {
	f, err := e.OpenFileNoFollow(os.O_RDONLY|unix.O_NONBLOCK, 0)
	if err != nil {
		return err
	}
	defer f.Close()

	if err := unix.Fsetxattr(int(f.Fd()), key, value, flags); err != nil {
		return errors.WithStack(&os.PathError{Op: "fsetxattr", Path: e.path, Err: err})
	}
	return nil
}
