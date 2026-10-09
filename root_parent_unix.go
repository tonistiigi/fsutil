//go:build linux || darwin || freebsd || netbsd || openbsd || dragonfly

package fsutil

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/sys/unix"
)

var _ RootLChtimes = (*root)(nil)

func (r *root) LChtimes(name string, mtime time.Time) error {
	entry, err := OpenRootEntry(r, name)
	if err != nil {
		return err
	}
	defer entry.Close()
	return entry.ChtimesNoFollow(mtime, mtime)
}

func (r *root) openRootParent(name string) (*os.File, string, bool, error) {
	if r == nil {
		return nil, "", false, errors.New("nil root")
	}

	// fast path for direct basename
	if !strings.ContainsRune(name, filepath.Separator) {
		if name == "" || name == "." || name == ".." {
			return nil, "", false, errors.WithStack(&os.PathError{Op: "openat", Path: name, Err: syscall.EINVAL})
		}
		parent, err := r.rootDirFile()
		if err != nil {
			return nil, "", false, errors.WithStack(err)
		}
		return parent, name, false, nil
	}

	cleaned := filepath.Clean(name)
	base := filepath.Base(cleaned)
	if base == "." || base == ".." {
		return nil, "", false, errors.WithStack(&os.PathError{Op: "openat", Path: name, Err: syscall.EINVAL})
	}

	dir := filepath.Dir(cleaned)
	if dir == "." {
		parent, err := r.rootDirFile()
		if err != nil {
			return nil, "", false, errors.WithStack(err)
		}
		return parent, base, false, nil
	}

	parent, err := r.OpenFile(dir, os.O_RDONLY|unix.O_DIRECTORY, 0)
	if err != nil {
		return nil, "", false, errors.WithStack(err)
	}
	return parent, base, true, nil
}

func (r *root) rootDirFile() (*os.File, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return nil, os.ErrClosed
	}
	if r.Root == nil {
		return nil, errors.New("nil root")
	}
	r.rootDirOnce.Do(func() {
		r.rootDir, r.rootDirErr = r.OpenFile(".", os.O_RDONLY|unix.O_DIRECTORY, 0)
	})
	if r.rootDirErr != nil {
		return nil, r.rootDirErr
	}
	return r.rootDir, nil
}
