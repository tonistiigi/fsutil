package compression

import (
	"bufio"
	"bytes"
	"compress/bzip2"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"sync"
)

type (
	// Compression is the state represents if compressed or not.
	Compression int
)

const (
	// Uncompressed represents the uncompressed.
	Uncompressed Compression = iota
	// Bzip2 is bzip2 compression algorithm.
	Bzip2
	// Gzip is gzip compression algorithm.
	Gzip
	// Xz is xz compression algorithm.
	Xz
)

var (
	bufioReader32KPool = &sync.Pool{
		New: func() interface{} { return bufio.NewReaderSize(nil, 32*1024) },
	}
)

type readCloserWrapper struct {
	io.ReadCloser
	closer func() error
}

func (r *readCloserWrapper) Close() error {
	err := r.ReadCloser.Close()
	if r.closer != nil {
		if err1 := r.closer(); err == nil {
			return err1
		}
	}
	return err
}

// DetectCompression detects the compression algorithm of the source.
func DetectCompression(source []byte) Compression {
	for compression, m := range map[Compression][]byte{
		Bzip2: {0x42, 0x5A, 0x68},
		Gzip:  {0x1F, 0x8B, 0x08},
		Xz:    {0xFD, 0x37, 0x7A, 0x58, 0x5A, 0x00},
	} {
		if len(source) < len(m) {
			// Len too short
			continue
		}
		if bytes.Equal(m, source[:len(m)]) {
			return compression
		}
	}
	return Uncompressed
}

// DecompressStream decompresses the archive and returns a ReaderCloser with the decompressed archive.
func DecompressStream(archive io.Reader) (io.ReadCloser, error) {
	buf := bufioReader32KPool.Get().(*bufio.Reader)
	buf.Reset(archive)
	bs, err := buf.Peek(10)
	if err != nil && err != io.EOF {
		// Note: we'll ignore any io.EOF error because there are some odd
		// cases where the layer.tar file will be empty (zero bytes) and
		// that results in an io.EOF from the Peek() call. So, in those
		// cases we'll just treat it as a non-compressed stream and
		// that means just create an empty layer.
		// See Issue docker/docker#18170
		return nil, err
	}

	closer := func() error {
		buf.Reset(nil)
		bufioReader32KPool.Put(buf)
		return nil
	}
	switch compression := DetectCompression(bs); compression {
	case Uncompressed:
		readBufWrapper := &readCloserWrapper{ioutil.NopCloser(buf), closer}
		return readBufWrapper, nil
	case Gzip:
		gzReader, err := gzip.NewReader(buf)
		if err != nil {
			return nil, err
		}
		readBufWrapper := &readCloserWrapper{gzReader, closer}
		return readBufWrapper, nil
	case Bzip2:
		bz2Reader := bzip2.NewReader(buf)
		readBufWrapper := &readCloserWrapper{ioutil.NopCloser(bz2Reader), closer}
		return readBufWrapper, nil
	case Xz:
		xzReader, err := xzDecompress(buf)
		if err != nil {
			return nil, err
		}
		readBufWrapper := &readCloserWrapper{xzReader, closer}
		return readBufWrapper, nil
	default:
		return nil, fmt.Errorf("unsupported compression format %s", compression)
	}
}
