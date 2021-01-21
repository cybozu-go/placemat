package vm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/cybozu-go/placemat/v2/pkg/types"
	"github.com/cybozu-go/placemat/v2/pkg/util"
	"github.com/cybozu-go/well"
)

// NodeVolume defines the interface for Node volumes.
type NodeVolume interface {
	Create(context.Context, string) (VolumeArgs, error)
	Prepare(ctx context.Context, c *util.Cache) error
}

// NewNodeVolume creates NodeVolume from specs
func NewNodeVolume(spec types.NodeVolumeSpec, imageSpecs []*types.ImageSpec) (NodeVolume, error) {
	var cache types.NodeVolumeCache
	switch spec.Cache {
	case "":
		cache = types.NodeVolumeCacheNone
	case types.NodeVolumeCacheWriteback, types.NodeVolumeCacheNone, types.NodeVolumeCacheWritethrough, types.NodeVolumeCacheDirectSync, types.NodeVolumeCacheUnsafe:
		cache = spec.Cache
	default:
		return nil, errors.New("invalid cache type for volume")
	}

	switch spec.Kind {
	case types.NodeVolumeKindImage:
		for _, imageSpec := range imageSpecs {
			if spec.Image == imageSpec.Name {
				image, err := NewImage(imageSpec)
				if err != nil {
					return nil, fmt.Errorf("failed to create the image %s: %w", imageSpec.Name, err)
				}
				return NewImageVolume(spec.Name, cache, image, spec.CopyOnWrite), nil
			}
		}
		return nil, fmt.Errorf("failed to find the image %s", spec.Image)
	case types.NodeVolumeKindLocalds:
		return newLocalDSVolume(spec.Name, cache, spec.UserData, spec.NetworkConfig), nil
	case types.NodeVolumeKindRaw:
		var format types.NodeVolumeFormat
		switch spec.Format {
		case "":
			format = types.NodeVolumeFormatQcow2
		case types.NodeVolumeFormatQcow2, types.NodeVolumeFormatRaw:
			format = spec.Format
		default:
			return nil, errors.New("invalid format for raw volume")
		}
		return newRawVolume(spec.Name, cache, spec.Size, format), nil
	case types.NodeVolumeKindHostPath:
		return newHosPathVolume(spec.Name, spec.Path, spec.Writable), nil
	default:
		return nil, errors.New("unknown volume kind: " + string(spec.Kind))
	}
}

func volumePath(dataDir, name string) string {
	return filepath.Join(dataDir, name+".img")
}

type imageVolume struct {
	name        string
	cache       types.NodeVolumeCache
	image       *Image
	copyOnWrite bool
}

// NewImageVolume creates a volume for type "image".
func NewImageVolume(name string, cache types.NodeVolumeCache, image *Image, cow bool) NodeVolume {
	return &imageVolume{
		name:        name,
		cache:       cache,
		image:       image,
		copyOnWrite: cow,
	}
}

func (v *imageVolume) Create(ctx context.Context, dataDir string) (VolumeArgs, error) {
	vPath := volumePath(dataDir, v.name)
	_, err := os.Stat(vPath)
	if err == nil {
		return &ImageVolumeArgs{
			volumePath: vPath,
			cache:      v.cache,
		}, nil
	}

	if !os.IsNotExist(err) {
		return nil, err
	}

	if v.image.file != "" {
		fp, err := filepath.Abs(v.image.file)
		if err != nil {
			return nil, err
		}
		if v.copyOnWrite {
			err = createCoWImageFromBase(ctx, fp, vPath)
			if err != nil {
				return nil, err
			}
		} else {
			err = util.WriteToFile(fp, vPath, v.image.decomp)
			if err != nil {
				return nil, err
			}
		}
		return &ImageVolumeArgs{
			volumePath: vPath,
			cache:      v.cache,
		}, nil
	}

	baseImage := v.image.Path()
	if v.copyOnWrite {
		err = createCoWImageFromBase(ctx, baseImage, vPath)
		if err != nil {
			return nil, err
		}
		return &ImageVolumeArgs{
			volumePath: vPath,
			cache:      v.cache,
		}, nil
	}

	f, err := os.Open(baseImage)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	g, err := os.Create(vPath)
	if err != nil {
		return nil, err
	}
	defer g.Close()

	_, err = io.Copy(g, f)
	if err != nil {
		return nil, err
	}

	return &ImageVolumeArgs{
		volumePath: vPath,
		cache:      v.cache,
	}, nil
}

func createCoWImageFromBase(ctx context.Context, base, dest string) error {
	c := well.CommandContext(ctx, "qemu-img", "create", "-f", "qcow2", "-o", "backing_file="+base, dest)
	return c.Run()
}

func (v *imageVolume) Prepare(ctx context.Context, c *util.Cache) error {
	return v.image.Prepare(ctx, c)
}

type localDSVolume struct {
	name          string
	cache         types.NodeVolumeCache
	userData      string
	networkConfig string
}

// NewLocalDSVolume creates a volume for type "localds".
func newLocalDSVolume(name string, cache types.NodeVolumeCache, u, n string) NodeVolume {
	return &localDSVolume{
		name:          name,
		cache:         cache,
		userData:      u,
		networkConfig: n,
	}
}

func (v *localDSVolume) Create(ctx context.Context, dataDir string) (VolumeArgs, error) {
	vPath := volumePath(dataDir, v.name)

	_, err := os.Stat(vPath)
	switch {
	case os.IsNotExist(err):
		if v.networkConfig == "" {
			err := well.CommandContext(ctx, "cloud-localds", vPath, v.userData).Run()
			if err != nil {
				return nil, err
			}
		} else {
			err := well.CommandContext(ctx, "cloud-localds", vPath, v.userData, "--network-config", v.networkConfig).Run()
			if err != nil {
				return nil, err
			}
		}
	case err == nil:
	default:
		return nil, err
	}

	return &LocalDSVolumeArgs{
		volumePath: vPath,
		cache:      v.cache,
	}, nil
}

func (v *localDSVolume) Prepare(ctx context.Context, c *util.Cache) error {
	return nil
}

type rawVolume struct {
	name   string
	cache  types.NodeVolumeCache
	size   string
	format types.NodeVolumeFormat
}

// NewRawVolume creates a volume for type "raw".
func newRawVolume(name string, cache types.NodeVolumeCache, size string, format types.NodeVolumeFormat) NodeVolume {
	return &rawVolume{
		name:   name,
		cache:  cache,
		size:   size,
		format: format,
	}
}

func (v *rawVolume) Create(ctx context.Context, dataDir string) (VolumeArgs, error) {
	vPath := volumePath(dataDir, v.name)
	_, err := os.Stat(vPath)
	switch {
	case os.IsNotExist(err):
		err = well.CommandContext(ctx, "qemu-img", "create", "-f", string(v.format), vPath, v.size).Run()
		if err != nil {
			return nil, err
		}
	case err == nil:
	default:
		return nil, err
	}

	return &RawVolumeArgs{
		volumePath: vPath,
		cache:      v.cache,
		format:     v.format,
	}, nil
}

func (v *rawVolume) Prepare(ctx context.Context, c *util.Cache) error {
	return nil
}

type hostPathVolume struct {
	name     string
	path     string
	writable bool
}

func newHosPathVolume(name string, path string, writable bool) NodeVolume {
	return &hostPathVolume{
		name:     name,
		path:     path,
		writable: writable,
	}
}

func (v *hostPathVolume) Create(ctx context.Context, _ string) (VolumeArgs, error) {
	st, err := os.Stat(v.path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat the path %s: %w", v.path, err)
	}
	if !st.IsDir() {
		return nil, errors.New(v.path + " is not a directory")
	}
	p, err := filepath.Abs(v.path)
	if err != nil {
		return nil, err
	}
	return &hostPathVolumeArgs{
		volumePath: p,
		cache:      "",
		writable:   v.writable,
		mountTag:   v.name,
	}, nil
}

func (v *hostPathVolume) Prepare(ctx context.Context, c *util.Cache) error {
	return nil
}
