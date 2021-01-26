package vm

import (
	"context"
	"net/url"

	"github.com/cybozu-go/placemat/v2/pkg/types"
	"github.com/cybozu-go/placemat/v2/pkg/util"
)

type image struct {
	name   string
	url    *url.URL
	file   string
	decomp util.Decompressor
	p      string
}

func newImage(spec *types.ImageSpec) (*image, error) {
	i := &image{
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

func (i *image) prepare(ctx context.Context, c *util.Cache) error {
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

func (i *image) path() string {
	return i.p
}
