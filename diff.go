package fsutil

import (
	"context"
	"hash"
	gofs "io/fs"
	"os"
	"path/filepath"
	"runtime"

	"github.com/pkg/errors"
	"github.com/tonistiigi/fsutil/types"
)

type walkerFn func(ctx context.Context, pathC chan<- *currentPath) error

func Changes(ctx context.Context, a, b walkerFn, changeFn ChangeFunc) error {
	return nil
}

type HandleChangeFn func(ChangeKind, string, os.FileInfo, error) error

type ContentHasher func(*types.Stat) (hash.Hash, error)

func getWalkerFn(root string) walkerFn {
	return func(ctx context.Context, pathC chan<- *currentPath) error {
		return errors.Wrap(Walk(ctx, root, nil, func(path string, f os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			stat, ok := f.Sys().(*types.Stat)
			if !ok {
				return errors.Errorf("%T invalid file without stat information", f.Sys())
			}

			p := &currentPath{
				path: path,
				stat: stat,
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case pathC <- p:
				return nil
			}
		}), "failed to walk")
	}
}

func getRootWalkerFn(root Root) walkerFn {
	return func(ctx context.Context, pathC chan<- *currentPath) error {
		seenFiles := make(map[uint64]string)
		return errors.Wrap(gofs.WalkDir(root.FS(), ".", func(path string, d gofs.DirEntry, err error) error {
			if err != nil {
				if isNotExist(err) {
					return filepath.SkipDir
				}
				return err
			}
			if path == "." {
				return nil
			}

			rootPath := filepath.FromSlash(path)
			fi, err := root.Lstat(rootPath)
			if err != nil {
				if isNotExist(err) {
					return filepath.SkipDir
				}
				return errors.WithStack(err)
			}
			stat, err := mkrootstat(root, rootPath, fi, seenFiles)
			if err != nil {
				return err
			}

			p := &currentPath{
				path: rootPath,
				stat: stat,
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case pathC <- p:
				return nil
			}
		}), "failed to walk")
	}
}

func mkrootstat(root Root, relpath string, fi os.FileInfo, inodemap map[uint64]string) (*types.Stat, error) {
	stat := &types.Stat{
		Path:    filepath.FromSlash(filepath.ToSlash(relpath)),
		Mode:    uint32(fi.Mode()),
		ModTime: fi.ModTime().UnixNano(),
	}

	setUnixOpt(fi, stat, relpath, inodemap)

	if !fi.IsDir() {
		stat.Size = fi.Size()
		if fi.Mode()&os.ModeSymlink != 0 {
			link, err := root.Readlink(relpath)
			if err != nil {
				return nil, errors.WithStack(err)
			}
			stat.Linkname = link
		}
	}

	if runtime.GOOS == "windows" {
		permPart := stat.Mode & uint32(os.ModePerm)
		noPermPart := stat.Mode &^ uint32(os.ModePerm)
		// Add the x bit: make everything +x from windows
		permPart |= 0111
		permPart &= 0755
		stat.Mode = noPermPart | permPart
	}

	// Clear the socket bit since archive/tar.FileInfoHeader does not handle it
	stat.Mode &^= uint32(os.ModeSocket)

	return stat, nil
}

func emptyWalker(ctx context.Context, pathC chan<- *currentPath) error {
	return nil
}
