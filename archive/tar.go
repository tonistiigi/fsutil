package archive

import (
	"archive/tar"
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"

	"github.com/containerd/containerd/fs"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

var bufferPool = &sync.Pool{
	New: func() interface{} {
		buffer := make([]byte, 32*1024)
		return &buffer
	},
}

// Apply applies a tar stream of an OCI style diff tar.
// See https://github.com/opencontainers/image-spec/blob/master/layer.md#applying-changesets
func Apply(ctx context.Context, root string, r io.Reader) (int64, error) {
	root = filepath.Clean(root)

	var (
		tr   = tar.NewReader(r)
		size int64
		dirs []*tar.Header

		// Used for handling opaque directory markers which
		// may occur out of order
		unpackedPaths = make(map[string]struct{})
	)

	// Iterate through the files in the archive.
	for {
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}

		hdr, err := tr.Next()
		if err == io.EOF {
			// end of tar archive
			break
		}
		if err != nil {
			return 0, err
		}

		size += hdr.Size

		// Normalize name, for safety and for a simple is-root check
		hdr.Name = filepath.Clean(hdr.Name)

		if skipFile(hdr) {
			logrus.Warnf("file %q ignored: archive may not be supported on system", hdr.Name)
			continue
		}

		// Split name and resolve symlinks for root directory.
		ppath, base := filepath.Split(hdr.Name)
		ppath, err = fs.RootPath(root, ppath)
		if err != nil {
			return 0, errors.Wrap(err, "failed to get root path")
		}

		// Join to root before joining to parent path to ensure relative links are
		// already resolved based on the root before adding to parent.
		path := filepath.Join(ppath, filepath.Join("/", base))
		if path == root {
			logrus.Debugf("file %q ignored: resolved to root", hdr.Name)
			continue
		}

		// If file is not directly under root, ensure parent directory
		// exists or is created.
		if ppath != root {
			parentPath := ppath
			if base == "" {
				parentPath = filepath.Dir(path)
			}
			if _, err := os.Lstat(parentPath); err != nil && os.IsNotExist(err) {
				err = mkdirAll(parentPath, 0700)
				if err != nil {
					return 0, err
				}
			}
		}

		// If path exits we almost always just want to remove and replace it.
		// The only exception is when it is a directory *and* the file from
		// the layer is also a directory. Then we want to merge them (i.e.
		// just apply the metadata from the layer).
		if fi, err := os.Lstat(path); err == nil {
			if !(fi.IsDir() && hdr.Typeflag == tar.TypeDir) {
				if err := os.RemoveAll(path); err != nil {
					return 0, err
				}
			}
		}

		srcData := io.Reader(tr)
		srcHdr := hdr

		if err := createTarFile(ctx, path, root, srcHdr, srcData); err != nil {
			return 0, err
		}

		// Directory mtimes must be handled at the end to avoid further
		// file creation in them to modify the directory mtime
		if hdr.Typeflag == tar.TypeDir {
			dirs = append(dirs, hdr)
		}
		unpackedPaths[path] = struct{}{}
	}

	for _, hdr := range dirs {
		path, err := fs.RootPath(root, hdr.Name)
		if err != nil {
			return 0, err
		}
		if err := chtimes(path, boundTime(latestTime(hdr.AccessTime, hdr.ModTime)), boundTime(hdr.ModTime)); err != nil {
			return 0, err
		}
	}

	return size, nil
}

func createTarFile(ctx context.Context, path, extractDir string, hdr *tar.Header, reader io.Reader) error {
	// hdr.Mode is in linux format, which we can use for syscalls,
	// but for os.Foo() calls we need the mode converted to os.FileMode,
	// so use hdrInfo.Mode() (they differ for e.g. setuid bits)
	hdrInfo := hdr.FileInfo()

	switch hdr.Typeflag {
	case tar.TypeDir:
		// Create directory unless it exists as a directory already.
		// In that case we just want to merge the two
		if fi, err := os.Lstat(path); !(err == nil && fi.IsDir()) {
			if err := mkdir(path, hdrInfo.Mode()); err != nil {
				return err
			}
		}

	case tar.TypeReg, tar.TypeRegA:
		file, err := openFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, hdrInfo.Mode())
		if err != nil {
			return err
		}

		_, err = copyBuffered(ctx, file, reader)
		if err1 := file.Close(); err == nil {
			err = err1
		}
		if err != nil {
			return err
		}

	case tar.TypeBlock, tar.TypeChar:
		// Handle this is an OS-specific way
		if err := handleTarTypeBlockCharFifo(hdr, path); err != nil {
			return err
		}

	case tar.TypeFifo:
		// Handle this is an OS-specific way
		if err := handleTarTypeBlockCharFifo(hdr, path); err != nil {
			return err
		}

	case tar.TypeLink:
		targetPath, err := fs.RootPath(extractDir, hdr.Linkname)
		if err != nil {
			return err
		}
		if err := os.Link(targetPath, path); err != nil {
			return err
		}

	case tar.TypeSymlink:
		if err := os.Symlink(hdr.Linkname, path); err != nil {
			return err
		}

	case tar.TypeXGlobalHeader:
		logrus.Debug("PAX Global Extended Headers found and ignored")
		return nil

	default:
		return errors.Errorf("unhandled tar header type %d\n", hdr.Typeflag)
	}

	// Lchown is not supported on Windows.
	if runtime.GOOS != "windows" {
		if err := os.Lchown(path, hdr.Uid, hdr.Gid); err != nil {
			return err
		}
	}

	for key, value := range hdr.Xattrs {
		if err := setxattr(path, key, value); err != nil {
			if errors.Cause(err) == syscall.ENOTSUP {
				logrus.WithError(err).Warnf("ignored xattr %s in archive", key)
				continue
			}
			return err
		}
	}

	// There is no LChmod, so ignore mode for symlink. Also, this
	// must happen after chown, as that can modify the file mode
	if err := handleLChmod(hdr, path, hdrInfo); err != nil {
		return err
	}

	return chtimes(path, boundTime(latestTime(hdr.AccessTime, hdr.ModTime)), boundTime(hdr.ModTime))
}

func copyBuffered(ctx context.Context, dst io.Writer, src io.Reader) (written int64, err error) {
	buf := bufferPool.Get().(*[]byte)
	defer bufferPool.Put(buf)

	for {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			return
		default:
		}

		nr, er := src.Read(*buf)
		if nr > 0 {
			nw, ew := dst.Write((*buf)[0:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if ew != nil {
				err = ew
				break
			}
			if nr != nw {
				err = io.ErrShortWrite
				break
			}
		}
		if er != nil {
			if er != io.EOF {
				err = er
			}
			break
		}
	}
	return written, err

}
