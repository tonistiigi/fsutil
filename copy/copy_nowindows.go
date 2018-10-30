// +build !windows

package fs

import (
	"github.com/pkg/errors"

	"github.com/containerd/continuity/sysx"
)

func copyXAttrs(dst, src string) error {
	xattrKeys, err := sysx.LListxattr(src)
	if err != nil {
		return errors.Wrapf(err, "failed to list xattrs on %s", src)
	}
	for _, xattr := range xattrKeys {
		data, err := sysx.LGetxattr(src, xattr)
		if err != nil {
			return errors.Wrapf(err, "failed to get xattr %q on %s", xattr, src)
		}
		if err := sysx.LSetxattr(dst, xattr, data, 0); err != nil {
			return errors.Wrapf(err, "failed to set xattr %q on %s", xattr, dst)
		}
	}

	return nil
}
