//go:build windows

package fsutil

import (
	"os"
	"time"

	"github.com/tonistiigi/fsutil/types"
)

func rewriteRootEntryMetadata(entry *RootEntry, stat *types.Stat) error {
	if _, err := entry.Parent(); err != nil {
		return err
	}
	return chtimesRootEntry(entry, stat.ModTime)
}

func openRootEntryFile(entry *RootEntry, mode os.FileMode) (*os.File, error) {
	if _, err := entry.Parent(); err != nil {
		return nil, err
	}
	return entry.root.OpenFile(entry.path, os.O_CREATE|os.O_WRONLY, mode)
}

func openRootEntryWriteFile(entry *RootEntry) (*os.File, error) {
	if _, err := entry.Parent(); err != nil {
		return nil, err
	}
	return entry.root.OpenFile(entry.path, os.O_WRONLY, 0)
}

func chmodRootEntry(entry *RootEntry, mode os.FileMode) error {
	if _, err := entry.Parent(); err != nil {
		return err
	}
	return entry.root.Chmod(entry.path, mode)
}

func chtimesRootEntry(entry *RootEntry, un int64) error {
	if _, err := entry.Parent(); err != nil {
		return err
	}
	return entry.root.LChtimes(entry.path, time.Unix(0, un))
}
