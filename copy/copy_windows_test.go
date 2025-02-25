//go:build windows
// +build windows

package fs

import "os"

func readUidGid(fi os.FileInfo) (uid, gid int, ok bool) {
	return 0, 0, false
}
