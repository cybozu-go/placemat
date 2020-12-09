package vm

import (
	"context"
	"net/url"

	"github.com/cybozu-go/placemat/v2/pkg/types"
	"github.com/cybozu-go/placemat/v2/pkg/util"
)

// Image represents an image configuration
type Image struct {
	name   string
	url    *url.URL
	file   string
	decomp util.Decompressor
	p      string
}

// NewImage creates *Image from spec.
func NewImage(spec *types.ImageSpec) (*Image, error) {
	i := &Image{
		name: spec.Name,
		file: spec.File,
	}

	if len(spec.URL) > 0 {
		u, err := url.Parse(spec.URL)
		if err != nil {
			return nil, err
		}
		i.url = u
	}

	decomp, err := util.NewDecompressor(spec.CompressionMethod)
	if err != nil {
		return nil, err
	}
	i.decomp = decomp

	return i, nil
}

// Prepare downloads the image if it is not in the cache.
func (i *Image) Prepare(ctx context.Context, c *util.Cache) error {
	if i.url == nil {
		return nil
	}
	err := util.DownloadData(ctx, i.url, i.decomp, c)
	if err != nil {
		return err
	}

	i.p = c.Path(i.url.String())
	return nil
}

// Path returns the filesystem path to the image file.
func (i *Image) Path() string {
	return i.p
}
