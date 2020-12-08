package dcnet

import (
	"context"
	"fmt"
	"path"
	"strings"

	"github.com/containernetworking/plugins/pkg/ns"
	"github.com/cybozu-go/placemat/v2/pkg/types"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/vishvananda/netlink"
)

var _ = Describe("NetworkNamespace resource", func() {
	BeforeEach(func() {
		Expect(createNatRules()).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		Expect(cleanupNatRules()).ToNot(HaveOccurred())
	})

	It("should create a network namespace as specified with a yaml representation", func() {
		clusterYaml := `
kind: Network
name: internet
type: external
use-nat: true
address: 10.0.0.1/24
---
kind: Network
name: core-to-s1
type: internal
use-nat: false
---
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

		cluster, err := types.Parse(strings.NewReader(clusterYaml))
		Expect(err).NotTo(HaveOccurred())
		internetSpec := cluster.Networks[0]
		internet, err := NewNetwork(internetSpec)
		Expect(err).NotTo(HaveOccurred())
		Expect(internet.Create(1460)).NotTo(HaveOccurred())
		defer internet.Cleanup()

		coreToS1Spec := cluster.Networks[1]
		coreToS1, err := NewNetwork(coreToS1Spec)
		Expect(err).NotTo(HaveOccurred())
		Expect(coreToS1.Create(1460)).NotTo(HaveOccurred())
		defer coreToS1.Cleanup()

		nnsSpec := cluster.NetNSs[0]
		nns, err := NewNetNS(nnsSpec)
		Expect(err).NotTo(HaveOccurred())
		Expect(nns.Setup(context.Background(), 1460)).NotTo(HaveOccurred())
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
})
