package fsutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWrapRootDoesNotCloseOSRoot(t *testing.T) {
	dest := t.TempDir()

	osroot, err := os.OpenRoot(dest)
	require.NoError(t, err)
	defer osroot.Close()

	root := WrapRoot(osroot)
	require.NoError(t, root.Mkdir("before", 0700))
	require.NoError(t, root.Close())

	f, err := osroot.OpenFile("after", os.O_CREATE|os.O_WRONLY, 0600)
	require.NoError(t, err)
	require.NoError(t, f.Close())

	_, err = os.Lstat(filepath.Join(dest, "after"))
	require.NoError(t, err)
}

func TestNewRootClosesOSRoot(t *testing.T) {
	dest := t.TempDir()

	osroot, err := os.OpenRoot(dest)
	require.NoError(t, err)

	root := NewRoot(osroot)
	require.NoError(t, root.Close())

	_, err = osroot.OpenFile("after", os.O_CREATE|os.O_WRONLY, 0600)
	require.ErrorIs(t, err, os.ErrClosed)
}
