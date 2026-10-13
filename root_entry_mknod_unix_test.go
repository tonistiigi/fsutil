//go:build linux || freebsd || netbsd || openbsd || dragonfly

package fsutil

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRootEntryMknodFIFO(t *testing.T) {
	dest := t.TempDir()

	osroot, err := os.OpenRoot(dest)
	require.NoError(t, err)
	destRoot := NewRoot(osroot)
	defer destRoot.Close()

	entry, err := OpenRootEntry(destRoot, "fifo")
	require.NoError(t, err)
	defer entry.Close()

	require.NoError(t, entry.Mknod(syscall.S_IFIFO|0600, 0))

	fi, err := os.Lstat(filepath.Join(dest, "fifo"))
	require.NoError(t, err)
	require.NotZero(t, fi.Mode()&os.ModeNamedPipe)
	require.Equal(t, os.FileMode(0600), fi.Mode().Perm())
}
