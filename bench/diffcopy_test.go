package bench

import (
	"os"
	"testing"
)

func TestDiffCopyFakeConnEOF(t *testing.T) {
	for _, tc := range []struct {
		name string
		mode string
		fn   func(string, string) error
	}{
		{name: "packet_default", fn: diffCopyReg},
		{name: "proto_default", fn: diffCopyProto},
		{name: "packet_path", mode: "path", fn: diffCopyReg},
		{name: "proto_path", mode: "path", fn: diffCopyProto},
		{name: "packet_osroot", mode: "osroot", fn: diffCopyReg},
		{name: "proto_osroot", mode: "osroot", fn: diffCopyProto},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("BENCH_FS_MODE", tc.mode)

			src, err := createTestDir(10)
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(src)

			dest := t.TempDir()
			if err := tc.fn(src, dest); err != nil {
				t.Fatal(err)
			}
		})
	}
}
