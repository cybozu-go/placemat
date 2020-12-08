package types

import (
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Cluster resource types", func() {
	It("should create an external network", func() {
		clusterYaml := `
kind: Network
name: internet
type: external
use-nat: true
address: 10.0.0.1/24
---
kind: Network
name: core-to-op
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
		cluster, err := Parse(strings.NewReader(clusterYaml))
		Expect(err).NotTo(HaveOccurred())
		Expect(*cluster).To(Equal(Cluster{
			Networks: []*NetworkSpec{{
				Kind:    "Network",
				Name:    "internet",
				Type:    "external",
				UseNAT:  true,
				Address: "10.0.0.1/24",
			}, {
				Kind:   "Network",
				Name:   "core-to-op",
				Type:   "internal",
				UseNAT: false,
			}},
			NetNSs: []*NetNSSpec{{
				Kind: "NetworkNamespace",
				Name: "core",
				Interfaces: []*NetNSInterfaceSpec{
					{
						Network: "internet",
						Addresses: []string{
							"10.0.0.2/24",
						},
					},
					{
						Network: "core-to-s1",
						Addresses: []string{
							"10.0.2.0/31",
						},
					},
				},
				Apps: []*NetNSAppSpec{
					{
						Name: "link",
						Command: []string{
							"ip",
							"link",
							"add",
							"t1",
							"type",
							"veth",
							"peer",
							"name",
							"t2",
						},
					},
				},
				InitScripts: []string{
					"pkg/dcnet/netns_test.sh",
				},
			}},
		}))
	})

	It("should NOT create a network whose name is more than 15 characters", func() {
		clusterYaml := `
kind: Network
name: 1234567890123456
type: external
use-nat: false
`
		cluster, err := Parse(strings.NewReader(clusterYaml))
		Expect(err).To(HaveOccurred())
		Expect(cluster).To(BeNil())
	})

	It("should NOT create an invalid network", func() {
		clusterYaml := `
kind: Network
name: invalid
type: invalid
use-nat: false
`
		cluster, err := Parse(strings.NewReader(clusterYaml))
		Expect(err).To(HaveOccurred())
		Expect(cluster).To(BeNil())
	})

	It("should NOT create a network namespace with an empty name", func() {
		clusterYaml := `
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
interfaces:
- addresses:
  - 10.0.0.2/24
  network: internet
`
		cluster, err := Parse(strings.NewReader(clusterYaml))
		Expect(err).To(HaveOccurred())
		Expect(cluster).To(BeNil())
	})

	It("should NOT create a network namespace without interfaces", func() {
		clusterYaml := `
kind: NetworkNamespace
name: core
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
		cluster, err := Parse(strings.NewReader(clusterYaml))
		Expect(err).To(HaveOccurred())
		Expect(cluster).To(BeNil())
	})

	It("should NOT create a network namespace with an app without commands", func() {
		clusterYaml := `
kind: NetworkNamespace
name: core
apps:
  - name: link
interfaces:
- addresses:
  - 10.0.0.2/24
  network: internet
`
		cluster, err := Parse(strings.NewReader(clusterYaml))
		Expect(err).To(HaveOccurred())
		Expect(cluster).To(BeNil())
	})
})
