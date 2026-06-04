//go:build !go1.25

package fsutil

import (
	iofs "io/fs"
	"os"

	"github.com/pkg/errors"
)

func (r *root) RemoveAll(name string) error {
	if name == "" {
		name = "."
	}
	fi, err := r.Lstat(name)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if !fi.IsDir() {
		err := r.Remove(name)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	osroot, err := r.OpenRoot(name)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	subroot := NewRoot(osroot)

	entries, err := iofs.ReadDir(subroot.FS(), ".")
	if err == nil {
		for _, entry := range entries {
			if err = subroot.RemoveAll(entry.Name()); err != nil {
				break
			}
		}
	}
	if closeErr := subroot.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}

	if name == "" || name == "." {
		return nil
	}
	err = r.Remove(name)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}
