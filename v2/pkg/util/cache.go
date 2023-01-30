package util

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"

	"github.com/cybozu-go/log"
	"github.com/cybozu-go/well"
)

type Cache struct {
	dir string
}

func NewCache(dir string) *Cache {
	return &Cache{dir: dir}
}

func escapeKey(key string) string {
	h := sha256.New()
	h.Write([]byte(key))
	return hex.EncodeToString(h.Sum(nil))
}

func (c *Cache) Put(key string, data io.Reader) error {
	ek := escapeKey(key)
	f, err := os.CreateTemp(c.dir, ".tmp")
	if err != nil {
		return err
	}
	dstName := f.Name()
	defer func() {
		if f != nil {
			f.Close()
		}
		os.Remove(dstName)
	}()

	_, err = io.Copy(f, data)
	if err != nil {
		return err
	}
	err = f.Sync()
	if err != nil {
		return err
	}

	f.Close()
	f = nil

	return os.Rename(dstName, filepath.Join(c.dir, ek))
}

func (c *Cache) Get(key string) (io.ReadCloser, error) {
	return os.Open(c.Path(key))
}

func (c *Cache) Contains(key string) bool {
	_, err := os.Stat(c.Path(key))
	return !os.IsNotExist(err)
}

func (c *Cache) Path(key string) string {
	ek := escapeKey(key)
	return filepath.Join(c.dir, ek)
}

func DownloadData(ctx context.Context, u *url.URL, decomp Decompressor, c *Cache) error {
	urlString := u.String()

	if c.Contains(urlString) {
		return nil
	}

	req, err := http.NewRequest("GET", urlString, nil)
	if err != nil {
		return err
	}
	req = req.WithContext(ctx)

	client := &well.HTTPClient{
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
