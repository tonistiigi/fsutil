package fsutil

import (
	"bytes"
	"context"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tonistiigi/fsutil/types"
)

func TestWalkerSimple(t *testing.T) {
	d, err := tmpDir(changeStream([]string{
		"ADD foo file",
		"ADD foo2 file",
	}))
	assert.NoError(t, err)
	defer os.RemoveAll(d)
	b := &bytes.Buffer{}
	err = Walk(context.Background(), d, nil, bufWalk(b))
	assert.NoError(t, err)

	assert.Equal(t, string(b.Bytes()), `file foo
file foo2
`)

}

func TestWalkerInclude(t *testing.T) {
	d, err := tmpDir(changeStream([]string{
		"ADD bar dir",
		"ADD bar/foo file",
		"ADD foo2 file",
	}))
	assert.NoError(t, err)
	defer os.RemoveAll(d)
	b := &bytes.Buffer{}
	err = Walk(context.Background(), d, &WalkOpt{
		IncludePatterns: []string{"bar"},
	}, bufWalk(b))
	assert.NoError(t, err)

	assert.Equal(t, `dir bar
file bar/foo
`, string(b.Bytes()))

	b.Reset()
	err = Walk(context.Background(), d, &WalkOpt{
		IncludePatterns: []string{"bar/foo"},
	}, bufWalk(b))
	assert.NoError(t, err)

	assert.Equal(t, `dir bar
file bar/foo
`, string(b.Bytes()))

	b.Reset()
	err = Walk(context.Background(), d, &WalkOpt{
		IncludePatterns: []string{"b*"},
	}, bufWalk(b))
	assert.NoError(t, err)

	assert.Equal(t, `dir bar
file bar/foo
`, string(b.Bytes()))

	b.Reset()
	err = Walk(context.Background(), d, &WalkOpt{
		IncludePatterns: []string{"bar/f*"},
	}, bufWalk(b))
	assert.NoError(t, err)

	assert.Equal(t, `dir bar
file bar/foo
`, string(b.Bytes()))

	b.Reset()
	err = Walk(context.Background(), d, &WalkOpt{
		IncludePatterns: []string{"bar/g*"},
	}, bufWalk(b))
	assert.NoError(t, err)

	assert.Empty(t, b.Bytes())

	b.Reset()
	err = Walk(context.Background(), d, &WalkOpt{
		IncludePatterns: []string{"f*"},
	}, bufWalk(b))
	assert.NoError(t, err)

	assert.Equal(t, `file foo2
`, string(b.Bytes()))

	b.Reset()
	err = Walk(context.Background(), d, &WalkOpt{
		IncludePatterns: []string{"b*/f*"},
	}, bufWalk(b))
	assert.NoError(t, err)

	assert.Equal(t, `dir bar
file bar/foo
`, string(b.Bytes()))

	b.Reset()
	err = Walk(context.Background(), d, &WalkOpt{
		IncludePatterns: []string{"b*/foo"},
	}, bufWalk(b))
	assert.NoError(t, err)

	assert.Equal(t, `dir bar
file bar/foo
`, string(b.Bytes()))

	b.Reset()
	err = Walk(context.Background(), d, &WalkOpt{
		IncludePatterns: []string{"b*/"},
	}, bufWalk(b))
	assert.NoError(t, err)

	assert.Equal(t, `dir bar
file bar/foo
`, string(b.Bytes()))
}

func TestWalkerExclude(t *testing.T) {
	d, err := tmpDir(changeStream([]string{
		"ADD bar file",
		"ADD foo dir",
		"ADD foo2 file",
		"ADD foo/bar2 file",
	}))
	assert.NoError(t, err)
	defer os.RemoveAll(d)
	b := &bytes.Buffer{}
	err = Walk(context.Background(), d, &WalkOpt{
		ExcludePatterns: []string{"foo*", "!foo/bar2"},
	}, bufWalk(b))
	assert.NoError(t, err)

	assert.Equal(t, `file bar
dir foo
file foo/bar2
`, string(b.Bytes()))
}

func TestWalkerFollowLinks(t *testing.T) {
	d, err := tmpDir(changeStream([]string{
		"ADD bar file",
		"ADD foo dir",
		"ADD foo/l1 symlink /baz/one",
		"ADD foo/l2 symlink /baz/two",
		"ADD baz dir",
		"ADD baz/one file",
		"ADD baz/two symlink ../bax",
		"ADD bax file",
		"ADD bay file", // not included
	}))
	assert.NoError(t, err)
	defer os.RemoveAll(d)
	b := &bytes.Buffer{}
	err = Walk(context.Background(), d, &WalkOpt{
		FollowPaths: []string{"foo/l*", "bar"},
	}, bufWalk(b))
	assert.NoError(t, err)

	assert.Equal(t, `file bar
file bax
dir baz
file baz/one
symlink:../bax baz/two
dir foo
symlink:/baz/one foo/l1
symlink:/baz/two foo/l2
`, string(b.Bytes()))
}

func TestWalkerFollowLinksToRoot(t *testing.T) {
	d, err := tmpDir(changeStream([]string{
		"ADD foo symlink .",
		"ADD bar file",
		"ADD bax file",
		"ADD bay dir",
		"ADD bay/baz file",
	}))
	assert.NoError(t, err)
	defer os.RemoveAll(d)
	b := &bytes.Buffer{}
	err = Walk(context.Background(), d, &WalkOpt{
		FollowPaths: []string{"foo"},
	}, bufWalk(b))
	assert.NoError(t, err)

	assert.Equal(t, `file bar
file bax
dir bay
file bay/baz
symlink:. foo
`, string(b.Bytes()))
}

func TestWalkerMap(t *testing.T) {
	d, err := tmpDir(changeStream([]string{
		"ADD bar file",
		"ADD foo dir",
		"ADD foo2 file",
		"ADD foo/bar2 file",
	}))
	assert.NoError(t, err)
	defer os.RemoveAll(d)
	b := &bytes.Buffer{}
	err = Walk(context.Background(), d, &WalkOpt{
		Map: func(_ string, s *types.Stat) bool {
			if strings.HasPrefix(s.Path, "foo") {
				s.Path = "_" + s.Path
				return true
			}
			return false
		},
	}, bufWalk(b))
	assert.NoError(t, err)

	assert.Equal(t, `dir _foo
file _foo/bar2
file _foo2
`, string(b.Bytes()))
}

func TestWalkerPermissionDenied(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("os.Chmod not fully supported on Windows")
	}
	if os.Getuid() == 0 {
		t.Skip("test cannot run as root")
	}

	d, err := tmpDir(changeStream([]string{
		"ADD foo dir",
		"ADD foo/bar dir",
	}))
	assert.NoError(t, err)
	err = os.Chmod(filepath.Join(d, "foo", "bar"), 0000)
	require.NoError(t, err)
	defer func() {
		os.Chmod(filepath.Join(d, "bar"), 0700)
		os.RemoveAll(d)
	}()

	b := &bytes.Buffer{}
	err = Walk(context.Background(), d, &WalkOpt{}, bufWalk(b))
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "permission denied")
	}

	b.Reset()
	err = Walk(context.Background(), d, &WalkOpt{
		ExcludePatterns: []string{"**/bar"},
	}, bufWalk(b))
	assert.NoError(t, err)
	assert.Equal(t, `dir foo
`, string(b.Bytes()))

	b.Reset()
	err = Walk(context.Background(), d, &WalkOpt{
		ExcludePatterns: []string{"**/bar", "!foo/bar/baz"},
	}, bufWalk(b))
	assert.NoError(t, err)
	assert.Equal(t, `dir foo
`, string(b.Bytes()))

	b.Reset()
	err = Walk(context.Background(), d, &WalkOpt{
		ExcludePatterns: []string{"**/bar", "!foo/bar"},
	}, bufWalk(b))
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "permission denied")
	}

	b.Reset()
	err = Walk(context.Background(), d, &WalkOpt{
		IncludePatterns: []string{"foo", "!**/bar"},
	}, bufWalk(b))
	assert.NoError(t, err)
	assert.Equal(t, `dir foo
`, string(b.Bytes()))
}

func bufWalk(buf *bytes.Buffer) filepath.WalkFunc {
	return func(path string, fi os.FileInfo, err error) error {
		stat, ok := fi.Sys().(*types.Stat)
		if !ok {
			return errors.Errorf("invalid symlink %s", path)
		}
		t := "file"
		if fi.IsDir() {
			t = "dir"
		}
		if fi.Mode()&os.ModeSymlink != 0 {
			t = "symlink:" + stat.Linkname
		}
		fmt.Fprintf(buf, "%s %s", t, path)
		if fi.Mode()&os.ModeSymlink == 0 && stat.Linkname != "" {
			fmt.Fprintf(buf, " >%s", stat.Linkname)
		}
		fmt.Fprintln(buf)
		return nil
	}
}

func tmpDir(inp []*change) (dir string, retErr error) {
	tmpdir, err := ioutil.TempDir("", "diff")
	if err != nil {
		return "", err
	}
	defer func() {
		if retErr != nil {
			os.RemoveAll(tmpdir)
		}
	}()
	for _, c := range inp {
		if c.kind == ChangeKindAdd {
			p := filepath.Join(tmpdir, c.path)
			stat, ok := c.fi.Sys().(*types.Stat)
			if !ok {
				return "", errors.Errorf("invalid symlink change %s", p)
			}
			if c.fi.IsDir() {
				if err := os.Mkdir(p, 0700); err != nil {
					return "", err
				}
			} else if c.fi.Mode()&os.ModeSymlink != 0 {
				if err := os.Symlink(stat.Linkname, p); err != nil {
					return "", err
				}
			} else if len(stat.Linkname) > 0 {
				if err := os.Link(filepath.Join(tmpdir, stat.Linkname), p); err != nil {
					return "", err
				}
			} else if c.fi.Mode()&os.ModeSocket != 0 {
				// not closing listener because it would remove the socket file
				if _, err := net.Listen("unix", p); err != nil {
					return "", err
				}
			} else {
				f, err := os.Create(p)
				if err != nil {
					return "", err
				}

				// Make sure all files start with the same default permissions,
				// regardless of OS settings.
				err = os.Chmod(p, 0644)
				if err != nil {
					return "", err
				}

				if len(c.data) > 0 {
					if _, err := f.Write([]byte(c.data)); err != nil {
						return "", err
					}
				}
				f.Close()
			}
		}
	}
	return tmpdir, nil
}

func BenchmarkWalker(b *testing.B) {
	for _, scenario := range []struct {
		maxDepth int
		pattern  string
		expected int
	}{{
		maxDepth: 1,
		pattern:  "target",
		expected: 1,
	}, {
		maxDepth: 1,
		pattern:  "**/target",
		expected: 1,
	}, {
		maxDepth: 2,
		pattern:  "*/target",
		expected: 52,
	}, {
		maxDepth: 2,
		pattern:  "**/target",
		expected: 52,
	}, {
		maxDepth: 3,
		pattern:  "*/*/target",
		expected: 1378,
	}, {
		maxDepth: 3,
		pattern:  "**/target",
		expected: 1378,
	}, {
		maxDepth: 4,
		pattern:  "*/*/*/target",
		expected: 2794,
	}, {
		maxDepth: 4,
		pattern:  "**/target",
		expected: 2794,
	}, {
		maxDepth: 5,
		pattern:  "*/*/*/*/target",
		expected: 1405,
	}, {
		maxDepth: 5,
		pattern:  "**/target",
		expected: 1405,
	}, {
		maxDepth: 6,
		pattern:  "*/*/*/*/*/target",
		expected: 2388,
	}, {
		maxDepth: 6,
		pattern:  "**/target",
		expected: 2388,
	}} {
		scenario := scenario // copy loop var
		b.Run(fmt.Sprintf("[%d]-%s", scenario.maxDepth, scenario.pattern), func(b *testing.B) {
			tmpdir, err := ioutil.TempDir("", "walk")
			if err != nil {
				b.Error(err)
			}
			defer func() {
				b.StopTimer()
				os.RemoveAll(tmpdir)
			}()
			mkBenchTree(tmpdir, scenario.maxDepth, 1)

			// don't include time to setup dirs in benchmark
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				count := 0
				err = Walk(context.Background(), tmpdir, &WalkOpt{
					IncludePatterns: []string{scenario.pattern},
				}, func(path string, fi os.FileInfo, err error) error {
					count++
					return nil
				})
				if err != nil {
					b.Error(err)
				}
				if count != scenario.expected {
					b.Errorf("Got count %d, expected %d", count, scenario.expected)
				}
			}
		})
	}

}

func TestWalkerDoublestarInclude(t *testing.T) {
	d, err := tmpDir(changeStream([]string{
		"ADD a dir",
		"ADD a/b dir",
		"ADD a/b/baz dir",
		"ADD a/b/bar dir ",
		"ADD a/b/bar/foo file",
		"ADD a/b/bar/fop file",
		"ADD bar dir",
		"ADD bar/foo file",
		"ADD baz dir",
		"ADD foo2 file",
		"ADD foo dir",
		"ADD foo/bar dir",
		"ADD foo/bar/bee file",
	}))

	assert.NoError(t, err)
	defer os.RemoveAll(d)
	b := &bytes.Buffer{}
	err = Walk(context.Background(), d, &WalkOpt{
		IncludePatterns: []string{"**"},
	}, bufWalk(b))
	assert.NoError(t, err)

	trimEqual(t, `
		dir a
		dir a/b
		dir a/b/bar
		file a/b/bar/foo
		file a/b/bar/fop
		dir a/b/baz
		dir bar
		file bar/foo
		dir baz
		dir foo
		dir foo/bar
		file foo/bar/bee
		file foo2
	`, string(b.Bytes()))

	b.Reset()
	err = Walk(context.Background(), d, &WalkOpt{
		IncludePatterns: []string{"**/bar"},
	}, bufWalk(b))
	assert.NoError(t, err)

	trimEqual(t, `
		dir a
		dir a/b
		dir a/b/bar
		file a/b/bar/foo
		file a/b/bar/fop
		dir bar
		file bar/foo
		dir foo
		dir foo/bar
		file foo/bar/bee
	`, string(b.Bytes()))

	b.Reset()
	err = Walk(context.Background(), d, &WalkOpt{
		IncludePatterns: []string{"**/bar/foo"},
	}, bufWalk(b))
	assert.NoError(t, err)

	trimEqual(t, `
		dir a
		dir a/b
		dir a/b/bar
		file a/b/bar/foo
		dir bar
		file bar/foo
	`, string(b.Bytes()))

	b.Reset()
	err = Walk(context.Background(), d, &WalkOpt{
		IncludePatterns: []string{"**/b*"},
	}, bufWalk(b))
	assert.NoError(t, err)

	trimEqual(t, `
		dir a
		dir a/b
		dir a/b/bar
		file a/b/bar/foo
		file a/b/bar/fop
		dir a/b/baz
		dir bar
		file bar/foo
		dir baz
		dir foo
		dir foo/bar
		file foo/bar/bee
	`, string(b.Bytes()))

	b.Reset()
	err = Walk(context.Background(), d, &WalkOpt{
		IncludePatterns: []string{"**/bar/f*"},
	}, bufWalk(b))
	assert.NoError(t, err)

	trimEqual(t, `
			dir a
			dir a/b
			dir a/b/bar
			file a/b/bar/foo
			file a/b/bar/fop
			dir bar
			file bar/foo
	`, string(b.Bytes()))

	b.Reset()
	err = Walk(context.Background(), d, &WalkOpt{
		IncludePatterns: []string{"**/bar/g*"},
	}, bufWalk(b))
	assert.NoError(t, err)

	trimEqual(t, ``, string(b.Bytes()))

	b.Reset()
	err = Walk(context.Background(), d, &WalkOpt{
		IncludePatterns: []string{"**/f*"},
	}, bufWalk(b))
	assert.NoError(t, err)

	trimEqual(t, `
		dir a
		dir a/b
		dir a/b/bar
		file a/b/bar/foo
		file a/b/bar/fop
		dir bar
		file bar/foo
		dir foo
		dir foo/bar
		file foo/bar/bee
		file foo2
	`, string(b.Bytes()))

	b.Reset()
	err = Walk(context.Background(), d, &WalkOpt{
		IncludePatterns: []string{"**/b*/f*"},
	}, bufWalk(b))
	assert.NoError(t, err)

	trimEqual(t, `
		dir a
		dir a/b
		dir a/b/bar
		file a/b/bar/foo
		file a/b/bar/fop
		dir bar
		file bar/foo
	`, string(b.Bytes()))

	b.Reset()
	err = Walk(context.Background(), d, &WalkOpt{
		IncludePatterns: []string{"**/b*/foo"},
	}, bufWalk(b))
	assert.NoError(t, err)

	trimEqual(t, `
		dir a
		dir a/b
		dir a/b/bar
		file a/b/bar/foo
		dir bar
		file bar/foo
	`, string(b.Bytes()))

	b.Reset()
	err = Walk(context.Background(), d, &WalkOpt{
		IncludePatterns: []string{"**/foo/**"},
	}, bufWalk(b))
	assert.NoError(t, err)

	trimEqual(t, `
		dir foo
		dir foo/bar
		file foo/bar/bee
	`, string(b.Bytes()))

	b.Reset()
	err = Walk(context.Background(), d, &WalkOpt{
		IncludePatterns: []string{"**/baz"},
	}, bufWalk(b))
	assert.NoError(t, err)

	trimEqual(t, `
		dir a
		dir a/b
		dir a/b/baz
		dir baz
	`, string(b.Bytes()))
}

func trimEqual(t assert.TestingT, expected, actual string, msgAndArgs ...interface{}) bool {
	lines := []string{}
	for _, line := range strings.Split(expected, "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	lines = append(lines, "") // we expect a trailing newline
	expected = strings.Join(lines, "\n")

	return assert.Equal(t, expected, actual, msgAndArgs)
}

// mkBenchTree will create directories named a-z recursively
// up to 3 layers deep.  If maxDepth is > 3 we will shorten
// the last letter to prevent the generated inodes going over
// 25k. The final directory in the tree will contain only files.
// Additionally there is a single file named `target`
// in each leaf directory.
func mkBenchTree(dir string, maxDepth, depth int) error {
	end := 'z'
	switch maxDepth {
	case 1, 2, 3:
		end = 'z' // max 19682 inodes
	case 4:
		end = 'k' // max 19030 inodes
	case 5:
		end = 'e' // max 12438 inodes
	case 6:
		end = 'd' // max 8188 inodes
	case 7, 8:
		end = 'c' // max 16398 inodes
	case 9, 10, 11, 12:
		end = 'b' // max 16378 inodes
	default:
		panic("depth cannot be > 12, would create too many files")
	}

	if depth == maxDepth {
		fd, err := os.Create(filepath.Join(dir, "target"))
		if err != nil {
			return err
		}
		fd.Close()
	}
	for r := 'a'; r <= end; r++ {
		p := filepath.Join(dir, string(r))
		if depth == maxDepth {
			fd, err := os.Create(p)
			if err != nil {
				return err
			}
			fd.Close()
		} else {
			err := os.Mkdir(p, 0755)
			if err != nil {
				return err
			}
			err = mkBenchTree(p, maxDepth, depth+1)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
