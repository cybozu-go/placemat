package dcnet

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Nat Rule", func() {
	It("should create nat rules", func() {
		Expect(CreateNatRules()).NotTo(HaveOccurred())
		defer CleanupNatRules()

		// Check if the nat rules are properly configured.
		ipt4, ipt6, err := newIptables()
		Expect(err).NotTo(HaveOccurred())
		exists, err := ipt4.Exists("nat", "POSTROUTING", "-j", "PLACEMAT")
		Expect(err).NotTo(HaveOccurred())
		Expect(exists).To(BeTrue())
		exists, err = ipt4.Exists("filter", "FORWARD", "-j", "PLACEMAT")
		Expect(err).NotTo(HaveOccurred())
		Expect(exists).To(BeTrue())
		exists, err = ipt6.Exists("nat", "POSTROUTING", "-j", "PLACEMAT")
		Expect(err).NotTo(HaveOccurred())
		Expect(exists).To(BeTrue())
		exists, err = ipt6.Exists("filter", "FORWARD", "-j", "PLACEMAT")
		Expect(err).NotTo(HaveOccurred())
		Expect(exists).To(BeTrue())
	})

	It("should clean up nat rules", func() {
		Expect(CreateNatRules()).NotTo(HaveOccurred())
		CleanupNatRules()

		// Check if the nat rules are wiped out.
		ipt4, ipt6, err := newIptables()
		Expect(err).NotTo(HaveOccurred())
		exists, _ := ipt4.Exists("nat", "POSTROUTING", "-j", "PLACEMAT")
		Expect(exists).To(BeFalse())
		exists, _ = ipt4.Exists("filter", "FORWARD", "-j", "PLACEMAT")
		Expect(exists).To(BeFalse())
		exists, _ = ipt6.Exists("nat", "POSTROUTING", "-j", "PLACEMAT")
		Expect(exists).To(BeFalse())
		exists, _ = ipt6.Exists("filter", "FORWARD", "-j", "PLACEMAT")
		Expect(exists).To(BeFalse())
	})
})
