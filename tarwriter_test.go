package fsutil

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"github.com/containerd/continuity/fs/fstest"
	"github.com/stretchr/testify/require"
	"github.com/tonistiigi/fsutil/types"
)

func TestWriteTarSubDirFSRoot(t *testing.T) {
	tmpDir := t.TempDir()
	apply := fstest.Apply(
		fstest.CreateFile("payload.txt", []byte("payload"), 0600),
	)
	require.NoError(t, apply.Apply(tmpDir))

	tmpFS, err := NewFS(tmpDir)
	require.NoError(t, err)

	f, err := SubDirFS([]Dir{
		{
			Stat: &types.Stat{
				Mode: uint32(os.ModeDir | 0755),
				Path: ".._buildkit-outside",
			},
			FS: tmpFS,
		},
	})
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, WriteTar(context.TODO(), f, &buf))

	tr := tar.NewReader(bytes.NewReader(buf.Bytes()))
	expected := ".._buildkit-outside/payload.txt"
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)

		// The tar can contain directory entries and unrelated files; this test
		// only verifies that the payload is exported under the sanitized root.
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		if hdr.Name != expected {
			continue
		}
		dt, err := io.ReadAll(tr)
		require.NoError(t, err)
		require.Equal(t, []byte("payload"), dt)
		return
	}

	require.Failf(t, "expected payload", "expected payload under %q", expected)
}

func TestWriteTarClosesFileOnCopyError(t *testing.T) {
	tmpDir := t.TempDir()
	require.NoError(t, fstest.CreateFile("payload.txt", []byte("payload"), 0600).Apply(tmpDir))
	f, err := NewFS(tmpDir)
	require.NoError(t, err)
	tracked := &tarTrackingFS{FS: f}
	t.Cleanup(func() {
		if tracked.file != nil {
			tracked.file.Close()
		}
	})

	writeErr := errors.New("archive write failed")
	w := tarWriterFunc(func(p []byte) (int, error) {
		if tracked.file != nil {
			return 0, writeErr
		}
		return len(p), nil
	})
	require.ErrorIs(t, WriteTar(context.Background(), tracked, w), writeErr)
	require.NotNil(t, tracked.file)
	_, err = tracked.file.Read(make([]byte, 1))
	require.ErrorIs(t, err, os.ErrClosed)
}

func TestWriteTarKeepsSubSecondModTime(t *testing.T) {
	tmpDir := t.TempDir()
	apply := fstest.Apply(
		fstest.CreateFile("fraction.txt", []byte("fraction"), 0600),
		fstest.CreateFile("whole.txt", []byte("whole"), 0600),
	)
	require.NoError(t, apply.Apply(tmpDir))

	// The stat times are set here, so the test does not depend on the
	// timestamp precision of the host filesystem.
	modTimes := map[string]time.Time{
		"fraction.txt": time.Unix(1704164645, 987654321),
		"whole.txt":    time.Unix(1704164645, 0),
	}
	f, err := NewFS(tmpDir)
	require.NoError(t, err)
	f, err = NewFilterFS(f, &FilterOpt{
		Map: func(path string, st *types.Stat) MapResult {
			if tm, ok := modTimes[path]; ok {
				st.ModTime = tm.UnixNano()
			}
			return MapResultKeep
		},
	})
	require.NoError(t, err)

	var buf bytes.Buffer
	require.NoError(t, WriteTar(context.TODO(), f, &buf))

	formats := map[string]tar.Format{}
	tr := tar.NewReader(&buf)
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		require.NoError(t, err)
		tm, ok := modTimes[hdr.Name]
		if !ok {
			continue
		}
		require.True(t, tm.Equal(hdr.ModTime), "%s: got %s, want %s", hdr.Name, hdr.ModTime, tm)
		formats[hdr.Name] = hdr.Format
	}

	// Only the sub-second time needs a PAX record. The other header has none.
	require.Equal(t, map[string]tar.Format{
		"fraction.txt": tar.FormatPAX,
		"whole.txt":    tar.FormatUSTAR,
	}, formats)
}

type tarTrackingFS struct {
	FS
	file io.ReadCloser
}

func (f *tarTrackingFS) Open(path string) (io.ReadCloser, error) {
	rc, err := f.FS.Open(path)
	f.file = rc
	return rc, err
}

type tarWriterFunc func([]byte) (int, error)

func (f tarWriterFunc) Write(p []byte) (int, error) {
	return f(p)
}
