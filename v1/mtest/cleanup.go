package mtest

import (
	"os"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

// TestCleanup tests -force option test
func TestCleanup() {
	It("should remove remaining resources and launch placemat", func() {
		var session *gexec.Session
		By("running placemat", func() {
			session = runPlacemat(clusterYAML, "-force")
			err := prepareSSHClients(node1, node2)
			Expect(err).To(Succeed())
		})

		By("checking that socket files exist on a host", func() {
			_, err := os.Stat("/tmp/node1/swtpm.socket")
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Stat("/tmp/node2/swtpm.socket")
			Expect(err).NotTo(HaveOccurred())
		})

		By("checking that a device file exists on guests", func() {
			execSafeAt(node1, "test", "-c", "/dev/tpm0")
			execSafeAt(node2, "test", "-c", "/dev/tpm0")
		})

		By("killing placemat process", func() {
			killPlacemat(session)
			Eventually(session.Exited).Should(BeClosed())
		})

		By("running placemat without -force option", func() {
			session = runPlacemat(clusterYAML)
			Eventually(session.Exited).Should(BeClosed())
		})

		By("running placemat with -force option", func() {
			session = runPlacemat(clusterYAML, "-force")
			err := prepareSSHClients(node1, node2)
			Expect(err).To(Succeed())
		})

		By("terminating placemat", func() {
			terminatePlacemat(session)
			Eventually(session.Exited).Should(BeClosed())
		})
	})
}
