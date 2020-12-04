package dcnet

import (
	"context"
	"fmt"
	"path"

	"github.com/containernetworking/plugins/pkg/ns"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vishvananda/netlink"
	"sigs.k8s.io/yaml"
)

var _ = Describe("Network Namespace", func() {
	BeforeEach(func() {
		Expect(createNatRules()).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		Expect(cleanupNatRules()).ToNot(HaveOccurred())
	})

	It("should create a network namespace", func() {
		internetYaml := `
kind: Network
name: internet
type: external
use-nat: true
address: 10.0.0.1/24
`
		internetSpec := &NetworkSpec{}
		Expect(yaml.Unmarshal([]byte(internetYaml), internetSpec)).NotTo(HaveOccurred())
		internet, err := NewNetwork(internetSpec)
		Expect(err).NotTo(HaveOccurred())
		Expect(internet.Create(1460)).NotTo(HaveOccurred())
		defer internet.Cleanup()

		coreToS1Yaml := `
kind: Network
name: core-to-s1
type: internal
use-nat: false
`
		coreToS1Spec := &NetworkSpec{}
		Expect(yaml.Unmarshal([]byte(coreToS1Yaml), coreToS1Spec)).NotTo(HaveOccurred())
		coreToS1, err := NewNetwork(coreToS1Spec)
		Expect(err).NotTo(HaveOccurred())
		Expect(coreToS1.Create(1460)).NotTo(HaveOccurred())
		defer coreToS1.Cleanup()

		netnsYaml := `
kind: NetworkNamespace
name: core
init-scripts:
  - pkg/dcnet/netns_test.sh
apps:
  - name: link
    command:
    - ip
    - link
    - add
    - t1
    - type
    - veth
    - peer
    - name
    - t2
interfaces:
- addresses:
  - 10.0.0.2/24
  network: internet
- addresses:
  - 10.0.2.0/31
  network: core-to-s1
`
		nnsSpec := &NetNSSpec{}
		Expect(yaml.Unmarshal([]byte(netnsYaml), nnsSpec)).NotTo(HaveOccurred())
		nns, err := NewNetNS(nnsSpec)
		Expect(err).NotTo(HaveOccurred())
		Expect(nns.Start(context.Background(), 1460)).NotTo(HaveOccurred())
		defer nns.Cleanup()

		// Check if a networks namespace is properly created
		created, err := ns.GetNS(path.Join(getNsRunDir(), nns.name))
		Expect(err).NotTo(HaveOccurred())

		// Check inside the network namespace
		err = created.Do(func(hostNS ns.NetNS) error {
			// Check if veths are properly created
			// eth0
			eth0, err := netlink.LinkByName("eth0")
			if err != nil {
				return fmt.Errorf("failed to find eth0: %w", err)
			}
			if eth0.Type() != "veth" {
				return fmt.Errorf("eth0 type is not veth: actual type is %s", eth0.Type())
			}
			if eth0.Attrs().MTU != 1460 {
				return fmt.Errorf("eth0 mtu is not 1460: actual mtu is %d", eth0.Attrs().MTU)
			}
			addrs, err := netlink.AddrList(eth0, netlink.FAMILY_V4)
			if err != nil {
				return fmt.Errorf("failed to list eth0 addresses: %w", err)
			}
			if len(addrs) != 1 {
				return fmt.Errorf("eth0 address length is not 1: acrual addresses are %v", addrs)
			}
			if addrs[0].IP.String() != "10.0.0.2" {
				return fmt.Errorf("eth0 address is not 10.0.0.2: actual address is %s", addrs[0].IP.String())
			}
			// eth1
			eth1, err := netlink.LinkByName("eth1")
			if err != nil {
				return fmt.Errorf("failed to find eth1: %w", err)
			}
			if eth1.Type() != "veth" {
				return fmt.Errorf("eth1 type is not veth: actual type is %s", eth1.Type())
			}
			if eth1.Attrs().MTU != 1460 {
				return fmt.Errorf("eth1 mtu is not 1460: actual mtu is %d", eth1.Attrs().MTU)
			}
			addrs, err = netlink.AddrList(eth1, netlink.FAMILY_V4)
			if err != nil {
				return fmt.Errorf("failed to list eth1 addresses: %w", err)
			}
			if len(addrs) != 1 {
				return fmt.Errorf("eth1 address length is not 1: acrual addresses are %v", addrs)
			}
			if addrs[0].IP.String() != "10.0.2.0" {
				return fmt.Errorf("eth1 address is not 10.0.2.0: actual address is %s", addrs[0].IP.String())
			}

			// Check if the init script is properly executed.
			ipt4, _, err := newIptables()
			if err != nil {
				return fmt.Errorf("failed to create iptables: %w", err)
			}
			exists, err := ipt4.Exists("nat", "POSTROUTING", "-o", "eth0", "-j", "MASQUERADE")
			if err != nil {
				return fmt.Errorf("failed to find iptables rule: %w", err)
			}
			if !exists {
				return fmt.Errorf("failed to run the init script")
			}

			// Check if commands are properly executed
			if _, err := netlink.LinkByName("t1"); err != nil {
				return fmt.Errorf("failed to find the veth t1: %w", err)
			}

			return nil
		})
		Expect(err).NotTo(HaveOccurred())

		// Check if host veths' masters are properly configured
		bridge, err := netlink.LinkByName("internet")
		Expect(err).NotTo(HaveOccurred())
		veth0, err := netlink.LinkByName(nns.hostVethNames[0])
		Expect(err).NotTo(HaveOccurred())
		Expect(veth0.Attrs().MasterIndex).To(Equal(bridge.Attrs().Index))

		bridge, err = netlink.LinkByName("core-to-s1")
		Expect(err).NotTo(HaveOccurred())
		veth1, err := netlink.LinkByName(nns.hostVethNames[1])
		Expect(err).NotTo(HaveOccurred())
		Expect(veth1.Attrs().MasterIndex).To(Equal(bridge.Attrs().Index))

	})

	It("should NOT create a network namespace with an empty name", func() {
		netnsYaml := `
kind: NetworkNamespace
name:
apps:
  - name: link
    command:
    - ip
    - link
    - add
    - t1
    - type
    - veth
    - peer
    - name
    - t2
`
		nnsSpec := &NetNSSpec{}
		Expect(yaml.Unmarshal([]byte(netnsYaml), nnsSpec)).NotTo(HaveOccurred())
		// Check if a networks namespace is NOT created
		nns, err := NewNetNS(nnsSpec)
		Expect(err).To(HaveOccurred())
		Expect(nns).To(BeNil())
	})

	It("should NOT create a network namespace without apps", func() {
		netnsYaml := `
kind: NetworkNamespace
name: core
`
		nnsSpec := &NetNSSpec{}
		Expect(yaml.Unmarshal([]byte(netnsYaml), nnsSpec)).NotTo(HaveOccurred())
		// Check if a networks namespace is NOT created
		nns, err := NewNetNS(nnsSpec)
		Expect(err).To(HaveOccurred())
		Expect(nns).To(BeNil())
	})

	It("should NOT create a network namespace with an app without commands", func() {
		netnsYaml := `
kind: NetworkNamespace
name: core
apps:
  - name: link
`
		nnsSpec := &NetNSSpec{}
		Expect(yaml.Unmarshal([]byte(netnsYaml), nnsSpec)).NotTo(HaveOccurred())
		// Check if a networks namespace is NOT created
		nns, err := NewNetNS(nnsSpec)
		Expect(err).To(HaveOccurred())
		Expect(nns).To(BeNil())
	})
})
