package placemat

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"github.com/cybozu-go/cmd"
	"github.com/cybozu-go/log"
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
	dirPath     string
}

// Path returns a filepath to the directory having folder contents.
func (d *DataFolder) Path() string {
	return d.dirPath
}

func (d *DataFolder) setup(ctx context.Context) error {
	if len(d.Spec.Dir) != 0 {
		st, err := os.Stat(d.Spec.Dir)
		if err != nil {
			return err
		}
		if !st.IsDir() {
			return errors.New(d.Spec.Dir + " is not a directory")
		}
		absPath, err := filepath.Abs(d.Spec.Dir)
		if err != nil {
			return err
		}
		d.dirPath = absPath
		return nil
	}

	p := filepath.Join(d.baseTempDir, d.Name)
	err := os.MkdirAll(p, 0755)
	if err != nil {
		return err
	}

	for _, file := range d.Spec.Files {
		dstPath := filepath.Join(p, file.Name)
		if file.File != "" {
			err = writeToFile(file.File, dstPath, nil)
			if err != nil {
				return err
			}
		} else {
			err = downloadData(ctx, file.URL, nil, d.cache)
			if err != nil {
				return err
			}
			err := copyDownloadedData(file.URL, dstPath, d.cache)
			if err != nil {
				return err
			}
		}
	}

	d.dirPath = p
	return nil
}

// CloudConfigSpec represents a cloud-config configuration
type CloudConfigSpec struct {
	NetworkConfig string
	UserData      string
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

func writeToFile(srcPath, destPath string, decomp Decompressor) error {
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
	if decomp != nil {
		newSrc, err := decomp.Decompress(src)
		if err != nil {
			return err
		}
		src = newSrc
	}

	_, err = io.Copy(destFile, src)
	return err
}

func downloadData(ctx context.Context, u *url.URL, decomp Decompressor, c *cache) error {
	urlString := u.String()

	if c.Contains(urlString) {
		return nil
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

	var src io.Reader = res.Body
	if decomp != nil {
		newSrc, err := decomp.Decompress(res.Body)
		if err != nil {
			return err
		}
		src = newSrc
	}

	return c.Put(urlString, src)

}
func copyDownloadedData(u *url.URL, dest string, c *cache) error {
	r, err := c.Get(u.String())
	if err != nil {
		return err
	}
	defer r.Close()

	d, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE, 0644)
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
