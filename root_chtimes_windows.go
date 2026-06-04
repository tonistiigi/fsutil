//go:build !go1.25 && windows

package fsutil

import (
	"os"
	"time"

	"golang.org/x/sys/windows"
)

func (r *root) Chtimes(name string, atime time.Time, mtime time.Time) error {
	f, err := r.OpenFile(name, os.O_RDONLY, 0)
	if err != nil {
		return err
	}
	defer f.Close()

	a := windows.NsecToFiletime(atime.UnixNano())
	w := windows.NsecToFiletime(mtime.UnixNano())
	return windows.SetFileTime(windows.Handle(f.Fd()), nil, &a, &w)
}
