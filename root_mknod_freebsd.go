//go:build freebsd

package fsutil

var _ RootMknod = (*root)(nil)

func (r *root) Mknod(name string, mode uint32, dev int) error {
	entry, err := OpenRootEntry(r, name)
	if err != nil {
		return err
	}
	defer entry.Close()
	return entry.Mknod(mode, dev)
}
