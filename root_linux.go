//go:build linux

package fsutil

import (
	"os"
	"path/filepath"
	"strconv"

	"github.com/pkg/errors"
	"golang.org/x/sys/unix"
)

var (
	_ RootXattr = (*root)(nil)
)

func (r *root) LSetxattr(name, key string, value []byte, flags int) error {
	parent, base, closeParent, err := r.openRootParent(name)
	if err != nil {
		return err
	}
	if closeParent {
		defer parent.Close()
	}

	if err := unix.Lsetxattr(procSelfFdPath(int(parent.Fd()), base), key, value, flags); err != nil {
		return errors.WithStack(&os.PathError{Op: "lsetxattr", Path: name, Err: err})
	}
	return nil
}

func procSelfFdPath(parent int, base string) string {
	return filepath.Join("/proc/self/fd", strconv.FormatUint(uint64(parent), 10), base)
}
