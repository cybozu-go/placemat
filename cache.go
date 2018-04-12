package placemat

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

type cache struct {
	dir string
}

func escapeKey(key string) string {
	h := sha256.New()
	h.Write([]byte(key))
	return hex.EncodeToString(h.Sum(nil))
}

func (c *cache) Put(key string, data io.Reader) error {
	ek := escapeKey(key)
	f, err := ioutil.TempFile(c.dir, ".tmp")
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

func (c *cache) Get(key string) (io.ReadCloser, error) {
	return os.Open(c.Path(key))
}

func (c *cache) Contains(key string) bool {
	_, err := os.Stat(c.Path(key))
	return !os.IsNotExist(err)
}

func (c *cache) Path(key string) string {
	ek := escapeKey(key)
	return filepath.Join(c.dir, ek)
}
