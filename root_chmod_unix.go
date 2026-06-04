//go:build !go1.25 && (linux || darwin || freebsd || netbsd || openbsd || dragonfly)

package fsutil

import (
	"os"

	"github.com/pkg/errors"
	"golang.org/x/sys/unix"
)

func (r *root) Chmod(name string, mode os.FileMode) error {
	if name == "" || name == "." {
		rootDir, err := r.rootDirFile()
		if err != nil {
			return errors.WithStack(err)
		}
		if err := unix.Fchmod(int(rootDir.Fd()), rootChmodMode(mode)); err != nil {
			return errors.WithStack(&os.PathError{Op: "fchmod", Path: name, Err: err})
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

	if err := unix.Fchmodat(int(parent.Fd()), base, rootChmodMode(mode), 0); err != nil {
		return errors.WithStack(&os.PathError{Op: "fchmodat", Path: name, Err: err})
	}
	return nil
}

func rootChmodMode(mode os.FileMode) uint32 {
	m := uint32(mode.Perm())
	if mode&os.ModeSetuid != 0 {
		m |= unix.S_ISUID
	}
	if mode&os.ModeSetgid != 0 {
		m |= unix.S_ISGID
	}
	if mode&os.ModeSticky != 0 {
		m |= unix.S_ISVTX
	}
	return m
}
