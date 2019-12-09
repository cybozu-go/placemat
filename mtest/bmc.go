package mtest

import (
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

const virtualBMCPort = "/dev/virtio-ports/placemat"

// TestBMC tests the behaviour of virtual BMC
func TestBMC() {
	It("should launch pods", func() {
		var session *gexec.Session
		By("launch placemat", func() {
			session = runPlacemat(clusterYAML, "-force", "-bmc-cert", bmcCert, "-bmc-key", bmcKey)
			err := prepareSSHClients(node1, node2)
			Expect(err).To(Succeed())
		})

		By("writing to "+virtualBMCPort, func() {
			execSafeAt(node1, "echo", bmc1, "|", "sudo", "dd", "of="+virtualBMCPort)
		})

		By("starting HTTPS server", func() {
			Eventually(func() error {
				stdout, err := execAtLocal("curl", "--insecure", "https://"+bmc1)
				if err != nil {
					return fmt.Errorf("failed to curl; stdout: %s, err: %v", stdout, err)
				}
				return nil
			}).Should(Succeed())
		})

		By("terminating placemat", func() {
			terminatePlacemat(session)
			Eventually(session.Exited).Should(BeClosed())
		})
	})
}
