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

	runPlacemt("-force")

	SetDefaultEventuallyPollingInterval(5 * time.Second)
	SetDefaultEventuallyTimeout(30 * time.Second)

	err := prepareSSHClients(node1, node2)
	Expect(err).NotTo(HaveOccurred())

	// sync VM root filesystem to store newly generated SSH host keys.
	for h := range sshClients {
		execSafeAt(h, "sync")
	}

	time.Sleep(time.Second)

	fmt.Println("Begin tests...")
})

var _ = AfterSuite(func() {
	fmt.Println("Terminating...")
	terminatePlacemat()

	select {
	case <-placematSession.Exited:
		fmt.Println("exited")
	case <-time.After(30 * time.Second):
		fmt.Println("waited")
	}
})
