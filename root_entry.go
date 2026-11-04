package fsutil

import (
	"os"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/pkg/errors"
)

// RootEntry is an opened parent directory and final path component for a
// root-relative path.
//
// On Unix, operations use an opened parent directory but do not pin the leaf
// inode between calls. Callers must prevent concurrent relocation of that
// directory outside the intended tree. RootEntry is not safe for concurrent use.
// On other platforms, supported operations resolve the path through the root.
type RootEntry struct {
	root        *os.Root
	path        string
	base        string
	parent      *os.File
	closeParent bool
	closed      bool
}

// OpenRootEntry opens the parent directory for name within root and returns an
// entry object for operations on name's final path component.
// The root remains owned by the caller and must outlive the entry. Close the
// entry to release its parent descriptor. Names must be relative and must not
// contain .. components; archive-specific symlink resolution belongs to callers.
func OpenRootEntry(root *os.Root, name string) (*RootEntry, error) {
	if root == nil {
		return nil, errors.New("nil root")
	}
	path, dir, base, err := cleanRootEntryPath(name)
	if err != nil {
		return nil, err
	}
	parent, err := root.OpenFile(dir, rootEntryParentFlags, 0)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	if fi, err := parent.Stat(); err != nil {
		parent.Close()
		return nil, errors.WithStack(err)
	} else if !fi.IsDir() {
		parent.Close()
		return nil, errors.WithStack(&os.PathError{Op: "openat", Path: name, Err: syscall.ENOTDIR})
	}
	return &RootEntry{
		root:        root,
		path:        path,
		base:        base,
		parent:      parent,
		closeParent: true,
	}, nil
}

func cleanRootEntryPath(name string) (path, dir, base string, err error) {
	if name == "" || name == "." || name == ".." || filepath.IsAbs(name) || filepath.VolumeName(name) != "" {
		return "", "", "", errors.WithStack(&os.PathError{Op: "openat", Path: name, Err: syscall.EINVAL})
	}
	for component := range strings.SplitSeq(filepath.ToSlash(name), "/") {
		if component == ".." {
			return "", "", "", errors.WithStack(&os.PathError{Op: "openat", Path: name, Err: syscall.EINVAL})
		}
	}

	path = filepath.Clean(name)
	if path == "." {
		return "", "", "", errors.WithStack(&os.PathError{Op: "openat", Path: name, Err: syscall.EINVAL})
	}
	base = filepath.Base(path)
	if base == string(filepath.Separator) {
		return "", "", "", errors.WithStack(&os.PathError{Op: "openat", Path: name, Err: syscall.EINVAL})
	}
	dir = filepath.Dir(path)
	return path, dir, base, nil
}

// Path returns the cleaned root-relative path for the entry.
func (e *RootEntry) Path() string {
	if e == nil {
		return ""
	}
	return e.path
}

// Base returns the final path component for the entry.
func (e *RootEntry) Base() string {
	if e == nil {
		return ""
	}
	return e.base
}

// Parent returns the opened parent directory for the entry.
//
// The returned file is borrowed and must not be closed by the caller. Internal
// entries may borrow a cached descriptor from their root, which must remain open.
// On Linux it is an O_PATH descriptor, suitable for *at operations and procfs
// paths, but not for reading directory entries or calling File.Chmod.
func (e *RootEntry) Parent() (*os.File, error) {
	if e == nil || e.closed || e.parent == nil {
		return nil, errors.WithStack(os.ErrClosed)
	}
	return e.parent, nil
}

// Close releases resources held by the entry.
func (e *RootEntry) Close() error {
	if e == nil || e.closed {
		return nil
	}
	e.closed = true
	if !e.closeParent || e.parent == nil {
		e.parent = nil
		return nil
	}
	err := e.parent.Close()
	e.parent = nil
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}
