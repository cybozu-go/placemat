package placemat

import (
	"compress/bzip2"
	"compress/gzip"
	"errors"
	"io"
)

// Decompressor defines an interface to decompress data from io.Reader.
type Decompressor interface {
	Decompress(io.Reader) (io.Reader, error)
}

type bzip2Decompressor struct{}

func (d bzip2Decompressor) Decompress(r io.Reader) (io.Reader, error) {
	return bzip2.NewReader(r), nil
}

type gzipDecompressor struct{}

func (d gzipDecompressor) Decompress(r io.Reader) (io.Reader, error) {
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
