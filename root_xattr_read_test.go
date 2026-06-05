//go:build linux || darwin || freebsd || netbsd

package fsutil

import (
	"context"
	gofs "io/fs"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/tonistiigi/fsutil/types"
	"golang.org/x/sys/unix"
)

func TestRootFSWalkXattrs(t *testing.T) {
	dest := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dest, "file"), []byte("data"), 0600))

	key := "user.fsutil.rootfs"
	value := []byte("value")
	err := unix.Setxattr(filepath.Join(dest, "file"), key, value, 0)
	skipUnsupportedXattr(t, err)
	require.NoError(t, err)

	osroot, err := os.OpenRoot(dest)
	require.NoError(t, err)
	root := NewRoot(osroot)
	defer root.Close()

	var stat *types.Stat
	err = NewRootFS(root).Walk(context.Background(), "/", func(path string, entry gofs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if path != "file" {
			return nil
		}
		fi, err := entry.Info()
		if err != nil {
			return err
		}
		stat = fi.Sys().(*types.Stat)
		return nil
	})
	require.NoError(t, err)
	require.NotNil(t, stat)
	require.Equal(t, value, stat.Xattrs[key])
}

func TestLoadRootXattrIgnoresOpenPermissionError(t *testing.T) {
	err := loadRootXattr(&rootXattrOpenErrorRoot{err: syscall.EACCES}, "file", &types.Stat{})
	require.NoError(t, err)

	err = loadRootXattr(&rootXattrOpenErrorRoot{err: syscall.EPERM}, "file", &types.Stat{})
	require.NoError(t, err)
}

type rootXattrOpenErrorRoot struct {
	Root
	err error
}

func (r *rootXattrOpenErrorRoot) OpenFile(string, int, os.FileMode) (*os.File, error) {
	return nil, r.err
}
