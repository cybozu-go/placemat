package placemat

import (
	"net/url"

	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
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
	URL          *url.URL
	File         string
	Decompressor Decompressor
	CopyOnWrite  bool
}

// Image represents an image configuration
type Image struct {
	Name  string
	Spec  ImageSpec
	cache *cache
}

// CloudConfigSpec represents a cloud-config configuration
type CloudConfigSpec struct {
	NetworkConfig string
	UserData      string
}

// Volume defines the interface for Node volumes.
type Volume interface {
	Kind() string
	Name() string
	RecreatePolicy() VolumeRecreatePolicy
	Resolve(*Cluster) error
	Create(ctx context.Context, p string) error
}

type baseVolume struct {
	name   string
	policy VolumeRecreatePolicy
}

func (v baseVolume) Name() string {
	return v.name
}

func (v baseVolume) RecreatePolicy() VolumeRecreatePolicy {
	return v.policy
}

type imageVolume struct {
	baseVolume
	imageName string
	image     *Image
}

// NewImageVolume creates a volume for type "image".
func NewImageVolume(name string, policy VolumeRecreatePolicy, imageName string) *imageVolume {
	return &imageVolume{
		baseVolume: baseVolume{name, policy},
		imageName:  imageName,
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

func (v imageVolume) Create(ctx context.Context, p string) error {
	return v.image.writeToFile(ctx, p)
}

type localDSVolume struct {
	baseVolume
	userData      string
	networkConfig string
}

// NewLocalDSVolume creates a volume for type "localds".
func NewLocalDSVolume(name string, policy VolumeRecreatePolicy, u, n string) *localDSVolume {
	return &localDSVolume{
		baseVolume:    baseVolume{name, policy},
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

func (v localDSVolume) Create(ctx context.Context, p string) error {
	if v.networkConfig == "" {
		c := cmd.CommandContext(ctx, "cloud-localds", p, v.userData)
		return c.Run()
	}

	c := cmd.CommandContext(ctx, "cloud-localds", p, v.userData, "--network-config", v.networkConfig)
	return c.Run()
}

type rawVolume struct {
	baseVolume
	size string
}

// NewRawVolume creates a volume for type "raw".
func NewRawVolume(name string, policy VolumeRecreatePolicy, size string) *rawVolume {
	return &rawVolume{
		baseVolume: baseVolume{name, policy},
		size:       size,
	}
}

func (v rawVolume) Kind() string {
	return "raw"
}

func (v *rawVolume) Resolve(c *Cluster) error {
	return nil
}

func (v rawVolume) Create(ctx context.Context, p string) error {
	return cmd.CommandContext(ctx, "qemu-img", "create", "-f", "qcow2", p, v.size).Run()
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
	Interfaces   []string
	Volumes      []Volume
	IgnitionFile string
	Resources    ResourceSpec
	BIOS         BIOSMode
	SMBIOS       SMBIOSSpec
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
func (c *Cluster) Resolve(pv Provider) error {
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

	ic := pv.ImageCache()
	for _, img := range c.Images {
		img.cache = ic
	}
	return nil
}

func (img *Image) lookupFile(ctx context.Context, c *cache) (*cachedReadCloser, error) {
	if img.Spec.File != "" {
		f, err := os.Open(img.Spec.File)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		var src io.ReadCloser = f
		if img.Spec.Decompressor != nil {
			newSrc, err := img.Spec.Decompressor.Decompress(src)
			if err != nil {
				return nil, err
			}
			src = newSrc
		}

		return &cachedReadCloser{path: img.Spec.File, ReadCloser: src}, nil
	}

	return img.downloadImage(ctx, c)
}

func (img *Image) downloadImage(ctx context.Context, c *cache) (*cachedReadCloser, error) {
	urlString := img.Spec.URL.String()
	c := img.cache
RETRY:
	r, err := c.Get(urlString)
	if err == nil {
		return r, nil
	}

	req, err := http.NewRequest("GET", urlString, nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	client := &cmd.HTTPClient{
		Client:   &http.Client{},
		Severity: log.LvDebug,
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download: %s: %s", res.Status, urlString)
	}

	size, err := strconv.Atoi(res.Header.Get("Content-Length"))
	if err != nil {
		return nil, err
	}

	log.Info("Downloading image...", map[string]interface{}{
		"size": size,
	})

	var src io.Reader = res.Body
	if img.Spec.Decompressor != nil {
		newSrc, err := img.Spec.Decompressor.Decompress(res.Body)
		if err != nil {
			return nil, err
		}
		src = newSrc
	}

	err = c.Put(urlString, src)
	if err != nil {
		return nil, err
	}

	goto RETRY
}
