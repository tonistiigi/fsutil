package fs

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	_ "crypto/sha256"

	"github.com/containerd/continuity/fs/fstest"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

// TODO: Create copy directory which requires privilege
//  chown
//  mknod
//  setxattr fstest.SetXAttr("/home", "trusted.overlay.opaque", "y"),

func TestCopyDirectory(t *testing.T) {
	apply := fstest.Apply(
		fstest.CreateDir("/etc/", 0755),
		fstest.CreateFile("/etc/hosts", []byte("localhost 127.0.0.1"), 0644),
		fstest.Link("/etc/hosts", "/etc/hosts.allow"),
		fstest.CreateDir("/usr/local/lib", 0755),
		fstest.CreateFile("/usr/local/lib/libnothing.so", []byte{0x00, 0x00}, 0755),
		fstest.Symlink("libnothing.so", "/usr/local/lib/libnothing.so.2"),
		fstest.CreateDir("/home", 0755),
	)

	if err := testCopy(apply); err != nil {
		t.Fatalf("Copy test failed: %+v", err)
	}
}

// This test used to fail because link-no-nothing.txt would be copied first,
// then file operations in dst during the CopyDir would follow the symlink and
// fail.
func TestCopyDirectoryWithLocalSymlink(t *testing.T) {
	apply := fstest.Apply(
		fstest.CreateFile("nothing.txt", []byte{0x00, 0x00}, 0755),
		fstest.Symlink("nothing.txt", "link-no-nothing.txt"),
	)

	if err := testCopy(apply); err != nil {
		t.Fatalf("Copy test failed: %+v", err)
	}
}

func TestCopySingleFile(t *testing.T) {
	t1, err := ioutil.TempDir("", "test")
	require.NoError(t, err)
	defer os.RemoveAll(t1)

	apply := fstest.Apply(
		fstest.CreateFile("foo.txt", []byte("contents"), 0755),
	)

	require.NoError(t, apply.Apply(t1))

	t2, err := ioutil.TempDir("", "test")
	require.NoError(t, err)
	defer os.RemoveAll(t2)

	err = Copy(context.TODO(), filepath.Join(t1, "foo.txt"), t2)
	require.NoError(t, err)

	err = fstest.CheckDirectoryEqual(t1, t2)
	require.NoError(t, err)

	t3, err := ioutil.TempDir("", "test")
	require.NoError(t, err)
	defer os.RemoveAll(t2)

	err = Copy(context.TODO(), filepath.Join(t1, "foo.txt"), filepath.Join(t3, "foo.txt"))
	require.NoError(t, err)

	err = fstest.CheckDirectoryEqual(t1, t2)
	require.NoError(t, err)

	t4, err := ioutil.TempDir("", "test")
	require.NoError(t, err)
	defer os.RemoveAll(t2)

	err = Copy(context.TODO(), filepath.Join(t1, "foo.txt"), filepath.Join(t4, "foo2.txt"))
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(t4, "foo2.txt"))
	require.NoError(t, err)
}

func TestCopyOverrideFile(t *testing.T) {
	t1, err := ioutil.TempDir("", "test")
	require.NoError(t, err)
	defer os.RemoveAll(t1)

	apply := fstest.Apply(
		fstest.CreateFile("foo.txt", []byte("contents"), 0755),
	)

	require.NoError(t, apply.Apply(t1))

	t2, err := ioutil.TempDir("", "test")
	require.NoError(t, err)
	defer os.RemoveAll(t2)

	err = Copy(context.TODO(), filepath.Join(t1, "foo.txt"), filepath.Join(t2, "foo.txt"))
	require.NoError(t, err)

	err = fstest.CheckDirectoryEqual(t1, t2)
	require.NoError(t, err)

	err = Copy(context.TODO(), filepath.Join(t1, "foo.txt"), filepath.Join(t2, "foo.txt"))
	require.NoError(t, err)

	err = fstest.CheckDirectoryEqual(t1, t2)
	require.NoError(t, err)

	err = Copy(context.TODO(), t1+"/.", t2)
	require.NoError(t, err)

	err = fstest.CheckDirectoryEqual(t1, t2)
	require.NoError(t, err)
}

func TestCopyDirectoryBasename(t *testing.T) {
	t1, err := ioutil.TempDir("", "test")
	require.NoError(t, err)
	defer os.RemoveAll(t1)

	apply := fstest.Apply(
		fstest.CreateDir("foo", 0755),
		fstest.CreateDir("foo/bar", 0755),
		fstest.CreateFile("foo/bar/baz.txt", []byte("contents"), 0755),
	)
	require.NoError(t, apply.Apply(t1))

	t2, err := ioutil.TempDir("", "test")
	require.NoError(t, err)
	defer os.RemoveAll(t2)

	err = Copy(context.TODO(), filepath.Join(t1, "foo"), filepath.Join(t2, "foo"))
	require.NoError(t, err)

	err = fstest.CheckDirectoryEqual(t1, t2)
	require.NoError(t, err)

	err = Copy(context.TODO(), filepath.Join(t1, "foo"), filepath.Join(t2, "foo"))
	require.NoError(t, err)

	err = fstest.CheckDirectoryEqual(t1, t2)
	require.NoError(t, err)
}

func TestCopyWildcards(t *testing.T) {
	t1, err := ioutil.TempDir("", "test")
	require.NoError(t, err)
	defer os.RemoveAll(t1)

	apply := fstest.Apply(
		fstest.CreateFile("foo.txt", []byte("foo-contents"), 0755),
		fstest.CreateFile("foo.go", []byte("go-contents"), 0755),
		fstest.CreateFile("bar.txt", []byte("bar-contents"), 0755),
	)

	require.NoError(t, apply.Apply(t1))

	t2, err := ioutil.TempDir("", "test")
	require.NoError(t, err)
	defer os.RemoveAll(t2)

	err = Copy(context.TODO(), filepath.Join(t1, "foo*"), t2)
	require.Error(t, err)

	err = Copy(context.TODO(), filepath.Join(t1, "foo*"), t2, AllowWildcards)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(t2, "foo.txt"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(t2, "foo.go"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(t2, "bar.txt"))
	require.Error(t, err)
	require.True(t, os.IsNotExist(err))

	t2, err = ioutil.TempDir("", "test")
	require.NoError(t, err)
	defer os.RemoveAll(t2)

	err = Copy(context.TODO(), filepath.Join(t1, "bar*"), filepath.Join(t2, "foo.txt"), AllowWildcards)
	require.NoError(t, err)
	dt, err := ioutil.ReadFile(filepath.Join(t2, "foo.txt"))
	require.NoError(t, err)
	require.Equal(t, "bar-contents", string(dt))
}

func testCopy(apply fstest.Applier) error {
	t1, err := ioutil.TempDir("", "test-copy-src-")
	if err != nil {
		return errors.Wrap(err, "failed to create temporary directory")
	}
	defer os.RemoveAll(t1)

	t2, err := ioutil.TempDir("", "test-copy-dst-")
	if err != nil {
		return errors.Wrap(err, "failed to create temporary directory")
	}
	defer os.RemoveAll(t2)

	if err := apply.Apply(t1); err != nil {
		return errors.Wrap(err, "failed to apply changes")
	}

	if err := Copy(context.TODO(), t1+"/.", t2); err != nil {
		return errors.Wrap(err, "failed to copy")
	}

	return fstest.CheckDirectoryEqual(t1, t2)
}
