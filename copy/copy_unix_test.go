//go:build !windows
// +build !windows

package fs

import (
	"context"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func TestCopyDevicesAndFifo(t *testing.T) {
	requiresRoot(t)

	t1 := t.TempDir()

	err := mknod(filepath.Join(t1, "char"), unix.S_IFCHR|0444, int(unix.Mkdev(1, 9)))
	require.NoError(t, err)

	err = mknod(filepath.Join(t1, "block"), unix.S_IFBLK|0441, int(unix.Mkdev(3, 2)))
	require.NoError(t, err)

	err = mknod(filepath.Join(t1, "socket"), unix.S_IFSOCK|0555, 0)
	require.NoError(t, err)

	err = unix.Mkfifo(filepath.Join(t1, "fifo"), 0555)
	require.NoError(t, err)

	t2 := t.TempDir()

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

func TestCopySetuid(t *testing.T) {
	requiresRoot(t)

	t1 := t.TempDir()

	err := mknod(filepath.Join(t1, "char"), unix.S_IFCHR|0444, int(unix.Mkdev(1, 9)))
	require.NoError(t, err)

	t2 := t.TempDir()

	err = Copy(context.TODO(), t1, ".", t2, ".")
	require.NoError(t, err)

	fi, err := os.Lstat(filepath.Join(t2, "char"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0444), fi.Mode().Perm())
	assert.Equal(t, os.FileMode(0), fi.Mode()&os.ModeSetuid)
	assert.Equal(t, os.FileMode(0), fi.Mode()&os.ModeSetgid)
	assert.Equal(t, os.FileMode(0), fi.Mode()&os.ModeSticky)

	t3 := t.TempDir()

	p := 0444 | syscall.S_ISUID
	err = Copy(context.TODO(), t1, ".", t3, ".", WithCopyInfo(CopyInfo{
		Mode: &p,
	}))
	require.NoError(t, err)

	fi, err = os.Lstat(filepath.Join(t3, "char"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0444), fi.Mode().Perm())
	assert.Equal(t, os.ModeSetuid, fi.Mode()&os.ModeSetuid)
	assert.Equal(t, os.FileMode(0), fi.Mode()&os.ModeSetgid)
	assert.Equal(t, os.FileMode(0), fi.Mode()&os.ModeSticky)
}

func TestCopyModeTextFormat(t *testing.T) {
	t1 := t.TempDir()

	err := os.WriteFile(filepath.Join(t1, "file"), []byte("hello"), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(t1, "executable_file"), []byte("world"), 0755)
	require.NoError(t, err)

	err = os.Mkdir(filepath.Join(t1, "dir"), 0750)
	require.NoError(t, err)

	err = os.Mkdir(filepath.Join(t1, "restricted_dir"), 0700)
	require.NoError(t, err)

	testCases := []struct {
		name                    string
		modeStr                 string
		expectedFilePerm        os.FileMode
		expectedExecPerm        os.FileMode
		expectedDirPerm         os.FileMode
		expectedRestrictDirPerm os.FileMode
	}{
		{"remove write for others", "go-w", 0644, 0755, 0750, 0700},
		{"add execute for user", "u+x", 0744, 0755, 0750, 0700},
		{"remove all permissions for group", "g-rwx", 0604, 0705, 0700, 0700},
		{"add read for others", "o+r", 0644, 0755, 0754, 0704},
		{"remove execute for all", "a-x", 0644, 0644, 0640, 0600},
		{"remove others and add execute for group", "o-rwx,g+x", 0650, 0750, 0750, 0710},
		{"capital X (apply execute only if directory)", "a+X", 0644, 0755, 0751, 0711},
		{"capital u-go X (apply execute only if directory)", "u=rwX,go=rX", 0644, 0755, 0755, 0755},
		{"capital u-go X (apply execute only if directory)", "u=rX,go=r", 0444, 0544, 0544, 0544},
		{"remove execute and add write for user", "u-x,u+w", 0644, 0655, 0650, 0600},
		{"add execute for user and others", "u+x,o+x", 0745, 0755, 0751, 0701},
		{"add write and read for group and others", "g+rw,o+rw", 0666, 0777, 0776, 0766},
		{"set read-only for all", "a=r", 0444, 0444, 0444, 0444},
		{"set full permissions for user only", "u=rwx,g=,o=", 0700, 0700, 0700, 0700},
		{"remove all permissions for others", "o-rwx", 0640, 0750, 0750, 0700},
		{"remove read for group, add execute for all", "g-r,a+x", 0715, 0715, 0711, 0711},
		{"complex permissions change", "u+rw,g+r,o-x,o+w", 0646, 0756, 0752, 0742},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t2 := t.TempDir()

			err := Copy(context.TODO(), t1, ".", t2, ".", WithCopyInfo(CopyInfo{
				ModeStr: tc.modeStr,
			}))
			require.NoError(t, err)

			fi, err := os.Lstat(filepath.Join(t2, "file"))
			require.NoError(t, err)
			assert.Equal(t, tc.expectedFilePerm, fi.Mode().Perm(), "file %04o, got %04o", tc.expectedFilePerm, fi.Mode().Perm())

			execFileInfo, err := os.Lstat(filepath.Join(t2, "executable_file"))
			require.NoError(t, err)
			assert.Equal(t, tc.expectedExecPerm, execFileInfo.Mode().Perm(), "executable file %04o, got %04o", tc.expectedExecPerm, execFileInfo.Mode().Perm())

			dirInfo, err := os.Lstat(filepath.Join(t2, "dir"))
			require.NoError(t, err)
			assert.Equal(t, tc.expectedDirPerm, dirInfo.Mode().Perm(), "dir %04o, got %04o", tc.expectedDirPerm, dirInfo.Mode().Perm())

			restrictDirInfo, err := os.Lstat(filepath.Join(t2, "restricted_dir"))
			require.NoError(t, err)
			assert.Equal(t, tc.expectedRestrictDirPerm, restrictDirInfo.Mode().Perm(), "restricted dir %04o, got %04o", tc.expectedRestrictDirPerm, restrictDirInfo.Mode().Perm())
		})
	}
}

// requiresRoot skips tests that require root
func requiresRoot(t *testing.T) {
	t.Helper()
	if os.Getuid() != 0 {
		t.Skip("skipping test that requires root")
		return
	}
}

func readUidGid(fi os.FileInfo) (uid, gid int, ok bool) {
	if stat, ok := fi.Sys().(*syscall.Stat_t); ok {
		return int(stat.Uid), int(stat.Gid), true
	}
	return 0, 0, false
}
