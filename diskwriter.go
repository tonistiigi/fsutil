package fsutil

import (
	"context"
	"hash"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/opencontainers/go-digest"
	"github.com/pkg/errors"
	"github.com/tonistiigi/fsutil/types"
	"golang.org/x/sync/errgroup"
)

// WriteToFunc arranges to save data for the file at path
// to wc
type WriteToFunc func(ctx context.Context, path string, wc io.WriteCloser) error

type DiskWriterOpt struct {
	AsyncDataCb   WriteToFunc
	SyncDataCb    WriteToFunc
	NotifyCb      func(ChangeKind, string, os.FileInfo, error) error
	ContentHasher ContentHasher
	Filter        FilterFunc

	rewriteMetadata func(path string, st *types.Stat) error
}

type FilterFunc func(string, *types.Stat) bool

type DiskWriter struct {
	opt   DiskWriterOpt
	dests []string

	ctx    context.Context
	cancel func()
	eg     *errgroup.Group
	filter FilterFunc
}

func NewDiskWriter(ctx context.Context, dest string, opt DiskWriterOpt) (*DiskWriter, error) {
	return NewDiskWriterMultiple(ctx, []string{dest}, opt)
}

func NewDiskWriterMultiple(ctx context.Context, dests []string, opt DiskWriterOpt) (*DiskWriter, error) {
	if opt.SyncDataCb == nil && opt.AsyncDataCb == nil {
		return nil, errors.New("no data callback specified")
	}
	if opt.SyncDataCb != nil && opt.AsyncDataCb != nil {
		return nil, errors.New("can't specify both sync and async data callbacks")
	}
	if opt.rewriteMetadata == nil {
		opt.rewriteMetadata = rewriteMetadata
	}

	ctx, cancel := context.WithCancel(ctx)
	eg, ctx := errgroup.WithContext(ctx)

	return &DiskWriter{
		opt:    opt,
		dests:  dests,
		eg:     eg,
		ctx:    ctx,
		cancel: cancel,
		filter: opt.Filter,
	}, nil
}

func (dw *DiskWriter) Wait(ctx context.Context) error {
	return dw.eg.Wait()
}

func (dw *DiskWriter) HandleChange(kind ChangeKind, path string, fi os.FileInfo, err error) (retErr error) {
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

	if kind == ChangeKindDelete {
		return dw.handleDelete(path)
	}

	if err := dw.handleChange(kind, path, fi); err != nil {
		return errors.WithStack(err)
	}

	return dw.processChange(kind, path, fi, nil)
}

func (dw *DiskWriter) handleDelete(path string) error {
	for _, dest := range dw.dests {
		destPath := filepath.Join(dest, filepath.FromSlash(path))
		if dw.filter != nil {
			var empty types.Stat
			if ok := dw.filter(path, &empty); !ok {
				return nil
			}
		}
		// todo: no need to validate if diff is trusted but is it always?
		if err := os.RemoveAll(destPath); err != nil {
			return errors.Wrapf(err, "failed to remove: %s", destPath)
		}
		if dw.opt.NotifyCb != nil {
			if err := dw.opt.NotifyCb(ChangeKindDelete, path, nil, nil); err != nil {
				return err
			}
		}
	}
	return nil
}

func (dw *DiskWriter) handleChange(kind ChangeKind, path string, fi os.FileInfo) (err error) {
	stat, ok := fi.Sys().(*types.Stat)
	if !ok {
		return errors.WithStack(&os.PathError{Path: path, Err: syscall.EBADMSG, Op: "change without stat info"})
	}

	var changes fileChanges
	for _, dest := range dw.dests {
		destPath := filepath.Join(dest, filepath.FromSlash(path))
		statCopy := *stat
		if dw.filter != nil {
			if ok := dw.filter(path, &statCopy); !ok {
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
			if err := dw.opt.rewriteMetadata(destPath, &statCopy); err != nil {
				return errors.Wrapf(err, "error setting dir metadata for %s", destPath)
			}
			return nil
		}

		newPath := destPath
		if rename {
			newPath = filepath.Join(filepath.Dir(destPath), ".tmp."+nextSuffix())
		}

		var file *os.File
		switch {
		case fi.IsDir():
			if err := os.Mkdir(newPath, fi.Mode()); err != nil {
				return errors.Wrapf(err, "failed to create dir %s", newPath)
			}
		case fi.Mode()&os.ModeDevice != 0 || fi.Mode()&os.ModeNamedPipe != 0:
			if err := handleTarTypeBlockCharFifo(newPath, &statCopy); err != nil {
				return errors.Wrapf(err, "failed to create device %s", newPath)
			}
		case fi.Mode()&os.ModeSymlink != 0:
			if err := os.Symlink(stat.Linkname, newPath); err != nil {
				return errors.Wrapf(err, "failed to symlink %s", newPath)
			}
		case statCopy.Linkname != "":
			if err := os.Link(filepath.Join(dest, statCopy.Linkname), newPath); err != nil {
				return errors.Wrapf(err, "failed to link %s to %s", newPath, statCopy.Linkname)
			}
		default:
			file, err = os.OpenFile(newPath, os.O_CREATE|os.O_WRONLY, fi.Mode()) // todo: windows
			if err != nil {
				return errors.Wrapf(err, "failed to create %s", newPath)
			}
			if dw.opt.SyncDataCb != nil {
				break
			}
			// Close the file for the side-effect of it being created.
			if err := file.Close(); err != nil {
				return errors.Wrapf(err, "failed to close %s", newPath)
			}
		}
		changes = append(changes, fileChange{
			newPath:   newPath,
			destPath:  destPath,
			rename:    rename,
			file:      file,
			stat:      statCopy,
			typChange: rename && oldFi.IsDir() != fi.IsDir(),
		})
	}

	files := changes.files()
	if len(files) != 0 && dw.opt.SyncDataCb != nil {
		wc := newMultiWriter(files)
		if err := dw.processChange(ChangeKindAdd, path, fi, wc); err != nil {
			wc.Close()
			return err
		}
	}
	for _, change := range changes {
		if err := dw.opt.rewriteMetadata(change.newPath, &change.stat); err != nil {
			return errors.Wrapf(err, "error setting metadata for %s", change.newPath)
		}

		if change.rename {
			if change.typChange {
				if err := os.RemoveAll(change.destPath); err != nil {
					return errors.Wrapf(err, "failed to remove %s", change.destPath)
				}
			}
			if err := os.Rename(change.newPath, change.destPath); err != nil {
				return errors.Wrapf(err, "failed to rename %s to %s", change.newPath, change.destPath)
			}
		}
	}
	if len(files) != 0 {
		if dw.opt.AsyncDataCb != nil {
			dw.requestAsyncFileData(path, fi, files)
		}
	}

	return nil
}

func (dw *DiskWriter) requestAsyncFileData(path string, fi os.FileInfo, changes []fileChange) {
	// todo: limit worker threads
	dw.eg.Go(func() error {
		ws := make([]io.Writer, 0, len(changes))
		for _, change := range changes {
			ws = append(ws, &lazyFileWriter{
				dest: change.destPath,
			})
		}
		wc := &multiWriter{
			Writer: io.MultiWriter(ws...),
		}
		if err := dw.processChange(ChangeKindAdd, path, fi, wc); err != nil {
			return err
		}
		var errs []error
		for _, change := range changes {
			if err := chtimes(change.destPath, change.stat.ModTime); err != nil { // TODO: parent dirs
				errs = append(errs, err)
			}
		}
		return newMultiErr(errs...)
	})
}

// processChange invokes the corresponding callback (SyncDataCb or AsyncDataCb, in that order)
// for the file at path with file info fi, using wc to write file's data.
func (dw *DiskWriter) processChange(kind ChangeKind, path string, fi os.FileInfo, wc io.WriteCloser) error {
	origwc := wc
	var hw *hashedWriter
	if dw.opt.NotifyCb != nil {
		var err error
		if hw, err = newHashWriter(dw.opt.ContentHasher, fi, wc); err != nil {
			return err
		}
		wc = hw
	}
	if origwc != nil {
		fn := dw.opt.SyncDataCb
		if fn == nil && dw.opt.AsyncDataCb != nil {
			fn = dw.opt.AsyncDataCb
		}
		if err := fn(dw.ctx, path, wc); err != nil {
			return err
		}
	} else {
		if hw != nil {
			hw.Close()
		}
	}
	if hw != nil {
		return dw.opt.NotifyCb(kind, path, hw, nil)
	}
	return nil
}

type hashedWriter struct {
	os.FileInfo
	io.Writer
	h    hash.Hash
	w    io.WriteCloser
	dgst digest.Digest
}

func newHashWriter(ch ContentHasher, fi os.FileInfo, w io.WriteCloser) (*hashedWriter, error) {
	stat, ok := fi.Sys().(*types.Stat)
	if !ok {
		return nil, errors.Errorf("invalid change without stat information")
	}

	h, err := ch(stat)
	if err != nil {
		return nil, err
	}
	hw := &hashedWriter{
		FileInfo: fi,
		Writer:   io.MultiWriter(w, h),
		h:        h,
		w:        w,
	}
	return hw, nil
}

func (hw *hashedWriter) Close() error {
	hw.dgst = digest.NewDigest(digest.SHA256, hw.h)
	if hw.w != nil {
		return hw.w.Close()
	}
	return nil
}

func (hw *hashedWriter) Digest() digest.Digest {
	return hw.dgst
}

type lazyFileWriter struct {
	dest     string
	f        *os.File
	fileMode *os.FileMode
}

func (lfw *lazyFileWriter) Write(dt []byte) (int, error) {
	if lfw.f == nil {
		file, err := os.OpenFile(lfw.dest, os.O_WRONLY, 0) //todo: windows
		if os.IsPermission(err) {
			// retry after chmod
			fi, er := os.Stat(lfw.dest)
			if er == nil {
				mode := fi.Mode()
				lfw.fileMode = &mode
				er = os.Chmod(lfw.dest, mode|0222)
				if er == nil {
					file, err = os.OpenFile(lfw.dest, os.O_WRONLY, 0)
				}
			}
		}
		if err != nil {
			return 0, errors.Wrapf(err, "failed to open %s", lfw.dest)
		}
		lfw.f = file
	}
	return lfw.f.Write(dt)
}

func (lfw *lazyFileWriter) Close() error {
	var err error
	if lfw.f != nil {
		err = lfw.f.Close()
	}
	if err == nil && lfw.fileMode != nil {
		err = os.Chmod(lfw.dest, *lfw.fileMode)
	}
	return err
}

// Random number state.
// We generate random temporary file names so that there's a good
// chance the file doesn't exist yet - keeps the number of tries in
// TempFile to a minimum.
var rand uint32
var randmu sync.Mutex

func reseed() uint32 {
	return uint32(time.Now().UnixNano() + int64(os.Getpid()))
}

func nextSuffix() string {
	randmu.Lock()
	r := rand
	if r == 0 {
		r = reseed()
	}
	r = r*1664525 + 1013904223 // constants from Numerical Recipes
	rand = r
	randmu.Unlock()
	return strconv.Itoa(int(1e9 + r%1e9))[1:]
}

// files returns the list of changes that change
// regular files (e.g. where the file reference is non-nil)
func (r fileChanges) files() (res []fileChange) {
	for _, change := range r {
		if change.file != nil {
			res = append(res, change)
		}
	}
	return res
}

type fileChanges []fileChange

type fileChange struct {
	// newPath specifies the new absolute path to the file/directory
	// in case of a rename
	newPath string
	// destPath is the absolute path to the file
	// in destination directory
	destPath string
	rename   bool
	// typChange indicates that a file is being switched to a directory
	// or vice versa and the old location should be cleaned up
	typChange bool
	// file optionally specifies the file
	file *os.File
	stat types.Stat
}

func (r *multiWriter) Close() (err error) {
	var errs []error
	for _, wc := range r.wcs {
		if err := wc.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	return newMultiErr(errs...)
}

func newMultiWriter(changes []fileChange) *multiWriter {
	var ws []io.Writer
	var wcs []io.WriteCloser
	for _, change := range changes {
		ws = append(ws, change.file)
		wcs = append(wcs, change.file)
	}
	return &multiWriter{
		Writer: io.MultiWriter(ws...),
		wcs:    wcs,
	}
}

type multiWriter struct {
	io.Writer
	wcs []io.WriteCloser
}

func newMultiErr(errs ...error) error {
	switch len(errs) {
	case 0:
		return nil
	case 1:
		return errs[0]
	default:
		return multiErr{errs: errs}
	}
}

func (r multiErr) Error() string {
	msgs := make([]string, 0, len(r.errs))
	for _, err := range r.errs {
		msgs = append(msgs, err.Error())
	}
	return strings.Join(msgs, ",")
}

type multiErr struct {
	errs []error
}
