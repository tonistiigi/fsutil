//go:build linux || darwin || freebsd || netbsd

package fsutil

import (
	"errors"
	"testing"

	"golang.org/x/sys/unix"
)

func skipUnsupportedXattr(t *testing.T, err error) {
	t.Helper()

	if err == nil {
		return
	}
	if errors.Is(err, unix.ENOTSUP) || errors.Is(err, unix.EOPNOTSUPP) || errors.Is(err, unix.ENOSYS) {
		t.Skipf("xattrs unsupported: %v", err)
	}
}
