package placemat

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"
)

func TestCache(t *testing.T) {
	key := "https://foo/bar.img"

	d, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		os.RemoveAll(d)
	}()
	c := cache{d}

	_, err = c.Get(key)
	if err == nil {
		t.Error("key must not exist")
	}

	data := []byte("foobar")
	err = c.Put(key, bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}

	r, err := c.Get(key)
	if err != nil {
		t.Fatal(err)
	}

	data2, err := ioutil.ReadAll(r)
	if err != nil {
		t.Error(err)
	}

	if !bytes.Equal(data, data2) {
		t.Error("data corrupted:", string(data2))
	}

}
