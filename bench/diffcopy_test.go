package bench

import (
	"os"
	"testing"
)

func TestDiffCopyFakeConnEOF(t *testing.T) {
	for _, tc := range []struct {
		name string
		fn   func(string, string) error
	}{
		{name: "packet_default", fn: diffCopyReg},
		{name: "proto_default", fn: diffCopyProto},
		{name: "packet_path", fn: func(src, dest string) error { return diffCopyPath(false, src, dest) }},
		{name: "proto_path", fn: func(src, dest string) error { return diffCopyPath(true, src, dest) }},
		{name: "packet_osroot", fn: func(src, dest string) error { return diffCopyRoot(false, src, dest) }},
		{name: "proto_osroot", fn: func(src, dest string) error { return diffCopyRoot(true, src, dest) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
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
