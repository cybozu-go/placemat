package vm

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/cybozu-go/placemat/v2/pkg/types"
	"github.com/cybozu-go/placemat/v2/pkg/util"
	"github.com/cybozu-go/well"
)

// NodeVolume defines the interface for Node volumes.
type NodeVolume interface {
	Create(context.Context, string) (VolumeArgs, error)
}

// NewNodeVolume creates NodeVolume from specs
func NewNodeVolume(spec types.NodeVolumeSpec, imageSpecs []*types.ImageSpec) (NodeVolume, error) {
	var cache string
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
	case types.NodeVolumeFormatRaw:
		var format string
		switch spec.Format {
		case "":
			format = types.NodeVolumeFormatQcow2
		case types.NodeVolumeFormatQcow2, types.NodeVolumeFormatRaw:
			format = spec.Format
		default:
			return nil, errors.New("invalid format for raw volume")
		}
		return newRawVolume(spec.Name, cache, spec.Size, format), nil
	case types.NodeVolumeKindLv:
		return newLVVolume(spec.Name, cache, spec.Size, spec.VG), nil
	case types.NodeVolumeKind9p:
		return new9pVolume(spec.Name, spec.Folder, spec.Writable), nil
	default:
		return nil, errors.New("unknown volume kind: " + spec.Kind)
	}
}

func volumePath(dataDir, name string) string {
	return filepath.Join(dataDir, name+".img")
}

type imageVolume struct {
	name        string
	cache       string
	image       *Image
	copyOnWrite bool
}

// NewImageVolume creates a volume for type "image".
func NewImageVolume(name string, cache string, image *Image, cow bool) NodeVolume {
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

type localDSVolume struct {
	name          string
	cache         string
	userData      string
	networkConfig string
}

// NewLocalDSVolume creates a volume for type "localds".
func newLocalDSVolume(name, cache string, u, n string) NodeVolume {
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

type rawVolume struct {
	name   string
	cache  string
	size   string
	format string
}

// NewRawVolume creates a volume for type "raw".
func newRawVolume(name string, cache string, size, format string) NodeVolume {
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
		err = well.CommandContext(ctx, "qemu-img", "create", "-f", v.format, vPath, v.size).Run()
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

type lvVolume struct {
	name  string
	cache string
	size  string
	vg    string
}

// NewLVVolume creates a volume for type "lv".
func newLVVolume(name string, cache string, size, vg string) NodeVolume {
	return &lvVolume{
		name:  name,
		cache: cache,
		size:  size,
		vg:    vg,
	}
}

func (v *lvVolume) Create(ctx context.Context, dataDir string) (VolumeArgs, error) {
	nodeName := filepath.Base(dataDir)
	lvName := nodeName + "." + v.name

	output, err := well.CommandContext(ctx, "lvs", "--noheadings", "--unbuffered", "-o", "lv_name", v.vg).Output()
	if err != nil {
		return nil, err
	}

	found := false
	for _, line := range strings.Split(string(output), "\n") {
		if strings.TrimSpace(line) == lvName {
			found = true
		}
	}
	if !found {
		err := well.CommandContext(ctx, "lvcreate", "-n", lvName, "-L", v.size, v.vg).Run()
		if err != nil {
			return nil, err
		}
	}

	output, err = well.CommandContext(ctx, "lvs", "--noheadings", "--unbuffered", "-o", "lv_path", v.vg+"/"+lvName).Output()
	if err != nil {
		return nil, err
	}
	vPath := strings.TrimSpace(string(output))

	return &LVVolumeArgs{
		volumePath: vPath,
		cache:      v.cache,
	}, nil
}

type qemu9pVolume struct {
	name     string
	folder   string
	writable bool
}

func new9pVolume(name string, folderName string, writable bool) NodeVolume {
	return &qemu9pVolume{
		name:     name,
		folder:   folderName,
		writable: writable,
	}
}

func (v *qemu9pVolume) Create(ctx context.Context, _ string) (VolumeArgs, error) {
	st, err := os.Stat(v.folder)
	if err != nil {
		return nil, fmt.Errorf("failed to stat the folder %s", v.folder)
	}
	if !st.IsDir() {
		return nil, errors.New(v.folder + " is not a directory")
	}
	p, err := filepath.Abs(v.folder)
	if err != nil {
		return nil, err
	}
	return &Qemu9pVolumeArgs{
		volumePath: p,
		cache:      "",
		writable:   v.writable,
		mountTag:   v.name,
	}, nil
}
