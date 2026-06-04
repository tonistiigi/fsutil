//go:build !go1.25 && windows

package fsutil

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

func (r *root) rootPath(name string) (string, error) {
	if name == "" {
		name = "."
	}
	if !filepath.IsLocal(name) {
		return "", errors.WithStack(&os.PathError{Op: "rootpath", Path: name, Err: os.ErrInvalid})
	}
	return filepath.Join(r.Name(), name), nil
}

func (r *root) Lchown(name string, uid, gid int) error {
	p, err := r.rootPath(name)
	if err != nil {
		return err
	}
	return os.Lchown(p, uid, gid)
}

func (r *root) Link(oldname, newname string) error {
	oldPath, err := r.rootPath(oldname)
	if err != nil {
		return err
	}
	newPath, err := r.rootPath(newname)
	if err != nil {
		return err
	}
	return os.Link(oldPath, newPath)
}

func (r *root) Readlink(name string) (string, error) {
	p, err := r.rootPath(name)
	if err != nil {
		return "", err
	}
	return os.Readlink(p)
}

func (r *root) Rename(oldname, newname string) error {
	oldPath, err := r.rootPath(oldname)
	if err != nil {
		return err
	}
	newPath, err := r.rootPath(newname)
	if err != nil {
		return err
	}
	return os.Rename(oldPath, newPath)
}

func (r *root) Symlink(oldname, newname string) error {
	newPath, err := r.rootPath(newname)
	if err != nil {
		return err
	}
	return os.Symlink(oldname, newPath)
}
