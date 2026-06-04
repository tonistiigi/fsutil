//go:build !go1.25 && !linux && !darwin && !freebsd && !netbsd && !openbsd && !dragonfly

package fsutil

import "os"

func (r *root) Chmod(name string, mode os.FileMode) error {
	f, err := r.OpenFile(name, os.O_RDONLY, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	return f.Chmod(mode)
}
