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

	entry, err := OpenRootEntry(osroot, "link")
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

	entry, err := OpenRootEntry(osroot, "link")
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

func TestRootEntryKeepsOpenedParent(t *testing.T) {
	dest := t.TempDir()
	outside := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dest, "parent"), 0700))
	root, err := os.OpenRoot(dest)
	require.NoError(t, err)
	defer root.Close()
	entry, err := OpenRootEntry(root, "parent/file")
	require.NoError(t, err)
	defer entry.Close()
	require.NoError(t, root.Rename("parent", "moved"))
	require.NoError(t, root.Symlink(outside, "parent"))
	f, err := entry.OpenFileNoFollow(os.O_CREATE|os.O_WRONLY, 0600)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	_, err = os.Stat(filepath.Join(dest, "moved", "file"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(outside, "file"))
	require.ErrorIs(t, err, os.ErrNotExist)
	_, err = OpenRootEntry(root, "parent/another")
	require.Error(t, err)
}

func TestRootEntryInternalCloseKeepsCachedParent(t *testing.T) {
	osroot, err := os.OpenRoot(t.TempDir())
	require.NoError(t, err)
	r := NewRoot(osroot).(*root)
	defer r.Close()
	entry, err := r.openRootEntry("first")
	require.NoError(t, err)
	parent, err := entry.Parent()
	require.NoError(t, err)
	require.NoError(t, entry.Close())
	_, err = parent.Stat()
	require.NoError(t, err)
	entry, err = r.openRootEntry("second")
	require.NoError(t, err)
	defer entry.Close()
	require.NoError(t, entry.Mkdir(0700))
}

func TestRootEntryFollowsSymlinkedParent(t *testing.T) {
	dest := t.TempDir()
	root, err := os.OpenRoot(dest)
	require.NoError(t, err)
	defer root.Close()
	require.NoError(t, root.Mkdir("dir", 0700))
	require.NoError(t, root.Symlink("dir", "link"))
	entry, err := OpenRootEntry(root, "link/child")
	require.NoError(t, err)
	defer entry.Close()
	require.NoError(t, entry.Mkdir(0700))
	fi, err := root.Lstat("dir/child")
	require.NoError(t, err)
	require.True(t, fi.IsDir())
}
