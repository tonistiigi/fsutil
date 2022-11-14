//go:build !windows
// +build !windows

package fsutil

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func TestWriterSimple(t *testing.T) {
	requiresRoot(t)

	changes := changeStream([]string{
		"ADD bar dir",
		"ADD bar/foo file",
		"ADD bar/foo2 symlink ../foo",
		"ADD foo file",
		"ADD foo2 file >foo",
	})

	dest := t.TempDir()

	dw, err := NewDiskWriter(context.TODO(), dest, DiskWriterOpt{
		SyncDataCb: noOpWriteTo,
	})
	assert.NoError(t, err)

	for _, c := range changes {
		err := dw.HandleChange(c.kind, c.path, c.fi, nil)
		assert.NoError(t, err)
	}

	b := &bytes.Buffer{}
	err = Walk(context.Background(), dest, nil, bufWalk(b))
	assert.NoError(t, err)

	assert.Equal(t, b.String(), `dir bar
file bar/foo
symlink:../foo bar/foo2
file foo
file foo2 >foo
`)

}

func TestWriterFileToDir(t *testing.T) {
	requiresRoot(t)

	changes := changeStream([]string{
		"ADD foo dir",
		"ADD foo/bar file data2",
	})

	dest, err := tmpDir(changeStream([]string{
		"ADD foo file data1",
	}))
	assert.NoError(t, err)
	defer os.RemoveAll(dest)

	dw, err := NewDiskWriter(context.TODO(), dest, DiskWriterOpt{
		SyncDataCb: noOpWriteTo,
	})
	assert.NoError(t, err)

	for _, c := range changes {
		err := dw.HandleChange(c.kind, c.path, c.fi, nil)
		assert.NoError(t, err)
	}

	b := &bytes.Buffer{}
	err = Walk(context.Background(), dest, nil, bufWalk(b))
	assert.NoError(t, err)

	assert.Equal(t, b.String(), `dir foo
file foo/bar
`)
}

func TestWriterDirToFile(t *testing.T) {
	requiresRoot(t)

	changes := changeStream([]string{
		"ADD foo file data1",
	})

	dest, err := tmpDir(changeStream([]string{
		"ADD foo dir",
		"ADD foo/bar file data2",
	}))
	assert.NoError(t, err)
	defer os.RemoveAll(dest)

	dw, err := NewDiskWriter(context.TODO(), dest, DiskWriterOpt{
		SyncDataCb: noOpWriteTo,
	})
	assert.NoError(t, err)

	for _, c := range changes {
		err := dw.HandleChange(c.kind, c.path, c.fi, nil)
		assert.NoError(t, err)
	}

	b := &bytes.Buffer{}
	err = Walk(context.Background(), dest, nil, bufWalk(b))
	assert.NoError(t, err)

	assert.Equal(t, b.String(), `file foo
`)
}

func TestWalkerWriterSimple(t *testing.T) {
	d, err := tmpDir(changeStream([]string{
		"ADD bar dir",
		"ADD bar/foo file",
		"ADD bar/foo2 symlink ../foo",
		"ADD foo file mydata",
		"ADD foo2 file",
	}))
	assert.NoError(t, err)
	defer os.RemoveAll(d)

	dest := t.TempDir()

	dw, err := NewDiskWriter(context.TODO(), dest, DiskWriterOpt{
		SyncDataCb: newWriteToFunc(d, 0),
	})
	assert.NoError(t, err)

	err = Walk(context.Background(), d, nil, readAsAdd(dw.HandleChange))
	assert.NoError(t, err)

	b := &bytes.Buffer{}
	err = Walk(context.Background(), dest, nil, bufWalk(b))
	assert.NoError(t, err)

	assert.Equal(t, b.String(), `dir bar
file bar/foo
symlink:../foo bar/foo2
file foo
file foo2
`)

	dt, err := os.ReadFile(filepath.Join(dest, "foo"))
	assert.NoError(t, err)
	assert.Equal(t, []byte("mydata"), dt)

}

func TestWalkerWriterAsync(t *testing.T) {
	d, err := tmpDir(changeStream([]string{
		"ADD foo dir",
		"ADD foo/foo1 file data1",
		"ADD foo/foo2 file data2",
		"ADD foo/foo3 file data3",
		"ADD foo/foo4 file >foo/foo3",
		"ADD foo5 file data5",
	}))
	assert.NoError(t, err)
	defer os.RemoveAll(d)

	dest := t.TempDir()

	dw, err := NewDiskWriter(context.TODO(), dest, DiskWriterOpt{
		AsyncDataCb: newWriteToFunc(d, 300*time.Millisecond),
	})
	assert.NoError(t, err)

	st := time.Now()

	err = Walk(context.Background(), d, nil, readAsAdd(dw.HandleChange))
	assert.NoError(t, err)

	err = dw.Wait(context.TODO())
	assert.NoError(t, err)

	dt, err := os.ReadFile(filepath.Join(dest, "foo/foo3"))
	assert.NoError(t, err)
	assert.Equal(t, "data3", string(dt))

	dt, err = os.ReadFile(filepath.Join(dest, "foo/foo4"))
	assert.NoError(t, err)
	assert.Equal(t, "data3", string(dt))

	fi1, err := os.Lstat(filepath.Join(dest, "foo/foo3"))
	assert.NoError(t, err)
	fi2, err := os.Lstat(filepath.Join(dest, "foo/foo4"))
	assert.NoError(t, err)
	stat1, ok1 := fi1.Sys().(*syscall.Stat_t)
	stat2, ok2 := fi2.Sys().(*syscall.Stat_t)
	if ok1 && ok2 {
		assert.Equal(t, stat1.Ino, stat2.Ino)
	}

	dt, err = os.ReadFile(filepath.Join(dest, "foo5"))
	assert.NoError(t, err)
	assert.Equal(t, "data5", string(dt))

	duration := time.Since(st)
	assert.True(t, duration < 500*time.Millisecond)
}

func TestWalkerWriterDevices(t *testing.T) {
	requiresRoot(t)

	d, err := tmpDir(changeStream([]string{
		"ADD foo dir",
		"ADD foo/foo1 file data1",
	}))
	require.NoError(t, err)

	err = unix.Mknod(filepath.Join(d, "foo/block"), syscall.S_IFBLK|0600, mkdev(2, 3))
	require.NoError(t, err)

	err = unix.Mknod(filepath.Join(d, "foo/char"), syscall.S_IFCHR|0400, mkdev(1, 9))
	require.NoError(t, err)

	dest := t.TempDir()

	dw, err := NewDiskWriter(context.TODO(), dest, DiskWriterOpt{
		SyncDataCb: newWriteToFunc(d, 0),
	})
	assert.NoError(t, err)

	err = Walk(context.Background(), d, nil, readAsAdd(dw.HandleChange))
	assert.NoError(t, err)

	err = dw.Wait(context.TODO())
	assert.NoError(t, err)

	fi, err := os.Lstat(filepath.Join(dest, "foo/char"))
	require.NoError(t, err)

	stat, ok := fi.Sys().(*syscall.Stat_t)
	require.True(t, ok)

	assert.Equal(t, uint64(1), stat.Rdev>>8)
	assert.Equal(t, uint64(9), stat.Rdev&0xff)

	fi, err = os.Lstat(filepath.Join(dest, "foo/block"))
	require.NoError(t, err)

	stat, ok = fi.Sys().(*syscall.Stat_t)
	require.True(t, ok)

	assert.Equal(t, uint64(2), stat.Rdev>>8)
	assert.Equal(t, uint64(3), stat.Rdev&0xff)
}

func readAsAdd(f HandleChangeFn) filepath.WalkFunc {
	return func(path string, fi os.FileInfo, err error) error {
		return f(ChangeKindAdd, path, fi, err)
	}
}

func noOpWriteTo(_ context.Context, _ string, _ io.WriteCloser) error {
	return nil
}

func newWriteToFunc(baseDir string, delay time.Duration) WriteToFunc {
	return func(ctx context.Context, path string, wc io.WriteCloser) error {
		if delay > 0 {
			time.Sleep(delay)
		}
		f, err := os.Open(filepath.Join(baseDir, path))
		if err != nil {
			return err
		}
		if _, err := io.Copy(wc, f); err != nil {
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
		return nil
	}
}
