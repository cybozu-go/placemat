package mtest

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

// TestVolume tests behavior of volume
func TestVolume() {
	It("should mount volumes", func() {
		var session *gexec.Session
		By("launching placemat", func() {
			session = runPlacemat(clusterYAML, "-force")
			err := prepareSSHClients(node1, node2)
			Expect(err).To(Succeed())
		})

		By("writing to vdc (raw volume, qcow2 format)", func() {
			_, _, err := execAt(node1, "sudo", "dd", "if=/dev/zero", "of=/dev/vdc", "bs=1M", "count=1")
			Expect(err).To(Succeed())
		})

		By("writing to vdd (raw volume, raw format)", func() {
			_, _, err := execAt(node1, "sudo", "dd", "if=/dev/zero", "of=/dev/vdd", "bs=1M", "count=1")
			Expect(err).To(Succeed())
		})

		if vg != "" {
			By("writing to vde (lv volume)", func() {
				_, _, err := execAt(node1, "sudo", "dd", "if=/dev/zero", "of=/dev/vde", "bs=1M", "count=1")
				Expect(err).To(Succeed())
			})
		} else {
			By("skipping test for vde (lv volume)")
		}

		By("terminating placemat", func() {
			terminatePlacemat(session)
			Eventually(session.Exited).Should(BeClosed())
		})
	})
}
