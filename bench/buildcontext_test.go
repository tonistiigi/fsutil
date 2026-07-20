package bench

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

const (
	buildContextManyFileDirs          = 100
	buildContextManyFilesPerDir       = 100
	buildContextManyFileMinSizeBytes  = 10 * 1024
	buildContextManyFileMaxSizeBytes  = 2 * 1024 * 1024
	buildContextManyFileMaxTotalBytes = 512 * 1024 * 1024
	buildContextLargeFileSizeBytes    = 512 * 1024 * 1024
)

type buildContextTransfer struct {
	name string
	fn   func(string, string) error
}

func BenchmarkBuildContextTransfer(b *testing.B) {
	transfers := []buildContextTransfer{
		{name: "diffcopy", fn: diffCopyReg},
		{name: "diffcopy_proto", fn: diffCopyProto},
	}

	for _, scenario := range []struct {
		name   string
		create func(testing.TB, string) int64
	}{
		{name: "ManyFiles", create: createManyFilesBuildContext},
		{name: "LargeFile", create: createLargeFileBuildContext},
	} {
		b.Run(scenario.name, func(b *testing.B) {
			src := buildContextTempDir(b, "build-context-src")
			bytes := scenario.create(b, src)

			for _, transfer := range transfers {
				b.Run(transfer.name, func(b *testing.B) {
					benchmarkBuildContextTransfer(b, transfer.fn, src, bytes)
				})
			}
		})
	}
}

func benchmarkBuildContextTransfer(b *testing.B, fn func(string, string) error, src string, bytes int64) {
	b.ReportAllocs()
	b.SetBytes(bytes)

	baseDir := os.Getenv("BENCH_BASE_DIR")
	// StopTimer also pauses allocation counters, so fixture setup and cleanup
	// stay out of ns/op and B/op.
	b.StopTimer()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		dest, err := os.MkdirTemp(baseDir, "build-context-transfer")
		if err != nil {
			b.Fatal(err)
		}

		b.StartTimer()
		err = fn(src, dest)
		b.StopTimer()

		if err != nil {
			os.RemoveAll(dest)
			b.Fatal(err)
		}
		if err := os.RemoveAll(dest); err != nil {
			b.Fatal(err)
		}
	}
}

func buildContextTempDir(tb testing.TB, prefix string) string {
	tb.Helper()

	dir, err := os.MkdirTemp(os.Getenv("BENCH_BASE_DIR"), prefix)
	if err != nil {
		tb.Fatal(err)
	}
	tb.Cleanup(func() {
		os.RemoveAll(dir)
	})
	return dir
}

func createManyFilesBuildContext(tb testing.TB, root string) int64 {
	tb.Helper()

	sizes := buildContextManyFileSizes()
	buf := make([]byte, 1024*1024)
	var total int64
	fileNum := 0
	for dirIndex := 0; dirIndex < buildContextManyFileDirs; dirIndex++ {
		dirName := filepath.Join(root, "payload", fmt.Sprintf("dir-%03d", dirIndex))
		if err := os.MkdirAll(dirName, 0755); err != nil {
			tb.Fatal(err)
		}
		for fileIndex := 0; fileIndex < buildContextManyFilesPerDir; fileIndex++ {
			size := sizes[fileNum]
			fileName := filepath.Join(dirName, fmt.Sprintf("file-%03d.txt", fileIndex))
			writeSizedFile(tb, fileName, size, buf)
			total += size
			fileNum++
		}
	}
	return total
}

func createLargeFileBuildContext(tb testing.TB, root string) int64 {
	tb.Helper()

	writeSizedFile(tb, filepath.Join(root, "blob.bin"), buildContextLargeFileSizeBytes, make([]byte, 1024*1024))
	return buildContextLargeFileSizeBytes
}

func buildContextManyFileSizes() []int64 {
	totalFiles := buildContextManyFileDirs * buildContextManyFilesPerDir
	if int64(totalFiles*buildContextManyFileMinSizeBytes) > buildContextManyFileMaxTotalBytes {
		panic("build context many-file minimum size exceeds total size")
	}

	sizes := make([]int64, 0, totalFiles)
	remainingBudget := int64(buildContextManyFileMaxTotalBytes)
	for fileNum := 0; fileNum < totalFiles; fileNum++ {
		remainingFiles := totalFiles - fileNum
		maxSize := remainingBudget - int64(remainingFiles-1)*buildContextManyFileMinSizeBytes
		if maxSize < int64(buildContextManyFileMinSizeBytes) {
			panic("build context many-file remaining budget below minimum file size")
		}
		maxSize = min(maxSize, int64(buildContextManyFileMaxSizeBytes))

		size := buildContextManyFileSize(fileNum)
		size = min(size, maxSize)
		sizes = append(sizes, size)
		remainingBudget -= size
	}
	if sumInt64(sizes) > buildContextManyFileMaxTotalBytes {
		panic("build context many-file sizes exceed total budget")
	}
	return sizes
}

func buildContextManyFileSize(fileNum int) int64 {
	hash := buildContextManyFileHash(uint64(fileNum))
	switch bucket := hash % 100; {
	case bucket == 0:
		return int64Range(hash>>8, 512*1024, buildContextManyFileMaxSizeBytes)
	case bucket < 9:
		return int64Range(hash>>8, 64*1024, 512*1024)
	default:
		return int64Range(hash>>8, buildContextManyFileMinSizeBytes, 32*1024)
	}
}

// Keep the many-file build-context payload independent from math/rand changes.
func buildContextManyFileHash(v uint64) uint64 {
	v += 0x9e3779b97f4a7c15
	v = (v ^ (v >> 30)) * 0xbf58476d1ce4e5b9
	v = (v ^ (v >> 27)) * 0x94d049bb133111eb
	return v ^ (v >> 31)
}

func int64Range(v uint64, minValue, maxValue int64) int64 {
	if maxValue <= minValue {
		return minValue
	}
	return minValue + int64(v%uint64(maxValue-minValue+1))
}

func sumInt64(values []int64) int64 {
	var total int64
	for _, value := range values {
		total += value
	}
	return total
}

func writeSizedFile(tb testing.TB, name string, size int64, buf []byte) {
	tb.Helper()

	f, err := os.Create(name)
	if err != nil {
		tb.Fatal(err)
	}

	for remaining := size; remaining > 0; {
		n := min(int64(len(buf)), remaining)
		if _, err := f.Write(buf[:n]); err != nil {
			f.Close()
			tb.Fatal(err)
		}
		remaining -= n
	}
	if err := f.Close(); err != nil {
		tb.Fatal(err)
	}
}
