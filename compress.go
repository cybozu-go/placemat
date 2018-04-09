package placemat

import (
	"compress/bzip2"
	"compress/gzip"
	"io"

	"github.com/pkg/errors"
)

// Decompressor defines an interface to decompress data from io.Reader.
type Decompressor interface {
	Decompress(closer io.ReadCloser) (io.ReadCloser, error)
}

type bzip2Decompressor struct {
	io.Reader
}

func (d *bzip2Decompressor) Read(p []byte) (n int, err error) {
	return d.Read(p)
}
func (d *bzip2Decompressor) Close() error {
	return nil
}

func (d bzip2Decompressor) Decompress(r io.ReadCloser) (io.ReadCloser, error) {
	return &bzip2Decompressor{Reader: bzip2.NewReader(r)}, nil
}

type gzipDecompressor struct{}

func (d gzipDecompressor) Decompress(r io.ReadCloser) (io.ReadCloser, error) {
	return gzip.NewReader(r)
}

// NewDecompressor returns a Decompressor for "format".
// If format is not supported, this returns a non-nil error.
func NewDecompressor(format string) (Decompressor, error) {
	switch format {
	case "bzip2":
		return bzip2Decompressor{}, nil
	case "gzip":
		return gzipDecompressor{}, nil
	case "":
		return nil, nil
	}

	return nil, errors.New("unsupported compression format: " + format)
}
