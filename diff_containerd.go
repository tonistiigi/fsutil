package fsutil

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/tonistiigi/fsutil/types"
	"golang.org/x/sync/errgroup"
)

// Everything below is copied from containerd/fs. TODO: remove duplication @dmcgowan

// Const redefined because containerd/fs doesn't build on !linux

// ChangeKind is the type of modification that
// a change is making.
type ChangeKind int

const (
	// ChangeKindAdd represents an addition of
	// a file
	ChangeKindAdd ChangeKind = iota

	// ChangeKindModify represents a change to
	// an existing file
	ChangeKindModify

	// ChangeKindDelete represents a delete of
	// a file
	ChangeKindDelete
)

func (k ChangeKind) String() string {
	switch k {
	case ChangeKindAdd:
		return "add"
	case ChangeKindModify:
		return "modify"
	case ChangeKindDelete:
		return "delete"
	default:
		return "unknown"
	}
}

// ChangeFunc is the type of function called for each change
// computed during a directory changes calculation.
type ChangeFunc func(ChangeKind, string, os.FileInfo, error) error

type currentPath struct {
	path        string
	stat        *types.Stat
	contentHash func(context.Context) ([]byte, error)
}

// doubleWalkDiff walks both directories to create a diff
func doubleWalkDiff(ctx context.Context, changeFn ChangeFunc, a, b walkerFn, filter FilterFunc, differ DiffType) (err error) {
	g, ctx := errgroup.WithContext(ctx)

	var (
		c1 = make(chan *currentPath, 128)
		c2 = make(chan *currentPath, 128)

		f1, f2 *currentPath
		rmdir  string

		changes []diffChange
		pending []pendingContentCheck
	)
	emitChange := func(kind ChangeKind, path string, stat *types.Stat) error {
		if differ == DiffContent {
			changes = append(changes, diffChange{kind: kind, path: path, stat: stat})
			return nil
		}
		return changeFn(kind, path, &StatInfo{stat}, nil)
	}
	g.Go(func() error {
		defer close(c1)
		return a(ctx, c1)
	})
	g.Go(func() error {
		defer close(c2)
		return b(ctx, c2)
	})
	g.Go(func() error {
	loop0:
		for c1 != nil || c2 != nil {
			if f1 == nil && c1 != nil {
				f1, err = nextPath(ctx, c1)
				if err != nil {
					return err
				}
				if f1 == nil {
					c1 = nil
				}
			}

			if f2 == nil && c2 != nil {
				f2, err = nextPath(ctx, c2)
				if err != nil {
					return err
				}
				if f2 == nil {
					c2 = nil
				}
			}
			if f1 == nil && f2 == nil {
				continue
			}

			var f *types.Stat
			var f2copy *currentPath
			if f2 != nil {
				statCopy := f2.stat.Clone()
				if filter != nil {
					filter(f2.path, statCopy)
				}
				f2copy = &currentPath{path: f2.path, stat: statCopy, contentHash: f2.contentHash}
			}
			k, p := pathChange(f1, f2copy)
			switch k {
			case ChangeKindAdd:
				if rmdir != "" {
					rmdir = ""
				}
				f = f2.stat
				f2 = nil
			case ChangeKindDelete:
				// Check if this file is already removed by being
				// under of a removed directory
				if rmdir != "" && strings.HasPrefix(f1.path, rmdir) {
					f1 = nil
					continue
				} else if rmdir == "" && f1.stat.IsDir() {
					rmdir = f1.path + string(filepath.Separator)
				} else if rmdir != "" {
					rmdir = ""
				}
				f1 = nil
			case ChangeKindModify:
				result, err := compareFile(f1, f2copy, differ)
				if err != nil {
					return err
				}
				lower := f1
				upper := f2copy
				if f1.stat.IsDir() && !f2copy.stat.IsDir() {
					rmdir = f1.path + string(filepath.Separator)
				} else if rmdir != "" {
					rmdir = ""
				}
				f = f2.stat
				f1 = nil
				f2 = nil
				switch result {
				case fileCompareSame:
					continue loop0
				case fileCompareNeedsContentCheck:
					pending = append(pending, pendingContentCheck{
						changeIndex: len(changes),
						lower:       lower,
						upper:       upper,
					})
				}
			}
			if err := emitChange(k, p, f); err != nil {
				return err
			}
		}
		if differ == DiffContent {
			if err := resolveContentChecks(ctx, changes, pending); err != nil {
				return err
			}
			for _, change := range changes {
				if change.skip {
					continue
				}
				if err := changeFn(change.kind, change.path, &StatInfo{change.stat}, nil); err != nil {
					return err
				}
			}
		}
		return nil
	})

	return g.Wait()
}

func pathChange(lower, upper *currentPath) (ChangeKind, string) {
	if lower == nil {
		if upper == nil {
			panic("cannot compare nil paths")
		}
		return ChangeKindAdd, upper.path
	}
	if upper == nil {
		return ChangeKindDelete, lower.path
	}

	switch i := ComparePath(lower.path, upper.path); {
	case i < 0:
		// File in lower that is not in upper
		return ChangeKindDelete, lower.path
	case i > 0:
		// File in upper that is not in lower
		return ChangeKindAdd, upper.path
	default:
		return ChangeKindModify, upper.path
	}
}

type fileCompareResult int

const (
	fileCompareChanged fileCompareResult = iota
	fileCompareSame
	fileCompareNeedsContentCheck
)

type diffChange struct {
	kind ChangeKind
	path string
	stat *types.Stat
	skip bool
}

type pendingContentCheck struct {
	changeIndex int
	lower       *currentPath
	upper       *currentPath
}

func compareFile(f1, f2 *currentPath, differ DiffType) (fileCompareResult, error) {
	if differ == DiffNone {
		return fileCompareChanged, nil
	}
	// If not a directory also check size, modtime, and content
	if !f1.stat.IsDir() {
		if f1.stat.Size != f2.stat.Size {
			return fileCompareChanged, nil
		}

		if f1.stat.ModTime != f2.stat.ModTime {
			return fileCompareChanged, nil
		}
	}

	same, err := compareStat(f1.stat, f2.stat)
	if err != nil || !same || differ == DiffMetadata {
		if same {
			return fileCompareSame, err
		}
		return fileCompareChanged, err
	}

	if !canContentCheck(f1.stat) || !canContentCheck(f2.stat) || f1.stat.Size == 0 {
		return fileCompareSame, nil
	}
	if f1.contentHash == nil || f2.contentHash == nil {
		return fileCompareChanged, nil
	}
	return fileCompareNeedsContentCheck, nil
}

func resolveContentChecks(ctx context.Context, changes []diffChange, pending []pendingContentCheck) error {
	if len(pending) == 0 {
		return nil
	}

	lowerHashes := make([][]byte, len(pending))
	upperHashes := make([][]byte, len(pending))
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(16)

	for i, check := range pending {
		g.Go(func() error {
			hash, err := check.upper.contentHash(ctx)
			if err != nil {
				return err
			}
			upperHashes[i] = hash
			return nil
		})
		g.Go(func() error {
			hash, err := check.lower.contentHash(ctx)
			if err != nil {
				return err
			}
			lowerHashes[i] = hash
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return err
	}

	for i, check := range pending {
		if bytes.Equal(lowerHashes[i], upperHashes[i]) {
			changes[check.changeIndex].skip = true
		}
	}
	return nil
}

// compareStat returns whether the stats are equivalent,
// whether the files are considered the same file, and
// an error
func compareStat(ls1, ls2 *types.Stat) (bool, error) {
	return ls1.Mode == ls2.Mode && ls1.Uid == ls2.Uid && ls1.Gid == ls2.Gid && ls1.Devmajor == ls2.Devmajor && ls1.Devminor == ls2.Devminor && ls1.Linkname == ls2.Linkname, nil
}

func nextPath(ctx context.Context, pathC <-chan *currentPath) (*currentPath, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case p := <-pathC:
		return p, nil
	}
}
