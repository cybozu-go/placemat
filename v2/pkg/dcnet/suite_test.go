package dcnet_test

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDCNet(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("run as root")
	}
	RegisterFailHandler(Fail)
	RunSpecs(t, "DCNet Suite")
}
