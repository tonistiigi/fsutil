package fsutil

import (
	"archive/tar"
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync"

	"github.com/docker/docker/builder"
	"github.com/docker/docker/pkg/symlink"
	iradix "github.com/hashicorp/go-immutable-radix"
	"github.com/pkg/errors"
)

type Tarsum struct {
	mu   sync.Mutex
	root string
	tree *iradix.Tree
	txn  *iradix.Txn
}

func NewTarsum(root string) *Tarsum {
	ts := &Tarsum{
		tree: iradix.New(),
		root: root,
	}
	return ts
}

func (ts *Tarsum) HandleChange(kind ChangeKind, p string, fi os.FileInfo, err error) (retErr error) {
	ts.mu.Lock()
	if ts.txn == nil {
		ts.txn = ts.tree.Txn()
	}
	if kind == ChangeKindDelete {
		ts.txn.Delete([]byte(p))
		ts.mu.Unlock()
		return
	}

	h, ok := fi.(builder.Hashed)
	if !ok {
		ts.mu.Unlock()
		return errors.Errorf("invalid fileinfo: %p", p)
	}

	hfi := &fileInfo{
		FileInfo: fi,
		Hashed:   h,
		path:     p,
	}
	ts.txn.Insert([]byte(p), hfi)
	ts.mu.Unlock()
	return nil
}

func (ts *Tarsum) getRoot() *iradix.Node {
	ts.mu.Lock()
	if ts.txn != nil {
		ts.tree = ts.txn.Commit()
		ts.txn = nil
	}
	t := ts.tree
	ts.mu.Unlock()
	return t.Root()
}

func (ts *Tarsum) Close() error {
	return nil
}

func (ts *Tarsum) normalize(path string) (cleanpath, fullpath string, err error) {
	cleanpath = filepath.Clean(string(os.PathSeparator) + path)[1:]
	fullpath, err = symlink.FollowSymlinkInScope(filepath.Join(ts.root, path), ts.root)
	if err != nil {
		return "", "", fmt.Errorf("Forbidden path outside the context: %s (%s)", path, fullpath)
	}
	_, err = os.Lstat(fullpath)
	if err != nil {
		return "", "", convertPathError(err, path)
	}
	return
}

func (c *Tarsum) Open(path string) (io.ReadCloser, error) {
	cleanpath, fullpath, err := c.normalize(path)
	if err != nil {
		return nil, err
	}
	r, err := os.Open(fullpath)
	if err != nil {
		return nil, convertPathError(err, cleanpath)
	}
	return r, nil
}

func (c *Tarsum) Stat(path string) (string, builder.FileInfo, error) {
	n := c.getRoot()
	v, ok := n.Get([]byte(path))

	if !ok {
		return "", nil, errors.Wrapf(os.ErrNotExist, "failed to stat %s", path)
	}
	hfi := v.(*fileInfo)

	return path, hfi, nil
}

func (c *Tarsum) Walk(root string, walkFn builder.WalkFunc) error {
	n := c.getRoot()
	var walkErr error
	n.WalkPrefix([]byte(root), func(k []byte, v interface{}) bool {
		hfi := v.(*fileInfo)
		if err := walkFn(string(k), hfi, nil); err != nil {
			walkErr = err
			return true
		}
		return false
	})
	return walkErr
}

type tarsumHash struct {
	hash.Hash
	h *tar.Header
}

func NewTarsumHash(fi os.FileInfo) (hash.Hash, error) {
	stat, ok := fi.Sys().(*Stat)
	link := ""
	if ok {
		link = stat.Linkname
	}
	h, err := tar.FileInfoHeader(fi, link)
	if err != nil {
		return nil, err
	}
	if ok {
		h.Uid = int(stat.Uid)
		h.Gid = int(stat.Gid)
		h.Linkname = stat.Linkname
		if stat.Xattrs != nil {
			h.Xattrs = make(map[string]string)
			for k, v := range stat.Xattrs {
				h.Xattrs[k] = string(v)
			}
		}
	}
	tsh := &tarsumHash{h: h, Hash: sha256.New()}
	tsh.Reset()
	return tsh, nil
}

// Reset resets the Hash to its initial state.
func (tsh *tarsumHash) Reset() {
	tsh.Hash.Reset()
	for _, elem := range v1TarHeaderSelect(tsh.h) {
		tsh.Hash.Write([]byte(elem[0] + elem[1]))
	}
}

func v1TarHeaderSelect(h *tar.Header) (orderedHeaders [][2]string) {
	// Get extended attributes.
	xAttrKeys := make([]string, len(h.Xattrs))
	for k := range h.Xattrs {
		xAttrKeys = append(xAttrKeys, k)
	}
	sort.Strings(xAttrKeys)

	headers := [][2]string{
		{"name", h.Name},
		{"mode", strconv.FormatInt(h.Mode, 10)},
		{"uid", strconv.Itoa(h.Uid)},
		{"gid", strconv.Itoa(h.Gid)},
		{"size", strconv.FormatInt(h.Size, 10)},
		// {"mtime", strconv.FormatInt(h.ModTime.UTC().Unix(), 10)},
		{"typeflag", string([]byte{h.Typeflag})},
		{"linkname", h.Linkname},
		{"uname", h.Uname},
		{"gname", h.Gname},
		{"devmajor", strconv.FormatInt(h.Devmajor, 10)},
		{"devminor", strconv.FormatInt(h.Devminor, 10)},
	}

	// Make the slice with enough capacity to hold the 11 basic headers
	// we want from the v0 selector plus however many xattrs we have.
	orderedHeaders = make([][2]string, 0, 11+len(xAttrKeys))

	// Copy all headers from v0 excluding the 'mtime' header (the 5th element).
	orderedHeaders = append(orderedHeaders, headers[:]...)

	// Finally, append the sorted xattrs.
	for _, k := range xAttrKeys {
		orderedHeaders = append(orderedHeaders, [2]string{k, h.Xattrs[k]})
	}

	return
}

type fileInfo struct {
	os.FileInfo
	builder.Hashed
	path string
}

func (fi *fileInfo) Path() string {
	return fi.path
}

func convertPathError(err error, cleanpath string) error {
	if err, ok := err.(*os.PathError); ok {
		err.Path = cleanpath
		return err
	}
	return err
}
