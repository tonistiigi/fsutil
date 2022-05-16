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

	fws := make(fileWriters, 0, len(dw.dests))
	for _, dest := range dw.dests {
		destPath := filepath.Join(dest, filepath.FromSlash(path))
		switch kind {
		case ChangeKindDelete:
			if err := dw.handleDelete(destPath, path); err != nil {
				return err
			}
		default:
			fw := fileWriter{
				destPath: destPath,
				destDir:  dest,
			}
			if err := dw.handleChange(kind, path, fi, &fw); err != nil {
				return err
			}
			fws = append(fws, fw)
		}
	}
	// TODO(dima): rewrite this to match the original closer
	if kind == ChangeKindDelete {
		return nil
	}
	files := fws.files()
	if len(files) != 0 && dw.opt.SyncDataCb != nil {
		wc := newMultiWriter(fws...)
		if err := dw.processChange(ChangeKindAdd, path, fi, wc); err != nil {
			wc.Close()
			return err
		}
	}
	for _, fw := range fws {
		if err := dw.opt.rewriteMetadata(fw.newPath, &fw.stat); err != nil {
			return errors.Wrapf(err, "error setting metadata for %s", fw.newPath)
		}

		if fw.rename {
			if fw.typChange {
				if err := os.RemoveAll(fw.destPath); err != nil {
					return errors.Wrapf(err, "failed to remove %s", fw.destPath)
				}
			}
			if err := os.Rename(fw.newPath, fw.destPath); err != nil {
				return errors.Wrapf(err, "failed to rename %s to %s", fw.newPath, fw.destPath)
			}
		}
	}
	if len(files) != 0 {
		if dw.opt.AsyncDataCb != nil {
			dw.requestAsyncFileData(path, fi, files)
		}
		return nil
	}
	return dw.processChange(kind, path, fi, nil)
}

func (dw *DiskWriter) handleDelete(destPath, path string) error {
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
	return nil
}

func (dw *DiskWriter) handleChange(kind ChangeKind, path string, fi os.FileInfo, res *fileWriter) (err error) {
	stat, ok := fi.Sys().(*types.Stat)
	if !ok {
		return errors.WithStack(&os.PathError{Path: path, Err: syscall.EBADMSG, Op: "change without stat info"})
	}

	res.stat = *stat

	if dw.filter != nil {
		if ok := dw.filter(path, &res.stat); !ok {
			return nil
		}
	}

	res.rename = true
	oldFi, err := os.Lstat(res.destPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			if kind != ChangeKindAdd {
				return errors.Wrap(err, "modify/rm")
			}
			res.rename = false
		} else {
			return errors.WithStack(err)
		}
	}

	if oldFi != nil && fi.IsDir() && oldFi.IsDir() {
		if err := dw.opt.rewriteMetadata(res.destPath, &res.stat); err != nil {
			return errors.Wrapf(err, "error setting dir metadata for %s", res.destPath)
		}
		return nil
	}

	res.newPath = res.destPath
	if res.rename {
		res.newPath = filepath.Join(res.destDir, ".tmp."+nextSuffix())
	}

	res.typChange = oldFi != nil && oldFi.IsDir() != fi.IsDir()
	switch {
	case fi.IsDir():
		if err := os.Mkdir(res.newPath, fi.Mode()); err != nil {
			return errors.Wrapf(err, "failed to create dir %s", res.newPath)
		}
	case fi.Mode()&os.ModeDevice != 0 || fi.Mode()&os.ModeNamedPipe != 0:
		if err := handleTarTypeBlockCharFifo(res.newPath, &res.stat); err != nil {
			return errors.Wrapf(err, "failed to create device %s", res.newPath)
		}
	case fi.Mode()&os.ModeSymlink != 0:
		if err := os.Symlink(res.stat.Linkname, res.newPath); err != nil {
			return errors.Wrapf(err, "failed to symlink %s", res.newPath)
		}
	case res.stat.Linkname != "":
		if err := os.Link(filepath.Join(res.destDir, res.stat.Linkname), res.newPath); err != nil {
			return errors.Wrapf(err, "failed to link %s to %s", res.newPath, res.stat.Linkname)
		}
	default:
		res.file, err = os.OpenFile(res.newPath, os.O_CREATE|os.O_WRONLY, fi.Mode()) // todo: windows
		if err != nil {
			return errors.Wrapf(err, "failed to create %s", res.newPath)
		}
		if dw.opt.SyncDataCb != nil {
			break
		}
		// Close the file for the side-effect of it being created.
		if err := res.file.Close(); err != nil {
			return errors.Wrapf(err, "failed to close %s", res.newPath)
		}
	}

	if res.rename {
		res.typChange = oldFi.IsDir() != fi.IsDir()
	}

	return nil
}

func (dw *DiskWriter) requestAsyncFileData(path string, fi os.FileInfo, fws []fileWriter) {
	// todo: limit worker threads
	// TODO(dima): handle multiple dests
	dw.eg.Go(func() error {
		ws := make([]io.Writer, 0, len(fws))
		for _, fw := range fws {
			ws = append(ws, &lazyFileWriter{
				dest: fw.destPath,
			})
		}
		wc := &multiWriter{
			Writer: io.MultiWriter(ws...),
		}
		if err := dw.processChange(ChangeKindAdd, path, fi, wc); err != nil {
			return err
		}
		var errs []error
		for _, fw := range fws {
			if err := chtimes(fw.destPath, fw.stat.ModTime); err != nil { // TODO: parent dirs
				errs = append(errs, err)
			}
		}
		return newMultiErr(errs...)
	})
}

// processChange invokes the corresponding callback (SyncDataCb or AsyncDataCb, in that order)
// for the file at path with file info fi, using wc to write file's data.
// assumes that wc is non-nil
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
		file, err := os.OpenFile(lfw.dest, os.O_WRONLY, 0) // todo: windows
		if os.IsPermission(err) {
			// retry after chmod
			fi, er := os.Stat(lfw.dest)
			if er == nil {
				mode := fi.Mode()
				lfw.fileMode = &mode
				er = os.Chmod(lfw.dest, mode|0o222)
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

func (r fileWriters) files() (res []fileWriter) {
	for _, fw := range r {
		if fw.file != nil {
			res = append(res, fw)
		}
	}
	return res
}

type fileWriters []fileWriter

type fileWriter struct {
	// newPath specifies the new absolute path to the file/directory
	// in case of a rename
	newPath string
	// TODO(dima): clearer description
	// destPath is the destination path
	destPath string
	// destDir specifies the destination directry
	destDir string
	rename  bool
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

func newMultiWriter(fws ...fileWriter) *multiWriter {
	var ws []io.Writer
	var wcs []io.WriteCloser
	for _, fw := range fws {
		if fw.file != nil {
			ws = append(ws, fw.file)
			wcs = append(wcs, fw.file)
		}
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
