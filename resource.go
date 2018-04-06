package placemat

import (
	"net/url"

	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/cybozu-go/cmd"
	"github.com/cybozu-go/log"
	"github.com/pkg/errors"
)

// VolumeRecreatePolicy represents a policy to recreate a volume
type VolumeRecreatePolicy int

// Common recreate policies.  The default recreate policy is
// RecreateIfNotPresent which causes Placemat to skip creating an image if it
// already exists RecreateAlways causes Placemat to create always create
// an image even if the image is exists.  QEMU will be failed if no images
// exist and RecreateNever is specified.
const (
	RecreateIfNotPresent VolumeRecreatePolicy = iota
	RecreateAlways
	RecreateNever
)

// BIOSMode represents a bios mode
type BIOSMode int

// BIOS mode, For LegacyBIOS, QEMU launch a vm with no options about bios. For
// UEFI, QEMU launch a vm with OVMF.
const (
	LegacyBIOS BIOSMode = iota
	UEFI
)

// NetworkSpec represents a network specification
type NetworkSpec struct {
	Internal  bool
	UseNAT    bool
	Addresses []string
}

// Network represents a network configuration
type Network struct {
	Name string
	Spec NetworkSpec
}

// ImageSpec represents an image specification
type ImageSpec struct {
	URL  *url.URL
	File string
}

// Image represents an image configuration
type Image struct {
	Name string
	Spec ImageSpec
}

// CloudConfigSpec represents a cloud-config configuration
type CloudConfigSpec struct {
	NetworkConfig string
	UserData      string
}

// VolumeSpec represents a volume specification
type VolumeSpec struct {
	Name           string
	Size           string
	Source         string
	CloudConfig    CloudConfigSpec
	RecreatePolicy VolumeRecreatePolicy
	image          *Image
}

// Resolve resolves image source reference
func (vs *VolumeSpec) Resolve(c *Cluster) error {
	if vs.Source == "" {
		return nil
	}
	for _, img := range c.Images {
		if img.Name == vs.Source {
			vs.image = img
			return nil
		}
	}
	return errors.New("no such image: " + vs.Source)
}

// ResourceSpec represents a resource specification
type ResourceSpec struct {
	CPU    string
	Memory string
}

// SMBIOSSpec represents a manufacturer name, product name, and serial number in smbios
type SMBIOSSpec struct {
	Manufacturer string
	Product      string
	Serial       string
}

// NodeSpec represents a node specification
type NodeSpec struct {
	Interfaces []string
	Volumes    []*VolumeSpec
	Resources  ResourceSpec
	BIOS       BIOSMode
	SMBIOS     SMBIOSSpec
}

// Node represents a node configuration
type Node struct {
	Name string
	Spec NodeSpec
}

// NodeSetSpec represents a node-set specification
type NodeSetSpec struct {
	Replicas int
	Template NodeSpec
}

// NodeSet represents a node-set configuration
type NodeSet struct {
	Name string
	Spec NodeSetSpec
}

// Cluster represents cluster configuration
type Cluster struct {
	Networks []*Network
	Images   []*Image
	Nodes    []*Node
	NodeSets []*NodeSet
}

// Append appends the other cluster into the receiver
func (c *Cluster) Append(other *Cluster) *Cluster {
	c.Networks = append(c.Networks, other.Networks...)
	c.Nodes = append(c.Nodes, other.Nodes...)
	c.NodeSets = append(c.NodeSets, other.NodeSets...)
	c.Images = append(c.Images, other.Images...)
	return c
}

// Resolve resolves references between resources
func (c *Cluster) Resolve() error {
	for _, node := range c.Nodes {
		for _, vs := range node.Spec.Volumes {
			err := vs.Resolve(c)
			if err != nil {
				return err
			}
		}
	}
	for _, nodeSet := range c.NodeSets {
		for _, vs := range nodeSet.Spec.Template.Volumes {
			err := vs.Resolve(c)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (img *Image) writeToFile(ctx context.Context, destPath string, c *cache) error {
	if img.Spec.File != "" {
		f, err := os.Open(img.Spec.File)
		if err != nil {
			return err
		}
		defer f.Close()

		destFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		defer destFile.Close()

		_, err = io.Copy(destFile, f)
		return err
	}

	return img.downloadImage(ctx, destPath, c)
}

func (img *Image) downloadImage(ctx context.Context, destPath string, c *cache) error {
	urlString := img.Spec.URL.String()
RETRY:
	r, err := c.Get(urlString)
	if err == nil {
		defer r.Close()

		d, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		defer d.Close()

		_, err = io.Copy(d, r)
		if err != nil {
			return err
		}

		return d.Sync()
	}

	req, err := http.NewRequest("GET", urlString, nil)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)

	client := &cmd.HTTPClient{
		Client:   &http.Client{},
		Severity: log.LvDebug,
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download: %s: %s", res.Status, urlString)
	}

	size, err := strconv.Atoi(res.Header.Get("Content-Length"))
	if err != nil {
		return err
	}

	log.Info("Downloading image...", map[string]interface{}{
		"size": size,
	})

	err = c.Put(urlString, res.Body)
	if err != nil {
		return err
	}

	goto RETRY
}
