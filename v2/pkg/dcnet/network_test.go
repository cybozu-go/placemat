package dcnet

import (
	"github.com/containernetworking/plugins/pkg/utils/sysctl"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vishvananda/netlink"
	"sigs.k8s.io/yaml"
)

var _ = Describe("Bridge Network", func() {
	BeforeEach(func() {
		Expect(createNatRules()).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		Expect(cleanupNatRules()).ToNot(HaveOccurred())
	})

	It("should create an external network", func() {
		networkYaml := `
kind: Network
name: internet
type: external
use-nat: true
address: 10.0.0.1/24
`
		spec := &NetworkSpec{}
		Expect(yaml.Unmarshal([]byte(networkYaml), spec)).NotTo(HaveOccurred())
		network, err := NewNetwork(spec)
		Expect(err).NotTo(HaveOccurred())
		Expect(network.Create(1460)).NotTo(HaveOccurred())
		defer network.Cleanup()

		// Check if the bridge network is properly created.
		bridge, err := netlink.LinkByName(network.name)
		Expect(err).NotTo(HaveOccurred())
		Expect(bridge).NotTo(BeNil())
		Expect(bridge.Type()).To(Equal("bridge"))
		Expect(bridge.Attrs().MTU).To(Equal(1460))

		// Check if the ip forwarding is properly configured.
		Expect(isForwarding("net.ipv4.ip_forward")).To(BeTrue())
		Expect(isForwarding("net.ipv6.conf.all.forwarding")).To(BeTrue())

		// Check if the masquerade rule is properly configured.
		ipt4, _, err := newIptables()
		Expect(err).NotTo(HaveOccurred())
		exists, err := ipt4.Exists("nat", "PLACEMAT", "-s", "10.0.0.0/24", "!", "--destination", "10.0.0.0/24", "-j", "MASQUERADE")
		Expect(err).NotTo(HaveOccurred())
		Expect(exists).To(BeTrue())
	})

	It("should create an internal network", func() {
		networkYaml := `
kind: Network
name: core-to-op
type: internal
use-nat: false
`
		spec := &NetworkSpec{}
		Expect(yaml.Unmarshal([]byte(networkYaml), spec)).NotTo(HaveOccurred())
		network, err := NewNetwork(spec)
		Expect(err).NotTo(HaveOccurred())
		Expect(network.Create(1460)).NotTo(HaveOccurred())
		defer network.Cleanup()

		// Check if the bridge network is properly created.
		bridge, err := netlink.LinkByName(network.name)
		Expect(err).NotTo(HaveOccurred())
		Expect(bridge).NotTo(BeNil())
		Expect(bridge.Type()).To(Equal("bridge"))
		Expect(bridge.Attrs().MTU).To(Equal(1460))

		// Check if the accept rules are properly configured.
		ipt4, ipt6, err := newIptables()
		Expect(err).NotTo(HaveOccurred())
		exists, err := ipt4.Exists("filter", "PLACEMAT", "-i", network.name, "-j", "ACCEPT")
		Expect(err).NotTo(HaveOccurred())
		Expect(exists).To(BeTrue())
		exists, err = ipt4.Exists("filter", "PLACEMAT", "-o", network.name, "-j", "ACCEPT")
		Expect(err).NotTo(HaveOccurred())
		Expect(exists).To(BeTrue())
		exists, err = ipt6.Exists("filter", "PLACEMAT", "-i", network.name, "-j", "ACCEPT")
		Expect(err).NotTo(HaveOccurred())
		Expect(exists).To(BeTrue())
		exists, err = ipt6.Exists("filter", "PLACEMAT", "-o", network.name, "-j", "ACCEPT")
		Expect(err).NotTo(HaveOccurred())
		Expect(exists).To(BeTrue())
	})

	It("should create a bmc network", func() {
		networkYaml := `
kind: Network
name: bmc
type: bmc
use-nat: false
address: 10.72.16.1/20
`
		spec := &NetworkSpec{}
		Expect(yaml.Unmarshal([]byte(networkYaml), spec)).NotTo(HaveOccurred())
		network, err := NewNetwork(spec)
		Expect(err).NotTo(HaveOccurred())
		Expect(network.Create(1460)).NotTo(HaveOccurred())
		defer network.Cleanup()

		// Check if the bridge network is properly created.
		bridge, err := netlink.LinkByName(network.name)
		Expect(err).NotTo(HaveOccurred())
		Expect(bridge).NotTo(BeNil())
		Expect(bridge.Type()).To(Equal("bridge"))
		Expect(bridge.Attrs().MTU).To(Equal(1460))

		// Check if the accept rules are NOT configured.
		ipt4, ipt6, err := newIptables()
		Expect(err).NotTo(HaveOccurred())
		exists, err := ipt4.Exists("filter", "PLACEMAT", "-i", network.name, "-j", "ACCEPT")
		Expect(err).NotTo(HaveOccurred())
		Expect(exists).To(BeFalse())
		exists, err = ipt4.Exists("filter", "PLACEMAT", "-o", network.name, "-j", "ACCEPT")
		Expect(err).NotTo(HaveOccurred())
		Expect(exists).To(BeFalse())
		exists, err = ipt6.Exists("filter", "PLACEMAT", "-i", network.name, "-j", "ACCEPT")
		Expect(err).NotTo(HaveOccurred())
		Expect(exists).To(BeFalse())
		exists, err = ipt6.Exists("filter", "PLACEMAT", "-o", network.name, "-j", "ACCEPT")
		Expect(err).NotTo(HaveOccurred())
		Expect(exists).To(BeFalse())
	})

	It("should NOT create a network whose name is more than 15 characters", func() {
		networkYaml := `
kind: Network
name: 1234567890123456
type: external
use-nat: false
`
		spec := &NetworkSpec{}
		Expect(yaml.Unmarshal([]byte(networkYaml), spec)).NotTo(HaveOccurred())
		_, err := NewNetwork(spec)
		Expect(err).To(HaveOccurred())
	})

	It("should NOT create an invalid network", func() {
		networkYaml := `
kind: Network
name: invalid
type: invalid
use-nat: false
`
		spec := &NetworkSpec{}
		Expect(yaml.Unmarshal([]byte(networkYaml), spec)).NotTo(HaveOccurred())
		_, err := NewNetwork(spec)
		Expect(err).To(HaveOccurred())
	})
})

func isForwarding(name string) bool {
	val, err := sysctl.Sysctl(name)
	if err != nil {
		return false
	}
	return len(val) > 0 && val[0] != '0'
}
