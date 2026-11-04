//go:build linux || darwin || freebsd || netbsd

package fsutil

var _ RootXattr = (*root)(nil)

func (r *root) LSetxattr(name, key string, value []byte, flags int) error {
	entry, err := OpenRootEntry(r, name)
	if err != nil {
		return err
	}
	defer entry.Close()
	return entry.LSetxattr(key, value, flags)
}
