package fsutil

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func TestRootEntryChmodFallback(t *testing.T) {
	for _, kind := range []string{"file", "directory", "fifo", "symlink"} {
		t.Run(kind, func(t *testing.T) {
			root, err := os.OpenRoot(t.TempDir())
			require.NoError(t, err)
			defer root.Close()
			entry, err := OpenRootEntry(root, "entry")
			require.NoError(t, err)
			defer entry.Close()
			switch kind {
			case "file":
				f, err := entry.OpenFileNoFollow(os.O_CREATE|os.O_WRONLY, 0)
				require.NoError(t, err)
				require.NoError(t, f.Close())
			case "directory":
				require.NoError(t, entry.Mkdir(0))
			case "fifo":
				require.NoError(t, entry.Mknod(unix.S_IFIFO, 0))
			case "symlink":
				require.NoError(t, root.Mkdir("target", 0700))
				require.NoError(t, entry.Symlink("target"))
			}
			parent, err := entry.Parent()
			require.NoError(t, err)
			// Exercise the old-kernel path even on hosts with fchmodat2.
			err = entry.chmodFallback(parent, 0750)
			if kind == "symlink" {
				require.ErrorIs(t, err, unix.EOPNOTSUPP)
				require.ErrorIs(t, entry.ChmodNoFollow(0750), unix.EOPNOTSUPP)
				fi, err := root.Stat("target")
				require.NoError(t, err)
				require.Equal(t, os.FileMode(0700), fi.Mode().Perm())
				return
			}
			require.NoError(t, err)
			fi, err := root.Lstat("entry")
			require.NoError(t, err)
			require.Equal(t, os.FileMode(0750), fi.Mode().Perm())
		})
	}
}

func TestRootEntryMetadataWithoutReadPermission(t *testing.T) {
	dest := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dest, "parent"), 0300))
	t.Cleanup(func() { require.NoError(t, os.Chmod(filepath.Join(dest, "parent"), 0700)) })
	root, err := os.OpenRoot(dest)
	require.NoError(t, err)
	defer root.Close()
	entry, err := OpenRootEntry(root, "parent/file")
	require.NoError(t, err)
	defer entry.Close()
	f, err := entry.OpenFileNoFollow(os.O_CREATE|os.O_WRONLY, 0)
	require.NoError(t, err)
	require.NoError(t, f.Close())
	require.NoError(t, entry.ChmodNoFollow(0200))
	err = entry.LSetxattr("user.fsutil", []byte("value"), 0)
	skipUnsupportedXattr(t, err)
	require.NoError(t, err)
	// Reading user xattrs requires read access; setting them only needs write.
	require.NoError(t, entry.ChmodNoFollow(0600))
	buf := make([]byte, 16)
	n, err := unix.Getxattr(filepath.Join(dest, "parent", "file"), "user.fsutil", buf)
	require.NoError(t, err)
	require.Equal(t, "value", string(buf[:n]))
	atime, mtime := time.Unix(1000, 123), time.Unix(2000, 456)
	require.NoError(t, entry.ChtimesNoFollow(atime, mtime))
	parent, err := entry.Parent()
	require.NoError(t, err)
	var st unix.Stat_t
	require.NoError(t, unix.Fstatat(int(parent.Fd()), entry.Base(), &st, unix.AT_SYMLINK_NOFOLLOW))
	require.Equal(t, atime.UnixNano(), st.Atim.Nano())
	require.Equal(t, mtime.UnixNano(), st.Mtim.Nano())
}

func TestRootEntryXattrDanglingSymlink(t *testing.T) {
	const label = "system_u:object_r:container_file_t:s0"
	probe := filepath.Join(t.TempDir(), "link")
	require.NoError(t, os.Symlink("missing", probe))
	if err := unix.Lsetxattr(probe, "security.selinux", []byte(label), 0); err != nil {
		if errors.Is(err, unix.EACCES) || errors.Is(err, unix.EPERM) || errors.Is(err, unix.EINVAL) {
			t.Skipf("host SELinux policy rejects label: %v", err)
		}
		skipUnsupportedXattr(t, err)
		require.NoError(t, err)
	}
	dest := t.TempDir()
	root, err := os.OpenRoot(dest)
	require.NoError(t, err)
	defer root.Close()
	entry, err := OpenRootEntry(root, "link")
	require.NoError(t, err)
	defer entry.Close()
	require.NoError(t, entry.Symlink("missing"))
	require.NoError(t, entry.LSetxattr("security.selinux", []byte(label), 0))
	buf := make([]byte, 256)
	n, err := unix.Lgetxattr(filepath.Join(dest, "link"), "security.selinux", buf)
	require.NoError(t, err)
	require.Equal(t, label, string(bytes.TrimSuffix(buf[:n], []byte{0})))
}
