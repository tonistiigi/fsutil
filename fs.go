package fsutil

import (
	"context"
	io "io"
	"os"
	"path/filepath"
)

type FS interface {
	Walk(context.Context, filepath.WalkFunc) error
	Open(string) (io.ReadCloser, error)
}

func NewFS(root string, opt *WalkOpt) FS {
	return &fs{
		root: root,
		opt:  opt,
	}
}

type fs struct {
	root string
	opt  *WalkOpt
}

func (fs *fs) Walk(ctx context.Context, fn filepath.WalkFunc) error {
	return Walk(ctx, fs.root, fs.opt, fn)
}

func (fs *fs) Open(p string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(fs.root, p))
}
