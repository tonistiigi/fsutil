package fsutil

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/docker/docker/pkg/fileutils"
	"github.com/pkg/errors"
	"github.com/stevvooe/continuity/sysx"
)

type WalkOpt struct {
	IncludePaths    []string // todo: remove?
	ExcludePatterns []string
}

func Walk(ctx context.Context, p string, opt *WalkOpt, fn filepath.WalkFunc) error {
	root, err := filepath.EvalSymlinks(p)
	if err != nil {
		return errors.Wrapf(err, "failed to resolve %s", root)
	}
	fi, err := os.Stat(root)
	if err != nil {
		return errors.Wrapf(err, "failed to stat: %s", root)
	}
	if !fi.IsDir() {
		return errors.Errorf("%s is not a directory", root)
	}

	var patterns []string
	var patDirs [][]string
	var exceptions bool
	if opt != nil && opt.ExcludePatterns != nil {
		patterns, patDirs, exceptions, err = fileutils.CleanPatterns(opt.ExcludePatterns)

		if err != nil {
			return errors.Wrapf(err, "invalid excludepaths %s", opt.ExcludePatterns)
		}
	}

	seenFiles := make(map[uint64]string)
	return filepath.Walk(root, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		origpath := path
		path, err = filepath.Rel(root, path)
		if err != nil {
			return err
		}
		// Skip root
		if path == "." {
			return nil
		}

		if opt != nil {
			if opt.IncludePaths != nil {
				matched := false
				for _, p := range opt.IncludePaths {
					if m, _ := filepath.Match(p, path); m {
						matched = true
						break
					}
				}
				if !matched {
					if fi.IsDir() {
						return filepath.SkipDir
					}
					return nil
				}
			}
			if opt.ExcludePatterns != nil {
				m, err := fileutils.OptimizedMatches(path, patterns, patDirs)
				if err != nil {
					return errors.Wrap(err, "failed to match excludepatterns")
				}

				if m {
					if fi.IsDir() {
						if !exceptions {
							return filepath.SkipDir
						}
						dirSlash := path + string(filepath.Separator)
						for _, pat := range patterns {
							if pat[0] != '!' {
								continue
							}
							pat = pat[1:] + string(filepath.Separator)
							if strings.HasPrefix(pat, dirSlash) {
								goto passedFilter
							}
						}
						return filepath.SkipDir
					}
					return nil
				}
			}
		}

	passedFilter:
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

			if fi.Mode()&os.ModeSymlink != 0 {
				link, err := os.Readlink(origpath)
				if err != nil {
					return errors.Wrapf(err, "failed to readlink %s", origpath)
				}
				stat.Linkname = link
			}
		}

		xattrs, err := sysx.LListxattr(origpath)
		if err != nil {
			return errors.Wrapf(err, "failed to xattr %s", path)
		}
		if len(xattrs) > 0 {
			m := make(map[string][]byte)
			for _, key := range xattrs {
				v, err := sysx.LGetxattr(origpath, key)
				if err == nil {
					m[key] = v
				}
			}
			stat.Xattrs = m
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := fn(path, &StatInfo{stat}, nil); err != nil {
				return err
			}
		}
		return nil
	})
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
