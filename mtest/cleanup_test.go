package mtest

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("-force option test", func() {
	It("should remove remaining resources and launch placemat", func() {
		By("kill placemat process", func() {
			killPlacemat()
			Eventually(placematSession.Exited).Should(BeClosed())
		})

		By("run placemat without -force option", func() {
			runPlacemt()
			Eventually(placematSession.Exited).Should(BeClosed())
		})

		By("run placemat with -force option", func() {
			runPlacemt("-force")
			err := prepareSSHClients(node1, node2)
			Expect(err).To(Succeed())
		})
	})
})
