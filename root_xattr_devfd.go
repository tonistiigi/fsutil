//go:build darwin || freebsd || netbsd

package fsutil

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/pkg/errors"
	"golang.org/x/sys/unix"
)

var _ RootXattr = (*root)(nil)

func (r *root) LSetxattr(name, key string, value []byte, flags int) error {
	parent, base, closeParent, err := r.openRootParent(name)
	if err != nil {
		return err
	}
	if closeParent {
		defer parent.Close()
	}

	if err := unix.Lsetxattr(devFdPath(int(parent.Fd()), base), key, value, flags); err != nil {
		return errors.WithStack(&os.PathError{Op: "lsetxattr", Path: name, Err: err})
	}
	return nil
}

func devFdPath(parent int, base string) string {
	return filepath.Join("/dev/fd", strconv.FormatUint(uint64(parent), 10), base)
}
