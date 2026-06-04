//go:build !go1.25 && !linux && !darwin && !freebsd && !netbsd && !openbsd && !dragonfly && !windows

package fsutil

import (
	"os"
	"time"
)

func (r *root) Chtimes(name string, atime time.Time, mtime time.Time) error {
	return unsupportedRootOp("utimensat", name, os.ErrInvalid)
}
