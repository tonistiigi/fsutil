package bench

import (
	"archive/tar"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
)

// copyWithTar benchmarks the standard tar writer with a minimal extractor for the
// generated regular files and directories, not full archive metadata restoration.
func copyWithTar(src, dest string) error {
	pr, pw := io.Pipe()
	var eg errgroup.Group

	eg.Go(func() error {
		tw := tar.NewWriter(pw)
		err := tw.AddFS(os.DirFS(src))
		if err == nil {
			err = tw.Close()
		}
		_ = pw.CloseWithError(err)
		return err
	})

	eg.Go(func() error {
		err := extractTar(dest, pr)
		_ = pr.CloseWithError(err)
		return err
	})

	return eg.Wait()
}

func extractTar(dest string, r io.Reader) error {
	tr := tar.NewReader(r)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}

		target, err := tarTarget(dest, hdr.Name)
		if err != nil {
			return err
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(hdr.Mode)); err != nil {
				return errors.Wrapf(err, "failed to create directory %s", hdr.Name)
			}
			if err := os.Chmod(target, os.FileMode(hdr.Mode)); err != nil {
				return errors.Wrapf(err, "failed to chmod directory %s", hdr.Name)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return errors.Wrapf(err, "failed to create parent directory for %s", hdr.Name)
			}
			if err := writeTarFile(target, hdr, tr); err != nil {
				return errors.Wrapf(err, "failed to write file %s", hdr.Name)
			}
		default:
			return errors.Errorf("unsupported tar entry %s: type %v", hdr.Name, hdr.Typeflag)
		}
	}
}

func tarTarget(dest, name string) (string, error) {
	name = filepath.FromSlash(name)
	if !filepath.IsLocal(name) {
		return "", errors.Errorf("invalid tar path %s", name)
	}
	return filepath.Join(dest, name), nil
}

func writeTarFile(target string, hdr *tar.Header, r io.Reader) error {
	f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(f, r)
	closeErr := f.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Chtimes(target, hdr.ModTime, hdr.ModTime)
}
