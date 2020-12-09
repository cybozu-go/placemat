package vm

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestVM(t *testing.T) {
	if os.Getuid() != 0 {
		t.Skip("run as root")
	}
	RegisterFailHandler(Fail)
	RunSpecs(t, "VM Suite")
}
