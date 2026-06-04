//go:build linux || darwin || freebsd || netbsd || openbsd || dragonfly

package fsutil

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRootLChtimesDoesNotFollowSymlink(t *testing.T) {
	dest := t.TempDir()
	target := filepath.Join(dest, "target")
	link := filepath.Join(dest, "link")
	require.NoError(t, os.WriteFile(target, []byte("data"), 0600))
	require.NoError(t, os.Symlink("target", link))

	targetTime := time.Unix(1000, 0)
	linkTime := time.Unix(2000, 0)
	require.NoError(t, os.Chtimes(target, targetTime, targetTime))

	osroot, err := os.OpenRoot(dest)
	require.NoError(t, err)
	destRoot := NewRoot(osroot)
	defer destRoot.Close()

	err = destRoot.LChtimes("link", linkTime)
	require.NoError(t, err)

	linkFi, err := os.Lstat(link)
	require.NoError(t, err)
	require.Equal(t, linkTime, linkFi.ModTime())

	targetFi, err := os.Stat(target)
	require.NoError(t, err)
	require.Equal(t, targetTime, targetFi.ModTime())
}

func TestRootChmodNoReadPermission(t *testing.T) {
	dest := t.TempDir()
	target := filepath.Join(dest, "target")
	require.NoError(t, os.WriteFile(target, []byte("data"), 0600))

	osroot, err := os.OpenRoot(dest)
	require.NoError(t, err)
	destRoot := NewRoot(osroot)
	defer destRoot.Close()

	require.NoError(t, destRoot.Chmod("target", 0000))
	fi, err := os.Lstat(target)
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0000), fi.Mode().Perm())

	require.NoError(t, destRoot.Chmod("target", 0600))
}

func TestRootChtimes(t *testing.T) {
	dest := t.TempDir()
	target := filepath.Join(dest, "target")
	require.NoError(t, os.WriteFile(target, []byte("data"), 0600))

	osroot, err := os.OpenRoot(dest)
	require.NoError(t, err)
	destRoot := NewRoot(osroot)
	defer destRoot.Close()

	targetTime := time.Unix(1000, 0)
	require.NoError(t, destRoot.Chtimes("target", targetTime, targetTime))
	fi, err := os.Stat(target)
	require.NoError(t, err)
	require.Equal(t, targetTime, fi.ModTime())

	rootTime := time.Unix(2000, 0)
	require.NoError(t, destRoot.Chtimes(".", rootTime, rootTime))
	rootFi, err := os.Stat(dest)
	require.NoError(t, err)
	require.Equal(t, rootTime, rootFi.ModTime())
}
