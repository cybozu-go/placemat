package mtest

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("-force option test", func() {
	It("should remove remaining resources and launch placemat", func() {
		var session *gexec.Session
		By("launch placemat", func() {
			session = runPlacemt(clusterYaml, "-force")
			err := prepareSSHClients(node1, node2)
			Expect(err).To(Succeed())
		})

		By("kill placemat process", func() {
			killPlacemat(session)
			Eventually(session.Exited).Should(BeClosed())
		})

		By("run placemat without -force option", func() {
			session = runPlacemt(clusterYaml)
			Eventually(session.Exited).Should(BeClosed())
		})

		By("run placemat with -force option", func() {
			session = runPlacemt(clusterYaml, "-force")
			err := prepareSSHClients(node1, node2)
			Expect(err).To(Succeed())
		})

		By("terminate placemat", func() {
			terminatePlacemat(session)
			Eventually(session.Exited).Should(BeClosed())
		})
	})
})
