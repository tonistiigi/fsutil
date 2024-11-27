//go:build !unix
// +build !unix

package fsutil

import "syscall"

const (
	EBADMSG = syscall.EBADMSG
	EEXIST  = syscall.EEXIST
	EINVAL  = syscall.EINVAL
	EISDIR  = syscall.EISDIR
	ENOENT  = syscall.ENOENT
	ENOTDIR = syscall.ENOTDIR
)
