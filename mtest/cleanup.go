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
		By("launch placemat", func() {
			session = runPlacemat(clusterYAML, "-force")
			err := prepareSSHClients(node1, node2)
			Expect(err).To(Succeed())
		})

		By("confirm that socket files exist on host", func() {
			_, err := os.Stat("/tmp/node1/swtpm.socket")
			Expect(err).NotTo(HaveOccurred())
			_, err = os.Stat("/tmp/node2/swtpm.socket")
			Expect(err).NotTo(HaveOccurred())
		})

		By("confirm that a device file exists on guest", func() {
			execSafeAt(node1, "test", "-c", "/dev/tpm0")
			execSafeAt(node2, "test", "-c", "/dev/tpm0")
		})

		By("kill placemat process", func() {
			killPlacemat(session)
			Eventually(session.Exited).Should(BeClosed())
		})

		By("run placemat without -force option", func() {
			session = runPlacemat(clusterYAML)
			Eventually(session.Exited).Should(BeClosed())
		})

		By("run placemat with -force option", func() {
			session = runPlacemat(clusterYAML, "-force")
			err := prepareSSHClients(node1, node2)
			Expect(err).To(Succeed())
		})

		By("terminate placemat", func() {
			terminatePlacemat(session)
			Eventually(session.Exited).Should(BeClosed())
		})
	})
}
