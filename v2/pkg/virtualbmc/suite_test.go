package virtualbmc_test

import (
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestVirtualBMC(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("run as root")
	}
	RegisterFailHandler(Fail)
	SetDefaultEventuallyTimeout(10 * time.Second)
	SetDefaultEventuallyPollingInterval(1 * time.Second)
	RunSpecs(t, "VirtualBMC Suite")
}
