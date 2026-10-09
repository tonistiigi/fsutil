//go:build linux || darwin || freebsd || netbsd || openbsd || dragonfly

package fsutil

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/tonistiigi/fsutil/types"
)

func TestRootDiskWriterSymlinkMetadata(t *testing.T) {
	dest := t.TempDir()
	target := filepath.Join(dest, "target")
	require.NoError(t, os.WriteFile(target, []byte("data"), 0600))

	targetTime := time.Unix(1000, 0)
	linkTime := time.Unix(2000, 0)
	require.NoError(t, os.Chtimes(target, targetTime, targetTime))

	osroot, err := os.OpenRoot(dest)
	require.NoError(t, err)
	destRoot := NewRoot(osroot)
	defer destRoot.Close()

	dw, err := NewRootDiskWriter(context.TODO(), destRoot, DiskWriterOpt{
		SyncDataCb: noOpWriteTo,
	})
	require.NoError(t, err)

	stat := &types.Stat{
		Mode:     uint32(os.ModeSymlink | 0777),
		Linkname: "target",
		Uid:      uint32(os.Getuid()),
		Gid:      uint32(os.Getgid()),
		ModTime:  linkTime.UnixNano(),
	}
	require.NoError(t, dw.HandleChange(ChangeKindAdd, "link", &StatInfo{stat}, nil))
	require.NoError(t, dw.Wait(context.Background()))

	linkFi, err := os.Lstat(filepath.Join(dest, "link"))
	require.NoError(t, err)
	require.Equal(t, linkTime, linkFi.ModTime())

	targetFi, err := os.Stat(target)
	require.NoError(t, err)
	require.Equal(t, targetTime, targetFi.ModTime())
}
