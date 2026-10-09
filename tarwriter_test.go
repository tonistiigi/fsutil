package fsutil

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"testing"

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
