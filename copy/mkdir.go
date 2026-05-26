package fs

import (
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

// MkdirAll is forked os.MkdirAll
func MkdirAll(path string, perm os.FileMode, user Chowner, tm *time.Time) ([]string, error) {
	fixUmask := needsUmaskFix(perm)
	return mkdirAll(path, perm, user, tm, fixUmask)
}

func mkdirAll(path string, perm os.FileMode, user Chowner, tm *time.Time, fixUmask bool) ([]string, error) {
	// Fast path: if we can tell whether path is a directory or file, stop with success or error.
	dir, err := os.Stat(path)
	if err == nil {
		if dir.IsDir() {
			return nil, nil
		}
		return nil, &os.PathError{Op: "mkdir", Path: path, Err: syscall.ENOTDIR}
	}

	// Slow path: make sure parent exists and then call Mkdir for path.
	i := len(path)
	for i > 0 && os.IsPathSeparator(path[i-1]) { // Skip trailing path separator.
		i--
	}

	j := i
	for j > 0 && !os.IsPathSeparator(path[j-1]) { // Scan backward over element.
		j--
	}

	var createdDirs []string

	if j > 1 {
		// Create parent.
		createdDirs, err = MkdirAll(fixRootDirectory(path[:j-1]), perm, user, tm)
		if err != nil {
			return nil, err
		}
	}

	dir, err1 := os.Lstat(path)
	if err1 == nil && dir.IsDir() {
		return createdDirs, nil
	}

	// Parent now exists; invoke Mkdir and use its result.
	err = os.Mkdir(path, perm)
	if err != nil {
		// Handle arguments like "foo/." by
		// double-checking that directory doesn't exist.
		dir, err1 := os.Lstat(path)
		if err1 == nil && dir.IsDir() {
			return createdDirs, nil
		}
		return nil, err
	}

	// In general, this code should run with umask unset.
	// At the same time, there are certain environments where we rely
	// on this behavior and the umask causes the directory to be created
	// with the wrong mode.
	if fixUmask {
		if err := os.Chmod(path, perm); err != nil {
			return nil, err
		}
	}
	createdDirs = append(createdDirs, path)

	if err := Chown(path, nil, user); err != nil {
		return nil, err
	}

	if err := Utimes(path, tm); err != nil {
		return nil, err
	}

	return createdDirs, nil
}

var (
	systemUmask     os.FileMode
	systemUmaskOnce sync.Once
)

// needsUmaskFix will check if the requested permission would be affected by the umask.
func needsUmaskFix(perm os.FileMode) bool {
	systemUmaskOnce.Do(func() {
		dir, err := os.MkdirTemp("", "fsutil-umask-probe-")
		if err != nil {
			return
		}
		defer os.RemoveAll(dir)

		f, err := os.OpenFile(filepath.Join(dir, "probe"), os.O_CREATE|os.O_RDWR, 0777)
		if err != nil {
			return
		}
		defer f.Close()

		fi, err := f.Stat()
		if err != nil {
			return
		}

		// The bits that were masked out by the system umask are the bits
		// that were requested (0777) but not present in the resulting mode.
		systemUmask = 0777 &^ (fi.Mode() & os.ModePerm)
	})
	return perm&systemUmask != 0
}
