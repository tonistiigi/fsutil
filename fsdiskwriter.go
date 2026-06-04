package fsutil

import (
	"context"
	"io"
	gofs "io/fs"
	"os"
	"path/filepath"
	"syscall"

	"github.com/pkg/errors"
	"github.com/tonistiigi/fsutil/types"
	"golang.org/x/sync/errgroup"
)

type FSDiskWriterOpt = DiskWriterOpt

type FSDiskWriter struct {
	opt  FSDiskWriterOpt
	dest string

	ctx         context.Context
	cancel      func()
	eg          *errgroup.Group
	egCtx       context.Context
	filter      FilterFunc
	dirModTimes map[string]int64
}

func NewFSDiskWriter(ctx context.Context, dest string, opt FSDiskWriterOpt) (*FSDiskWriter, error) {
	if opt.SyncDataCb == nil && opt.AsyncDataCb == nil {
		return nil, errors.New("no data callback specified")
	}
	if opt.SyncDataCb != nil && opt.AsyncDataCb != nil {
		return nil, errors.New("can't specify both sync and async data callbacks")
	}

	ctx, cancel := context.WithCancel(ctx)
	eg, egCtx := errgroup.WithContext(ctx)

	return &FSDiskWriter{
		opt:         opt,
		dest:        dest,
		eg:          eg,
		ctx:         ctx,
		egCtx:       egCtx,
		cancel:      cancel,
		filter:      opt.Filter,
		dirModTimes: map[string]int64{},
	}, nil
}

func (dw *FSDiskWriter) Wait(ctx context.Context) error {
	if err := dw.eg.Wait(); err != nil {
		return err
	}
	return filepath.WalkDir(dw.dest, func(path string, d gofs.DirEntry, prevErr error) error {
		if prevErr != nil {
			return prevErr
		}
		if !d.IsDir() {
			return nil
		}
		if mtime, ok := dw.dirModTimes[path]; ok {
			return chtimes(path, mtime)
		}
		return nil
	})
}

func (dw *FSDiskWriter) HandleChange(kind ChangeKind, p string, fi os.FileInfo, err error) (retErr error) {
	if err != nil {
		return err
	}

	select {
	case <-dw.ctx.Done():
		return dw.ctx.Err()
	default:
	}

	defer func() {
		if retErr != nil {
			dw.cancel()
		}
	}()

	destPath := filepath.Join(dw.dest, p)

	if kind == ChangeKindDelete {
		if dw.filter != nil {
			var empty types.Stat
			if ok := dw.filter(p, &empty); !ok {
				return nil
			}
		}
		// todo: no need to validate if diff is trusted but is it always?
		if err := os.RemoveAll(destPath); err != nil {
			return errors.Wrapf(err, "failed to remove: %s", destPath)
		}
		if dw.opt.NotifyCb != nil {
			if err := dw.opt.NotifyCb(kind, p, nil, nil); err != nil {
				return err
			}
		}
		return nil
	}

	stat, ok := fi.Sys().(*types.Stat)
	if !ok {
		return errors.WithStack(&os.PathError{Path: p, Err: syscall.EBADMSG, Op: "change without stat info"})
	}

	statCopy := stat.Clone()

	if dw.filter != nil {
		if ok := dw.filter(p, statCopy); !ok {
			return nil
		}
	}

	rename := true
	oldFi, err := os.Lstat(destPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if kind != ChangeKindAdd {
				return errors.Wrap(err, "modify/rm")
			}
			rename = false
		} else {
			return errors.WithStack(err)
		}
	}

	if oldFi != nil && fi.IsDir() && oldFi.IsDir() {
		if err := rewriteMetadata(destPath, statCopy); err != nil {
			return errors.Wrapf(err, "error setting dir metadata for %s", destPath)
		}
		return nil
	}

	newPath := destPath
	if rename {
		newPath = filepath.Join(filepath.Dir(destPath), ".tmp."+nextSuffix())
	}

	isRegularFile := false

	switch {
	case fi.IsDir():
		if err := os.Mkdir(newPath, fi.Mode()); err != nil {
			if errors.Is(err, syscall.EEXIST) {
				// we saw a race to create this directory, so try again
				return dw.HandleChange(kind, p, fi, nil)
			}
			return errors.Wrapf(err, "failed to create dir %s", newPath)
		}
		dw.dirModTimes[destPath] = statCopy.ModTime
	case fi.Mode()&os.ModeDevice != 0 || fi.Mode()&os.ModeNamedPipe != 0:
		if err := handleTarTypeBlockCharFifo(newPath, statCopy); err != nil {
			return errors.Wrapf(err, "failed to create device %s", newPath)
		}
	case fi.Mode()&os.ModeSymlink != 0:
		if err := os.Symlink(statCopy.Linkname, newPath); err != nil {
			return errors.Wrapf(err, "failed to symlink %s", newPath)
		}
	case statCopy.Linkname != "":
		if err := os.Link(filepath.Join(dw.dest, statCopy.Linkname), newPath); err != nil {
			return errors.Wrapf(err, "failed to link %s to %s", newPath, statCopy.Linkname)
		}
	default:
		isRegularFile = true
		file, err := os.OpenFile(newPath, os.O_CREATE|os.O_WRONLY, fi.Mode())
		if err != nil {
			return errors.Wrapf(err, "failed to create %s", newPath)
		}
		if dw.opt.SyncDataCb != nil {
			if err := dw.processChange(dw.ctx, ChangeKindAdd, p, fi, file); err != nil {
				file.Close()
				return err
			}
		}
		if err := file.Close(); err != nil {
			return errors.Wrapf(err, "failed to close %s", newPath)
		}
	}

	if err := rewriteMetadata(newPath, statCopy); err != nil {
		return errors.Wrapf(err, "error setting metadata for %s", newPath)
	}

	if rename {
		if oldFi.IsDir() != fi.IsDir() {
			if err := os.RemoveAll(destPath); err != nil {
				return errors.Wrapf(err, "failed to remove %s", destPath)
			}
		}

		if err := renameFile(newPath, destPath); err != nil {
			return errors.Wrapf(err, "failed to rename %s to %s", newPath, destPath)
		}
	}

	if isRegularFile {
		if dw.opt.AsyncDataCb != nil {
			dw.requestAsyncFileData(p, destPath, fi, statCopy)
		}
	} else {
		return dw.processChange(dw.ctx, kind, p, fi, nil)
	}

	return nil
}

func (dw *FSDiskWriter) requestAsyncFileData(p, dest string, fi os.FileInfo, st *types.Stat) {
	// todo: limit worker threads
	dw.eg.Go(func() error {
		if err := dw.processChange(dw.egCtx, ChangeKindAdd, p, fi, &lazyFileWriter{
			dest: dest,
		}); err != nil {
			return err
		}
		return chtimes(dest, st.ModTime) // TODO: parent dirs
	})
}

func (dw *FSDiskWriter) processChange(ctx context.Context, kind ChangeKind, p string, fi os.FileInfo, w io.WriteCloser) error {
	origw := w
	var hw *hashedWriter
	if dw.opt.NotifyCb != nil {
		var err error
		if hw, err = newHashWriter(dw.opt.ContentHasher, fi, w); err != nil {
			return err
		}
		w = hw
	}
	if origw != nil {
		fn := dw.opt.SyncDataCb
		if fn == nil && dw.opt.AsyncDataCb != nil {
			fn = dw.opt.AsyncDataCb
		}
		if err := fn(ctx, p, w); err != nil {
			return err
		}
	} else {
		if hw != nil {
			hw.Close()
		}
	}
	if hw != nil {
		return dw.opt.NotifyCb(kind, p, hw, nil)
	}
	return nil
}
