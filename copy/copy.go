package fs

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"
)

var bufferPool = &sync.Pool{
	New: func() interface{} {
		buffer := make([]byte, 32*1024)
		return &buffer
	},
}

func Copy(ctx context.Context, src, dst string, opts ...Opt) error {
	var ci CopyInfo
	for _, o := range opts {
		o(&ci)
	}

	srcFollowed, err := filepath.EvalSymlinks(src)
	if err != nil {
		return err
	}

	srcBase := filepath.Base(src)

	ensureDstPath := dst
	if d, f := filepath.Split(dst); f == "" {
		ensureDstPath = d
	}
	if err := os.MkdirAll(ensureDstPath, 0700); err != nil {
		return err
	}

	return newCopier(ci.Chown).copy(ctx, srcBase, srcFollowed, filepath.Clean(dst))
}

type ChownOpt struct {
	Uid, Gid int
}

type CopyInfo struct {
	Chown *ChownOpt
}

type Opt func(*CopyInfo)

func WithChown(uid, gid int) Opt {
	return func(ci *CopyInfo) {
		ci.Chown = &ChownOpt{Uid: uid, Gid: gid}
	}
}

type copier struct {
	chown  *ChownOpt
	inodes map[uint64]string
}

func newCopier(chown *ChownOpt) *copier {
	return &copier{inodes: map[uint64]string{}, chown: chown}
}

func (c *copier) copy(ctx context.Context, base, src, dst string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	dstBase := base
	fi, err := os.Lstat(src)
	if err != nil {
		return errors.Wrapf(err, "failed to stat %s", src)
	}

	if _, err := os.Lstat(dst); err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		dst, dstBase = filepath.Split(dst)
	}

	target := filepath.Join(dst, dstBase)
	if !fi.IsDir() {
		if err := ensureEmptyFileTarget(target); err != nil {
			return err
		}
	}

	switch {
	case fi.IsDir():
		if err := c.copyDirectory(ctx, src, target, fi); err != nil {
			return err
		}
	case (fi.Mode() & os.ModeType) == 0:
		link, err := getLinkSource(target, fi, c.inodes)
		if err != nil {
			return errors.Wrap(err, "failed to get hardlink")
		}
		if link != "" {
			if err := os.Link(link, target); err != nil {
				return errors.Wrap(err, "failed to create hard link")
			}
		} else if err := copyFile(src, target); err != nil {
			return errors.Wrap(err, "failed to copy files")
		}
	case (fi.Mode() & os.ModeSymlink) == os.ModeSymlink:
		link, err := os.Readlink(src)
		if err != nil {
			return errors.Wrapf(err, "failed to read link: %s", src)
		}
		if err := os.Symlink(link, target); err != nil {
			return errors.Wrapf(err, "failed to create symlink: %s", target)
		}
	case (fi.Mode() & os.ModeDevice) == os.ModeDevice:
		if err := copyDevice(target, fi); err != nil {
			return errors.Wrapf(err, "failed to create device")
		}
	default:
		// TODO: Support pipes and sockets
		return errors.Wrapf(err, "unsupported mode %s", fi.Mode())
	}
	if err := c.copyFileInfo(fi, target); err != nil {
		return errors.Wrap(err, "failed to copy file info")
	}

	if err := copyXAttrs(target, src); err != nil {
		return errors.Wrap(err, "failed to copy xattrs")
	}
	return nil
}

func (c *copier) copyDirectory(ctx context.Context, src, dst string, stat os.FileInfo) error {
	if !stat.IsDir() {
		return errors.Errorf("source is not directory")
	}

	if st, err := os.Lstat(dst); err != nil {
		if err := os.Mkdir(dst, stat.Mode()); err != nil {
			return errors.Wrapf(err, "failed to mkdir %s", dst)
		}
	} else if !st.IsDir() {
		return errors.Errorf("cannot copy to non-directory: %s", dst)
	} else {
		if err := os.Chmod(dst, stat.Mode()); err != nil {
			return errors.Wrapf(err, "failed to chmod on %s", dst)
		}
	}

	fis, err := ioutil.ReadDir(src)
	if err != nil {
		return errors.Wrapf(err, "failed to read %s", src)
	}

	for _, fi := range fis {
		if err := c.copy(ctx, fi.Name(), filepath.Join(src, fi.Name()), filepath.Join(dst, fi.Name())); err != nil {
			return err
		}
	}

	return nil
}

func ensureEmptyFileTarget(dst string) error {
	fi, err := os.Lstat(dst)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if fi.IsDir() {
		return errors.Errorf("cannot replace to directory %s with file", dst)
	}
	return os.Remove(dst)
}

func copyFile(source, target string) error {
	src, err := os.Open(source)
	if err != nil {
		return errors.Wrapf(err, "failed to open source %s", source)
	}
	defer src.Close()
	tgt, err := os.Create(target)
	if err != nil {
		return errors.Wrapf(err, "failed to open target %s", target)
	}
	defer tgt.Close()

	return copyFileContent(tgt, src)
}
