package main

import (
	"bufio"
	"bytes"
	"reflect"
	"testing"

	"github.com/cybozu-go/placemat"
)

func TestUnmarshalNetwork(t *testing.T) {
	cases := []struct {
		source string

		expected placemat.Network
	}{
		{
			source: `
kind: Network
name: net1
spec:
  addresses:
    - 10.0.0.1
    - 10.0.0.2
`,
			expected: placemat.Network{
				Name: "net1",
				Spec: placemat.NetworkSpec{
					Addresses: []string{"10.0.0.1", "10.0.0.2"},
				},
			},
		},
		{
			source: `

kind: Network
name: net2
`,
			expected: placemat.Network{
				Name: "net2",
				Spec: placemat.NetworkSpec{},
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
}

func TestUnmarshalNode(t *testing.T) {
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
    - name: boot
      size: 10GB
      recreatePolicy: Always
    - name: data
      size: 20GB
`,

			expected: placemat.Node{
				Name: "node1",
				Spec: placemat.NodeSpec{
					Interfaces: []string{"br0", "br1"},
					Volumes: []*placemat.VolumeSpec{
						{Name: "boot", Size: "10GB", RecreatePolicy: placemat.RecreateAlways},
						{Name: "data", Size: "20GB", RecreatePolicy: placemat.RecreateIfNotPresent},
					},
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
					Volumes:    []*placemat.VolumeSpec{},
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

	errorSources := []string{
		`kind: Node`,
	}
	for _, c := range errorSources {
		_, err := unmarshalNode([]byte(c))
		if err == nil {
			t.Error("err == nil, ", err)
		}
	}

}

func TestUnmarshalCluster(t *testing.T) {
	yaml := `
kind: Network
name: net1
---
kind: Node
name: node1
---
kind: Node
name: node2
`

	cluster, err := readYaml(bufio.NewReader(bytes.NewReader([]byte(yaml))))
	if err != nil {
		t.Error(err)
	}
	if len(cluster.Networks) != 1 {
		t.Error("len(cluster.Networks) != 2, ", len(cluster.Networks))
	}
	if len(cluster.Nodes) != 2 {
		t.Error("len(cluster.Nodes) != 2, ", len(cluster.Nodes))
	}

}
