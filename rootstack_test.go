package fsutil

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRootStackUsesSubrootAndBasename(t *testing.T) {
	dest := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dest, "foo"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dest, "foo", "bar"), []byte("data"), 0600))

	osroot, err := os.OpenRoot(dest)
	require.NoError(t, err)
	root := NewRoot(osroot)
	defer root.Close()

	stack := newRootStack(root)
	defer stack.Close()

	subroot, base, err := stack.get(filepath.Join("foo", "bar"))
	require.NoError(t, err)
	require.Equal(t, "bar", base)

	f, err := subroot.OpenFile(base, os.O_RDONLY, 0)
	require.NoError(t, err)
	defer f.Close()

	dt, err := io.ReadAll(f)
	require.NoError(t, err)
	require.Equal(t, []byte("data"), dt)

	rootAgain, base, err := stack.get("baz")
	require.NoError(t, err)
	require.Equal(t, "baz", base)
	require.True(t, root == rootAgain)
}
