//go:build linux || darwin || freebsd || netbsd

package fsutil

import (
	"context"
	gofs "io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func TestRootDiskWriterXattrs(t *testing.T) {
	d, err := tmpDir(changeStream([]string{
		"ADD foo file data1",
	}))
	require.NoError(t, err)
	defer os.RemoveAll(d)

	key := "user.fsutil.rootdiskwriter"
	value := []byte("value")
	err = unix.Setxattr(filepath.Join(d, "foo"), key, value, 0)
	skipUnsupportedXattr(t, err)
	require.NoError(t, err)

	dest := t.TempDir()
	root, err := os.OpenRoot(dest)
	require.NoError(t, err)
	destRoot := NewRoot(root)
	defer destRoot.Close()

	dw, err := NewRootDiskWriter(context.TODO(), destRoot, DiskWriterOpt{
		SyncDataCb: newWriteToFunc(d, 0),
	})
	require.NoError(t, err)

	source, err := os.OpenRoot(d)
	require.NoError(t, err)
	sourceRoot := NewRoot(source)
	defer sourceRoot.Close()

	err = NewRootFS(sourceRoot).Walk(context.Background(), "/", func(path string, entry gofs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		fi, err := entry.Info()
		if err != nil {
			return err
		}
		return dw.HandleChange(ChangeKindAdd, path, fi, nil)
	})
	require.NoError(t, err)

	buf := make([]byte, len(value))
	n, err := unix.Getxattr(filepath.Join(dest, "foo"), key, buf)
	require.NoError(t, err)
	require.Equal(t, value, buf[:n])
}
