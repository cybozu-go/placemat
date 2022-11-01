package vm

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/cybozu-go/placemat/v2/pkg/types"
	"github.com/cybozu-go/placemat/v2/pkg/util"
	"github.com/cybozu-go/well"
)

type nodeVolume interface {
	create(context.Context, string, string) (volumeArgs, error)
	prepare(ctx context.Context, c *util.Cache) error
}

func newNodeVolume(spec types.NodeVolumeSpec, imageSpecs []*types.ImageSpec, deviceClassSpecs []*types.DeviceClassSpec) (nodeVolume, error) {
	var cache types.NodeVolumeCache
	switch spec.Cache {
	case "":
		cache = types.NodeVolumeCacheNone
	case types.NodeVolumeCacheWriteback, types.NodeVolumeCacheNone, types.NodeVolumeCacheWritethrough, types.NodeVolumeCacheDirectSync, types.NodeVolumeCacheUnsafe:
		cache = spec.Cache
	default:
		return nil, errors.New("invalid cache type for volume")
	}

	deviceClassDir := ""
	for _, deviceClassSpec := range deviceClassSpecs {
		if spec.DeviceClass == deviceClassSpec.Name {
			deviceClassDir = deviceClassSpec.Path
		}
	}
	if spec.DeviceClass != "" && deviceClassDir == "" {
		return nil, fmt.Errorf("invalid device-class %s", spec.DeviceClass)
	}

	switch spec.Kind {
	case types.NodeVolumeKindImage:
		for _, imageSpec := range imageSpecs {
			if spec.Image == imageSpec.Name {
				image, err := newImage(imageSpec)
				if err != nil {
					return nil, fmt.Errorf("failed to create the image %s: %w", imageSpec.Name, err)
				}
				return newImageVolume(spec.Name, cache, image, spec.CopyOnWrite, deviceClassDir), nil
			}
		}
		return nil, fmt.Errorf("failed to find the image %s", spec.Image)
	case types.NodeVolumeKindLocalds:
		return newLocalDSVolume(spec.Name, cache, spec.UserData, spec.NetworkConfig, deviceClassDir), nil
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
		return newRawVolume(spec.Name, cache, spec.Size, format, deviceClassDir), nil
	case types.NodeVolumeKindHostPath:
		return newHosPathVolume(spec.Name, spec.Path, spec.Writable), nil
	default:
		return nil, errors.New("unknown volume kind: " + string(spec.Kind))
	}
}

func volumePath(dataDir, name string) string {
	return filepath.Join(dataDir, name+".img")
}

func makeVolumeDir(dataDir, deviceClassDir, dataPathLastPart, name string) (string, error) {
	var volumePathFull string
	if deviceClassDir == "" {
		volumePathFull = filepath.Join(dataDir, dataPathLastPart)
	} else {
		volumePathFull = filepath.Join(deviceClassDir, dataPathLastPart)
	}
	if err := os.MkdirAll(volumePathFull, 0755); err != nil {
		return "", fmt.Errorf("failed to make the directory %s: %w", volumePathFull, err)
	}

	vPath := volumePath(volumePathFull, name)
	return vPath, nil
}

type imageVolume struct {
	name           string
	cache          types.NodeVolumeCache
	image          *image
	copyOnWrite    bool
	deviceClassDir string
}

func newImageVolume(name string, cache types.NodeVolumeCache, image *image, cow bool, deviceClassDir string) nodeVolume {
	return &imageVolume{
		name:           name,
		cache:          cache,
		image:          image,
		copyOnWrite:    cow,
		deviceClassDir: deviceClassDir,
	}
}

func (v *imageVolume) create(ctx context.Context, dataDir, dataPathLastPart string) (volumeArgs, error) {
	vPath, err := makeVolumeDir(dataDir, v.deviceClassDir, dataPathLastPart, v.name)
	if err != nil {
		return nil, err
	}

	_, err = os.Stat(vPath)
	if err == nil {
		return &imageVolumeArgs{
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
		return &imageVolumeArgs{
			volumePath: vPath,
			cache:      v.cache,
		}, nil
	}

	baseImage := v.image.path()
	if v.copyOnWrite {
		err = createCoWImageFromBase(ctx, baseImage, vPath)
		if err != nil {
			return nil, err
		}
		return &imageVolumeArgs{
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

	return &imageVolumeArgs{
		volumePath: vPath,
		cache:      v.cache,
	}, nil
}

func createCoWImageFromBase(ctx context.Context, base, dest string) error {
	var info map[string]interface{}
	out, err := well.CommandContext(ctx, "qemu-img", "info", "--output=json", base).Output()
	if err != nil {
		return err
	}

	if err := json.Unmarshal(out, &info); err != nil {
		return err
	}

	fileFormat := fmt.Sprintf("%v", info["format"])
	c := well.CommandContext(ctx, "qemu-img", "create", "-f", "qcow2", "-F", fileFormat, "-o", "backing_file="+base, dest)
	return c.Run()
}

func (v *imageVolume) prepare(ctx context.Context, c *util.Cache) error {
	return v.image.prepare(ctx, c)
}

type localDSVolume struct {
	name           string
	cache          types.NodeVolumeCache
	userData       string
	networkConfig  string
	deviceClassDir string
}

func newLocalDSVolume(name string, cache types.NodeVolumeCache, u, n, deviceClassDir string) nodeVolume {
	return &localDSVolume{
		name:           name,
		cache:          cache,
		userData:       u,
		networkConfig:  n,
		deviceClassDir: deviceClassDir,
	}
}

func (v *localDSVolume) create(ctx context.Context, dataDir, dataPathLastPart string) (volumeArgs, error) {
	vPath, err := makeVolumeDir(dataDir, v.deviceClassDir, dataPathLastPart, v.name)
	if err != nil {
		return nil, err
	}

	_, err = os.Stat(vPath)
	switch {
	case os.IsNotExist(err):
		if v.networkConfig == "" {
			err := well.CommandContext(ctx, "cloud-localds", vPath, v.userData).Run()
			if err != nil {
				return nil, err
			}
		} else {
			err := well.CommandContext(ctx, "cloud-localds", vPath, v.userData, "--network-config", v.networkConfig, "--disk-format", "qcow2").Run()
			if err != nil {
				return nil, err
			}
		}
	case err == nil:
	default:
		return nil, err
	}

	return &localDSVolumeArgs{
		volumePath: vPath,
		cache:      v.cache,
	}, nil
}

func (v *localDSVolume) prepare(ctx context.Context, c *util.Cache) error {
	return nil
}

type rawVolume struct {
	name           string
	cache          types.NodeVolumeCache
	size           string
	format         types.NodeVolumeFormat
	deviceClassDir string
}

func newRawVolume(name string, cache types.NodeVolumeCache, size string, format types.NodeVolumeFormat, deviceClassDir string) nodeVolume {
	return &rawVolume{
		name:           name,
		cache:          cache,
		size:           size,
		format:         format,
		deviceClassDir: deviceClassDir,
	}
}

func (v *rawVolume) create(ctx context.Context, dataDir, dataPathLastPart string) (volumeArgs, error) {
	vPath, err := makeVolumeDir(dataDir, v.deviceClassDir, dataPathLastPart, v.name)
	if err != nil {
		return nil, err
	}

	_, err = os.Stat(vPath)
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

	return &rawVolumeArgs{
		volumePath: vPath,
		cache:      v.cache,
		format:     v.format,
	}, nil
}

func (v *rawVolume) prepare(ctx context.Context, c *util.Cache) error {
	return nil
}

type hostPathVolume struct {
	name     string
	path     string
	writable bool
}

func newHosPathVolume(name string, path string, writable bool) nodeVolume {
	return &hostPathVolume{
		name:     name,
		path:     path,
		writable: writable,
	}
}

func (v *hostPathVolume) create(ctx context.Context, _, _ string) (volumeArgs, error) {
	p, err := filepath.Abs(v.path)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(p, 0755); err != nil {
		return nil, fmt.Errorf("failed to mkdir %s: %w", v.path, err)
	}

	return &hostPathVolumeArgs{
		volumePath: p,
		cache:      "",
		writable:   v.writable,
		mountTag:   v.name,
	}, nil
}

func (v *hostPathVolume) prepare(ctx context.Context, c *util.Cache) error {
	return nil
}
