package diffcopy

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/pkg/errors"
)

type Walker interface {
	NextPath() (Path, error)
}

type WalkOpt struct {
	IncludePaths []string
	ExcludePaths []string
}

type Path interface { // todo: change fs.Change type to avoid this
	Path() string
	os.FileInfo
}

func Walk(ctx context.Context, p string, opt *WalkOpt) (Walker, error) {
	evalPath, err := filepath.EvalSymlinks(p)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to resolve %s", evalPath)
	}
	fi, err := os.Stat(evalPath)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to stat: %s", evalPath)
	}
	if !fi.IsDir() {
		return nil, errors.Errorf("%s is not a directory", evalPath)
	}
	w := &walker{
		root: evalPath,
		opt:  opt,
		ctx:  ctx,
		ch:   make(chan Path, 128),
	}
	go w.run(ctx)
	return w, nil
}

type walker struct {
	root string
	opt  *WalkOpt
	ch   chan Path
	ctx  context.Context
	mu   sync.RWMutex
	err  error
}

func (w *walker) NextPath() (Path, error) {
	select {
	case <-w.ctx.Done():
		w.mu.RLock()
		defer w.mu.RLock()
		if w.err != nil {
			return nil, w.err
		} else {
			return nil, w.ctx.Err()
		}
	}
	select {
	case <-w.ctx.Done():
		w.mu.RLock()
		defer w.mu.RLock()
		if w.err != nil {
			return nil, w.err
		} else {
			return nil, w.ctx.Err()
		}
	case p := <-w.ch:
		return p, nil
	}
}

func (w *walker) run(ctx context.Context) error {
	seenFiles := make(map[uint64]string)
	defer close(w.ch)
	err := filepath.Walk(w.root, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		path, err = filepath.Rel(w.root, path)
		if err != nil {
			return err
		}

		if w.opt != nil {
			if w.opt.IncludePaths != nil {
				matched := false
				for _, p := range w.opt.IncludePaths {
					if m, _ := filepath.Match(p, path); m {
						matched = true
						break
					}
				}
				if !matched {
					return nil
				}
			}
			if w.opt.ExcludePaths != nil {
				for _, p := range w.opt.ExcludePaths {
					if m, _ := filepath.Match(p, path); m {
						return nil
					}
				}
			}
		}

		// Skip root
		if path == "." {
			return nil
		}

		s, ok := fi.Sys().(*syscall.Stat_t)
		if !ok {
			return errors.New("linux only atm")
		}

		stat := &Stat{
			Path:    path,
			Mode:    uint32(fi.Mode()),
			Uid:     s.Uid,
			Gid:     s.Gid,
			Size_:   fi.Size(),
			ModTime: fi.ModTime().UnixNano(),
		}

		if !fi.IsDir() {
			if s.Mode&syscall.S_IFBLK != 0 ||
				s.Mode&syscall.S_IFCHR != 0 {
				stat.Devmajor = int64(major(uint64(s.Rdev)))
				stat.Devminor = int64(minor(uint64(s.Rdev)))
			}

			ino := s.Ino
			if s.Nlink > 1 {
				if oldpath, ok := seenFiles[ino]; ok {
					stat.Linkname = oldpath
					stat.Size_ = 0
				}
			}
			seenFiles[ino] = path
		}

		p := &currentPath{
			path:     path,
			FileInfo: &StatInfo{stat},
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case w.ch <- p:
			return nil
		}
	})
	if err != nil {
		w.mu.Lock()
		w.err = err
		w.mu.Unlock()
	}
	return err
}

func major(device uint64) uint64 {
	return (device >> 8) & 0xfff
}

func minor(device uint64) uint64 {
	return (device & 0xff) | ((device >> 12) & 0xfff00)
}

type currentPath struct {
	os.FileInfo
	path string
}

func (p *currentPath) Path() string {
	return p.path
}

type StatInfo struct {
	*Stat
}

func (s *StatInfo) Name() string {
	return filepath.Base(s.Stat.Path)
}
func (s *StatInfo) Size() int64 {
	return s.Stat.Size_
}
func (s *StatInfo) Mode() os.FileMode {
	return os.FileMode(s.Stat.Mode)
}
func (s *StatInfo) ModTime() time.Time {
	return time.Unix(s.Stat.ModTime/1e9, s.Stat.ModTime%1e9)
}
func (s *StatInfo) IsDir() bool {
	return s.Mode().IsDir()
}
func (s *StatInfo) Sys() interface{} {
	return s.Stat
}
