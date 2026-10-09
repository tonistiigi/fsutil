//go:build linux || darwin || freebsd || netbsd || openbsd || dragonfly

package fsutil

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestRootEntryOpenFileNoFollowRejectsFinalSymlink(t *testing.T) {
	dest := t.TempDir()
	target := filepath.Join(dest, "target")
	require.NoError(t, os.WriteFile(target, []byte("target"), 0600))
	require.NoError(t, os.Symlink("target", filepath.Join(dest, "link")))

	osroot, err := os.OpenRoot(dest)
	require.NoError(t, err)
	destRoot := NewRoot(osroot)
	defer destRoot.Close()

	entry, err := OpenRootEntry(destRoot, "link")
	require.NoError(t, err)
	defer entry.Close()

	file, err := entry.OpenFileNoFollow(os.O_WRONLY|os.O_TRUNC, 0)
	if file != nil {
		require.NoError(t, file.Close())
	}
	require.Error(t, err)

	data, err := os.ReadFile(target)
	require.NoError(t, err)
	require.Equal(t, []byte("target"), data)
}

func TestRootEntryChtimesNoFollowDoesNotFollowSymlink(t *testing.T) {
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

	entry, err := OpenRootEntry(destRoot, "link")
	require.NoError(t, err)
	defer entry.Close()

	require.NoError(t, entry.ChtimesNoFollow(linkTime, linkTime))

	linkFi, err := os.Lstat(link)
	require.NoError(t, err)
	require.Equal(t, linkTime, linkFi.ModTime())

	targetFi, err := os.Stat(target)
	require.NoError(t, err)
	require.Equal(t, targetTime, targetFi.ModTime())
}
