//go:build linux || freebsd || netbsd || openbsd || dragonfly

package fsutil

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRootMknodFIFO(t *testing.T) {
	dest := t.TempDir()

	osroot, err := os.OpenRoot(dest)
	require.NoError(t, err)
	destRoot := NewRoot(osroot)
	defer destRoot.Close()

	err = destRoot.Mknod("fifo", syscall.S_IFIFO|0600, 0)
	require.NoError(t, err)

	fi, err := os.Lstat(filepath.Join(dest, "fifo"))
	require.NoError(t, err)
	require.NotZero(t, fi.Mode()&os.ModeNamedPipe)
	require.Equal(t, os.FileMode(0600), fi.Mode().Perm())
}

func TestRootMknodRejectsEscape(t *testing.T) {
	dest := t.TempDir()
	outside := filepath.Join(filepath.Dir(dest), "fsutil-root-escape")
	require.NoError(t, os.RemoveAll(outside))
	t.Cleanup(func() { os.RemoveAll(outside) })

	osroot, err := os.OpenRoot(dest)
	require.NoError(t, err)
	destRoot := NewRoot(osroot)
	defer destRoot.Close()

	err = destRoot.Mknod("../"+filepath.Base(outside), syscall.S_IFIFO|0600, 0)
	require.Error(t, err)

	_, err = os.Lstat(outside)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestRootCloseClosesWrappedRoot(t *testing.T) {
	dest := t.TempDir()

	osroot, err := os.OpenRoot(dest)
	require.NoError(t, err)

	destRoot := NewRoot(osroot)
	require.NoError(t, destRoot.Mknod("fifo", syscall.S_IFIFO|0600, 0))
	require.NoError(t, destRoot.Close())

	_, err = osroot.Lstat("fifo")
	require.ErrorIs(t, err, os.ErrClosed)

	err = destRoot.Mknod("fifo2", syscall.S_IFIFO|0600, 0)
	require.ErrorIs(t, err, os.ErrClosed)
}
