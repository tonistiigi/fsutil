package fsutil

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tonistiigi/fsutil/types"
)

func TestStat(t *testing.T) {
	requiresRoot(t)

	d, err := tmpDir(changeStream([]string{
		"ADD foo file data1",
		"ADD zzz dir",
		"ADD zzz/aa file data3",
		"ADD zzz/bb dir",
		"ADD zzz/bb/cc dir",
		"ADD zzz/bb/cc/dd symlink ../../",
		"ADD sock socket",
	}))
	assert.NoError(t, err)
	defer os.RemoveAll(d)

	st, err := Stat(filepath.Join(d, "foo"))
	assert.NoError(t, err)
	assert.NotZero(t, st.ModTime)
	st.ModTime = 0
	assert.Equal(t, &types.Stat{Path: "foo", Mode: 0644, Size: 5}, st)

	st, err = Stat(filepath.Join(d, "zzz"))
	assert.NoError(t, err)
	assert.NotZero(t, st.ModTime)
	st.ModTime = 0
	assert.Equal(t, &types.Stat{Path: "zzz", Mode: uint32(os.ModeDir | 0700)}, st)

	st, err = Stat(filepath.Join(d, "zzz/aa"))
	assert.NoError(t, err)
	assert.NotZero(t, st.ModTime)
	st.ModTime = 0
	assert.Equal(t, &types.Stat{Path: "aa", Mode: 0644, Size: 5}, st)

	st, err = Stat(filepath.Join(d, "zzz/bb/cc/dd"))
	assert.NoError(t, err)
	assert.NotZero(t, st.ModTime)
	st.ModTime = 0
	assert.Equal(t, &types.Stat{Path: "dd", Mode: uint32(os.ModeSymlink | 0777), Size: 6, Linkname: "../../"}, st)

	st, err = Stat(filepath.Join(d, "sock"))
	assert.NoError(t, err)
	assert.NotZero(t, st.ModTime)
	st.ModTime = 0
	assert.Equal(t, &types.Stat{Path: "sock", Mode: 0755 /* ModeSocket not set */}, st)
}

func TestStat_SkipAppleXattrs(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("skipping test that requires darwin")
	}

	st, err := Stat("Dockerfile")
	assert.NoError(t, err)

	for key := range st.Xattrs {
		assert.False(t, strings.HasPrefix(key, "com.apple."))
	}
}
