package placemat

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/cybozu-go/well"
)

// NodeVolumeSpec represents a Node's Volume specification in YAML
type NodeVolumeSpec struct {
	Kind          string `json:"kind"`
	Name          string `json:"name"`
	Image         string `json:"image,omitempty"`
	UserData      string `json:"user-data,omitempty"`
	NetworkConfig string `json:"network-config,omitempty"`
	Size          string `json:"size,omitempty"`
	Folder        string `json:"folder,omitempty"`
	CopyOnWrite   bool   `json:"copy-on-write,omitempty"`
	Cache         string `json:"cache,omitempty"`
}

// NodeVolume defines the interface for Node volumes.
type NodeVolume interface {
	Kind() string
	Name() string
	Resolve(*Cluster) error
	Create(context.Context, string) ([]string, error)
}

type baseVolume struct {
	name  string
	cache string
}

func (v baseVolume) Name() string {
	return v.name
}

func volumePath(dataDir, name string) string {
	return filepath.Join(dataDir, name+".img")
}

func (v baseVolume) qemuArgs(p string) []string {
	return []string{
		"-drive",
		fmt.Sprintf("if=virtio,cache=%s,aio=native,file=%s", v.cache, p),
	}
}

type imageVolume struct {
	baseVolume
	imageName   string
	image       *Image
	copyOnWrite bool
}

// NewImageVolume creates a volume for type "image".
func NewImageVolume(name string, cache string, imageName string, cow bool) NodeVolume {
	return &imageVolume{
		baseVolume:  baseVolume{name: name, cache: cache},
		imageName:   imageName,
		copyOnWrite: cow,
	}
}

func (v imageVolume) Kind() string {
	return "image"
}

func (v *imageVolume) Resolve(c *Cluster) error {
	img, err := c.GetImage(v.imageName)
	if err != nil {
		return err
	}
	v.image = img
	return nil
}

func (v *imageVolume) Create(ctx context.Context, dataDir string) ([]string, error) {
	p := volumePath(dataDir, v.name)
	args := v.qemuArgs(p)

	_, err := os.Stat(p)
	if err == nil {
		return args, nil
	}

	if !os.IsNotExist(err) {
		return nil, err
	}

	if v.image.File != "" {
		fp, err := filepath.Abs(v.image.File)
		if err != nil {
			return nil, err
		}
		if v.copyOnWrite {
			err = createCoWImageFromBase(ctx, fp, p)
			if err != nil {
				return nil, err
			}
		} else {
			err = writeToFile(fp, p, v.image.decomp)
			if err != nil {
				return nil, err
			}
		}
		return args, nil
	}

	baseImage := v.image.Path()
	if v.copyOnWrite {
		err = createCoWImageFromBase(ctx, baseImage, p)
		if err != nil {
			return nil, err
		}
		return args, nil
	}

	f, err := os.Open(baseImage)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	g, err := os.Create(p)
	if err != nil {
		return nil, err
	}
	defer g.Close()

	_, err = io.Copy(g, f)
	if err != nil {
		return nil, err
	}
	return args, nil
}

func createCoWImageFromBase(ctx context.Context, base, dest string) error {
	c := well.CommandContext(ctx, "qemu-img", "create", "-f", "qcow2", "-o", "backing_file="+base, dest)
	return c.Run()
}

type localDSVolume struct {
	baseVolume
	userData      string
	networkConfig string
}

// NewLocalDSVolume creates a volume for type "localds".
func NewLocalDSVolume(name, cache string, u, n string) NodeVolume {
	return &localDSVolume{
		baseVolume:    baseVolume{name: name, cache: cache},
		userData:      u,
		networkConfig: n,
	}
}

func (v localDSVolume) Kind() string {
	return "localds"
}

func (v *localDSVolume) Resolve(c *Cluster) error {
	return nil
}

func (v *localDSVolume) Create(ctx context.Context, dataDir string) ([]string, error) {
	p := volumePath(dataDir, v.name)

	_, err := os.Stat(p)
	switch {
	case os.IsNotExist(err):
		if v.networkConfig == "" {
			err := well.CommandContext(ctx, "cloud-localds", p, v.userData).Run()
			if err != nil {
				return nil, err
			}
		} else {
			err := well.CommandContext(ctx, "cloud-localds", p, v.userData, "--network-config", v.networkConfig).Run()
			if err != nil {
				return nil, err
			}
		}
	case err == nil:
	default:
		return nil, err
	}

	return v.qemuArgs(p), nil
}

type rawVolume struct {
	baseVolume
	size string
}

// NewRawVolume creates a volume for type "raw".
func NewRawVolume(name string, cache string, size string) NodeVolume {
	return &rawVolume{
		baseVolume: baseVolume{name: name, cache: cache},
		size:       size,
	}
}

func (v rawVolume) Kind() string {
	return "raw"
}

func (v *rawVolume) Resolve(c *Cluster) error {
	return nil
}

func (v *rawVolume) Create(ctx context.Context, dataDir string) ([]string, error) {
	p := volumePath(dataDir, v.name)
	_, err := os.Stat(p)
	switch {
	case os.IsNotExist(err):
		err = well.CommandContext(ctx, "qemu-img", "create", "-f", "qcow2", p, v.size).Run()
		if err != nil {
			return nil, err
		}
	case err == nil:
	default:
		return nil, err
	}
	return v.qemuArgs(p), nil
}

type vvfatVolume struct {
	baseVolume
	folderName string
	folder     *DataFolder
}

// NewVVFATVolume creates a volume for type "vvfat".
func NewVVFATVolume(name string, folderName string) NodeVolume {
	return &vvfatVolume{
		baseVolume: baseVolume{name: name},
		folderName: folderName,
	}
}

func (v vvfatVolume) Kind() string {
	return "vvfat"
}

func (v *vvfatVolume) Resolve(c *Cluster) error {
	df, err := c.GetDataFolder(v.folderName)
	if err != nil {
		return err
	}
	v.folder = df
	return nil
}

func (v *vvfatVolume) Create(ctx context.Context, _ string) ([]string, error) {
	return v.qemuArgs(v.folder.Path()), nil
}

func (v vvfatVolume) qemuArgs(p string) []string {
	return []string{
		"-drive",
		"file=fat:16:" + p + ",format=raw,if=virtio",
	}
}
