//go:build windows

package fs

import (
	"context"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func readUidGid(_ os.FileInfo) (uid, gid int, ok bool) {
	return 0, 0, false
}

// Test folder and file names
const (
	normalFolder     = "normal"
	normalFile       = "file.txt"
	excludedFolder   = "excluded"
	systemVolumeInfo = "System Volume Information"
	wcSandboxState   = "WcSandboxState"
	dirFolder        = "dir"
	includedFile     = "included.txt"
	excludedLogFile  = "excluded.log"
	logsFolder       = "logs"
	debugLogFile     = "debug.log"
	importantLogFile = "important.log"
)

// createLstatTracker creates a test hook that tracks os.Lstat() calls in a thread-safe map.
// Returns the tracking function and the map for verification.
func createLstatTracker() (func(string), map[string]int, *sync.Mutex) {
	var mu sync.Mutex
	lstatCalls := make(map[string]int)

	trackFunc := func(path string) {
		mu.Lock()
		lstatCalls[path]++
		mu.Unlock()
	}

	return trackFunc, lstatCalls, &mu
}

// TestCopyWithExcludeSkipsLstat verifies that excluded paths skip os.Lstat() when safe.
// This uses a test hook to directly verify Lstat is NOT called on excluded paths.
func TestCopyWithExcludeSkipsLstat(t *testing.T) {
	t1 := t.TempDir()
	t2 := t.TempDir()

	// Create test structure
	require.NoError(t, os.MkdirAll(filepath.Join(t1, normalFolder), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(t1, normalFolder, normalFile), []byte("data"), 0644))

	require.NoError(t, os.MkdirAll(filepath.Join(t1, excludedFolder), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(t1, excludedFolder, normalFile), []byte("excluded"), 0644))

	// Track which paths have Lstat called
	testHookLstat, lstatCalls, mu := createLstatTracker()

	// Copy with exclude pattern and test hook
	ci := CopyInfo{
		ExcludePatterns:   []string{excludedFolder},
		XAttrErrorHandler: func(string, string, string, error) error { return nil },
		testHookLstat:     testHookLstat,
	}

	err := Copy(context.Background(), t1, "/", t2, "/", WithCopyInfo(ci))
	require.NoError(t, err)

	// Verify normal folder was copied and Lstat WAS called
	_, err = os.Stat(filepath.Join(t2, normalFolder, normalFile))
	require.NoError(t, err)

	normalPath := filepath.Join(t1, normalFolder)
	mu.Lock()
	normalCalls := lstatCalls[normalPath]
	mu.Unlock()
	require.Greater(t, normalCalls, 0, "Lstat should be called on normal folder")

	// Verify excluded folder was NOT copied and Lstat was NOT called
	_, err = os.Stat(filepath.Join(t2, excludedFolder))
	require.True(t, os.IsNotExist(err))

	excludedPath := filepath.Join(t1, excludedFolder)
	mu.Lock()
	excludedCalls := lstatCalls[excludedPath]
	mu.Unlock()
	require.Equal(t, 0, excludedCalls, "Lstat should NOT be called on excluded folder (optimization working)")
}

// TestCopyMultipleExcludesSkipLstat verifies multiple exclude patterns work correctly.
// This simulates the real scenario of excluding multiple Windows protected folders.
func TestCopyMultipleExcludesSkipLstat(t *testing.T) {
	t1 := t.TempDir()
	t2 := t.TempDir()

	// Create structure mimicking container mount with protected folders
	require.NoError(t, os.MkdirAll(filepath.Join(t1, normalFolder), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(t1, normalFolder, normalFile), []byte("data"), 0644))

	require.NoError(t, os.MkdirAll(filepath.Join(t1, systemVolumeInfo), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(t1, systemVolumeInfo, normalFile), []byte("protected1"), 0644))

	require.NoError(t, os.MkdirAll(filepath.Join(t1, wcSandboxState), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(t1, wcSandboxState, normalFile), []byte("protected2"), 0644))

	// Track which paths have Lstat called
	testHookLstat, lstatCalls, mu := createLstatTracker()

	// Copy excluding both protected folders with test hook
	ci := CopyInfo{
		ExcludePatterns:   []string{systemVolumeInfo, wcSandboxState},
		XAttrErrorHandler: func(string, string, string, error) error { return nil },
		testHookLstat:     testHookLstat,
	}

	err := Copy(context.Background(), t1, "/", t2, "/", WithCopyInfo(ci))
	require.NoError(t, err)

	// Verify normal folder was copied and Lstat WAS called
	_, err = os.Stat(filepath.Join(t2, normalFolder, normalFile))
	require.NoError(t, err)

	normalPath := filepath.Join(t1, normalFolder)
	mu.Lock()
	normalCalls := lstatCalls[normalPath]
	mu.Unlock()
	require.Greater(t, normalCalls, 0, "Lstat should be called on normal folder")

	// Verify both protected folders were NOT copied and Lstat was NOT called
	_, err = os.Stat(filepath.Join(t2, systemVolumeInfo))
	require.True(t, os.IsNotExist(err))

	sviPath := filepath.Join(t1, systemVolumeInfo)
	mu.Lock()
	sviCalls := lstatCalls[sviPath]
	mu.Unlock()
	require.Equal(t, 0, sviCalls, "Lstat should NOT be called on System Volume Information (optimization working)")

	_, err = os.Stat(filepath.Join(t2, wcSandboxState))
	require.True(t, os.IsNotExist(err))

	wcsPath := filepath.Join(t1, wcSandboxState)
	mu.Lock()
	wcsCalls := lstatCalls[wcsPath]
	mu.Unlock()
	require.Equal(t, 0, wcsCalls, "Lstat should NOT be called on WcSandboxState (optimization working)")
}

// TestCopyWithIncludeAndExcludeMustCallLstat tests that when BOTH include and exclude
// patterns are used, os.Lstat() MUST be called even for excluded paths.
// This is necessary because include patterns might match files inside excluded directories.
func TestCopyWithIncludeAndExcludeMustCallLstat(t *testing.T) {
	t1 := t.TempDir()
	t2 := t.TempDir()

	// Create directory structure
	require.NoError(t, os.MkdirAll(filepath.Join(t1, dirFolder), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(t1, dirFolder, includedFile), []byte("include me"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(t1, dirFolder, excludedLogFile), []byte("exclude me"), 0644))

	// Track which paths have Lstat called
	testHookLstat, lstatCalls, mu := createLstatTracker()

	// Copy with BOTH include and exclude patterns with test hook
	// Use **/*.txt to match files in subdirectories
	ci := CopyInfo{
		IncludePatterns:   []string{"**/*.txt"},
		ExcludePatterns:   []string{"**/*.log"},
		XAttrErrorHandler: func(string, string, string, error) error { return nil },
		testHookLstat:     testHookLstat,
	}

	err := Copy(context.Background(), t1, "/", t2, "/", WithCopyInfo(ci))
	require.NoError(t, err)

	// Verify included.txt was copied
	_, err = os.Stat(filepath.Join(t2, dirFolder, includedFile))
	require.NoError(t, err, "included.txt should be copied")

	// Verify excluded.log was NOT copied
	_, err = os.Stat(filepath.Join(t2, dirFolder, excludedLogFile))
	require.True(t, os.IsNotExist(err), "excluded.log should not be copied")

	// When include patterns exist, Lstat MUST be called even for excluded paths
	// to properly evaluate the pattern matching
	dirPath := filepath.Join(t1, dirFolder)
	mu.Lock()
	dirCalls := lstatCalls[dirPath]
	mu.Unlock()
	require.Greater(t, dirCalls, 0, "Lstat MUST be called when include patterns exist")
}

// TestCopyWithNegationPatternsMustCallLstat tests that when exclude patterns
// contain negations (!pattern), os.Lstat() MUST be called.
// Negations create exceptions to exclusions, requiring file stats to evaluate properly.
func TestCopyWithNegationPatternsMustCallLstat(t *testing.T) {
	t1 := t.TempDir()
	t2 := t.TempDir()

	// Create directory structure
	require.NoError(t, os.MkdirAll(filepath.Join(t1, logsFolder), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(t1, logsFolder, debugLogFile), []byte("debug"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(t1, logsFolder, importantLogFile), []byte("important"), 0644))

	// Track which paths have Lstat called
	testHookLstat, lstatCalls, mu := createLstatTracker()

	// Copy with negation pattern: exclude all logs EXCEPT important.log
	ci := CopyInfo{
		ExcludePatterns:   []string{"logs/*.log", "!logs/important.log"},
		XAttrErrorHandler: func(string, string, string, error) error { return nil },
		testHookLstat:     testHookLstat,
	}

	err := Copy(context.Background(), t1, "/", t2, "/", WithCopyInfo(ci))
	require.NoError(t, err)

	// Verify important.log was copied (negation exception)
	_, err = os.Stat(filepath.Join(t2, logsFolder, importantLogFile))
	require.NoError(t, err, "important.log should be copied due to negation")

	// Verify debug.log was NOT copied
	_, err = os.Stat(filepath.Join(t2, logsFolder, debugLogFile))
	require.True(t, os.IsNotExist(err), "debug.log should be excluded")

	// When negation patterns exist, Lstat MUST be called to evaluate exceptions
	logsPath := filepath.Join(t1, logsFolder)
	mu.Lock()
	logsCalls := lstatCalls[logsPath]
	mu.Unlock()
	require.Greater(t, logsCalls, 0, "Lstat MUST be called when negation patterns exist")
}

// TestCopyFromRootWithProtectedFolders simulates copying from a root directory
// containing protected folders, which is the real-world buildkit scenario.
func TestCopyFromRootWithProtectedFolders(t *testing.T) {
	t1 := t.TempDir()
	t2 := t.TempDir()

	// Create a structure that simulates a Windows root with protected folders
	require.NoError(t, os.MkdirAll(filepath.Join(t1, normalFolder), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(t1, normalFolder, normalFile), []byte("data"), 0644))

	require.NoError(t, os.MkdirAll(filepath.Join(t1, systemVolumeInfo), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(t1, systemVolumeInfo, normalFile), []byte("protected1"), 0644))

	require.NoError(t, os.MkdirAll(filepath.Join(t1, wcSandboxState), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(t1, wcSandboxState, normalFile), []byte("protected2"), 0644))

	// Track operations
	testHookLstat, lstatCalls, mu := createLstatTracker()

	// Copy from root "/" with exclude patterns (like buildkit does)
	ci := CopyInfo{
		ExcludePatterns:   []string{systemVolumeInfo, wcSandboxState},
		XAttrErrorHandler: func(string, string, string, error) error { return nil },
		testHookLstat:     testHookLstat,
	}

	err := Copy(context.Background(), t1, "/", t2, "/", WithCopyInfo(ci))
	require.NoError(t, err)

	// Verify normal folder was copied
	_, err = os.Stat(filepath.Join(t2, normalFolder, normalFile))
	require.NoError(t, err, "normal folder should be copied")

	// Verify protected folders were NOT copied
	_, err = os.Stat(filepath.Join(t2, systemVolumeInfo))
	require.True(t, os.IsNotExist(err), "System Volume Information should not be copied")

	_, err = os.Stat(filepath.Join(t2, wcSandboxState))
	require.True(t, os.IsNotExist(err), "WcSandboxState should not be copied")

	// Check the operations log
	mu.Lock()
	defer mu.Unlock()

	// Verify Lstat was NOT called for protected folders
	sviPath := filepath.Join(t1, systemVolumeInfo)
	wcsPath := filepath.Join(t1, wcSandboxState)

	require.NotContains(t, lstatCalls, sviPath, "Lstat should NOT be called on System Volume Information")
	require.NotContains(t, lstatCalls, wcsPath, "Lstat should NOT be called on WcSandboxState")
}
