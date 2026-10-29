package fsutil

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenRootEntryMkdir(t *testing.T) {
	dest := t.TempDir()

	osroot, err := os.OpenRoot(dest)
	require.NoError(t, err)
	destRoot := NewRoot(osroot)
	defer destRoot.Close()

	require.NoError(t, destRoot.Mkdir("parent", 0700))

	entry, err := OpenRootEntry(destRoot, filepath.Join("parent", "child"))
	require.NoError(t, err)
	require.Equal(t, filepath.Join("parent", "child"), entry.Path())
	require.Equal(t, "child", entry.Base())
	require.NoError(t, entry.Mkdir(0750))
	require.NoError(t, entry.Close())

	fi, err := os.Lstat(filepath.Join(dest, "parent", "child"))
	require.NoError(t, err)
	require.True(t, fi.IsDir())
	if runtime.GOOS != "windows" {
		require.Equal(t, os.FileMode(0750), fi.Mode().Perm())
	}

	_, err = entry.Parent()
	require.ErrorIs(t, err, os.ErrClosed)
}

type fallbackRootEntryRoot struct {
	Root
}

func TestOpenRootEntryFallback(t *testing.T) {
	dest := t.TempDir()

	osroot, err := os.OpenRoot(dest)
	require.NoError(t, err)
	destRoot := fallbackRootEntryRoot{Root: NewRoot(osroot)}
	defer destRoot.Close()

	require.NoError(t, destRoot.Mkdir("parent", 0700))

	entry, err := OpenRootEntry(destRoot, filepath.Join("parent", "child"))
	require.NoError(t, err)
	require.Equal(t, filepath.Join("parent", "child"), entry.Path())
	require.Equal(t, "child", entry.Base())
	require.NoError(t, entry.Close())
}

func TestOpenRootEntryRejectsInvalidName(t *testing.T) {
	dest := t.TempDir()

	osroot, err := os.OpenRoot(dest)
	require.NoError(t, err)
	destRoot := NewRoot(osroot)
	defer destRoot.Close()

	names := []string{
		"",
		".",
		"..",
		filepath.Join("dir", ".."),
		filepath.Join("dir", "..", "..", "file"),
		filepath.VolumeName(dest) + string(filepath.Separator),
	}
	if volume := filepath.VolumeName(dest); volume != "" {
		names = append(names, volume+"file")
	}

	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			entry, err := OpenRootEntry(destRoot, name)
			require.Error(t, err)
			require.Nil(t, entry)
		})
	}
}
