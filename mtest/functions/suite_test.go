package mtest

import (
	"os"
	"testing"

	"github.com/cybozu-go/placemat/mtest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestMtest(t *testing.T) {
	if os.Getenv("SSH_PRIVKEY") == "" {
		t.Skip("no SSH_PRIVKEY envvar")
	}
	RegisterFailHandler(Fail)
	RunSpecs(t, "Multi-host test for sabakan")
}

var _ = BeforeSuite(func() {
	mtest.RunBeforeSuite()
})

// This must be the only top-level test container.
// Other tests and test containers must be listed in this.
var _ = Describe("Test placemat functions", func() {
	mtest.FunctionsSuite()
})
