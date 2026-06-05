//go:build linux

package fsutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func TestRootLSetxattr(t *testing.T) {
	testRootLSetxattr(t)
}

func testRootLSetxattr(t *testing.T) {
	t.Helper()

	dest := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dest, "file"), []byte("data"), 0600))

	osroot, err := os.OpenRoot(dest)
	require.NoError(t, err)
	destRoot := NewRoot(osroot)
	defer destRoot.Close()

	key := "user.fsutil.root"
	value := []byte("value")
	err = destRoot.LSetxattr("file", key, value, 0)
	skipUnsupportedXattr(t, err)
	require.NoError(t, err)

	buf := make([]byte, len(value))
	n, err := unix.Getxattr(filepath.Join(dest, "file"), key, buf)
	require.NoError(t, err)
	require.Equal(t, value, buf[:n])
}
