package placemat

import (
	"context"
	"errors"
	"os"
	"path/filepath"

	"github.com/cybozu-go/cmd"
)

// NodeVolumeSpec represents a Node's Volume specification in YAML
type NodeVolumeSpec struct {
	Kind          string `yaml:"kind"`
	Name          string `yaml:"name"`
	Image         string `yaml:"image,omitempty"`
	UserData      string `yaml:"user-data,omitempty"`
	NetworkConfig string `yaml:"network-config,omitempty"`
	Size          string `yaml:"size,omitempty"`
	Folder        string `yaml:"folder,omitempty"`
	CopyOnWrite   bool   `yaml:"copy-on-write,omitempty"`
}

// Volume defines the interface for Node volumes.
type NodeVolume interface {
	Kind() string
	Name() string
	Resolve(*Cluster) error
	Create(context.Context, string) ([]string, error)
}

type baseVolume struct {
	name string
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
		"if=virtio,cache=none,aio=native,file=" + p,
	}
}

type imageVolume struct {
	baseVolume
	imageName   string
	image       *Image
	copyOnWrite bool
}

// NewImageVolume creates a volume for type "image".
func NewImageVolume(name string, imageName string, cow bool) *imageVolume {
	return &imageVolume{
		baseVolume:  baseVolume{name: name},
		imageName:   imageName,
		copyOnWrite: cow,
	}
}

func (v imageVolume) Kind() string {
	return "image"
}

func (v *imageVolume) Resolve(c *Cluster) error {
	for _, img := range c.Images {
		if img.Name == v.imageName {
			v.image = img
			return nil
		}
	}
	return errors.New("no such image: " + v.imageName)
}

func (v *imageVolume) Create(ctx context.Context, dataDir string) ([]string, error) {
	p := volumePath(dataDir, v.name)

	_, err := os.Stat(p)
	switch {
	case os.IsNotExist(err):
		if v.image.Spec.File != "" {
			fp, err := filepath.Abs(v.image.Spec.File)
			if err != nil {
				return nil, err
			}
			if v.copyOnWrite {
				err = createCoWImageFromBase(ctx, fp, p)
				if err != nil {
					return nil, err
				}
			} else {
				err = writeToFile(fp, p, v.image.Spec.Decompressor)
				if err != nil {
					return nil, err
				}
			}
		} else {
			err := downloadData(ctx, v.image.Spec.URL, v.image.Spec.Decompressor, v.image.cache)
			if err != nil {
				return nil, err
			}
			if v.copyOnWrite {
				baseImage := v.image.cache.Path(v.image.Spec.URL.String())
				err = createCoWImageFromBase(ctx, baseImage, p)
				if err != nil {
					return nil, err
				}
			} else {
				err = copyDownloadedData(v.image.Spec.URL, p, v.image.cache)
				if err != nil {
					return nil, err
				}
			}
		}
	case err == nil:
	default:
		return nil, err
	}

	return v.qemuArgs(p), nil
}

func createCoWImageFromBase(ctx context.Context, base, dest string) error {
	c := cmd.CommandContext(ctx, "qemu-img", "create", "-f", "qcow2", "-o", "backing_file="+base, dest)
	return c.Run()
}

type localDSVolume struct {
	baseVolume
	userData      string
	networkConfig string
}

// NewLocalDSVolume creates a volume for type "localds".
func NewLocalDSVolume(name string, u, n string) *localDSVolume {
	return &localDSVolume{
		baseVolume:    baseVolume{name: name},
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
			err := cmd.CommandContext(ctx, "cloud-localds", p, v.userData).Run()
			if err != nil {
				return nil, err
			}
		} else {
			err := cmd.CommandContext(ctx, "cloud-localds", p, v.userData, "--network-config", v.networkConfig).Run()
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
func NewRawVolume(name string, size string) *rawVolume {
	return &rawVolume{
		baseVolume: baseVolume{name: name},
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
		err = cmd.CommandContext(ctx, "qemu-img", "create", "-f", "qcow2", p, v.size).Run()
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
func NewVVFATVolume(name string, folderName string) *vvfatVolume {
	return &vvfatVolume{
		baseVolume: baseVolume{name: name},
		folderName: folderName,
	}
}

func (v vvfatVolume) Kind() string {
	return "vvfat"
}

func (v *vvfatVolume) Resolve(c *Cluster) error {
	for _, folder := range c.DataFolders {
		if folder.Name == v.folderName {
			v.folder = folder
			return nil
		}
	}
	return errors.New("no such data folder: " + v.folderName)
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
