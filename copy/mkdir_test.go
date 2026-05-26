//go:build !windows

package fs

import (
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMkdirUmaskFix(t *testing.T) {
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

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()

			// Invoked both on start and cleanup just out of paranoia.
			resetSystemUmaskCheck()
			t.Cleanup(resetSystemUmaskCheck)

			// Set the umask to our tested value and then reset it back.
			umask := syscall.Umask(int(tt.umask))
			t.Cleanup(func() { syscall.Umask(umask) })

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

// resetSystemUmaskCheck resets the cached system umask so that the next call to
// needsUmaskFix re-probes the environment.
func resetSystemUmaskCheck() {
	systemUmask = 0
	systemUmaskOnce = sync.Once{}
}
