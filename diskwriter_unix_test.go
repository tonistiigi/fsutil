//go:build !windows
// +build !windows

package fsutil

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func diskWriterTestFactoriesSpecialFiles() []diskWriterTestFactory {
	switch runtime.GOOS {
	case "linux", "freebsd", "netbsd", "openbsd", "dragonfly":
		return diskWriterTestFactories()
	default:
		return diskWriterTestFactories()[:1]
	}
}

func TestWalkerWriterAsync(t *testing.T) {
	for _, factory := range diskWriterTestFactories() {
		t.Run(factory.name, func(t *testing.T) {
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

			dw := factory.new(t, context.TODO(), dest, DiskWriterOpt{
				AsyncDataCb: newWriteToFunc(d, 300*time.Millisecond),
			})

			st := time.Now()

			err = Walk(context.Background(), d, nil, readAsAdd(dw.handleChange))
			assert.NoError(t, err)

			err = dw.wait(context.TODO())
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
		})
	}
}

func TestWalkerWriterDevices(t *testing.T) {
	for _, factory := range diskWriterTestFactoriesSpecialFiles() {
		t.Run(factory.name, func(t *testing.T) {
			requiresRoot(t)

			d, err := tmpDir(changeStream([]string{
				"ADD foo dir",
				"ADD foo/foo1 file data1",
			}))
			require.NoError(t, err)
			defer os.RemoveAll(d)

			err = unix.Mknod(filepath.Join(d, "foo/block"), syscall.S_IFBLK|0600, mkdev(2, 3))
			require.NoError(t, err)

			err = unix.Mknod(filepath.Join(d, "foo/char"), syscall.S_IFCHR|0400, mkdev(1, 9))
			require.NoError(t, err)

			dest := t.TempDir()

			dw := factory.new(t, context.TODO(), dest, DiskWriterOpt{
				SyncDataCb: newWriteToFunc(d, 0),
			})

			err = Walk(context.Background(), d, nil, readAsAdd(dw.handleChange))
			assert.NoError(t, err)

			err = dw.wait(context.TODO())
			assert.NoError(t, err)

			fi, err := os.Lstat(filepath.Join(dest, "foo/char"))
			require.NoError(t, err)

			stat, ok := fi.Sys().(*syscall.Stat_t)
			require.True(t, ok)

			assert.Equal(t, uint64(1), major(uint64(stat.Rdev)))
			assert.Equal(t, uint64(9), minor(uint64(stat.Rdev)))

			fi, err = os.Lstat(filepath.Join(dest, "foo/block"))
			require.NoError(t, err)

			stat, ok = fi.Sys().(*syscall.Stat_t)
			require.True(t, ok)

			assert.Equal(t, uint64(2), major(uint64(stat.Rdev)))
			assert.Equal(t, uint64(3), minor(uint64(stat.Rdev)))
		})
	}
}
