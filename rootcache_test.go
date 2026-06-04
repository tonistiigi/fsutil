package fsutil

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRootCacheUsesCachedParentChain(t *testing.T) {
	dest := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dest, "a", "b", "c"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dest, "a", "b", "c", "file"), []byte("data"), 0600))

	osroot, err := os.OpenRoot(dest)
	require.NoError(t, err)
	root := NewRoot(osroot)
	defer root.Close()

	cache := newRootCache(root, 16)
	defer cache.Close()

	lease, err := cache.get(filepath.Join("a", "b", "c", "file"))
	require.NoError(t, err)
	defer lease.Release()
	require.Equal(t, "file", lease.base)

	f, err := lease.root.OpenFile(lease.base, os.O_RDONLY, 0)
	require.NoError(t, err)
	defer f.Close()

	dt, err := io.ReadAll(f)
	require.NoError(t, err)
	require.Equal(t, []byte("data"), dt)

	require.Contains(t, cache.entries, "a")
	require.Contains(t, cache.entries, filepath.Join("a", "b"))
	require.Contains(t, cache.entries, filepath.Join("a", "b", "c"))
}
