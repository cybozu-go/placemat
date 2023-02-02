package virtualbmc_test

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestVirtualBMC(t *testing.T) {
	RegisterFailHandler(Fail)
	SetDefaultEventuallyTimeout(20 * time.Second)
	SetDefaultEventuallyPollingInterval(1 * time.Second)
	RunSpecs(t, "VirtualBMC Suite")
}
