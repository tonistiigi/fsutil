//go:build linux || darwin || freebsd || netbsd

package fsutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func TestRootEntryLSetxattr(t *testing.T) {
	dest := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dest, "file"), []byte("data"), 0600))

	osroot, err := os.OpenRoot(dest)
	require.NoError(t, err)
	destRoot := NewRoot(osroot)
	defer destRoot.Close()

	entry, err := OpenRootEntry(destRoot, "file")
	require.NoError(t, err)
	defer entry.Close()

	key := "user.fsutil.rootentry"
	value := []byte("value")
	err = entry.LSetxattr(key, value, 0)
	skipUnsupportedXattr(t, err)
	require.NoError(t, err)

	buf := make([]byte, len(value))
	n, err := unix.Getxattr(filepath.Join(dest, "file"), key, buf)
	require.NoError(t, err)
	require.Equal(t, value, buf[:n])
}
