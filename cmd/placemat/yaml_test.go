package main

import (
	"bufio"
	"bytes"
	"net/url"
	"reflect"
	"testing"

	"github.com/cybozu-go/placemat"
)

func testUnmarshalImage(t *testing.T) {
	t.Parallel()

	url, _ := url.Parse("https://cloud-images.ubuntu.com/releases/16.04/release/ubuntu-16.04-server-cloudimg-amd64-disk1.img")

	cases := []struct {
		source   string
		expected placemat.Image
	}{
		{
			source: `
kind: Image
name: ubuntu-image
spec:
  url: https://cloud-images.ubuntu.com/releases/16.04/release/ubuntu-16.04-server-cloudimg-amd64-disk1.img
`,
			expected: placemat.Image{
				Name: "ubuntu-image",
				Spec: placemat.ImageSpec{
					URL: url,
				},
			},
		},
		{
			source: `
kind: Image
name: ubuntu-image
spec:
  file: /home/cybozu/ubuntu-18.04.img
`,
			expected: placemat.Image{
				Name: "ubuntu-image",
				Spec: placemat.ImageSpec{
					File: "/home/cybozu/ubuntu-18.04.img",
				},
			},
		},
	}

	for _, c := range cases {
		actual, err := unmarshalImage([]byte(c.source))
		if err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(*actual, c.expected) {
			t.Errorf("%v != %v", *actual, c.expected)
		}
	}

	errorCases := []string{
		`
kind: Image
spec:
  file: "/home/cybozu/ubuntu.img"
`,
		`
kind: Image
name: "empty-spec"
spec:
`,
		`
kind: Image
name: "both-spec"
spec:
  file: "/home/cybozu/ubuntu.img"
  url: https://cloud-images.ubuntu.com/releases/16.04/release/ubuntu-16.04-server-cloudimg-amd64-disk1.img
`,
		`
kind: Image
name: "invalid-url"
spec:
  url: $://cloud-images.ubuntu.com/releases/16.04/release/ubuntu-16.04-server-cloudimg-amd64-disk1.img
`,
	}

	for _, c := range errorCases {
		image, err := unmarshalImage([]byte(c))
		if err == nil {
			t.Errorf("%s should be error", image.Name)
		}
	}
}

func testUnmarshalNetwork(t *testing.T) {
	t.Parallel()
	cases := []struct {
		source string

		expected placemat.Network
	}{
		{
			source: `
kind: Network
name: net1
spec:
  internal: false
  use-nat: true
  addresses:
    - 10.0.0.1
    - 10.0.0.2
`,
			expected: placemat.Network{
				Name: "net1",
				Spec: placemat.NetworkSpec{
					Internal:  false,
					UseNAT:    true,
					Addresses: []string{"10.0.0.1", "10.0.0.2"},
				},
			},
		},
		{
			source: `

kind: Network
name: net2
spec:
  internal: true
`,
			expected: placemat.Network{
				Name: "net2",
				Spec: placemat.NetworkSpec{
					Internal: true,
				},
			},
		},
	}

	for _, c := range cases {
		actual, err := unmarshalNetwork([]byte(c.source))
		if err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(*actual, c.expected) {
			t.Errorf("%v != %v", *actual, c.expected)
		}
	}

	errorCases := []struct {
		source string

		expected string
	}{
		{
			source: `
kind: Network
name: net1
spec:
  internal: true
  use-nat: true
  addresses:
    - 10.0.0.1
    - 10.0.0.2
`,
			expected: "'use-nat' and 'addresses' are meaningless for internal network",
		},
		{
			source: `
kind: Network
name: net2
spec:
  internal: false
  use-nat: true
  addresses:
`,
			expected: "addresses is empty for non-internal network",
		},
	}

	for _, c := range errorCases {
		_, err := unmarshalNetwork([]byte(c.source))
		if err.Error() != c.expected {
			t.Errorf("%v != %v", err.Error(), c.expected)
		}
	}

}

func testUnmarshalNode(t *testing.T) {
	t.Parallel()
	cases := []struct {
		source string

		expected placemat.Node
	}{
		{
			source: `
kind: Node
name: node1
spec:
  interfaces:
    - br0
    - br1
  volumes:
    - kind: image
      name: ubuntu
      recreatePolicy: IfNotPresent
      spec:
        image: ubuntu-image
    - kind: localds
      name: seed
      recreatePolicy: Always
      spec:
        network-config: network.yml
        user-data: user-data.yml
    - kind: raw
      name: data
      spec:
        size: 20GB
  resources:
    cpu: 4
    memory: 8G
  bios: legacy
  smbios:
    manufacturer: QEMU
    product: Mk2
    serial: 1234abcd
`,

			expected: placemat.Node{
				Name: "node1",
				Spec: placemat.NodeSpec{
					Interfaces: []string{"br0", "br1"},
					Volumes: []placemat.Volume{
						placemat.NewImageVolume("ubuntu", placemat.RecreateIfNotPresent, "ubuntu-image"),
						placemat.NewLocalDSVolume("seed", placemat.RecreateAlways, "user-data.yml", "network.yml"),
						placemat.NewRawVolume("data", placemat.RecreateIfNotPresent, "20GB"),
					},
					Resources: placemat.ResourceSpec{CPU: "4", Memory: "8G"},
					BIOS:      placemat.LegacyBIOS,
					SMBIOS:    placemat.SMBIOSSpec{Manufacturer: "QEMU", Product: "Mk2", Serial: "1234abcd"},
				},
			},
		},
		{
			source: `
 kind: Node
 name: node2
 `,

			expected: placemat.Node{
				Name: "node2",
				Spec: placemat.NodeSpec{
					Interfaces: []string{},
					Volumes:    []placemat.Volume{},
				},
			},
		},
	}

	for _, c := range cases {
		actual, err := unmarshalNode([]byte(c.source))
		if err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(*actual, c.expected) {
			t.Errorf("%v != %v", *actual, c.expected)
		}
	}

	errorSources := []struct {
		source string

		expected string
	}{
		{
			source:   `kind: Node`,
			expected: "node name is empty",
		},
		{
			source: `
kind: Node
name: node1
spec:
  bios: None
`,
			expected: "invalid BIOS: None",
		},
		{
			source: `
kind: Node
name: node1
spec:
  volumes:
    - name: vol
      recreatePolicy: Sometime
`,
			expected: "invalid RecreatePolicy: Sometime",
		},
	}
	for _, c := range errorSources {
		_, err := unmarshalNode([]byte(c.source))
		if err.Error() != c.expected {
			t.Errorf("%v != %v", err.Error(), c.expected)
		}
	}

}

func testUnmarshalNodeSet(t *testing.T) {
	t.Parallel()
	cases := []struct {
		source   string
		expected placemat.NodeSet
	}{
		{
			source: `
kind: NodeSet
name: worker
spec:
  replicas: 3
  template:
    interfaces:
      - my-net
    volumes:
      - kind: raw
        name: data
        spec:
          size: 10GB
`,
			expected: placemat.NodeSet{
				Name: "worker",
				Spec: placemat.NodeSetSpec{
					Replicas: 3,
					Template: placemat.NodeSpec{
						Interfaces: []string{"my-net"},
						Volumes: []placemat.Volume{
							placemat.NewRawVolume("data", placemat.RecreateIfNotPresent, "10GB"),
						},
					},
				},
			},
		},
	}
	for _, c := range cases {
		actual, err := unmarshalNodeSet([]byte(c.source))
		if err != nil {
			t.Error(err)
		}
		if !reflect.DeepEqual(*actual, c.expected) {
			t.Errorf("%v != %v", *actual, c.expected)
		}
	}
	errorSources := []string{
		`kind: NodeSet`,
	}
	for _, c := range errorSources {
		_, err := unmarshalNodeSet([]byte(c))
		if err == nil {
			t.Error("err == nil, ", err)
		}
	}

}

func testUnmarshalCluster(t *testing.T) {
	t.Parallel()
	yaml := `
kind: Network
name: net1
spec:
  internal: true
---
kind: Image
name: ubuntu
spec:
  file: hoge
---
kind: Node
name: node1
---
kind: Node
name: node2
---
kind: NodeSet
name: nodeSet
`

	cluster, err := readYaml(bufio.NewReader(bytes.NewReader([]byte(yaml))))
	if err != nil {
		t.Error(err)
	}
	if len(cluster.Networks) != 1 {
		t.Error("len(cluster.Networks) != 1, ", len(cluster.Networks))
	}
	if len(cluster.Images) != 1 {
		t.Error("len(cluster.Images) != 1, ", len(cluster.Images))
	}
	if len(cluster.Nodes) != 2 {
		t.Error("len(cluster.Nodes) != 2, ", len(cluster.Nodes))
	}
	if len(cluster.NodeSets) != 1 {
		t.Error("len(cluster.NodeSets) != 1, ", len(cluster.NodeSets))
	}

}

func TestYAML(t *testing.T) {
	t.Run("image", testUnmarshalImage)
	t.Run("network", testUnmarshalNetwork)
	t.Run("node", testUnmarshalNode)
	t.Run("nodeSet", testUnmarshalNodeSet)
	t.Run("cluster", testUnmarshalCluster)
}
