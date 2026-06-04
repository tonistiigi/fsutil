//go:build !go1.25 && (linux || darwin || freebsd || netbsd || openbsd || dragonfly)

package fsutil

import (
	"os"

	"github.com/pkg/errors"
	"golang.org/x/sys/unix"
)

func (r *root) Lchown(name string, uid, gid int) error {
	if name == "" || name == "." {
		rootDir, err := r.rootDirFile()
		if err != nil {
			return errors.WithStack(err)
		}
		if err := unix.Fchown(int(rootDir.Fd()), uid, gid); err != nil {
			return errors.WithStack(&os.PathError{Op: "fchown", Path: name, Err: err})
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

	if err := unix.Fchownat(int(parent.Fd()), base, uid, gid, unix.AT_SYMLINK_NOFOLLOW); err != nil {
		return errors.WithStack(&os.PathError{Op: "fchownat", Path: name, Err: err})
	}
	return nil
}

func (r *root) Link(oldname, newname string) error {
	oldParent, oldBase, closeOldParent, err := r.openRootParent(oldname)
	if err != nil {
		return err
	}
	if closeOldParent {
		defer oldParent.Close()
	}

	newParent, newBase, closeNewParent, err := r.openRootParent(newname)
	if err != nil {
		return err
	}
	if closeNewParent {
		defer newParent.Close()
	}

	if err := unix.Linkat(int(oldParent.Fd()), oldBase, int(newParent.Fd()), newBase, 0); err != nil {
		return errors.WithStack(&os.PathError{Op: "linkat", Path: newname, Err: err})
	}
	return nil
}

func (r *root) Readlink(name string) (string, error) {
	parent, base, closeParent, err := r.openRootParent(name)
	if err != nil {
		return "", err
	}
	if closeParent {
		defer parent.Close()
	}

	for size := 128; ; size *= 2 {
		buf := make([]byte, size)
		n, err := unix.Readlinkat(int(parent.Fd()), base, buf)
		if err != nil {
			return "", errors.WithStack(&os.PathError{Op: "readlinkat", Path: name, Err: err})
		}
		if n < len(buf) {
			return string(buf[:n]), nil
		}
	}
}

func (r *root) Rename(oldname, newname string) error {
	oldParent, oldBase, closeOldParent, err := r.openRootParent(oldname)
	if err != nil {
		return err
	}
	if closeOldParent {
		defer oldParent.Close()
	}

	newParent, newBase, closeNewParent, err := r.openRootParent(newname)
	if err != nil {
		return err
	}
	if closeNewParent {
		defer newParent.Close()
	}

	if err := unix.Renameat(int(oldParent.Fd()), oldBase, int(newParent.Fd()), newBase); err != nil {
		return errors.WithStack(&os.PathError{Op: "renameat", Path: oldname, Err: err})
	}
	return nil
}

func (r *root) Symlink(oldname, newname string) error {
	parent, base, closeParent, err := r.openRootParent(newname)
	if err != nil {
		return err
	}
	if closeParent {
		defer parent.Close()
	}

	if err := unix.Symlinkat(oldname, int(parent.Fd()), base); err != nil {
		return errors.WithStack(&os.PathError{Op: "symlinkat", Path: newname, Err: err})
	}
	return nil
}
