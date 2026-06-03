//go:build !windows

package fs

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMkdirUmaskFix(t *testing.T) {
	modes := []struct {
		name  string
		setup func(t *testing.T, umask os.FileMode)
	}{
		{
			name: "umask",
			setup: func(t *testing.T, umask os.FileMode) {
				oldUmask := syscall.Umask(int(umask))
				t.Cleanup(func() { syscall.Umask(oldUmask) })
			},
		},
		{
			name: "zero umask",
			setup: func(t *testing.T, _ os.FileMode) {
				umask := syscall.Umask(0)
				t.Cleanup(func() { syscall.Umask(umask) })
				UmaskIsZero = true
				t.Cleanup(func() { UmaskIsZero = false })
			},
		},
	}
	tests := []struct {
		name  string
		umask os.FileMode
		perm  os.FileMode
	}{
		{
			name:  "default - world",
			umask: 0022,
			perm:  0777,
		},
		{
			name:  "none - world",
			umask: 0,
			perm:  0777,
		},
		{
			name:  "default - normal",
			umask: 0022,
			perm:  0755,
		},
		{
			name:  "none - normal",
			umask: 0,
			perm:  0755,
		},
	}

	for _, mode := range modes {
		for _, tt := range tests {
			t.Run(mode.name+"/"+tt.name, func(t *testing.T) {
				dir := t.TempDir()
				mode.setup(t, tt.umask)

				path := filepath.Join(dir, "a/b/c")

				createdDirs, err := MkdirAll(path, tt.perm, nil, nil)
				require.NoError(t, err)
				require.Len(t, createdDirs, 3)

				for _, p := range createdDirs {
					st, err := os.Stat(p)
					require.NoError(t, err)
					require.Equal(t, tt.perm, st.Mode()&os.ModePerm)
				}
			})
		}
	}
}

func TestMkdirUmaskFixExistingPath(t *testing.T) {
	modes := []struct {
		name  string
		setup func(t *testing.T)
	}{
		{
			name: "umask",
			setup: func(t *testing.T) {
				umask := syscall.Umask(0022)
				t.Cleanup(func() { syscall.Umask(umask) })
			},
		},
		{
			name: "zero umask",
			setup: func(t *testing.T) {
				umask := syscall.Umask(0)
				t.Cleanup(func() { syscall.Umask(umask) })
				UmaskIsZero = true
				t.Cleanup(func() { UmaskIsZero = false })
			},
		},
	}

	for _, mode := range modes {
		t.Run(mode.name, func(t *testing.T) {
			dir := t.TempDir()
			mode.setup(t)

			existing := filepath.Join(dir, "existing")
			require.NoError(t, os.Mkdir(existing, 0700))

			newPath := filepath.Join(existing, "new")
			createdDirs, err := MkdirAll(filepath.Join(newPath, "."), 0770, nil, nil)
			require.NoError(t, err)
			require.Equal(t, []string{newPath}, createdDirs)

			st, err := os.Stat(existing)
			require.NoError(t, err)
			require.Equal(t, os.FileMode(0700), st.Mode()&os.ModePerm)

			st, err = os.Stat(newPath)
			require.NoError(t, err)
			require.Equal(t, os.FileMode(0770), st.Mode()&os.ModePerm)
		})
	}
}
