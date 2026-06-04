package bench

import (
	"os"
	"testing"
)

func TestDiffCopyFakeConnEOF(t *testing.T) {
	for name, fn := range map[string]func(string, string) error{
		"packet": diffCopyReg,
		"proto":  diffCopyProto,
	} {
		t.Run(name, func(t *testing.T) {
			src, err := createTestDir(10)
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(src)

			dest := t.TempDir()
			if err := fn(src, dest); err != nil {
				t.Fatal(err)
			}
		})
	}
}
