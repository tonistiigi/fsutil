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
	"github.com/tonistiigi/fsutil/types"
)

type WalkOpt struct {
	IncludePatterns []string
	ExcludePatterns []string
	// FollowPaths contains symlinks that are resolved into include patterns
	// before performing the fs walk
	FollowPaths []string
	Map         FilterFunc
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

	var pm *fileutils.PatternMatcher
	if opt != nil && opt.ExcludePatterns != nil {
		pm, err = fileutils.NewPatternMatcher(opt.ExcludePatterns)
		if err != nil {
			return errors.Wrapf(err, "invalid excludepaths %s", opt.ExcludePatterns)
		}
	}

	var includePatterns []string
	if opt != nil && opt.IncludePatterns != nil {
		includePatterns = make([]string, len(opt.IncludePatterns))
		for k := range opt.IncludePatterns {
			includePatterns[k] = filepath.Clean(opt.IncludePatterns[k])
		}
	}
	if opt != nil && opt.FollowPaths != nil {
		targets, err := FollowLinks(p, opt.FollowPaths)
		if err != nil {
			return err
		}
		if targets != nil {
			includePatterns = append(includePatterns, targets...)
			includePatterns = dedupePaths(includePatterns)
		}
	}

	var lastIncludedDir string

	seenFiles := make(map[uint64]string)
	seenDirs := make(map[string]struct{})
	return filepath.Walk(root, func(path string, fi os.FileInfo, err error) (retErr error) {
		if err != nil {
			if os.IsNotExist(err) {
				return filepath.SkipDir
			}
			return err
		}
		defer func() {
			if retErr != nil && isNotExist(retErr) {
				retErr = filepath.SkipDir
			}
		}()
		origpath := path
		path, err = filepath.Rel(root, path)
		if err != nil {
			return err
		}
		// Skip root
		if path == "." {
			return nil
		}

		partial := false
		if opt != nil {
			if includePatterns != nil {
				skip := false
				if lastIncludedDir != "" {
					if strings.HasPrefix(path, lastIncludedDir+string(filepath.Separator)) {
						skip = true
					}
				}

				if !skip {
					matched := false
					partial = true
					for _, p := range includePatterns {
						if ok, p := matchPrefix(p, path, fi.IsDir()); ok {
							matched = true
							if !p {
								partial = false
								break
							}
						}
					}
					if !matched {
						if fi.IsDir() {
							return filepath.SkipDir
						}
						return nil
					}
					if !partial && fi.IsDir() {
						lastIncludedDir = path
					}
				}
			}
			if pm != nil {
				m, err := pm.Matches(path)
				if err != nil {
					return errors.Wrap(err, "failed to match excludepatterns")
				}

				if m {
					if fi.IsDir() {
						if !pm.Exclusions() {
							return filepath.SkipDir
						}
						dirSlash := path + string(filepath.Separator)
						for _, pat := range pm.Patterns() {
							if !pat.Exclusion() {
								continue
							}
							patStr := pat.String() + string(filepath.Separator)
							if strings.HasPrefix(patStr, dirSlash) {
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
		// don't report partial files yet, we might later if we eventually find a
		// non-partial file in this path
		if partial && fi.IsDir() {
			return nil
		}

		type pathInfo struct {
			OrigPath string
			Path     string
			Info     os.FileInfo
		}

		// collect any parent path that have not been previously reported
		// so we can report it first
		newPaths := []pathInfo{}
		origDir := filepath.Dir(origpath)
		if _, ok := seenDirs[origDir]; !ok {
			relDir, err := filepath.Rel(root, origDir)
			if err != nil {
				return err
			}
			dirParts := strings.Split(relDir, string(filepath.Separator))
			for i := 0; i < len(dirParts); i++ {
				dir := filepath.Join(dirParts[0 : i+1]...)
				if dir == "." {
					continue
				}
				origDir := filepath.Join(root, dir)
				if _, ok := seenDirs[origDir]; !ok {
					fi, err := os.Stat(origDir)
					if err != nil {
						return err
					}
					newPaths = append(newPaths, pathInfo{
						OrigPath: origDir,
						Path:     dir,
						Info:     fi,
					})
					seenDirs[origDir] = struct{}{}
				}
			}
		}
		// if we were reporting a directory then track it so
		// we don't report it again.
		if fi.IsDir() {
			seenDirs[origpath] = struct{}{}
		}
		newPaths = append(newPaths, pathInfo{
			OrigPath: origpath,
			Path:     path,
			Info:     fi,
		})

		for _, path := range newPaths {
			stat, err := mkstat(path.OrigPath, path.Path, path.Info, seenFiles)
			if err != nil {
				return err
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				if opt != nil && opt.Map != nil {
					if allowed := opt.Map(stat.Path, stat); !allowed {
						return nil
					}
				}
				if err := fn(stat.Path, &StatInfo{stat}, nil); err != nil {
					return err
				}
			}
		}
		return nil
	})
}

type StatInfo struct {
	*types.Stat
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

func matchPrefix(pattern, name string, isDir bool) (bool, bool) {
	partial := false
	if strings.Contains(pattern, "**") {
		// short-circuit for single "**"
		if pattern == "**" {
			return true, false
		}
		pattParts := strings.Split(pattern, string(filepath.Separator))
		lastPatt := len(pattParts) - 1
		fileParts := strings.Split(name, string(filepath.Separator))
		lastFile := len(fileParts) - 1
		// simple "stack" to allow for backtracking the cursors if there
		// is a partial match with doublestar, so that we can check if
		// there is a better (complete) match by being greedy with
		// the doublestar matching, ie if we have `**/b*` we want it to
		// completely match `foo/bar/baz` rather than return partial
		// match to `foo/bar`
		stack := [][2]int{{0, 0}}
		seenAttempt := map[[2]int]struct{}{}
	backtrack:
		for sp := 0; sp < len(stack); sp++ {
			frame := stack[sp]
			// track cursors from frame, first frame will be 0,0
			pattCursor, fileCursor := frame[0], frame[1]

			// if we are backtracking we start in doubleGlobbing mode,
			doubleGlobbing := pattCursor > 0
			// if doubleStar globbing is trailing then we match
			// files and dirs, otherwise just dirs
			allGlobbing := pattCursor == lastPatt

			// we will double loop, with outer loop being the pattern parts
			// and the inner loop being the file path parts.  We will
			// keep a cursor to identify where were are at in the pattern
			// matching.  If both the pattern parts and the file parts are
			// entirely consumed then  we have a match, otherwise it
			// might be a partial match
		pattern:
			for pattCursor < len(pattParts) && fileCursor < len(fileParts) {
				pattPart := pattParts[pattCursor]
				if pattPart == "**" {
					// advance pattern cursor so we can start looking for
					// the next pattern, but flag that we are double globbing
					// so we can consume file parts that don't match the next
					// pattern.
					allGlobbing = pattCursor == lastPatt
					doubleGlobbing = true
					pattCursor++
					continue
				}
				for fileCursor < len(fileParts) {
					if _, ok := seenAttempt[[2]int{pattCursor, fileCursor}]; ok {
						// been here before, so give up and try next backtrack
						continue backtrack
					}
					seenAttempt[[2]int{pattCursor, fileCursor}] = struct{}{}
					filePart := fileParts[fileCursor]
					if ok, _ := filepath.Match(pattPart, filePart); ok {
						if doubleGlobbing {
							// we are double globbing, so push on alternate to stack
							// in case we can get a better complete match being greedy
							stack = append(stack, [2]int{pattCursor, fileCursor + 1})
						}
						partial = true
						pattCursor++
						fileCursor++
						continue pattern
					}
					// not a pattern match but it might be captured by double glob.
					// If target is a file only capture filename if matching
					// with an allGlobbing pattern (ie `**` will match, `**/...` will not`)
					if doubleGlobbing && (allGlobbing || (!isDir && fileCursor < lastFile) || isDir) {
						partial = true
						fileCursor++
						continue
					}
					// no match and not double globbing, so give up
					return false, partial
				}
				// ran out of fileParts so break pattern loop
				break
			}
			if pattCursor == len(pattParts) && fileCursor == len(fileParts) {
				// all the pattern and file parts were consumed, so we have a complete match
				return true, false
			}
		}
		// partial matches are only relevant if the input is a directory, otherwise this is a failed match
		return isDir, partial
	}

	count := strings.Count(name, string(filepath.Separator))
	if strings.Count(pattern, string(filepath.Separator)) > count {
		pattern = trimUntilIndex(pattern, string(filepath.Separator), count)
		partial = true
	}
	m, _ := filepath.Match(pattern, name)
	return m, partial
}

func trimUntilIndex(str, sep string, count int) string {
	s := str
	i := 0
	c := 0
	for {
		idx := strings.Index(s, sep)
		s = s[idx+len(sep):]
		i += idx + len(sep)
		c++
		if c > count {
			return str[:i-len(sep)]
		}
	}
}

func isNotExist(err error) bool {
	err = errors.Cause(err)
	if os.IsNotExist(err) {
		return true
	}
	if pe, ok := err.(*os.PathError); ok {
		err = pe.Err
	}
	return err == syscall.ENOTDIR
}
