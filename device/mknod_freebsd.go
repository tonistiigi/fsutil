package device

import (
	"syscall"
)

func Mknod(path string, mode uint32, dev int) error {
	return syscall.Mknod(path, mode, uint64(dev))
}
