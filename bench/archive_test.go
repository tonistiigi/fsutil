package bench

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/containerd/continuity"
)

func TestCopyWithTar(t *testing.T) {
	src, err := createTestDir(10)
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(src)

	ctx, err := continuity.NewContext(src)
	if err != nil {
		t.Fatal(err)
	}
	m, err := continuity.BuildManifest(ctx)
	if err != nil {
		t.Fatal(err)
	}

	dest := t.TempDir()
	if err := copyWithTar(src, dest); err != nil {
		t.Fatal(err)
	}

	ctx, err = continuity.NewContext(dest)
	if err != nil {
		t.Fatal(err)
	}
	if err := continuity.VerifyManifest(ctx, m); err != nil {
		t.Fatal(err)
	}
}

func TestCopyWithTarOverwrite(t *testing.T) {
	src, dest := t.TempDir(), t.TempDir()
	for _, content := range []string{"original longer content", "short"} {
		if err := os.WriteFile(filepath.Join(src, "file"), []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		if err := copyWithTar(src, dest); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(filepath.Join(dest, "file"))
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != content {
			t.Fatalf("copied content = %q, want %q", got, content)
		}
	}
}

func TestCopyWithTarExtractionError(t *testing.T) {
	src, dest := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(src, "file"), make([]byte, 64*1024), 0600); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dest, "file")
	if err := os.Mkdir(target, 0700); err != nil {
		t.Fatal(err)
	}
	err := copyWithTar(src, dest)
	var pathErr *os.PathError
	if !errors.As(err, &pathErr) || pathErr.Path != target || pathErr.Op != "open" {
		t.Fatalf("expected destination open error, got %v", err)
	}
}
