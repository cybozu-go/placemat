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
---
kind: Node
name: boot-0
interfaces:
- r0-node1
- r0-node2
memory: 2G
cpu: 8
smbios:
  manufacturer: cybozu
  product: mk2
  serial: fb8f2417d0b4db30050719c31ce02a2e8141bbd8
ignition: my-node.ign
volumes:
- kind: image
  name: root
  image: custom-ubuntu-image
  size: 10G
  copy-on-write: true
  cache: writeback
- kind: localds
  name: seed
  network-config: network.yml
  user-data: seed_boot-0.yml
- kind: 9p
  name: sabakan
  folder: sabakan-data
uefi: false
tpm: true
---
kind: Image
name: custom-ubuntu-image
file: cybozu-ubuntu-18.04-server-cloudimg-amd64.img
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
			Nodes: []*NodeSpec{
				{
					Kind: "Node",
					Name: "boot-0",
					Interfaces: []string{
						"r0-node1",
						"r0-node2",
					},
					Volumes: []NodeVolumeSpec{
						{
							Kind:        "image",
							Name:        "root",
							Image:       "custom-ubuntu-image",
							Size:        "10G",
							CopyOnWrite: true,
							Cache:       "writeback",
						},
						{
							Kind:          "localds",
							Name:          "seed",
							NetworkConfig: "network.yml",
							UserData:      "seed_boot-0.yml",
						},
						{
							Kind:   "9p",
							Name:   "sabakan",
							Folder: "sabakan-data",
						},
					},
					IgnitionFile: "my-node.ign",
					CPU:          8,
					Memory:       "2G",
					UEFI:         false,
					TPM:          true,
					SMBIOS: SMBIOSConfigSpec{
						Manufacturer: "cybozu",
						Product:      "mk2",
						Serial:       "fb8f2417d0b4db30050719c31ce02a2e8141bbd8",
					},
				},
			},
			Images: []*ImageSpec{
				{
					Kind: "Image",
					Name: "custom-ubuntu-image",
					File: "cybozu-ubuntu-18.04-server-cloudimg-amd64.img",
				},
			},
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
