package fsutil

import (
	"context"
	gofs "io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
	"github.com/tonistiigi/fsutil/types"
)

func TestRootFSWalkSkipsConcurrentlyRemovedEntry(t *testing.T) {
	dest := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dest, "foo"), 0755))

	root := &rootWalkerTestRoot{
		fs: fstest.MapFS{
			"foo":     {Mode: os.ModeDir | 0755},
			"foo/bar": {Data: []byte("gone"), Mode: 0644},
		},
		dir: dest,
	}

	var paths []string
	err := NewRootFS(root).Walk(context.Background(), "/", func(path string, entry gofs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		fi, err := entry.Info()
		if err != nil {
			return err
		}
		paths = append(paths, fi.Sys().(*types.Stat).Path)
		return nil
	})
	require.NoError(t, err)

	require.Equal(t, []string{"foo"}, paths)
}

type rootWalkerTestRoot struct {
	Root
	fs  gofs.FS
	dir string
}

func (r *rootWalkerTestRoot) FS() gofs.FS {
	return r.fs
}

func (r *rootWalkerTestRoot) Lstat(name string) (os.FileInfo, error) {
	if name != "foo" {
		return nil, &os.PathError{Op: "statat", Path: name, Err: os.ErrNotExist}
	}
	return os.Lstat(filepath.Join(r.dir, name))
}
