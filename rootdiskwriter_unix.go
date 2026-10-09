//go:build !windows

package fsutil

import (
	"os"
	"syscall"
	"time"

	"github.com/pkg/errors"
	"github.com/tonistiigi/fsutil/types"
)

func rewriteRootEntryMetadata(entry *RootEntry, stat *types.Stat) error {
	for key, value := range stat.Xattrs {
		_ = entry.LSetxattr(key, value, 0)
	}

	if err := entry.Lchown(int(stat.Uid), int(stat.Gid)); err != nil {
		return errors.WithStack(err)
	}

	mtime := time.Unix(0, stat.ModTime)
	if os.FileMode(stat.Mode)&os.ModeSymlink != 0 {
		return entry.ChtimesNoFollow(mtime, mtime)
	}

	if err := chmodRootEntry(entry, os.FileMode(stat.Mode)); err != nil {
		return err
	}
	return entry.ChtimesNoFollow(mtime, mtime)
}

func chtimesRootEntry(entry *RootEntry, un int64) error {
	t := time.Unix(0, un)
	return entry.ChtimesNoFollow(t, t)
}

func openRootEntryFile(entry *RootEntry, mode os.FileMode) (*os.File, error) {
	return entry.OpenFileNoFollow(os.O_CREATE|os.O_WRONLY, mode)
}

func openRootEntryWriteFile(entry *RootEntry) (*os.File, error) {
	return entry.OpenFileNoFollow(os.O_WRONLY, 0)
}

func chmodRootEntry(entry *RootEntry, mode os.FileMode) error {
	if err := entry.ChmodNoFollow(mode); err != nil {
		if !errors.Is(err, syscall.ENOSYS) && !errors.Is(err, syscall.EOPNOTSUPP) {
			return errors.WithStack(err)
		}
		if err := entry.root.Chmod(entry.path, mode); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}
