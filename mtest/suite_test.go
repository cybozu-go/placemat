package mtest

import (
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestMtest(t *testing.T) {
	if len(sshKeyFile) == 0 {
		t.Skip("no SSH_PRIVKEY envvar")
	}
	RegisterFailHandler(Fail)
	RunSpecs(t, "Multi-host test for placemat")
}

var _ = BeforeSuite(func() {
	fmt.Println("Preparing...")
	SetDefaultEventuallyPollingInterval(5 * time.Second)
	SetDefaultEventuallyTimeout(60 * time.Second)
	fmt.Println("Begin tests...")
})
