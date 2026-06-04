//go:build !go1.25 && (linux || darwin || freebsd || netbsd || openbsd || dragonfly)

package fsutil

import (
	"os"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/sys/unix"
)

func (r *root) Chtimes(name string, atime time.Time, mtime time.Time) error {
	ts := []unix.Timespec{
		unix.NsecToTimespec(atime.UnixNano()),
		unix.NsecToTimespec(mtime.UnixNano()),
	}

	if name == "" || name == "." {
		rootDir, err := r.rootDirFile()
		if err != nil {
			return errors.WithStack(err)
		}
		if err := unix.UtimesNanoAt(int(rootDir.Fd()), ".", ts, 0); err != nil {
			return errors.WithStack(&os.PathError{Op: "utimensat", Path: name, Err: err})
		}
		return nil
	}

	parent, base, closeParent, err := r.openRootParent(name)
	if err != nil {
		return err
	}
	if closeParent {
		defer parent.Close()
	}

	if err := unix.UtimesNanoAt(int(parent.Fd()), base, ts, 0); err != nil {
		return errors.WithStack(&os.PathError{Op: "utimensat", Path: name, Err: err})
	}
	return nil
}
