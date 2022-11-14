package fsutil

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/containerd/continuity/fs/fstest"
	"github.com/stretchr/testify/require"
)

func TestFollowLinks(t *testing.T) {
	tmpDir := t.TempDir()

	apply := fstest.Apply(
		fstest.CreateDir("dir", 0700),
		fstest.CreateFile("dir/foo", []byte("contents"), 0600),
		fstest.Symlink("foo", "dir/l1"),
		fstest.Symlink("dir/l1", "l2"),
		fstest.CreateFile("bar", nil, 0600),
		fstest.CreateFile("baz", nil, 0600),
	)

	require.NoError(t, apply.Apply(tmpDir))

	out, err := FollowLinks(tmpDir, []string{"l2", "bar"})
	require.NoError(t, err)

	require.Equal(t, out, []string{"bar", filepath.FromSlash("dir/foo"), filepath.FromSlash("dir/l1"), "l2"})
}

func TestFollowLinksLoop(t *testing.T) {
	tmpDir := t.TempDir()

	apply := fstest.Apply(
		fstest.Symlink("l1", "l1"),
		fstest.Symlink("l2", "l3"),
		fstest.Symlink("l3", "l2"),
	)
	require.NoError(t, apply.Apply(tmpDir))

	out, err := FollowLinks(tmpDir, []string{"l1", "l3"})
	require.NoError(t, err)

	require.Equal(t, out, []string{"l1", "l2", "l3"})
}

func TestFollowLinksAbsolute(t *testing.T) {
	tmpDir := t.TempDir()

	abslutePathForBaz := "/foo/bar/baz"
	if runtime.GOOS == "windows" {
		abslutePathForBaz = "C:/foo/bar/baz"
	}

	apply := fstest.Apply(
		fstest.CreateDir("dir", 0700),
		fstest.Symlink(abslutePathForBaz, "dir/l1"),
		fstest.CreateDir("foo", 0700),
		fstest.Symlink("../", "foo/bar"),
		fstest.CreateFile("baz", nil, 0600),
	)
	require.NoError(t, apply.Apply(tmpDir))

	out, err := FollowLinks(tmpDir, []string{"dir/l1"})
	require.NoError(t, err)

	require.Equal(t, []string{"baz", filepath.FromSlash("dir/l1"), filepath.FromSlash("foo/bar")}, out)

	// same but a link outside root
	tmpDir = t.TempDir()

	apply = fstest.Apply(
		fstest.CreateDir("dir", 0700),
		fstest.Symlink(abslutePathForBaz, "dir/l1"),
		fstest.CreateDir("foo", 0700),
		fstest.Symlink("../../../", "foo/bar"),
		fstest.CreateFile("baz", nil, 0600),
	)
	require.NoError(t, apply.Apply(tmpDir))

	out, err = FollowLinks(tmpDir, []string{"dir/l1"})
	require.NoError(t, err)

	require.Equal(t, []string{"baz", filepath.FromSlash("dir/l1"), filepath.FromSlash("foo/bar")}, out)
}

func TestFollowLinksNotExists(t *testing.T) {
	tmpDir := t.TempDir()

	out, err := FollowLinks(tmpDir, []string{"foo/bar/baz", "bar/baz"})
	require.NoError(t, err)

	require.Equal(t, out, []string{filepath.FromSlash("bar/baz"), filepath.FromSlash("foo/bar/baz")})

	// root works fine with empty directory
	out, err = FollowLinks(tmpDir, []string{"."})
	require.NoError(t, err)

	require.Equal(t, out, []string(nil))

	out, err = FollowLinks(tmpDir, []string{"f*/foo/t*"})
	require.NoError(t, err)

	require.Equal(t, []string{filepath.FromSlash("f*/foo/t*")}, out)
}

func TestFollowLinksNormalized(t *testing.T) {
	tmpDir := t.TempDir()

	out, err := FollowLinks(tmpDir, []string{"foo/bar/baz", "foo/bar"})
	require.NoError(t, err)

	require.Equal(t, out, []string{filepath.FromSlash("foo/bar")})

	rootPath := "/"
	if runtime.GOOS == "windows" {
		rootPath = "C:/"
	}

	apply := fstest.Apply(
		fstest.CreateDir("dir", 0700),
		fstest.Symlink(filepath.Join(rootPath, "foo"), "dir/l1"),
		fstest.Symlink(rootPath, "dir/l2"),
		fstest.CreateDir("foo", 0700),
		fstest.CreateFile("foo/bar", nil, 0600),
	)
	require.NoError(t, apply.Apply(tmpDir))

	out, err = FollowLinks(tmpDir, []string{"dir/l1", "foo/bar"})
	require.NoError(t, err)

	require.Equal(t, out, []string{filepath.FromSlash("dir/l1"), "foo"})

	out, err = FollowLinks(tmpDir, []string{"dir/l2", "foo", "foo/bar"})
	require.NoError(t, err)

	require.Equal(t, out, []string(nil))
}

func TestFollowLinksWildcard(t *testing.T) {
	tmpDir := t.TempDir()

	absolutePathForFoo := "/foo"
	if runtime.GOOS == "windows" {
		absolutePathForFoo = "C:/foo"
	}

	apply := fstest.Apply(
		fstest.CreateDir("dir", 0700),
		fstest.CreateDir("foo", 0700),
		fstest.Symlink(filepath.Join(absolutePathForFoo, "bar1"), "dir/l1"),
		fstest.Symlink(filepath.Join(absolutePathForFoo, "bar2"), "dir/l2"),
		fstest.Symlink(filepath.Join(absolutePathForFoo, "bar3"), "dir/anotherlink"),
		fstest.Symlink("../baz", "foo/bar2"),
		fstest.CreateFile("foo/bar1", nil, 0600),
		fstest.CreateFile("foo/bar3", nil, 0600),
		fstest.CreateFile("baz", nil, 0600),
	)
	require.NoError(t, apply.Apply(tmpDir))

	out, err := FollowLinks(tmpDir, []string{"dir/l*"})
	require.NoError(t, err)

	require.Equal(t, []string{"baz", filepath.FromSlash("dir/l*"), filepath.FromSlash("foo/bar1"), filepath.FromSlash("foo/bar2")}, out)

	out, err = FollowLinks(tmpDir, []string{"dir"})
	require.NoError(t, err)

	require.Equal(t, out, []string{"dir"})

	out, err = FollowLinks(tmpDir, []string{"dir", "dir/*link"})
	require.NoError(t, err)

	require.Equal(t, out, []string{"dir", filepath.FromSlash("foo/bar3")})
}
