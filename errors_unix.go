//go:build unix
// +build unix

package fsutil

import "golang.org/x/sys/unix"

const (
	EBADMSG    = unix.EBADMSG
	EEXIST     = unix.EEXIST
	EINVAL     = unix.EINVAL
	EIO        = unix.EIO
	EISDIR     = unix.EISDIR
	ENOENT     = unix.ENOENT
	ENOSYS     = unix.ENOSYS
	ENOTDIR    = unix.ENOTDIR
	ENOTSUP    = unix.ENOTSUP
	EOPNOTSUPP = unix.EOPNOTSUPP
	EPERM      = unix.EPERM
	EXDEV      = unix.EXDEV
)
