package fsutil

import (
	"context"
	gofs "io/fs"
	"os"
	"path/filepath"
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

func TestRootWalkerSkipsConcurrentlyRemovedEntry(t *testing.T) {
	dest := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dest, "foo"), 0755))

	root := &rootWalkerTestRoot{
		fs: fstest.MapFS{
			"foo":     {Mode: os.ModeDir | 0755},
			"foo/bar": {Data: []byte("gone"), Mode: 0644},
		},
		dir: dest,
	}

	pathC := make(chan *currentPath, 10)
	err := getRootWalkerFn(root)(context.Background(), pathC)
	close(pathC)
	require.NoError(t, err)

	var paths []string
	for p := range pathC {
		paths = append(paths, p.path)
	}
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
