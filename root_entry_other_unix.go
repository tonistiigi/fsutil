//go:build darwin || freebsd || netbsd || openbsd || dragonfly

package fsutil

import (
	"os"

	"github.com/pkg/errors"
	"golang.org/x/sys/unix"
)

const rootEntryParentFlags = os.O_RDONLY | unix.O_DIRECTORY

// ChmodNoFollow changes mode without following a final symlink.
func (e *RootEntry) ChmodNoFollow(mode os.FileMode) error {
	parent, err := e.Parent()
	if err != nil {
		return err
	}
	if err := unix.Fchmodat(int(parent.Fd()), e.base, rootEntryPerm(mode), unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return errors.WithStack(&os.PathError{Op: "fchmodat", Path: e.path, Err: err})
	}
	return nil
}
