//go:build linux || darwin || freebsd || netbsd

package fsutil

import (
	"context"
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

	err = Walk(context.Background(), d, nil, readAsAdd(dw.HandleChange))
	require.NoError(t, err)

	buf := make([]byte, len(value))
	n, err := unix.Getxattr(filepath.Join(dest, "foo"), key, buf)
	require.NoError(t, err)
	require.Equal(t, value, buf[:n])
}
