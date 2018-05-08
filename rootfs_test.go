package placemat

import (
	"os"
	"testing"
)

func TestRootfs(t *testing.T) {
	if len(os.Getenv("TEST_ROOTFS")) == 0 {
		t.Skip("DANGER! To run this test, set environment variable TEST_ROOTFS=1.")
	}

	rootfs, err := NewRootfs()
	if err != nil {
		t.Fatal(err)
	}

	err = rootfs.Destroy()
	if err != nil {
		t.Error(err)
	}
}
