package fsutil

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAllocAndBytes(t *testing.T) {
	buf := &buffer{}

	out1 := buf.alloc(10)
	require.Len(t, out1, 10)
	copy(out1, []byte("abcdefghij"))

	out2 := buf.alloc(5)
	require.Len(t, out2, 5)
	copy(out2, []byte("12345"))

	res := &bytes.Buffer{}
	n, err := buf.WriteTo(res)
	require.NoError(t, err)
	require.Equal(t, int64(15), n)
	require.Equal(t, []byte("abcdefghij12345"), res.Bytes())
}

func TestLargeAllocGetsOwnChunk(t *testing.T) {
	buf := &buffer{}

	out := buf.alloc(100000)
	require.Len(t, out, 100000)

	for i := range out {
		out[i] = byte(i % 256)
	}

	res := &bytes.Buffer{}
	n, err := buf.WriteTo(res)
	require.NoError(t, err)
	require.Equal(t, int64(100000), n)
	require.Equal(t, 100000, res.Len())
}

func TestMultipleChunkBoundary(t *testing.T) {
	buf := &buffer{}

	var written []byte
	for i := 0; i < 100; i++ {
		b := buf.alloc(400)
		require.Len(t, b, 400)
		for j := range b {
			b[j] = byte((i + j) % 256)
		}
		written = append(written, b...)
	}

	res := &bytes.Buffer{}
	n, err := buf.WriteTo(res)
	require.NoError(t, err)
	require.Equal(t, int64(40000), n)
	require.Equal(t, 40000, res.Len())
	dt := res.Bytes()
	for i := 0; i < 100; i++ {
		for j := 0; j < 400; j++ {
			require.Equal(t, byte((i+j)%256), dt[i*400+j])
		}
	}
	require.Equal(t, written, dt)
}
