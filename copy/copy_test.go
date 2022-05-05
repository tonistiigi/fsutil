package fs

import (
	"context"
	_ "crypto/sha256"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/containerd/continuity/fs/fstest"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tonistiigi/fsutil"
	"golang.org/x/sys/unix"
)

// requiresRoot skips tests that require root
func requiresRoot(t *testing.T) {
	t.Helper()
	if os.Getuid() != 0 {
		t.Skip("skipping test that requires root")
		return
	}
}

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

	exp := "add:/etc,add:/etc/hosts,add:/etc/hosts.allow,add:/home,add:/usr,add:/usr/local,add:/usr/local/lib,add:/usr/local/lib/libnothing.so,add:/usr/local/lib/libnothing.so.2"

	if err := testCopy(apply, exp); err != nil {
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

	exp := "add:/link-no-nothing.txt,add:/nothing.txt"

	if err := testCopy(apply, exp); err != nil {
		t.Fatalf("Copy test failed: %+v", err)
	}
}

func TestCopyToWorkDir(t *testing.T) {
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

	err = Copy(context.TODO(), t1, "foo.txt", t2, "foo.txt")
	require.NoError(t, err)

	err = fstest.CheckDirectoryEqual(t1, t2)
	require.NoError(t, err)
}

func TestCopyDevicesAndFifo(t *testing.T) {
	requiresRoot(t)

	t1, err := ioutil.TempDir("", "test")
	require.NoError(t, err)
	defer os.RemoveAll(t1)

	err = mknod(filepath.Join(t1, "char"), unix.S_IFCHR|0444, int(unix.Mkdev(1, 9)))
	require.NoError(t, err)

	err = mknod(filepath.Join(t1, "block"), unix.S_IFBLK|0441, int(unix.Mkdev(3, 2)))
	require.NoError(t, err)

	err = mknod(filepath.Join(t1, "socket"), unix.S_IFSOCK|0555, 0)
	require.NoError(t, err)

	err = unix.Mkfifo(filepath.Join(t1, "fifo"), 0555)
	require.NoError(t, err)

	t2, err := ioutil.TempDir("", "test")
	require.NoError(t, err)
	defer os.RemoveAll(t2)

	err = Copy(context.TODO(), t1, ".", t2, ".")
	require.NoError(t, err)

	fi, err := os.Lstat(filepath.Join(t2, "char"))
	require.NoError(t, err)
	assert.Equal(t, os.ModeCharDevice, fi.Mode()&os.ModeCharDevice)
	assert.Equal(t, os.FileMode(0444), fi.Mode()&0777)

	fi, err = os.Lstat(filepath.Join(t2, "block"))
	require.NoError(t, err)
	assert.Equal(t, os.ModeDevice, fi.Mode()&os.ModeDevice)
	assert.Equal(t, os.FileMode(0441), fi.Mode()&0777)

	fi, err = os.Lstat(filepath.Join(t2, "fifo"))
	require.NoError(t, err)
	assert.Equal(t, os.ModeNamedPipe, fi.Mode()&os.ModeNamedPipe)
	assert.Equal(t, os.FileMode(0555), fi.Mode()&0777)

	fi, err = os.Lstat(filepath.Join(t2, "socket"))
	require.NoError(t, err)
	assert.NotEqual(t, os.ModeSocket, fi.Mode()&os.ModeSocket) // socket copied as stub
	assert.Equal(t, os.FileMode(0555), fi.Mode()&0777)
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

	err = Copy(context.TODO(), t1, "foo.txt", t2, "/")
	require.NoError(t, err)

	err = fstest.CheckDirectoryEqual(t1, t2)
	require.NoError(t, err)

	t3, err := ioutil.TempDir("", "test")
	require.NoError(t, err)
	defer os.RemoveAll(t2)

	err = Copy(context.TODO(), t1, "foo.txt", t3, "foo.txt")
	require.NoError(t, err)

	err = fstest.CheckDirectoryEqual(t1, t2)
	require.NoError(t, err)

	t4, err := ioutil.TempDir("", "test")
	require.NoError(t, err)
	defer os.RemoveAll(t2)

	err = Copy(context.TODO(), t1, "foo.txt", t4, "foo2.txt")
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(t4, "foo2.txt"))
	require.NoError(t, err)

	ch := &changeCollector{}

	err = Copy(context.TODO(), t1, "foo.txt", t4, "a/b/c/foo2.txt", WithChangeNotifier(ch.onChange))
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(t4, "a/b/c/foo2.txt"))
	require.NoError(t, err)

	require.Equal(t, "add:/a/b/c/foo2.txt", ch.String())
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

	err = Copy(context.TODO(), t1, "foo.txt", t2, "foo.txt")
	require.NoError(t, err)

	err = fstest.CheckDirectoryEqual(t1, t2)
	require.NoError(t, err)

	err = Copy(context.TODO(), t1, "foo.txt", t2, "foo.txt")
	require.NoError(t, err)

	err = fstest.CheckDirectoryEqual(t1, t2)
	require.NoError(t, err)

	err = Copy(context.TODO(), t1, "/.", t2, "/")
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

	ch := &changeCollector{}

	err = Copy(context.TODO(), t1, "foo", t2, "foo", WithChangeNotifier(ch.onChange))
	require.NoError(t, err)

	err = fstest.CheckDirectoryEqual(t1, t2)
	require.NoError(t, err)

	require.Equal(t, "add:/foo,add:/foo/bar,add:/foo/bar/baz.txt", ch.String())

	ch = &changeCollector{}
	err = Copy(context.TODO(), t1, "foo", t2, "foo", WithCopyInfo(CopyInfo{
		CopyDirContents: true,
		ChangeFunc:      ch.onChange,
	}))
	require.NoError(t, err)

	require.Equal(t, "add:/foo/bar,add:/foo/bar/baz.txt", ch.String())

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

	err = Copy(context.TODO(), t1, "foo*", t2, "/")
	require.Error(t, err)

	err = Copy(context.TODO(), t1, "foo*", t2, "/", AllowWildcards)
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

	err = Copy(context.TODO(), t1, "bar*", t2, "foo.txt", AllowWildcards)
	require.NoError(t, err)
	dt, err := ioutil.ReadFile(filepath.Join(t2, "foo.txt"))
	require.NoError(t, err)
	require.Equal(t, "bar-contents", string(dt))
}

func TestCopyExistingDirDest(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip()
	}

	t1, err := ioutil.TempDir("", "test")
	require.NoError(t, err)
	defer os.RemoveAll(t1)

	apply := fstest.Apply(
		fstest.CreateDir("dir", 0755),
		fstest.CreateFile("dir/foo.txt", []byte("foo-contents"), 0644),
		fstest.CreateFile("dir/bar.txt", []byte("bar-contents"), 0644),
	)
	require.NoError(t, apply.Apply(t1))

	t2, err := ioutil.TempDir("", "test")
	require.NoError(t, err)
	defer os.RemoveAll(t2)

	apply = fstest.Apply(
		// notice how perms for destination and source are different
		fstest.CreateDir("dir", 0700),
		// dir/foo.txt does not exist, but dir/bar.txt does
		// notice how both perms and contents for destination and source are different
		fstest.CreateFile("dir/bar.txt", []byte("old-bar-contents"), 0600),
	)
	require.NoError(t, apply.Apply(t2))

	for _, x := range []string{"dir", "dir/bar.txt"} {
		err = os.Chown(filepath.Join(t2, x), 1, 1)
		require.NoErrorf(t, err, "x=%s", x)
	}

	err = Copy(context.TODO(), t1, "dir", t2, "dir", WithCopyInfo(CopyInfo{
		CopyDirContents: true,
	}))
	require.NoError(t, err)

	// verify that existing destination dir's metadata was not overwritten
	st, err := os.Lstat(filepath.Join(t2, "dir"))
	require.NoError(t, err)
	require.Equal(t, st.Mode()&os.ModePerm, os.FileMode(0700))
	uid, gid := getUIDGID(st)
	require.Equal(t, 1, uid)
	require.Equal(t, 1, gid)

	// verify that non-existing file was created
	_, err = os.Lstat(filepath.Join(t2, "dir/foo.txt"))
	require.NoError(t, err)

	// verify that existing file's content and metadata was overwritten
	st, err = os.Lstat(filepath.Join(t2, "dir/bar.txt"))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0644), st.Mode()&os.ModePerm)
	uid, gid = getUIDGID(st)
	require.Equal(t, 0, uid)
	require.Equal(t, 0, gid)
	dt, err := ioutil.ReadFile(filepath.Join(t2, "dir/bar.txt"))
	require.NoError(t, err)
	require.Equal(t, "bar-contents", string(dt))
}

func TestCopySymlinks(t *testing.T) {
	t1, err := ioutil.TempDir("", "test")
	require.NoError(t, err)
	defer os.RemoveAll(t1)

	apply := fstest.Apply(
		fstest.CreateDir("testdir", 0755),
		fstest.CreateFile("testdir/foo.txt", []byte("foo-contents"), 0644),
		fstest.Symlink("foo.txt", "testdir/link2"),
		fstest.Symlink("/testdir", "link"),
	)
	require.NoError(t, apply.Apply(t1))

	t2, err := ioutil.TempDir("", "test")
	require.NoError(t, err)
	defer os.RemoveAll(t2)

	err = Copy(context.TODO(), t1, "link/link2", t2, "foo", WithCopyInfo(CopyInfo{
		FollowLinks: true,
	}))
	require.NoError(t, err)

	// verify that existing destination dir's metadata was not overwritten
	st, err := os.Lstat(filepath.Join(t2, "foo"))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0644), st.Mode()&os.ModePerm)
	require.Equal(t, 0, int(st.Mode()&os.ModeSymlink))
	dt, err := ioutil.ReadFile(filepath.Join(t2, "foo"))
	require.NoError(t, err)
	require.Equal(t, "foo-contents", string(dt))

	t3, err := ioutil.TempDir("", "test")
	require.NoError(t, err)
	defer os.RemoveAll(t2)

	err = Copy(context.TODO(), t1, "link/link2", t3, "foo", WithCopyInfo(CopyInfo{}))
	require.NoError(t, err)

	// verify that existing destination dir's metadata was not overwritten
	st, err = os.Lstat(filepath.Join(t3, "foo"))
	require.NoError(t, err)
	require.Equal(t, os.ModeSymlink, st.Mode()&os.ModeSymlink)
	link, err := os.Readlink(filepath.Join(t3, "foo"))
	require.NoError(t, err)
	require.Equal(t, "foo.txt", link)
}

func testCopy(apply fstest.Applier, exp string) error {
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

	ch := &changeCollector{}
	if err := Copy(context.TODO(), t1, "/.", t2, "/", WithChangeNotifier(ch.onChange)); err != nil {
		return errors.Wrap(err, "failed to copy")
	}

	if exp != ch.String() {
		return errors.Errorf("unexpected changes: %s", ch)
	}

	return fstest.CheckDirectoryEqual(t1, t2)
}

func TestCopyIncludeExclude(t *testing.T) {
	t1, err := ioutil.TempDir("", "test")
	require.NoError(t, err)
	defer os.RemoveAll(t1)

	apply := fstest.Apply(
		fstest.CreateDir("bar", 0755),
		fstest.CreateFile("bar/foo", []byte("foo-contents"), 0755),
		fstest.CreateDir("bar/baz", 0755),
		fstest.CreateFile("bar/baz/foo3", []byte("foo3-contents"), 0755),
		fstest.CreateFile("foo2", []byte("foo2-contents"), 0755),
	)

	require.NoError(t, apply.Apply(t1))

	testCases := []struct {
		name            string
		opts            []Opt
		expectedResults []string
		expectedChanges string
	}{
		{
			name:            "include bar",
			opts:            []Opt{WithIncludePattern("bar")},
			expectedResults: []string{"bar", "bar/foo", "bar/baz", "bar/baz/foo3"},
			expectedChanges: "add:/bar,add:/bar/baz,add:/bar/baz/foo3,add:/bar/foo",
		},
		{
			name:            "include *",
			opts:            []Opt{WithIncludePattern("*")},
			expectedResults: []string{"bar", "bar/foo", "bar/baz", "bar/baz/foo3", "foo2"},
			expectedChanges: "add:/bar,add:/bar/baz,add:/bar/baz/foo3,add:/bar/foo,add:/foo2",
		},
		{
			name:            "include bar/foo",
			opts:            []Opt{WithIncludePattern("bar/foo")},
			expectedResults: []string{"bar", "bar/foo"},
			expectedChanges: "add:/bar/foo",
		},
		{
			name:            "include bar except bar/foo",
			opts:            []Opt{WithIncludePattern("bar"), WithIncludePattern("!bar/foo")},
			expectedResults: []string{"bar", "bar/baz", "bar/baz/foo3"},
			expectedChanges: "add:/bar,add:/bar/baz,add:/bar/baz/foo3",
		},
		{
			name:            "include bar/foo and foo*",
			opts:            []Opt{WithIncludePattern("bar/foo"), WithIncludePattern("foo*")},
			expectedResults: []string{"bar", "bar/foo", "foo2"},
			expectedChanges: "add:/bar/foo,add:/foo2",
		},
		{
			name:            "include b*",
			opts:            []Opt{WithIncludePattern("b*")},
			expectedResults: []string{"bar", "bar/foo", "bar/baz", "bar/baz/foo3"},
			expectedChanges: "add:/bar,add:/bar/baz,add:/bar/baz/foo3,add:/bar/foo",
		},
		{
			name:            "include bar/f*",
			opts:            []Opt{WithIncludePattern("bar/f*")},
			expectedResults: []string{"bar", "bar/foo"},
		},
		{
			name:            "include bar/g*",
			opts:            []Opt{WithIncludePattern("bar/g*")},
			expectedResults: nil,
		},
		{
			name:            "include b*/f*",
			opts:            []Opt{WithIncludePattern("b*/f*")},
			expectedResults: []string{"bar", "bar/foo"},
		},
		{
			name:            "include b*/foo",
			opts:            []Opt{WithIncludePattern("b*/foo")},
			expectedResults: []string{"bar", "bar/foo"},
		},
		{
			name:            "include b*/",
			opts:            []Opt{WithIncludePattern("b*/")},
			expectedResults: []string{"bar", "bar/foo", "bar/baz", "bar/baz/foo3"},
		},
		{
			name:            "include bar/*/foo3",
			opts:            []Opt{WithIncludePattern("bar/*/foo3")},
			expectedResults: []string{"bar", "bar/baz", "bar/baz/foo3"},
		},
		{
			name:            "exclude bar*, !bar/baz",
			opts:            []Opt{WithExcludePattern("bar*"), WithExcludePattern("!bar/baz")},
			expectedResults: []string{"bar", "bar/baz", "bar/baz/foo3", "foo2"},
		},
		{
			name:            "exclude **, !bar/baz",
			opts:            []Opt{WithExcludePattern("**"), WithExcludePattern("!bar/baz")},
			expectedResults: []string{"bar", "bar/baz", "bar/baz/foo3"},
		},
		{
			name:            "exclude **, !bar/baz, bar/baz/foo3",
			opts:            []Opt{WithExcludePattern("**"), WithExcludePattern("!bar/baz"), WithExcludePattern("bar/baz/foo3")},
			expectedResults: []string{"bar", "bar/baz"},
		},
		{
			name:            "include bar, exclude bar/baz",
			opts:            []Opt{WithIncludePattern("bar"), WithExcludePattern("bar/baz")},
			expectedResults: []string{"bar", "bar/foo"},
		},
		{
			name:            "doublestar include",
			opts:            []Opt{WithIncludePattern("**/foo3")},
			expectedResults: []string{"bar", "bar/baz", "bar/baz/foo3"},
		},
		{
			name:            "doublestar matching second item in path",
			opts:            []Opt{WithIncludePattern("**/baz")},
			expectedResults: []string{"bar", "bar/baz", "bar/baz/foo3"},
			expectedChanges: "add:/bar/baz,add:/bar/baz/foo3",
		},
		{
			name:            "doublestar matching first item in path",
			opts:            []Opt{WithIncludePattern("**/bar")},
			expectedResults: []string{"bar", "bar/foo", "bar/baz", "bar/baz/foo3"},
			expectedChanges: "add:/bar,add:/bar/baz,add:/bar/baz/foo3,add:/bar/foo",
		},
		{
			name:            "doublestar exclude",
			opts:            []Opt{WithIncludePattern("bar"), WithExcludePattern("**/foo3")},
			expectedResults: []string{"bar", "bar/foo", "bar/baz"},
			expectedChanges: "add:/bar,add:/bar/baz,add:/bar/foo",
		},
		{
			name:            "exclude bar/baz",
			opts:            []Opt{WithExcludePattern("bar/baz")},
			expectedResults: []string{"bar", "bar/foo", "foo2"},
			expectedChanges: "add:/bar,add:/bar/foo,add:/foo2",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t2, err := ioutil.TempDir("", "test")
			require.NoError(t, err)
			defer os.RemoveAll(t2)

			ch := &changeCollector{}
			tc.opts = append(tc.opts, WithChangeNotifier(ch.onChange))

			err = Copy(context.Background(), t1, "/", t2, "/", tc.opts...)
			require.NoError(t, err, tc.name)

			var results []string
			for _, path := range []string{"bar", "bar/foo", "bar/baz", "bar/baz/asdf", "bar/baz/asdf/x", "bar/baz/foo3", "foo2"} {
				_, err := os.Stat(filepath.Join(t2, path))
				if err == nil {
					results = append(results, path)
				}
			}

			require.Equal(t, tc.expectedResults, results, tc.name)

			if tc.expectedChanges != "" {
				require.Equal(t, tc.expectedChanges, ch.String())
			}
		})
	}
}

type changeCollector struct {
	changes []string
}

func (c *changeCollector) onChange(kind fsutil.ChangeKind, path string, _ os.FileInfo, _ error) error {
	c.changes = append(c.changes, fmt.Sprintf("%s:%s", kind, path))
	return nil
}

func (c *changeCollector) String() string {
	sort.Strings(c.changes)
	return strings.Join(c.changes, ",")
}
