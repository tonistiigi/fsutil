//go:build windows
// +build windows

package fsutil

import (
	"os"

	"github.com/tonistiigi/fsutil/types"
)

func loadXattr(_ string, _ *types.Stat) error {
	return nil
}

func setUnixOpt(_ os.FileInfo, _ *types.Stat, _ string, _ map[uint64]string) {
}

func modeBits(fi os.FileInfo) uint32 {
	mode := fi.Mode()
	// Go 1.23 an above will set ModeIrregular for directories with reparse tags
	// that it doesn't understand. Clear it to make the mode consistent with
	// the expectations for ModeIrregular on Unix.
	// See https://github.com/golang/go/blob/bd80d8956f3062d2b2bff2d7da6b879dfa909f12/src/os/types_windows.go#L227
	// and https://learn.microsoft.com/en-us/openspecs/windows_protocols/ms-fscc/c8e77b37-3909-4fe6-a4ea-2b9d423b1ee4
	if mode&os.ModeDir != 0 {
		mode &^= os.ModeIrregular
	}
	return uint32(mode)
}
