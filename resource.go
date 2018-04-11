package placemat

import (
	"net/url"
	"path/filepath"

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
	URL          *url.URL
	File         string
	Decompressor Decompressor
}

// Image represents an image configuration
type Image struct {
	Name  string
	Spec  ImageSpec
	cache *cache
}

// DataFolderFile represents a file in a data folder
type DataFolderFile struct {
	Name string
	URL  *url.URL
	File string
}

// DataFolderSpec represents a data folder specification
type DataFolderSpec struct {
	Dir   string
	Files []DataFolderFile
}

// DataFolder represents a data folder configuration
type DataFolder struct {
	Name        string
	Spec        DataFolderSpec
	cache       *cache
	baseTempDir string
}

func (d *DataFolder) tempDir(ctx context.Context) (string, error) {
	p := filepath.Join(d.baseTempDir, d.Name)
	exists, err := pathExists(p)
	if err != nil {
		return "", err
	}
	if exists {
		return p, nil
	}

	err = os.MkdirAll(p, 0755)
	if err != nil {
		return "", err
	}

	for _, file := range d.Spec.Files {
		dstPath := filepath.Join(p, file.Name)
		if file.File != "" {
			err = writeToFile(ctx, file.File, dstPath, nil)
			if err != nil {
				return "", err
			}
		} else {
			err = downloadData(ctx, file.URL, dstPath, nil, d.cache)
			if err != nil {
				return "", err
			}
		}
	}

	return p, nil
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
	Resolve(*Cluster) error
	Create(ctx context.Context, dataDir, node string) error
	QemuArgs() []string
}

type baseVolume struct {
	name   string
	policy VolumeRecreatePolicy
	path   string
}

func (v baseVolume) Name() string {
	return v.name
}

func volumePath(dataDir, node, name string) string {
	return filepath.Join(dataDir, "volumes", node+"_"+name+".img")
}

func pathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	return !os.IsNotExist(err), nil
}

func (v baseVolume) needRecreate() (bool, error) {
	exists, err := pathExists(v.path)
	if err != nil {
		return false, err
	}
	return (v.policy == RecreateAlways || v.policy == RecreateIfNotPresent && !exists), nil
}

func (v baseVolume) QemuArgs() []string {
	return []string{
		"-drive",
		"if=virtio,cache=none,aio=native,file=" + v.path,
	}
}

type imageVolume struct {
	baseVolume
	imageName string
	image     *Image
}

// NewImageVolume creates a volume for type "image".
func NewImageVolume(name string, policy VolumeRecreatePolicy, imageName string) *imageVolume {
	return &imageVolume{
		baseVolume: baseVolume{name: name, policy: policy},
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

func (v *imageVolume) Create(ctx context.Context, dataDir, node string) error {
	v.path = volumePath(dataDir, node, v.name)
	needRecreate, err := v.needRecreate()
	if err != nil {
		return err
	}
	if !needRecreate {
		return nil
	}
	if v.image.Spec.File != "" {
		return writeToFile(ctx, v.image.Spec.File, v.path, v.image.Spec.Decompressor)
	}
	return downloadData(ctx, v.image.Spec.URL, v.path, v.image.Spec.Decompressor, v.image.cache)
}

type localDSVolume struct {
	baseVolume
	userData      string
	networkConfig string
}

// NewLocalDSVolume creates a volume for type "localds".
func NewLocalDSVolume(name string, policy VolumeRecreatePolicy, u, n string) *localDSVolume {
	return &localDSVolume{
		baseVolume:    baseVolume{name: name, policy: policy},
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

func (v *localDSVolume) Create(ctx context.Context, dataDir, node string) error {
	v.path = volumePath(dataDir, node, v.name)
	needRecreate, err := v.needRecreate()
	if err != nil {
		return err
	}
	if !needRecreate {
		return nil
	}

	if v.networkConfig == "" {
		c := cmd.CommandContext(ctx, "cloud-localds", v.path, v.userData)
		return c.Run()
	}

	c := cmd.CommandContext(ctx, "cloud-localds", v.path, v.userData, "--network-config", v.networkConfig)
	return c.Run()
}

type rawVolume struct {
	baseVolume
	size string
}

// NewRawVolume creates a volume for type "raw".
func NewRawVolume(name string, policy VolumeRecreatePolicy, size string) *rawVolume {
	return &rawVolume{
		baseVolume: baseVolume{name: name, policy: policy},
		size:       size,
	}
}

func (v rawVolume) Kind() string {
	return "raw"
}

func (v *rawVolume) Resolve(c *Cluster) error {
	return nil
}

func (v *rawVolume) Create(ctx context.Context, dataDir, node string) error {
	v.path = volumePath(dataDir, node, v.name)
	needRecreate, err := v.needRecreate()
	if err != nil {
		return err
	}
	if !needRecreate {
		return nil
	}
	return cmd.CommandContext(ctx, "qemu-img", "create", "-f", "qcow2", v.path, v.size).Run()
}

type vvfatVolume struct {
	baseVolume
	folderName string
	folder     *DataFolder
}

// NewVVFATVolume creates a volume for type "vvfat".
func NewVVFATVolume(name string, policy VolumeRecreatePolicy, folderName string) *vvfatVolume {
	return &vvfatVolume{
		baseVolume: baseVolume{name: name, policy: policy},
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

func (v *vvfatVolume) Create(ctx context.Context, dataDir, node string) error {
	if v.folder.Spec.Dir != "" {
		v.path = v.folder.Spec.Dir
		return nil
	}

	d, err := v.folder.tempDir(ctx)
	if err != nil {
		return err
	}
	v.path = d
	return nil
}

func (v vvfatVolume) QemuArgs() []string {
	return []string{
		"-drive",
		"file=fat:32:" + v.path + ",format=raw,if=virtio",
	}
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
	Networks    []*Network
	Images      []*Image
	DataFolders []*DataFolder
	Nodes       []*Node
	NodeSets    []*NodeSet
}

// Append appends the other cluster into the receiver
func (c *Cluster) Append(other *Cluster) *Cluster {
	c.Networks = append(c.Networks, other.Networks...)
	c.Nodes = append(c.Nodes, other.Nodes...)
	c.NodeSets = append(c.NodeSets, other.NodeSets...)
	c.Images = append(c.Images, other.Images...)
	c.DataFolders = append(c.DataFolders, other.DataFolders...)
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

	dc := pv.DataCache()
	td := pv.TempDir()
	for _, folder := range c.DataFolders {
		folder.cache = dc
		folder.baseTempDir = td
	}

	return nil
}

func writeToFile(ctx context.Context, srcPath, destPath string, decompressor Decompressor) error {
	f, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer f.Close()

	destFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer destFile.Close()

	var src io.Reader = f
	if decompressor != nil {
		newSrc, err := decompressor.Decompress(src)
		if err != nil {
			return err
		}
		src = newSrc
	}

	_, err = io.Copy(destFile, src)
	return err
}

func downloadData(ctx context.Context, url *url.URL, destPath string, decompressor Decompressor, cache *cache) error {
	urlString := url.String()
RETRY:
	r, err := cache.Get(urlString)
	if err == nil {
		defer r.Close()

		d, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE, 0644)
		if err != nil {
			return err
		}
		defer d.Close()

		var src io.Reader = r
		if decompressor != nil {
			newSrc, err := decompressor.Decompress(src)
			if err != nil {
				return err
			}
			src = newSrc
		}

		_, err = io.Copy(d, src)
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

	log.Info("Downloading data...", map[string]interface{}{
		"url":  urlString,
		"size": size,
	})

	err = cache.Put(urlString, res.Body)
	if err != nil {
		return err
	}

	goto RETRY
}
