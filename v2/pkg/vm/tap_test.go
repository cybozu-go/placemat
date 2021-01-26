package vm

import (
	"strings"

	"github.com/cybozu-go/placemat/v2/pkg/dcnet"
	"github.com/cybozu-go/placemat/v2/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vishvananda/netlink"
)

var _ = Describe("Tap", func() {
	BeforeEach(func() {
		Expect(dcnet.CreateNatRules()).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		dcnet.CleanupNatRules()
	})

	It("should create a tap as specified", func() {
		clusterYaml := `
kind: Network
name: r0-node1
type: internal
use-nat: false
`
		cluster, err := types.Parse(strings.NewReader(clusterYaml))
		Expect(err).NotTo(HaveOccurred())

		networkSpec := cluster.Networks[0]
		network, err := dcnet.NewNetwork(networkSpec)
		Expect(err).NotTo(HaveOccurred())
		Expect(network.Setup(1460)).NotTo(HaveOccurred())
		defer network.Cleanup()

		tap, err := newTap("r0-node1")
		Expect(err).NotTo(HaveOccurred())
		tapInfo, err := tap.create(1460)
		Expect(err).NotTo(HaveOccurred())
		defer tap.Cleanup()

		link, err := netlink.LinkByName(tapInfo.tap)
		Expect(err).NotTo(HaveOccurred())
		Expect(link.Type()).To(Equal("tuntap"))
		Expect(link.Attrs().MTU).To(Equal(1460))
	})

	It("should create a tap with default MTU 1500", func() {
		clusterYaml := `
kind: Network
name: r0-node1
type: internal
use-nat: false
`
		cluster, err := types.Parse(strings.NewReader(clusterYaml))
		Expect(err).NotTo(HaveOccurred())

		networkSpec := cluster.Networks[0]
		network, err := dcnet.NewNetwork(networkSpec)
		Expect(err).NotTo(HaveOccurred())
		Expect(network.Setup(0)).NotTo(HaveOccurred())
		defer network.Cleanup()

		tap, err := newTap("r0-node1")
		Expect(err).NotTo(HaveOccurred())
		tapInfo, err := tap.create(0)
		Expect(err).NotTo(HaveOccurred())
		defer tap.Cleanup()

		link, err := netlink.LinkByName(tapInfo.tap)
		Expect(err).NotTo(HaveOccurred())
		Expect(link.Type()).To(Equal("tuntap"))
		Expect(link.Attrs().MTU).To(Equal(1500))
	})
})
