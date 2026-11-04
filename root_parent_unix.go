//go:build linux || darwin || freebsd || netbsd || openbsd || dragonfly

package fsutil

import (
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"
)

var _ RootLChtimes = (*root)(nil)

func (r *root) LChtimes(name string, mtime time.Time) error {
	entry, err := r.openRootEntry(name)
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

	// openRootEntry has already validated and cleaned name.
	base := filepath.Base(name)
	dir := filepath.Dir(name)
	if dir == "." {
		parent, err := r.rootDirFile()
		if err != nil {
			return nil, "", false, errors.WithStack(err)
		}
		return parent, base, false, nil
	}

	parent, err := r.OpenFile(dir, rootEntryParentFlags, 0)
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
		r.rootDir, r.rootDirErr = r.OpenFile(".", rootEntryParentFlags, 0)
	})
	if r.rootDirErr != nil {
		return nil, r.rootDirErr
	}
	return r.rootDir, nil
}
